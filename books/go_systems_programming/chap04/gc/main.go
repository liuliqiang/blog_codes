package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	var mem runtime.MemStats
	printStats(mem)

	for i := 0; i < 10; i++ {
		s := make([]byte, 100000000)
		if s == nil {
			fmt.Println("Operation failed!")
		}
	}
	printStats(mem)

	for i := 0; i < 10; i++ {
		s := make([]byte, 100000000)
		if s == nil {
			fmt.Println("Operation failed!")
		}
		fmt.Printf("Times: %d\n", i)
		time.Sleep(2 * time.Second)
	}
	printStats(mem)

}

func printStats(mem runtime.MemStats) {
	runtime.ReadMemStats(&mem)
	fmt.Println("mem.Alloc:", mem.Alloc)
	fmt.Println("mem.TotalAlloc:", mem.TotalAlloc)
	fmt.Println("mem.HeapAlloc:", mem.HeapAlloc)
	fmt.Println("mem.NumGC:", mem.NumGC)
	fmt.Println("-----")
}
