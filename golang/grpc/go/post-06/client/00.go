package main

import (
	"context"
	"log"

	"github.com/liuliqiang/blog-demos/microservices/rpc/grpc/go/post-06/proto-gens"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := protobuf.NewPost06Client(conn)
	sumCli, err := cli.Sum(context.Background())
	if err != nil {
		panic(err)
	}
	sumCli.Send(&protobuf.SumRequest{Num: int64(1)})
	sumCli.Send(&protobuf.SumRequest{Num: int64(2)})
	sumCli.Send(&protobuf.SumRequest{Num: int64(3)})
	if resp, err := sumCli.CloseAndRecv(); err != nil {
		panic(err)
	} else {
		log.Printf("[D] resp: %s", resp.Result)
	}
}
