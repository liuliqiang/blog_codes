package main

import (
	"time"
	"fmt"
)

func hehe(greet string) {
	fmt.Println("begin: ", greet)
	time.Sleep(time.Duration(1) * time.Second)
	fmt.Println("end: ", greet)
}


func main() {
	go hehe("hello")
	go hehe("world")
	fmt.Println("main routine")
	time.Sleep(time.Duration(2) * time.Second)
	fmt.Println("main again")
}
