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
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	life, err := internal.HandleSchedule(
		ctx,
		[]*internal.BackgroundConfiguration{
			{
				BackgroundJobFunc:           internal.BackgroundJob(internal.Test1),
				AppName:                     "api",
				BackgroundJobName:           "UpdateFeatureFlags1",
				BackgroundJobWaitDuration:   5 * time.Second,
				LifeCheckDuration:           1 * time.Second,
				MaxCheckerRestarts:          5,
				NumberOfMonitoredGoroutines: 4,
			},
			{
				BackgroundJobFunc:           internal.BackgroundJob(internal.Test2),
				AppName:                     "api",
				BackgroundJobName:           "UpdateFeatureFlags2",
				BackgroundJobWaitDuration:   5 * time.Second,
				LifeCheckDuration:           1 * time.Second,
				MaxCheckerRestarts:          5,
				NumberOfMonitoredGoroutines: 4,
			},
		},
	)

	if err != nil {
		log.Println("there was an error in the schedule")
		return
	}

	<-sigCh
	cancel()

	log.Println(fmt.Sprintf("Alive %v", life.Alive()))
	life.AwaitUntilAlive(1 * time.Second)
	log.Println(fmt.Sprintf("Alive %v", life.Alive()))

	gcCoont := runtime.NumGoroutine()
	log.Println(gcCoont)
}
