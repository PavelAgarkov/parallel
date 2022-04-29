package main

import "fmt"

func channel() {
	a := make(chan int)
	b := new(chan int)
	c := &a
	d := &b

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(c)
	fmt.Println(d)
}
