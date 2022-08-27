package interceptor

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"
)

func UnaryServerDumpInterceptor(optFuncs ...grpc.CallOption) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		strInfo, _ := json.MarshalIndent(req, "", "\t")
		fmt.Printf("request: %s\n", strInfo)

		resp, err = handler(ctx, req)
		if resp != nil {
			strInfo, _ = json.MarshalIndent(resp, "", "\t")
			fmt.Printf("response: %s\n", strInfo)
		} else {
			fmt.Println("response is empty")
		}
		return resp, nil
	}
}
