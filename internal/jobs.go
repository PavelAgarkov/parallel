package internal

import (
	"context"
	"log"
	"time"
)

func Test1(ctx context.Context) error {
	//logger := log.Logger{}
	log.Println(111111)

	time.Sleep(1 * time.Second)
	panic("panic 1111")
	return nil
}

func Test2(ctx context.Context) error {
	//logger := log.Logger{}
	log.Println(222222)

	time.Sleep(1 * time.Second)
	panic("panic 2222")
	return nil
}
