package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/grpc/grpc-go/examples/helloworld/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	port *int
)

func main() {
	port = new(int)
	flag.IntVar(port, "port", 8080, "port to serve")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	// CreateServiceMonitor the TLS credentials
	caFile := "/tmp/client.crt"
	crt := "/tmp/server.crt"
	key := "/tmp/server.key"

	// CreateServiceMonitor a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		panic(err)
	}
	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		panic(err)
	}
	certificate, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		panic(err)
	}
	// CreateServiceMonitor the TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	helloworld.RegisterGreeterServer(grpcServer, &helloLiqiangIO{})
	grpcServer.Serve(lis)
}

// hello to https://liqiang.io
type helloLiqiangIO struct {
}

func (*helloLiqiangIO) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{
		Message: fmt.Sprintf("Hello: %s, Welcome to https://liqiang.io.io", req.Name),
	}, nil
}
