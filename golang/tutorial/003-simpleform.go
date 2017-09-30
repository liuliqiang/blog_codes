package main

import (
	"net/http"
	"fmt"
	"html/template"
	"log"
)

func login(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	fmt.Println("method: ", req.Method)
	if req.Method == "GET" {
		t, _ := template.ParseFiles("templates/login.gtpl")
		t.Execute(res, nil)
	} else {
		fmt.Println("req form", len(req.Form["username"][0]))
		fmt.Println("username", req.Form["username"])
		fmt.Println("password", req.Form["password"])
	}
}


func main() {
	http.HandleFunc("/login", login)

	err := http.ListenAndServe(":9081", nil)
	if err != nil {
		log.Fatal("ListenAndServe", err)
	}
}