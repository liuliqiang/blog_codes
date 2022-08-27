package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/liuliqiang/blog-demos/microservices/rpc/grpc/go/post-06/proto-gens"
	"google.golang.org/grpc"
)

type server struct {
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 8080))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	protobuf.RegisterPost06Server(grpcServer, &server{})
	grpcServer.Serve(lis)
}

func (*server) Sum(req protobuf.Post06_SumServer) (err error) {
	var reqObj *protobuf.SumRequest

	var sum int64 = 0
	for {
		reqObj, err = req.Recv()
		if err == io.EOF {
			log.Printf("[E] err: %v", err)
			req.SendAndClose(&protobuf.SumResponse{Result: sum})
			return nil
		} else if err == nil {
			log.Printf("[D] recv: %v", reqObj)
			sum += reqObj.Num
		} else {
			return err
		}
	}
}

func (*server) Facb(req *protobuf.FacbRequest, stream protobuf.Post06_FacbServer) error {
	if req.Max < 2 {
		stream.Send(&protobuf.FacbResponse{Index: 1, Curr: 1})
		return nil
	}
	var sum int64 = 1
	var index int32 = 1
	var next int64
	for sum < req.Max {
		stream.Send(&protobuf.FacbResponse{Index: index, Curr: sum})
		index++
		next, sum = sum, sum+next
	}
	return nil
}

func (*server) Chat(req protobuf.Post06_ChatServer) error {
	for {
		reqObj, err := req.Recv()
		if err != nil {
			if err == io.EOF || err == nil {
				return nil
			}
			return err
		}

		msg := strings.Replace(reqObj.Msg, "吗", "", -1)
		msg = strings.Replace(msg, "?", "!", -1)
		msg = strings.Replace(msg, "？", "!", -1)
		req.Send(&protobuf.ChatResponse{Reply: msg})
	}
}
