package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"

	"github.com/grpc/grpc-go/examples/helloworld/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	crt := "/tmp/client.crt"
	key := "/tmp/client.key"
	caFile := "/tmp/server.crt"
	// Load the client certificates from disk
	certificate, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		panic(err)
	}

	// CreateServiceMonitor a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		panic(err)
	}

	// Append the certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		panic("")
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "local.liqiang.io.io",
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	conn, err := grpc.Dial("local.liqiang.io.io:8080", grpc.WithTransportCredentials(creds))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := helloworld.NewGreeterClient(conn)
	resp, err := cli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "lucifer"})
	if err != nil {
		panic(err)
	}
	log.Printf("[D] resp: %s", resp.Message)
}
