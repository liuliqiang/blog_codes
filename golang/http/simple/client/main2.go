package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	for i := 0; i < 101; i++ {
		func() {
			resp, err := http.Get("https://gobyexample.com")
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			fmt.Println("Response status:", resp.Status)
			fmt.Println("Connection Header:", resp.Header.Get("Connection"))

			_, _ = io.ReadAll(resp.Body)
		}()
		time.Sleep(time.Millisecond * 500)
	}
}
