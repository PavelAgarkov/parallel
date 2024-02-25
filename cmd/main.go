package main

import (
	"context"
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
	// создает отельную горутину
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	err := internal.HandleSchedule(
		ctx,
		[]*internal.BackgroundConfiguration{
			{
				Background:                  internal.Test1,
				AppName:                     "api",
				BackgroundName:              "UpdateFeatureFlags",
				BackgroundSleep:             5 * time.Second,
				ControllerSleep:             5 * time.Second,
				MaxControllerRestarts:       5,
				NumberOfMonitoredGoroutines: 4,
			},
			{
				Background:                  internal.Test2,
				AppName:                     "api",
				BackgroundName:              "UpdateFeatureFlags",
				BackgroundSleep:             5 * time.Second,
				ControllerSleep:             5 * time.Second,
				MaxControllerRestarts:       5,
				NumberOfMonitoredGoroutines: 4,
			},
		},
	)

	if err != nil {
		log.Println("there was an error in the schedule")
		return
	}

	<-sigCh
	log.Println("stop from signal")
	cancel()
	//close(sigCh)
	gcCoont := runtime.NumGoroutine()
	log.Println(gcCoont)
	time.Sleep(1 * time.Second)
	gcCoont = runtime.NumGoroutine()
	log.Println(gcCoont)
}

// 1. Параллельна обработка данных с ожиданием всех воркеров(указанное число обработчиков).
// 2. Параллельная обработка данных с ожиданием всех воркеров (число воркеров формируется динамически).
// 3. Асинхронное выполнение ветви без ожидания окончания.
// 4.
