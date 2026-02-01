package main

import (
	"fmt"
	"net/http"
	"net/http/httptrace"
)

func main() {
	req, _ := http.NewRequest("GET", "https://google.com", nil)
	trace := &httptrace.ClientTrace{
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			fmt.Printf("DNS Info: %+v\n", dnsInfo)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			fmt.Printf("Got Conn: %+v\n", connInfo)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	if _, err := http.DefaultClient.Do(req); err != nil {
		panic(err)
	}
}
