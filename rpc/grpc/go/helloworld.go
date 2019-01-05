package helloworld

import (
	"fmt"

	"golang.org/x/net/context"
)

var _ GreeterServer = &HelloLiqiangIO{}

// hello to https://liqiang.io
type HelloLiqiangIO struct {
}

func (*HelloLiqiangIO) SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error) {
	return &HelloReply{
		Message: fmt.Sprintf("Hello: %s, Welcome to https://liqiang.io", req.Name),
	}, nil
}
