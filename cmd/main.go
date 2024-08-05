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
	//f1, _ := os.Create("trace.out")
	//trace.Start(f1)
	//defer trace.Stop()
	//
	//// Start profiling
	//f, err := os.Create("myprogram.prof")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//
	//}
	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()

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
				BackgroundJobFunc:         internal.Test1,
				AppName:                   "api",
				BackgroundJobName:         "UpdateFeatureFlags1",
				BackgroundJobWaitDuration: 5 * time.Second,
				LifeCheckDuration:         10 * time.Second,
			},
			{
				BackgroundJobFunc:         internal.Test2,
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
	life.AwaitStopSchedule()
	log.Println(fmt.Sprintf("Alive %v", life.Alive()))

	logs := life.GetScheduleLogTime(time.DateTime)
	log.Println(logs)

	log.Println(runtime.NumGoroutine(), "in_main_end")
	log.Println(fmt.Sprintf("Alive %v", life.Alive()))
}
