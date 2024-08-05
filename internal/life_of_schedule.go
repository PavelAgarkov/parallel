package internal

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type ScheduleLog map[string]scheduleLog

type scheduleLog struct {
	lastStartOfExecution time.Time
	lastEndOfExecution   time.Time
}

type ScheduleLife struct {
	mu  sync.Mutex
	run bool

	lifeChecker      *sync.Map
	listOfSchedulers []*syncScheduler

	logMu       sync.Mutex
	scheduleLog ScheduleLog
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

func (l *ScheduleLife) setScheduleLogTime(start, end time.Time, name string) {
	l.logMu.Lock()
	defer l.logMu.Unlock()
	l.scheduleLog[name] = scheduleLog{
		lastStartOfExecution: start,
		lastEndOfExecution:   end,
	}
}

func (l *ScheduleLife) GetScheduleLogTime(format string) map[string]string {
	l.logMu.Lock()
	defer l.logMu.Unlock()
	newLog := make(map[string]string)
	for k, v := range l.scheduleLog {
		keyStart := k + "." + "[start]"
		keyEnd := k + "." + "[stop]"
		newLog[keyStart] = v.lastStartOfExecution.Format(format)
		newLog[keyEnd] = v.lastEndOfExecution.Format(format)
	}
	return newLog
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
		go scheduler.runSchedule(ctx, l)
	}
}

func (l *ScheduleLife) AwaitStopSchedule() {
	if l.isRunning() {
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

	return numberAliveSchedulers
}

func (l *ScheduleLife) awaitUntilAlive(aliveTimer time.Duration) int64 {
	for {
		select {
		case <-time.After(aliveTimer):
			numberAliveSchedulers := int64(0)
			l.lifeChecker.Range(func(key, value any) bool {
				scheduler, ok := value.(*syncScheduler)
				//log.Println(key.(string))
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
