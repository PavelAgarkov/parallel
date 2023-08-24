package main

import (
	"fmt"
	"time"
)

type FixWorker struct {
	semaphore chan Worker
}

type Worker func(fake string)

func NewFixWorker(semaphoreNumber byte) *FixWorker {
	return &FixWorker{
		semaphore: make(chan Worker, semaphoreNumber),
	}
}

func (f *FixWorker) Start() {

	//wg := sync.WaitGroup{}
	//wg.Add(1)
	//
	//go func(semaphore chan Worker) {
	//	defer wg.Done()
	//for worker := range f.semaphore {
	for {
		//worker, ok := <- semaphore
		//if ok {
		//	go func(Worker func(fake string)) {
		//		worker("work")
		//	}(worker)
		//}
		select {
		case worker := <-f.semaphore:
			go func(Worker func(fake string)) {
				worker("work")
			}(worker)
		case <-time.After(1 * time.Second):
			fmt.Println("wait")
		}
	}
	//}
	//}(f.semaphore)
	//
	//wg.Wait()
}

func (f *FixWorker) AddWorker(worker Worker) bool {
	if f.semaphore != nil {
		f.semaphore <- worker
		return true
	}

	return false
}
