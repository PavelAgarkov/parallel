package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"parallel/internal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	gcCoont := runtime.NumGoroutine()
	log.Println(gcCoont, "in_main_start")
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	gcCoont = runtime.NumGoroutine()
	log.Println(gcCoont, "in_main_middle")
	life, err := internal.CreateScheduleLife(
		[]internal.BackgroundConfiguration{
			{
				BackgroundJobFunc:         internal.BackgroundJob(internal.Test1),
				AppName:                   "api",
				BackgroundJobName:         "UpdateFeatureFlags1",
				BackgroundJobWaitDuration: 5 * time.Second,
				LifeCheckDuration:         10 * time.Second,
			},
			{
				BackgroundJobFunc:         internal.BackgroundJob(internal.Test2),
				AppName:                   "api",
				BackgroundJobName:         "UpdateFeatureFlags2",
				BackgroundJobWaitDuration: 5 * time.Second,
				LifeCheckDuration:         10 * time.Second,
			},
		},
	)
	if err != nil {
		log.Println("there was an error in the schedule")
		return
	}

	life.RunSchedule(ctx)

	<-sigCh
	cancel()

	log.Println(fmt.Sprintf("Alive %v", life.Alive()))
	life.StopSchedule()
	log.Println(fmt.Sprintf("Alive %v", life.Alive()))

	logs := life.GetScheduleLog()
	log.Println(logs)

	gcCoont = runtime.NumGoroutine()
	log.Println(gcCoont, "in_main_end")
}
