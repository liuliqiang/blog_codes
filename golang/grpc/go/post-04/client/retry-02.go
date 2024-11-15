package client

import (
	"context"
	"log"

	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"

	helloworld "github.com/liuliqiang/blog_codes/golang/grpc/go/proto-gens"
)

func main() {
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpcretry.UnaryClientInterceptor(
			grpcretry.WithMax(2))))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := helloworld.NewGreeterClient(conn)
	resp, err := cli.SayHello(context.Background(),
		&helloworld.HelloRequest{Name: "lucifer"},
		grpcretry.WithMax(3))
	if err != nil {
		panic(err)
	}
	log.Printf("[D] resp: %s", resp.Message)
}
