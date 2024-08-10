package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"parallel/internal"
	"parallel/internal/structs"
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

	//ml1 := &structs.MainLocator{Sn: "main1"}
	ml2 := &structs.ExampleLocator{Sn: "main2"}
	ml3 := &structs.ExampleLocator{Sn: "main3"}
	life, err := internal.CreateSchedule(
		[]internal.BackgroundConfiguration{
			{
				BackgroundJobFunc:         internal.Test1,
				AppName:                   "api",
				BackgroundJobName:         "UpdateFeatureFlags1",
				BackgroundJobWaitDuration: 5 * time.Second,
				LifeCheckDuration:         3 * time.Second,
				//Locator:                   ml1,
			},
			{
				BackgroundJobFunc:         internal.Test2,
				AppName:                   "api",
				BackgroundJobName:         "UpdateFeatureFlags2",
				BackgroundJobWaitDuration: 5 * time.Second,
				LifeCheckDuration:         3 * time.Second,
				DependsOf: map[string]struct{}{
					"api.UpdateFeatureFlags1": {},
				},
				Locator: ml2,
			},
			{
				BackgroundJobFunc:         internal.Test3,
				AppName:                   "non_api",
				BackgroundJobName:         "UpdateFeatureFlags3",
				BackgroundJobWaitDuration: 5 * time.Second,
				LifeCheckDuration:         3 * time.Second,
				DependsOf: map[string]struct{}{
					"api.UpdateFeatureFlags1": {},
					"api.UpdateFeatureFlags2": {},
					//"e.UpdateFeatureFlags4":   {},
				},
				Locator: ml3,
			},
		},
	)
	if err != nil {
		log.Println(err)
		return
	}

	life.Run(ctx)

	<-sigCh
	cancel()

	log.Println(fmt.Sprintf("Alive %v", life.Alive()))
	life.Stop()
	log.Println(fmt.Sprintf("Alive %v", life.Alive()))

	logs := life.GetScheduleLogTime(time.DateTime)
	log.Println(logs)

	log.Println(runtime.NumGoroutine(), "in_main_end")
	log.Println(fmt.Sprintf("Alive %v", life.Alive()))
}
