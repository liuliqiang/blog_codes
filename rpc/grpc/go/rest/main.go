package main

import (
	"flag"
	"net/http"

	gw "github.com/liuliqiang/blog_codes/rpc/grpc/go"

	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	helloEndpoint = flag.String("hello", "localhost:8080", "endpoint of hello liqiang.io")
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := gw.RegisterGreeterHandlerFromEndpoint(ctx, mux, *helloEndpoint, opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(":8088", mux)
}

func main() {
	flag.Parse()
	defer glog.Flush()

	if err := run(); err != nil {
		glog.Fatal(err)
	}
}
