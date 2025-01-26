package main

import (
	"crypto/tls"
	"log"
	"net/http"
)

var (
	CertFilePath = "/home/liqiang.liu/blog/ssl/ssl.pyer.dev/fullchain.pem"
	KeyFilePath  = "/home/liqiang.liu/blog/ssl/ssl.pyer.dev/privkey.pem"
)

func httpRequestHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Hello,World!\n"))
}
func main() {
	// load tls certificates
	serverTLSCert, err := tls.LoadX509KeyPair(CertFilePath, KeyFilePath)
	if err != nil {
		log.Fatalf("Error loading certificate and key file: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			serverTLSCert,
		},
	}
	server := http.Server{
		Addr:      ":443",
		Handler:   http.HandlerFunc(httpRequestHandler),
		TLSConfig: tlsConfig,
	}
	defer server.Close()

	print("Server is running at 443 port.\n")
	log.Fatal(server.ListenAndServeTLS("", ""))
}
