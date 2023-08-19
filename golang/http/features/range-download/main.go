package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

var (
	dirPath = "./static"
)

func main() {
	flag.StringVar(&dirPath, "dir", "./static", "Directory to serve static files from")
	flag.Parse()

	http.HandleFunc("/upload", uploadHandle)
	http.Handle("/", http.FileServer(http.Dir(dirPath)))

	log.Print("Listening on :3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

const boundary = "d083a547-33d1-41d1-94f7-d15a0f36d795"

func uploadHandle(w http.ResponseWriter, req *http.Request) {
		partReader := multipart.NewReader(req.Body, boundary)
		buf := make([]byte, 256)
		for {
			part, err := partReader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				return
			}
			var n int
			for {
				n, err = part.Read(buf)
				if err == io.EOF {
					break
				}
				fmt.Printf(string(buf[:n]))
			}
			fmt.Printf(string(buf[:n]))
		}
}