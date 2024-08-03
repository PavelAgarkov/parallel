package internal

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

type ScheduleLog map[string]scheduleLog

type scheduleLog struct {
	lastStartOfExecution time.Time
}

type ScheduleLife struct {
	mu  sync.Mutex
	run bool

	lifeChecker      *sync.Map
	listOfSchedulers []*syncScheduler
	scheduleLog      ScheduleLog
}

func CreateScheduleLife(scheduleConfigList []BackgroundConfiguration) (*ScheduleLife, error) {
	sl := &ScheduleLife{}
	for _, backgroundConfig := range scheduleConfigList {
		scheduler := newScheduler(backgroundConfig)
		err := scheduler.toValidate()
		if err != nil {
			return nil, err
		}
		sl.listOfSchedulers = append(sl.listOfSchedulers, scheduler)
	}
	sl.lifeChecker = &sync.Map{}
	sl.scheduleLog = make(ScheduleLog)
	return sl, nil
}

func (l *ScheduleLife) isRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.run
}

func (l *ScheduleLife) toRun() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.run = true
}

func (l *ScheduleLife) RunSchedule(ctx context.Context) {
	if l.isRunning() {
		return
	}
	l.toRun()
	for _, scheduler := range l.listOfSchedulers {
		l.lifeChecker.Store(scheduler.config.BackgroundJobName, scheduler)
		key := scheduler.config.AppName + "." + scheduler.config.BackgroundJobName
		go scheduler.runSchedule(ctx)
		l.scheduleLog[key] = scheduleLog{
			lastStartOfExecution: time.Now(),
		}
	}
}

func (l *ScheduleLife) GetScheduleLog() ScheduleLog {
	return l.scheduleLog
}

func (l *ScheduleLife) StopSchedule() {
	if l.isRunning() {
		l.awaitUntilAlive(1 * time.Second)
	}
}

func (l *ScheduleLife) Alive() bool {
	numberAliveSchedulers := int64(0)
	l.lifeChecker.Range(func(key, value any) bool {
		scheduler, ok := value.(*syncScheduler)
		if !ok {
			return false
		}
		load := scheduler.getAliveGo()

		if load != 0 {
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

func (l *ScheduleLife) awaitUntilAlive(aliveTimer time.Duration) bool {
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

				load := scheduler.getAliveGo()
				log.Println(fmt.Sprintf("load %v", load))
				gcCoont := runtime.NumGoroutine()
				log.Println(gcCoont, "in_await")

				if load != 0 {
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
