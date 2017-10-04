package main

import (
	"fmt"
	"time"
	"runtime"
)

var c chan string

func hehe(greet string) {
	fmt.Println(greet, " recv from channel: ", <- c)
}


func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	c = make(chan string)

	go hehe("hello")
	go hehe("world")

	c <- "a"
	c <- "b"

	time.Sleep(time.Duration(1) * time.Second)
}
