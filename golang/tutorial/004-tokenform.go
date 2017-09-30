package main

import (
	"net/http"
	"fmt"
	"html/template"
	"log"
	"time"
	"crypto/md5"
	"io"
	"strconv"
)

func webForm2Login(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	fmt.Println("method: ", req.Method)
	if req.Method == "GET" {
		// add token in html
		curTime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(curTime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))
		t, _ := template.ParseFiles("templates/interest.gtpl")
		t.Execute(res, token)
	} else {
		fmt.Println("req form", len(req.Form["username"][0]))
		fmt.Println("username", req.Form["username"])
		fmt.Println("password", req.Form["password"])
	}
}


func main() {
	http.HandleFunc("/login", webForm2Login)

	err := http.ListenAndServe(":9082", nil)
	if err != nil {
		log.Fatal("ListenAndServe", err)
	}
}