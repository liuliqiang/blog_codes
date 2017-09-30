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
	"os"
)

func webFileUpload(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	if req.Method == "GET" {
		// add token in html
		curTime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(curTime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		t, _ := template.ParseFiles("templates/upload.gtpl")
		t.Execute(res, token)

	} else {

		req.ParseMultipartForm(32 << 20)

		// 先保存到内存，再写出
		file, handler, err := req.FormFile("uploadfile")

		if err != nil {
			fmt.Println(err)
			return
		}

		defer file.Close()
		fmt.Fprintf(res, "%v", handler.Header)
		f, err := os.OpenFile("/tmp/" + handler.Filename, os.O_WRONLY | os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}

		defer f.Close()
		io.Copy(f, file)
		fmt.Println("password", req.Form["password"])
	}
}


func main() {
	http.HandleFunc("/upload", webFileUpload)

	err := http.ListenAndServe(":9083", nil)
	if err != nil {
		log.Fatal("ListenAndServe", err)
	}
}