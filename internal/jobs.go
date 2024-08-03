package internal

import (
	"context"
	"log"
	"time"
)

func Test1(ctx context.Context) error {
	log.Println(111111)

	time.Sleep(5 * time.Second)
	if ctx.Err() != nil {
		log.Println("ctx err Test1")
		return nil
	}
	//panic("panic 1111")
	return nil
}

func Test2(ctx context.Context) error {
	//logger := log.Logger{}
	log.Println(222222)

	time.Sleep(5 * time.Second)
	if ctx.Err() != nil {
		log.Println("ctx err Test2")
		return nil
	}
	//panic("panic 2222")
	return nil
}
