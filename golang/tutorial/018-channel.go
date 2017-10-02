package main

import (
	"fmt"
	"time"
)

var c chan string

func hehe(greet string) {
	fmt.Println(greet, " recv from channel: ", <- c)
}


func main() {
	c = make(chan string)

	go hehe("hello")
	go hehe("world")

	c <- "a"
	c <- "b"
	time.Sleep(time.Duration(1) * time.Second)
}
