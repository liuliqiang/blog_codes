package main

import (
	"context"
	"flag"
	"fmt"
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
	crt := "/tmp/server.crt"
	key := "/tmp/server.key"
	creds, err := credentials.NewServerTLSFromFile(crt, key)
	if err != nil {
		panic(err)
	}

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
