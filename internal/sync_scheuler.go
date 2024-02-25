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
	number, restart, maxRestarts, numberOfMonitoredGoroutines, activeGoroutines int64
	wg                                                                          sync.WaitGroup
}

type BackgroundConfiguration struct {
	Background                                         func(ctx context.Context) error
	AppName, BackgroundName                            string
	BackgroundSleep, ControllerSleep                   time.Duration
	MaxControllerRestarts, NumberOfMonitoredGoroutines int64
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

func (su *SyncScheduler) controller(
	ctx context.Context,
	background func(ctx context.Context) error,
	appName string,
	backgroundName string,
	backgroundSleep time.Duration,
	controllerSleep time.Duration,
) {
	go su.runner(ctx, background, appName, backgroundName, backgroundSleep, controllerSleep)
}

func (su *SyncScheduler) runner(
	importCtx context.Context,
	background func(ctx context.Context) error,
	appName string,
	backgroundName string,
	backgroundSleep time.Duration,
	controllerSleep time.Duration,
) {
	defer su.wg.Wait()

	ctx, cancel := context.WithCancel(importCtx)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Println(fmt.Sprintf("Controller %s %s Recovered. Error: %s", appName, backgroundName, r.(string)))
			if atomic.LoadInt64(&su.restart) < su.maxRestarts {
				atomic.AddInt64(&su.restart, 1)
				go su.runner(ctx, background, appName, backgroundName, backgroundSleep, controllerSleep)
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

	for {
		select {
		case <-importCtx.Done():
			log.Println("context DONE controller")
			return

		case <-rate:
			atomic.AddInt64(&su.activeGoroutines, -1)

		case <-time.After(controllerSleep):
			load := atomic.LoadInt64(&su.activeGoroutines)
			if load < 0 {
				diff := 0 - load
				for i := diff; i > 0; i-- {
					gcCoont := runtime.NumGoroutine()
					log.Println(gcCoont)

					su.wg.Add(1)
					// можно вынести код из горутины и запустить тут
					go su.run(ctx, background, appName, backgroundName, backgroundSleep, rate)
				}
			}
		}
	}
}

func (su *SyncScheduler) run(
	ctx context.Context,
	background func(ctx context.Context) error,
	appName string,
	backgroundName string,
	backgroundSleep time.Duration,
	rate chan<- struct{},
) {
	defer su.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			rate <- struct{}{}
			log.Println(fmt.Sprintf("sync %s %s Recovered. Error: %s", appName, backgroundName, r))
		}
	}()

	atomic.AddInt64(&su.activeGoroutines, 1)
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
}

func HandleSchedule(ctx context.Context, scheduleConfigList []*BackgroundConfiguration) error {
	for _, background := range scheduleConfigList {
		scheduler, err := newScheduler(background.MaxControllerRestarts, background.NumberOfMonitoredGoroutines)
		if err != nil {
			return err
		}

		scheduler.controller(
			ctx,
			background.Background,
			background.AppName,
			background.BackgroundName,
			background.BackgroundSleep,
			background.ControllerSleep,
		)
	}
	return nil
}
