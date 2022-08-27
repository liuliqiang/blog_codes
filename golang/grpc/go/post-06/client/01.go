package main

import (
	"context"
	"io"
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
	facbCli, err := cli.Facb(context.Background(), &protobuf.FacbRequest{Max: int64(100)})
	if err != nil {
		panic(err)
	}

	for {
		if resp, err := facbCli.Recv(); err != nil {
			if err == io.EOF {
				return
			} else {
				panic(err)
			}
		} else {
			log.Printf("[D] index: %d, facb: %d", resp.Index, resp.Curr)
		}
	}
}
