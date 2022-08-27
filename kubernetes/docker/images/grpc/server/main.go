package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	helloworld "github.com/liuliqiang/grpc-demo/proto"

	"github.com/liuliqiang/log4go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var port int
var sleep int

var cert = "/go/src/github.com/liuliqiang/grpc-demo/90-tls/httpbin.example.com/3_application"
var serverCrt = cert + "/certs/httpbin.example.com.cert.pem"
var serverKey = cert + "/private/httpbin.example.com.key.pem"

func main() {
	flag.IntVar(&port, "port", 9321, "port to serve")
	flag.IntVar(&sleep, "sleep", 1, "time to sleep")
	flag.Parse()

	creds, err := credentials.NewServerTLSFromFile(serverCrt, serverKey)
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	helloworld.RegisterGreeterServer(grpcServer, &helloServer{})
	log4go.Info(context.Background(), "Ready to start server @:%d...", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to server grpc: %v", err)
	}
}

type helloServer struct {
}

func (s *helloServer) SayHello(ctx context.Context, req *helloworld.HelloRequest) (resp *helloworld.HelloReply, err error) {
	log4go.Info(context.Background(), "%s say hello to you!", req.Name)
	time.Sleep(time.Duration(sleep) * time.Second)
	log4go.Info(context.Background(), "resp hi to %s!", req.Name)
	return &helloworld.HelloReply{
		Message: req.Name,
	}, nil
}
