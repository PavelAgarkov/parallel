package main

import "fmt"

//			---------------> направление потока(последовательная работа)
//			-------------		-------------		 -------------
//			|transmitter|  --->	|  buffchan	|	---> | receiver  |
//			-------------		-------------		 -------------
//				  |									        |
//				  V											V
//				default									 default

type BuffChan struct {
	numbers chan byte
}

func NewBC(capacity byte) *BuffChan {
	return &BuffChan{
		numbers: make(chan byte, capacity),
	}
}

func (bc *BuffChan) Start(counter byte) {
	bc.transmitter(counter)
	bc.receiver(counter + 5)
}

// цикл ограничивает число записей в буферизованный канал, канал имеет 5 мест до заполнения,
// после чего select пишет в дефолтный конвейер пока не закончатся витки
func (bc *BuffChan) transmitter(counter byte) {
	var i byte
	for i = 0; i < counter; i++ {
		select {
		case bc.numbers <- i:
			fmt.Println("insert in channel", i)
		default:
			fmt.Println("not enough space for", i)
		}
	}
}

// цикл читает ровно то, что успело записаться в последовательном выполнении функции
// трансмиттер(получился синхронный буфер или очередь). читает в порядке очереди, затем когда канал свободен
// select переключает на дефолтный конвейер
func (bc *BuffChan) receiver(counter byte) {
	var i byte
	for i = 0; i < counter; i++ {
		select {
		case num := <-bc.numbers:
			fmt.Println(num)
		default:
			fmt.Println("nothing more to be done!")
			break
		}
	}
}
