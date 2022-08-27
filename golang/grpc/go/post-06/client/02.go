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

	var msgs = []string{
		"在吗？",
		"你好",
		"能看懂中文吗?",
		"真的吗?",
	}

	chatCli, err := cli.Chat(context.Background())
	if err != nil {
		panic(err)
	}

	for _, msg := range msgs {
		log.Printf("[D] send: \t%s", msg)
		if err := chatCli.Send(&protobuf.ChatRequest{Msg: msg}); err != nil {
			if err == io.EOF {
				return
			} else {
				panic(err)
			}
		} else {
			if resp, err := chatCli.Recv(); err != nil {
				if err == io.EOF {
					return
				}
				panic(err)
			} else {
				log.Printf("[D] reply: \t%s", resp.Reply)
			}
		}
	}

	chatCli.CloseSend()
}
