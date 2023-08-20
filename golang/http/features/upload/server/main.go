package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var (
	dirPath = "./static"
)

func main() {
	flag.StringVar(&dirPath, "dir", "./static", "Directory to serve static files from")
	flag.Parse()

	http.HandleFunc("/upload", uploadHandle)
	http.HandleFunc("/upload_binary", uploadBinary)

	log.Print("Listening on :3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

const boundary = "d083a547-33d1-41d1-94f7-d15a0f36d795"

func uploadHandle(w http.ResponseWriter, req *http.Request) {
	for k, vs := range req.Form {
		fmt.Printf("req.Form[%s]: %+v\n", k, vs)
	}
	for k, vs := range req.PostForm {
		fmt.Printf("req.PostForm[%s]: %+v\n", k, vs)
	}

	fmt.Printf("req.MultipartForm: %+v\n", req.MultipartForm)

	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	fmt.Printf("req.GetBody()[:200]: %s\n", bytes[:200])
	fmt.Printf("req.GetBody(): %d bytes\n", len(bytes))
}

func uploadBinary(w http.ResponseWriter, req *http.Request) {
	for k, vs := range req.Header {
		fmt.Printf("req.Header[%s]: %+v\n", k, vs)
	}

	file, err := os.Create("/tmp/upload-binary.jpg")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if _, err = io.Copy(file, req.Body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
