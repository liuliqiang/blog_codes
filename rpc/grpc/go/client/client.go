package main

import (
	"context"
	"log"

	"github.com/liuliqiang/blog_codes/rpc/grpc/go"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
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
