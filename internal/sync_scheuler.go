package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"

	"sync/atomic"
	"time"
)

type SyncScheduler struct {
	number                      int64
	restart                     int64
	maxRestarts                 int64
	numberOfMonitoredGoroutines int64
}

type BackgroundConfiguration struct {
	Background                  func(ctx context.Context) error
	AppName                     string
	BackgroundName              string
	BackgroundSleep             time.Duration
	ControllerSleep             time.Duration
	MaxControllerRestarts       int64
	NumberOfMonitoredGoroutines int64
}

func newScheduler(maxRestarts, numberOfMonitoredGoroutines int64) (*SyncScheduler, error) {
	if maxRestarts < 1 || numberOfMonitoredGoroutines < 1 {
		return nil, errors.New("maxRestarts and numberOfMonitoredGoroutines must be change from 0")
	}
	return &SyncScheduler{
		maxRestarts:                 maxRestarts,
		numberOfMonitoredGoroutines: numberOfMonitoredGoroutines,
	}, nil
}

func (su *SyncScheduler) run(
	ctx context.Context,
	background func(ctx context.Context) error,
	appName string,
	backgroundName string,
	backgroundSleep time.Duration,
	rate chan<- struct{},
	wg *sync.WaitGroup,
	atom int64,
) {
	log.Println("Run function started")

	atomic.AddInt64(&atom, 1)

	defer wg.Done()

	defer func() {
		if r := recover(); r != nil {
			rate <- struct{}{}
			log.Println(fmt.Sprintf("sync %s %s Recovered. Error: %s", appName, backgroundName, r))
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("context DONE background")
		return

	case <-time.After(backgroundSleep):
		for {
			select {
			case <-time.After(backgroundSleep):
				select {
				case <-ctx.Done():
					log.Println("context DONE run 1")
					return
				default:
					err := background(ctx)
					if err != nil {
						log.Println(fmt.Sprintf("can't background for %s %s", appName, backgroundName))
					}
				}
			case <-ctx.Done():
				log.Println("context DONE run 2")
				return
			}
		}
	}
}

func (su *SyncScheduler) Controller(
	ctx context.Context,
	background func(ctx context.Context) error,
	appName string,
	backgroundName string,
	backgroundSleep time.Duration,
	controllerSleep time.Duration,
	wait chan int,
) {
	go su.controller(ctx, background, appName, backgroundName, backgroundSleep, controllerSleep, wait)
}

func (su *SyncScheduler) controller(
	importCtx context.Context,
	background func(ctx context.Context) error,
	appName string,
	backgroundName string,
	backgroundSleep time.Duration,
	controllerSleep time.Duration,
	wait chan int,
) {
	var wg sync.WaitGroup // Create a WaitGroup to wait for all goroutines to finish
	defer log.Println("exit 1")
	defer wg.Wait() // Wait for all launched goroutines to complete
	defer log.Println("exit 2")

	ctx, cancel := context.WithCancel(importCtx)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Println(fmt.Sprintf("Controller %s %s Recovered. Error: %s", appName, backgroundName, r.(string)))
			if atomic.LoadInt64(&su.restart) < su.maxRestarts {
				atomic.AddInt64(&su.restart, 1)
				go su.controller(ctx, background, appName, backgroundName, backgroundSleep, controllerSleep, wait)
			} else {
				log.Println(fmt.Sprintf("the maximum number of allowed restarts has been exceeded %s %s : %s", appName, backgroundName, r.(string)))
			}
		}
	}()

	rate := make(chan struct{}, su.numberOfMonitoredGoroutines)

	for i := int64(1); i <= su.numberOfMonitoredGoroutines; i++ {
		rate <- struct{}{}
	}
	defer close(rate)

	atom := int64(0)

	ticker2 := time.NewTicker(controllerSleep)
	for {
		select {
		case <-importCtx.Done():
			log.Println("context DONE controller")
			return

		case <-rate:
			atomic.AddInt64(&atom, -1)

		case <-ticker2.C:
			load := atomic.LoadInt64(&atom)
			if load < 0 {
				diff := 0 - load

				for i := diff; i > 0; i-- {
					gcCoont := runtime.NumGoroutine()
					log.Println(gcCoont)

					wg.Add(1)

					go func() {
						atomic.AddInt64(&atom, 1)

						defer wg.Done()

						defer func() {
							if r := recover(); r != nil {
								rate <- struct{}{}
								log.Println(fmt.Sprintf("sync %s %s Recovered. Error: %s", appName, backgroundName, r))
							}
						}()

						select {
						case <-ctx.Done():
							log.Println("context DONE background")
							return

						case <-time.After(1 * time.Second):
							for {
								select {
								case <-time.After(backgroundSleep - 1*time.Second):
									select {
									case <-ctx.Done():
										log.Println("context DONE run 1")
										return
									default:
										err := background(ctx)
										if err != nil {
											log.Println(fmt.Sprintf("can't background for %s %s", appName, backgroundName))
										}
									}
								case <-ctx.Done():
									log.Println("context DONE run 2")
									return
								}
							}
						}
					}()
				}
			}
		}
	}
}

func HandleSchedule(ctx context.Context, wait chan int, scheduleConfigList []*BackgroundConfiguration) error {
	for _, background := range scheduleConfigList {
		scheduler, err := newScheduler(background.MaxControllerRestarts, background.NumberOfMonitoredGoroutines)
		if err != nil {
			return err
		}

		scheduler.Controller(
			ctx,
			background.Background,
			background.AppName,
			background.BackgroundName,
			background.BackgroundSleep,
			background.ControllerSleep,
			wait,
		)
	}
	return nil
}
