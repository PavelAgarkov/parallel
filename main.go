package main

import (
	"fmt"
	"sync"
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

	var worker Worker
	worker = func(fake string) {
		fmt.Printf("\n %v \n", fake)
	}

	fixCountWOrker := NewFixWorker(4)

	fixCountWOrker.AddWorker(worker)
	fixCountWOrker.AddWorker(worker)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fixCountWOrker.Start()
	}()

	fixCountWOrker.AddWorker(worker)
	fixCountWOrker.AddWorker(worker)
	fixCountWOrker.AddWorker(worker)
	fixCountWOrker.AddWorker(worker)

	//wg.Wait()

}

// 1. Параллельна обработка данных с ожиданием всех воркеров(указанное число обработчиков).
// 2. Параллельная обработка данных с ожиданием всех воркеров (число воркеров формируется динамически).
// 3. Асинхронное выполнение ветви без ожидания окончания.
// 4.
