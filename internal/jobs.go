package internal

import (
	"context"
	"log"
	"parallel/internal/structs"
	"time"
)

func Test1(ctx context.Context, locator ServiceLocator) error {
	log.Println(111111)

	time.Sleep(1 * time.Second)
	if ctx.Err() != nil {
		log.Println("ctx err Test1")
		return nil
	}
	//panic("panic 1111")
	//ml := locator.(*structs.MainLocator)
	log.Println("end Test1")
	return nil
}

func Test2(ctx context.Context, locator ServiceLocator) error {
	//logger := log.Logger{}
	log.Println(222222)

	time.Sleep(1 * time.Second)
	if ctx.Err() != nil {
		log.Println("ctx err Test2")
		return nil
	}
	//panic("panic 2222")
	ml := locator.(*structs.ExampleLocator)
	log.Println("end Test2", ml.Sn)
	return nil
}

func Test3(ctx context.Context, locator ServiceLocator) error {
	//logger := log.Logger{}
	log.Println(333333)

	time.Sleep(1 * time.Second)
	if ctx.Err() != nil {
		log.Println("ctx err Test3")
		return nil
	}
	//panic("panic 333")
	ml := locator.(*structs.ExampleLocator)
	log.Println("end Test3", ml.Sn)
	return nil
}
