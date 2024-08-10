package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type ScheduleLife struct {
	mu  sync.Mutex
	run bool

	lifeChecker      *sync.Map
	listOfSchedulers []*syncScheduler

	logMu       sync.Mutex
	scheduleLog map[string]*scheduleLog

	cancelContext context.CancelFunc
}

type scheduleLog struct {
	lastStartOfExecution *time.Time
	lastEndOfExecution   *time.Time
}

func CreateSchedule(scheduleConfigList []BackgroundConfiguration) (*ScheduleLife, error) {
	sl := &ScheduleLife{}
	for _, backgroundConfig := range scheduleConfigList {
		scheduler := newScheduler(backgroundConfig)
		err := scheduler.toValidate()
		if err != nil {
			return nil, err
		}
		sl.listOfSchedulers = append(sl.listOfSchedulers, scheduler)
	}
	err := sl.toValidate()
	if err != nil {
		return nil, err
	}

	sl.lifeChecker = &sync.Map{}
	lo := make(map[string]*scheduleLog)
	sl.scheduleLog = lo
	return sl, nil
}

func (l *ScheduleLife) GetScheduleLogTime(format string) map[string]string {
	newLog := make(map[string]string)
	if l.isRunning() {
		for k, v := range l.scheduleLog {
			keyStart := k + "." + "[start]"
			keyEnd := k + "." + "[stop]"
			newLog[keyStart] = v.lastStartOfExecution.Format(format)
			newLog[keyEnd] = v.lastEndOfExecution.Format(format)
		}
	}
	return newLog
}

func (l *ScheduleLife) Run(ctx context.Context) {
	if l.isRunning() {
		return
	}
	runctx, cancel := context.WithCancel(ctx)
	l.cancelContext = cancel
	l.toRun()
	for _, scheduler := range l.listOfSchedulers {
		l.lifeChecker.Store(scheduler.config.BackgroundJobName, scheduler)
		go scheduler.runSchedule(runctx, l, scheduler.config.Locator)
	}
}

func (l *ScheduleLife) Stop() {
	if l.isRunning() {
		l.cancelContext()
		alive := l.awaitUntilAlive(1 * time.Second)
		if alive == 0 {
			log.Println(alive, "await alive")
			for _, v := range l.listOfSchedulers {
				atomic.StoreInt64(&v.aliveGo, 0)
			}
		}
	}
}

func (l *ScheduleLife) Alive() int64 {
	numberAliveSchedulers := int64(0)
	if l.isRunning() {
		l.lifeChecker.Range(func(key, value any) bool {
			scheduler, ok := value.(*syncScheduler)
			if !ok {
				return false
			}
			load := scheduler.getAliveGo()

			if load > 0 {
				numberAliveSchedulers++
			}
			return true
		})
	}

	return numberAliveSchedulers

}

func (l *ScheduleLife) awaitUntilAlive(aliveTimer time.Duration) int64 {
	for {
		select {
		case <-time.After(aliveTimer):
			numberAliveSchedulers := int64(0)
			l.lifeChecker.Range(func(key, value any) bool {
				scheduler, ok := value.(*syncScheduler)
				if !ok {
					return false
				}

				load := scheduler.getAliveGo()
				if load > 0 {
					numberAliveSchedulers++
				}
				return true
			})

			if numberAliveSchedulers < 1 {
				return numberAliveSchedulers
			}
		}
	}
}

func (l *ScheduleLife) setScheduleLogTime(start, end time.Time, name string) {
	l.logMu.Lock()
	defer l.logMu.Unlock()
	sl := &scheduleLog{
		lastStartOfExecution: &start,
		lastEndOfExecution:   &end,
	}
	l.scheduleLog[name] = sl
}

func (l *ScheduleLife) getScheduleLogTime(key string) *scheduleLog {
	l.logMu.Lock()
	defer l.logMu.Unlock()
	return l.scheduleLog[key]
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

func (l *ScheduleLife) toValidate() error {
	deps := make(map[string]struct{})
	for _, schedule := range l.listOfSchedulers {
		for k := range schedule.config.DependsOf {
			deps[k] = struct{}{}
		}
	}
	if len(deps) == 0 {
		return nil
	}

	for s := range deps {
		ok := func(s string) bool {
			for _, schedule := range l.listOfSchedulers {
				key := schedule.config.AppName + "." + schedule.config.BackgroundJobName
				if s == key {
					return true
				}
			}
			return false
		}(s)

		if !ok {
			return errors.New(fmt.Sprintf("%v key not found in passed jobs", s))
		}
	}

	return nil
}
