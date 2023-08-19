package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
)

const boundary = "d083a547-33d1-41d1-94f7-d15a0f36d795"

func main() {
	tr := http.DefaultTransport

	client := &http.Client{
		Transport: tr,
		Timeout:   0,
	}
	
	fmt.Println("Set up pipe")
	pR, pW := io.Pipe()

	go func() {
		// Set up multipart body for reading
		multipartW := multipart.NewWriter(pW)
		fmt.Println("Set up multipart writer")
		multipartW.SetBoundary(boundary)
		fmt.Println("Set up boundary")	
		partW, err0 := multipartW.CreateFormFile("fakefield", "fakefilename")
		fmt.Println("Set up part writer")	
		if err0 != nil {
			panic("Something is amiss creating a part")
		}

		connector := io.TeeReader(os.Stdin, partW)
		buf := make([]byte, 256)
		for {
			/* stdin -> connector -> partW -> multipartW -> pW -> pR */
			_, err := connector.Read(buf) 
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("The error reading from connector: %v", err)
			}
		}
		
	}()

	// Send http request chunk encoding the multipart message
	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:3000",
			Path:   "/upload",
		},
		ProtoMajor: 1,
		ProtoMinor: 1,
		ContentLength: -1,
		Body: pR, 
	}
	fmt.Printf("Doing request\n")
	_, err := client.Do(req)
	fmt.Printf("Done request. Err: %v\n", err)
}