package server

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"

	"github.com/liuliqiang/blog-demos/microservices/rpc/grpc/go/proto-gens"

	"google.golang.org/grpc"
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

	grpcServer := grpc.NewServer()
	helloworld.RegisterGreeterServer(grpcServer, &helloLiqiangIO{})
	grpcServer.Serve(lis)
}

// hello to https://liqiang.io
type helloLiqiangIO struct {
}

func (*helloLiqiangIO) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	log.Printf("[I] server")
	return nil, status.Error(codes.Canceled, "")
}
