package main

import (
	"context"
	"log"

	"github.com/grpc/grpc-go/examples/helloworld/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// CreateServiceMonitor the client TLS credentials
	cert := "/tmp/server.crt"
	creds, err := credentials.NewClientTLSFromFile(cert, "local.liqiang.io.io")
	if err != nil {
		panic(err)
	}

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
