package main

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"
)

const (
	OneMB = 1024 * 1024
)

func main() {
	go memoryLeakFunction1()

	http.HandleFunc("/pprof/", pprof.Index)
	http.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	http.HandleFunc("/pprof/profile", pprof.Profile)
	http.HandleFunc("/pprof/symbol", pprof.Symbol)
	http.HandleFunc("/pprof/trace", pprof.Trace)

	addr := "0.0.0.0:8881"
	fmt.Println("visit http://" + addr + "/debug/pprof/ to view memory leak")
	panic(http.ListenAndServe("0.0.0.0:8881", nil))
}

func memoryLeakFunction1() {
	memBlocks := make([][]byte, 0)
	for i := 0; i < 1024; i++ {
		memBlocks = append(memBlocks, make([]byte, OneMB))
		memoryLeakFunction2()
		time.Sleep(time.Second)
	}
}

func memoryLeakFunction2() {
	memBlocks := make([][]byte, 0)
	memBlocks = append(memBlocks, make([]byte, OneMB))
	memoryLeakFunction3()
}

var memBlocks3 [][]byte

func memoryLeakFunction3() {
	memBlocks3 = append(memBlocks3, make([]byte, OneMB))
}
