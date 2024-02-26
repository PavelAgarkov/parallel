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
	// debug with package
	//w := NewWait()
	//w.wait(6)
	//
	//sf := NewSelectFlag()
	//sf.Start(5, 10, 9)

	//channel()

	//bc := NewBC(5)
	//bc.Start(10)

	//var worker Worker
	//worker = func(fake string) {
	//	fmt.Printf("\n %v \n", fake)
	//}
	//
	//fixCountWOrker := NewFixWorker(4)
	//
	//fixCountWOrker.AddWorker(worker)
	//fixCountWOrker.AddWorker(worker)
	//
	//wg := sync.WaitGroup{}
	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	fixCountWOrker.Start()
	//}()
	//
	//fixCountWOrker.AddWorker(worker)
	//fixCountWOrker.AddWorker(worker)
	//fixCountWOrker.AddWorker(worker)
	//fixCountWOrker.AddWorker(worker)

	//wg.Wait()

	//ctx := context.Background()

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)
	// создает отдельную горутину
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

// 1. Параллельна обработка данных с ожиданием всех воркеров(указанное число обработчиков).
// 2. Параллельная обработка данных с ожиданием всех воркеров (число воркеров формируется динамически).
// 3. Асинхронное выполнение ветви без ожидания окончания.
// 4.
