package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

type syncScheduler struct {
	aliveGo int64
	config  BackgroundConfiguration
}

type ServiceLocator interface {
	ServiceName() string
}

type BackgroundJob func(ctx context.Context, locator ServiceLocator) error

type BackgroundConfiguration struct {
	BackgroundJobFunc         BackgroundJob
	AppName                   string
	BackgroundJobName         string
	BackgroundJobWaitDuration time.Duration
	LifeCheckDuration         time.Duration
	Locator                   ServiceLocator
	DependsOf                 map[string]struct{}
}

func newScheduler(config BackgroundConfiguration) *syncScheduler {
	return &syncScheduler{
		config: config,
	}
}

func (ss *syncScheduler) getAliveGo() int64 {
	return atomic.LoadInt64(&ss.aliveGo)
}

func (ss *syncScheduler) incrementAliveGo() {
	atomic.AddInt64(&ss.aliveGo, 1)
}

func (ss *syncScheduler) decrementAliveGo() {
	atomic.AddInt64(&ss.aliveGo, -1)
}

func (ss *syncScheduler) runSchedule(importCtx context.Context, scheduleLife *ScheduleLife, locator ServiceLocator) {
	ctx, cancel := context.WithCancel(importCtx)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Println(fmt.Sprintf("syncScheduler.runSchedule() %s %s was Recovered. Error: %s",
				ss.config.AppName,
				ss.config.BackgroundJobName,
				r.(string)),
			)
		}
	}()

	for {
		select {
		case <-importCtx.Done():
			log.Println("context DONE controller")
			return

		case <-time.After(ss.config.LifeCheckDuration):
			if ss.getAliveGo() == 0 {
				log.Println(runtime.NumGoroutine(), "in_check")

				go ss.runJob(ctx, scheduleLife, locator)
			}
		}
	}
}

func (ss *syncScheduler) runJob(ctx context.Context, scheduleLife *ScheduleLife, locator ServiceLocator) {
	ss.incrementAliveGo()

	defer func() {
		ss.decrementAliveGo()
	}()
	defer func() {
		if r := recover(); r != nil {
			log.Println(fmt.Sprintf("syncScheduler.runJob() %s %s Has Recovered. Error: %s", ss.config.AppName, ss.config.BackgroundJobName, r))
		}
	}()

dependsLoop:
	for {
		if len(ss.config.DependsOf) == 0 {
			break dependsLoop
		}
		select {
		case <-ctx.Done():
			log.Println("job context exit")
			return
		case <-time.After(100 * time.Millisecond):
			doneExpectedGoroutines := 0
			for k := range ss.config.DependsOf {
				logHit := scheduleLife.getScheduleLogTime(k)
				if logHit != nil {
					if logHit.lastEndOfExecution != nil {
						doneExpectedGoroutines++
					}
				}
			}
			if doneExpectedGoroutines == len(ss.config.DependsOf) {
				log.Println("the dependent task has been unblocked", fmt.Sprintf("%s.%s", ss.config.AppName, ss.config.BackgroundJobName))
				break dependsLoop
			}
		}
	}

	select {
	case <-ctx.Done():
		log.Println("job context exit")
		return
	case <-time.After(ss.config.BackgroundJobWaitDuration):
		key := ss.config.AppName + "." + ss.config.BackgroundJobName
		start := time.Now()
		err := ss.config.BackgroundJobFunc(ctx, locator)
		end := time.Now()
		scheduleLife.setScheduleLogTime(start, end, key)
		if err != nil {
			log.Println(fmt.Sprintf("can't background for %s %s", ss.config.AppName, ss.config.BackgroundJobName))
		}
		return
	}
	//case <-time.After(1 * time.Second):
	//select {
	//case <-time.After(ss.config.BackgroundJobWaitDuration - 1*time.Second):
	//	select {
	//	case <-ctx.Done():
	//		log.Println("context DONE run 1")
	//		return
	//	default:
	//		key := ss.config.AppName + "." + ss.config.BackgroundJobName
	//		start := time.Now()
	//		ss.incrementAliveGo()
	//		err := ss.config.BackgroundJobFunc(ctx)
	//		end := time.Now()
	//		scheduleLife.setScheduleLogTime(start, end, key)
	//		if err != nil {
	//			log.Println(fmt.Sprintf("can't background for %s %s", ss.config.AppName, ss.config.BackgroundJobName))
	//		}
	//		return
	//	}
	//}
}

func (ss *syncScheduler) toValidate() error {
	if ss.config.AppName == "" {
		return errors.New("app name must be filled in")
	}
	if ss.config.BackgroundJobName == "" {
		return errors.New("background job name must be filled in")
	}
	return nil
}
