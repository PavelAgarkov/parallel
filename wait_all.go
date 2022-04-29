package main

import (
	"fmt"
	"sync"
)

type waitAll struct {
	wg sync.WaitGroup
}

func (w *waitAll) wait(count int8) {
	var i int8

	for i = 0; i < count; i++ {
		w.wg.Add(1)

		go func(x int8) {
			fmt.Printf("current value !!%d!! \n", x)
			w.wg.Done()
		}(i)
	}

	fmt.Printf("%#v\n", w.wg)
	w.wg.Wait()
	fmt.Println("End\n")
}

func NewWait() *waitAll {
	return &waitAll{}
}
