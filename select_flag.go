package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type selectFlag struct{}

func NewSelectFlag() *selectFlag {
	return &selectFlag{}
}

func (s *selectFlag) generate(max, min int, crtNumber chan int, end chan bool) {
	for {
		fmt.Println("start select")
		random := rand.Intn(max-min) + min

		select {
		// тут пишется случайное число в канал
		// на каждый виток цикла новое число
		// следующее значение пишется в канал только после того, как было прочитано предыдущее
		// до тех пор блокируется ветка select пока не будет прочитано на другой стороне конвеера
		case crtNumber <- random:
			fmt.Printf("read from crtNumber %v\n", random)

		//срабатывает через 4 секунды ожидания
		case <-time.After(4 * time.Second):
			fmt.Println("\ntime.After()\n")

		// срабатывает от получения значения через канал
		// return закрывает горутину
		case <-end:
			fmt.Println("channel off")
			close(end)
			return
		}
	}
}

func (s *selectFlag) Start(jobSleep, max, min int) {
	rand.Seed(time.Now().Unix())

	crtNumber := make(chan int)
	end := make(chan bool)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		s.generate(max, min, crtNumber, end)
	}()

	defer wg.Wait()

	// читает из канала данныезаписанные в цикле select case crtNumber <- random
	for i := min; i < max; i++ {
		fmt.Printf("return  %d\n", <-crtNumber)
	}

	// бесконечный цикл, т.к. не ограничено число обращений к каналу для чтения
	//for range crtNumber {
	//	fmt.Printf("return  %d\n", <-crtNumber)
	//}

	time.Sleep(time.Duration(jobSleep) * time.Second)
	fmt.Println("time exit \n")
	end <- true
	// если не подождать, то case <-end из select не успеет отработать
	//time.Sleep(time.Duration(waitEndChannelSleep) * time.Millisecond)
	// решено через wg
}
