package main

import (
	"context"
	"sync"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		println("goroutine 1 start")
		<-ctx.Done()
		println("goroutine 1 done")
	}()

	go func() {
		defer wg.Done()

		println("goroutine 2 start")
		<-ctx.Done()
		println("goroutine 2 done")
	}()

	go func() {
		defer wg.Done()

		println("goroutine 3 start")
		time.Sleep(1 * time.Second)
		cancel()
		println("goroutine 3 fail, canceled")
	}()

	wg.Wait()
	println("main done")
}
