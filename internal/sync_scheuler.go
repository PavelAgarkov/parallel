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

type BackgroundJob func(ctx context.Context) error

type BackgroundConfiguration struct {
	BackgroundJobFunc BackgroundJob
	AppName,
	BackgroundJobName string
	BackgroundJobWaitDuration,
	LifeCheckDuration time.Duration
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

func (ss *syncScheduler) runSchedule(importCtx context.Context, scheduleLife *ScheduleLife) {
	ctx, cancel := context.WithCancel(importCtx)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Println(fmt.Sprintf("syncScheduler.check() %s %s was Recovered. Error: %s",
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

				go ss.runJob(ctx, scheduleLife)
			}
		}
	}
}

func (ss *syncScheduler) runJob(ctx context.Context, scheduleLife *ScheduleLife) {
	defer func() {
		if r := recover(); r != nil {
			load := ss.getAliveGo()
			log.Println(fmt.Sprintf("load runJob %v", load))
			ss.decrementAliveGo()

			log.Println(fmt.Sprintf("syncScheduler.runJob() %s %s was Recovered. Error: %s", ss.config.AppName, ss.config.BackgroundJobName, r))
		}
	}()

	ss.incrementAliveGo()
	select {
	case <-ctx.Done():
		log.Println("context DONE background")
		ss.decrementAliveGo()
		return
	case <-time.After(1 * time.Second):
		for {
			select {
			case <-time.After(ss.config.BackgroundJobWaitDuration - 1*time.Second):
				select {
				case <-ctx.Done():
					log.Println("context DONE run 1")
					ss.decrementAliveGo()
					return
				default:
					key := ss.config.AppName + "." + ss.config.BackgroundJobName
					start := time.Now()
					err := ss.config.BackgroundJobFunc(ctx)
					end := time.Now()
					scheduleLife.setScheduleLogTime(start, end, key)
					if err != nil {
						log.Println(fmt.Sprintf("can't background for %s %s", ss.config.AppName, ss.config.BackgroundJobName))
					}
				}
			case <-ctx.Done():
				log.Println("context DONE run 2")
				ss.decrementAliveGo()
				return
			}
		}
	}
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
