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

type syncScheduler struct {
	restart, maxCheckerRestarts, numberOfMonitoredGoroutines, activeGoroutines int64
	wg                                                                         sync.WaitGroup
}

type BackgroundConfiguration struct {
	BackgroundJobFunc                               func(ctx context.Context) error
	AppName, BackgroundJobName                      string
	BackgroundJobWaitDuration, LifeCheckDuration    time.Duration
	MaxCheckerRestarts, NumberOfMonitoredGoroutines int64
}

func newScheduler(maxCheckerRestarts, numberOfMonitoredGoroutines int64) *syncScheduler {
	return &syncScheduler{
		maxCheckerRestarts:          maxCheckerRestarts,
		numberOfMonitoredGoroutines: numberOfMonitoredGoroutines,
	}
}

func (su *syncScheduler) toControl(
	ctx context.Context,
	background func(ctx context.Context) error,
	appName, backgroundName string,
	backgroundSleep, lifeCheckDuration time.Duration,
) {
	go su.check(ctx, background, appName, backgroundName, backgroundSleep, lifeCheckDuration)
}

func (su *syncScheduler) check(
	importCtx context.Context,
	background func(ctx context.Context) error,
	appName, backgroundName string,
	backgroundSleep, lifeCheckDuration time.Duration,
) {
	rate := make(chan struct{}, su.numberOfMonitoredGoroutines)
	for i := int64(1); i <= su.numberOfMonitoredGoroutines; i++ {
		rate <- struct{}{}
	}
	defer close(rate)

	defer func() {
		su.wg.Wait()
	}()
	ctx, cancel := context.WithCancel(importCtx)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Println(fmt.Sprintf("Controller %s %s Recovered. Error: %s", appName, backgroundName, r.(string)))
			if atomic.LoadInt64(&su.restart) < su.maxCheckerRestarts {
				atomic.AddInt64(&su.restart, 1)
				go su.check(ctx, background, appName, backgroundName, backgroundSleep, lifeCheckDuration)
			} else {
				log.Println(fmt.Sprintf("the maximum number of allowed restarts has been exceeded %s %s : %s", appName, backgroundName, r.(string)))
			}
		}
	}()

	for {
		select {
		case <-importCtx.Done():
			log.Println("context DONE controller")
			return

		case <-rate:
			atomic.AddInt64(&su.activeGoroutines, -1)

		case <-time.After(lifeCheckDuration):
			load := atomic.LoadInt64(&su.activeGoroutines)
			if load < 0 {
				diff := 0 - load
				for i := diff; i > 0; i-- {
					gcCoont := runtime.NumGoroutine()
					log.Println(gcCoont)

					su.wg.Add(1)
					// можно вынести код из горутины и запустить тут
					go su.runJob(ctx, background, appName, backgroundName, backgroundSleep, rate)
				}
			}
		}
	}
}

func (su *syncScheduler) runJob(
	ctx context.Context,
	background func(ctx context.Context) error,
	appName, backgroundName string,
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
		atomic.AddInt64(&su.activeGoroutines, -1)
		return
	case <-time.After(1 * time.Second):
		for {
			select {
			case <-time.After(backgroundSleep - 1*time.Second):
				select {
				case <-ctx.Done():
					log.Println("context DONE run 1")
					atomic.AddInt64(&su.activeGoroutines, -1)
					return
				default:
					err := background(ctx)
					if err != nil {
						log.Println(fmt.Sprintf("can't background for %s %s", appName, backgroundName))
					}
				}
			case <-ctx.Done():
				log.Println("context DONE run 2")
				atomic.AddInt64(&su.activeGoroutines, -1)
				return
			}
		}
	}
}

type Life struct {
	lifeChecker *sync.Map
}

func HandleSchedule(ctx context.Context, scheduleConfigList []*BackgroundConfiguration) (*Life, error) {
	lifeMap := sync.Map{}
	for _, background := range scheduleConfigList {
		scheduler := newScheduler(background.MaxCheckerRestarts, background.NumberOfMonitoredGoroutines)
		err := scheduler.toValidate()
		if err != nil {
			return nil, err
		}

		scheduler.toControl(
			ctx,
			background.BackgroundJobFunc,
			background.AppName,
			background.BackgroundJobName,
			background.BackgroundJobWaitDuration,
			background.LifeCheckDuration,
		)

		lifeMap.Store(background.BackgroundJobName, scheduler)
	}

	return &Life{lifeChecker: &lifeMap}, nil
}

func (l *Life) Alive() bool {
	numberAliveSchedulers := int64(0)
	l.lifeChecker.Range(func(key, value any) bool {
		scheduler, ok := value.(*syncScheduler)
		if !ok {
			return false
		}
		load := atomic.LoadInt64(&scheduler.activeGoroutines)

		if load != -scheduler.numberOfMonitoredGoroutines {
			numberAliveSchedulers++
			log.Println(fmt.Sprintf("numberAlive %v", numberAliveSchedulers))
		}
		return true
	})

	if numberAliveSchedulers > 0 {
		return true
	}

	return false
}

func (l *Life) AwaitUntilAlive(aliveTimer time.Duration) bool {
	for {
		select {
		case <-time.After(aliveTimer):
			numberAliveSchedulers := int64(0)
			l.lifeChecker.Range(func(key, value any) bool {
				scheduler, ok := value.(*syncScheduler)
				log.Println(key.(string))
				if !ok {
					return false
				}
				load := atomic.LoadInt64(&scheduler.activeGoroutines)

				if load != -scheduler.numberOfMonitoredGoroutines {
					numberAliveSchedulers++
				}
				return true
			})

			if numberAliveSchedulers == 0 {
				log.Println(fmt.Sprintf("numberAlive zero %v", 0))
				return true
			}
			log.Println(fmt.Sprintf("numberAlive %v", numberAliveSchedulers))
		}
	}
}

func (su *syncScheduler) toValidate() error {
	if su.maxCheckerRestarts < 1 || su.numberOfMonitoredGoroutines < 1 {
		return errors.New("maxRestarts and numberOfMonitoredGoroutines must be change from 0")
	}
	return nil
}
