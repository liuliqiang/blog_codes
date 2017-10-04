package main

import (
	"fmt"
	"net/http"
	"log"
)

type dollars float32
type database map[string]dollars


func (d dollars) String() string {
	return fmt.Sprintf("$%.2f", d)
}


func (db database) list(w http.ResponseWriter, req *http.Request) {
	for item, price := range db {
		fmt.Fprintf(w, "%s: %s\n", item, price)
	}
}

func (db database) price(w http.ResponseWriter, req *http.Request) {
	item := req.URL.Query().Get("item")
	price, ok := db[item]
	if !ok {
		w.WriteHeader(http.StatusFound)
		fmt.Fprintf(w, "no such item: %q\n", item)
		return
	}
	fmt.Fprintf(w, "%s\n", price)
}


func main() {
	db := database{"shoes": 50, "socks": 5}

	mux := http.NewServeMux()
	mux.HandleFunc("/list", http.HandlerFunc(db.list))
	mux.HandleFunc("/price", http.HandlerFunc(db.price))
	log.Fatal(http.ListenAndServe("localhost:8000", mux))
}
