package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
)

var (
	Ascii      = "QWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnm"
	ServerName string
)

func init() {
	for i := 0; i < 10; i++ {
		var idx = rand.Int() % len(Ascii)
		ServerName = ServerName + Ascii[idx:idx+1]
	}
}

func main() {
	var serverAddr = ":80"
	flag.StringVar(&serverAddr, "server.addr", serverAddr, "addr for http server")
	flag.StringVar(&ServerName, "server.name", ServerName, "name for this http server")
	flag.Parse()

	if podId := os.Getenv("POD_ID"); podId != "" {
		ServerName = podId
	}

	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		if _, err := resp.Write([]byte(fmt.Sprintf("Hello, It's %s", ServerName))); err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			log.Printf("Failed to resp: %v", err)
			return
		}
		return
	})
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Panic("Failed to listen server", err)
	}
}
