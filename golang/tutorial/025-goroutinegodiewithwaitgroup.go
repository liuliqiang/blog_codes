package main

import (
	"fmt"
	"sync"
)

func wgTest(wg *sync.WaitGroup) {
	// make sure called before return
	defer wg.Done()
	fmt.Println("In test")
}

func main() {
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)  // make sure called before goroutine run
		go wgTest(&wg)
	}

	wg.Wait()
	fmt.Println("All Done!")
}


