package client

import (
	"context"
	"log"

	"google.golang.org/grpc"

	helloworld "github.com/liuliqiang/blog_codes/golang/grpc/go/proto-gens"
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
		resp, err = cli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "lucifer"})
		if err != nil {
			panic(err)
		}
	}
	log.Printf("[D] resp: %s", resp.Message)
}
