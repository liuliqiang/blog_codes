// Package interceptor provides reusable Kitex middleware (endpoint.Middleware)
// for both the server side and the client side.
//
// Kitex middleware signature:
//
//	type Middleware func(next endpoint.Endpoint) endpoint.Endpoint
//
// Each middleware wraps the next handler in the chain, giving you a "before"
// and "after" hook around every RPC.
package interceptor

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
)

// ServerLoggingAndMetrics returns a server-side middleware that logs every
// incoming RPC together with the caller's service name, the method that was
// invoked, and whether it succeeded or failed.
//
// Registered via server.WithMiddleware(...) and runs for every inbound request.
func ServerLoggingAndMetrics() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) (err error) {
			ri := rpcinfo.GetRPCInfo(ctx)
			method := "unknown"
			caller := "unknown"
			if ri != nil {
				if ri.To() != nil {
					method = ri.To().Method()
				}
				if ri.From() != nil {
					caller = ri.From().ServiceName()
				}
			}

			// ---- call the actual handler ----
			err = next(ctx, req, resp)

			status := "OK"
			if err != nil {
				status = fmt.Sprintf("ERROR(%v)", err)
			}
			klog.Infof("[Server:Middleware] caller=%s method=%s status=%s",
				caller, method, status)
			return err
		}
	}
}

// ClientLoggingAndRequestID returns a client-side middleware that logs every
// outgoing RPC call together with the destination service, method, wall-clock
// latency, and result status.
//
// Registered via client.WithMiddleware(...).  Runs once per outbound call,
// before any retry or circuit-breaker logic.
func ClientLoggingAndRequestID() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) (err error) {
			ri := rpcinfo.GetRPCInfo(ctx)
			method := "unknown"
			dest := "unknown"
			if ri != nil && ri.To() != nil {
				method = ri.To().Method()
				dest = ri.To().ServiceName()
			}

			start := time.Now()

			// ---- call the next layer (transport / retry / ...) ----
			err = next(ctx, req, resp)

			elapsed := time.Since(start)
			status := "OK"
			if err != nil {
				status = fmt.Sprintf("ERROR(%v)", err)
			}
			klog.Infof("[Client:Middleware] dest=%s method=%s latency=%v status=%s",
				dest, method, elapsed, status)
			return err
		}
	}
}
