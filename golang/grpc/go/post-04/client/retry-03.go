package client

import (
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc/codes"
	"log"

	"github.com/liuliqiang/blog-demos/microservices/rpc/grpc/go/proto-gens"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(
			grpc_retry.WithCodes(codes.Canceled, codes.DataLoss, codes.Unavailable),
			grpc_retry.WithMax(2))),)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := helloworld.NewGreeterClient(conn)
	resp, err := cli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "lucifer"}, )
	if err != nil {
		panic(err)
	}
	log.Printf("[D] resp: %s", resp.Message)
}
