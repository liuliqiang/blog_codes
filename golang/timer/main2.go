package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/arl/statsviz"
)

func main() {
	mux := http.NewServeMux()
	statsviz.Register(mux)

	fmt.Println("start...")
	ch1 := make(chan string, 120)

	go func() {
		// time.Sleep(time.Second * 1)
		i := 0
		for {
			time.Sleep(time.Millisecond * 10)
			i++
			ch1 <- fmt.Sprintf("%s %d", "hello", i)
		}

	}()

	go func() {
		// http 监听8080, 开启 pprof
		if err := http.ListenAndServe("127.0.0.1:8081", nil); err != nil {
			fmt.Println("listen failed")
		}
	}()

	for {
		time.Sleep(time.Millisecond * 10)
		go func() {
			select {
			case res := <-ch1:
				fmt.Println(res)
			case <-time.After(time.Minute * 3):
				fmt.Println("timeout")
			}

		}()
	}
}
