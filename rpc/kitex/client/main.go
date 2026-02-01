package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/callopt"
	"github.com/cloudwego/kitex/pkg/circuitbreak"
	"github.com/cloudwego/kitex/pkg/connpool"
	"github.com/cloudwego/kitex/pkg/fallback"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/loadbalance"
	"github.com/cloudwego/kitex/pkg/retry"
	"github.com/cloudwego/kitex/pkg/rpcinfo"

	"github.com/liuliqiang/blog_codes/rpc/kitex/discovery"
	"github.com/liuliqiang/blog_codes/rpc/kitex/interceptor"
	"github.com/liuliqiang/blog_codes/rpc/kitex/kitex_gen/kitexdemo"
	"github.com/liuliqiang/blog_codes/rpc/kitex/kitex_gen/kitexdemo/userservice"
)

func main() {
	klog.SetLevel(klog.LevelInfo)
	klog.SetOutput(os.Stdout)

	// ---------------------------------------------------------------
	// 1. Failure Retry policy
	//    Retries a call up to 3 times when it returns an error, with
	//    a random back-off between 100 ms and 500 ms between attempts.
	//    Total retry budget is capped at 5 s.
	// ---------------------------------------------------------------
	retryPolicy := retry.NewFailurePolicy()
	retryPolicy.WithMaxRetryTimes(3)
	retryPolicy.WithMaxDurationMS(5000)     // total retry budget: 5 s
	retryPolicy.WithRandomBackOff(100, 500) // wait 100–500 ms between retries

	// ---------------------------------------------------------------
	// 2. Circuit Breaker suite
	//    Each service+method gets its own independent breaker.
	//    The breaker trips when the error rate exceeds 50 % over the
	//    last window, with a minimum of 10 samples before tripping.
	// ---------------------------------------------------------------
	cbSuite := circuitbreak.NewCBSuite(
		// GenServiceCBKeyFunc: map each RPC to a unique CB key
		func(ri rpcinfo.RPCInfo) string {
			if ri != nil && ri.To() != nil {
				return ri.To().ServiceName() + "." + ri.To().Method()
			}
			return "unknown"
		},
	)
	cbSuite.UpdateServiceCBConfig("*", circuitbreak.CBConfig{
		Enable:    true,
		ErrRate:   0.5,  // trip at 50 % error rate
		MinSample: 10,   // need at least 10 requests before tripping
	})

	// ---------------------------------------------------------------
	// 3. Fallback policy
	//    Only fires on timeout or circuit-breaker-open errors.
	//    Returns a synthetic "degraded" response so the caller does
	//    not see a hard failure.
	// ---------------------------------------------------------------
	fbPolicy := fallback.TimeoutAndCBFallback(
		fallback.UnwrapHelper(func(ctx context.Context, req, resp interface{}, err error) (interface{}, error) {
			klog.Warnf("[Fallback] triggered: %v", err)
			switch req.(type) {
			case *kitexdemo.EchoRequest:
				return &kitexdemo.EchoResponse{
					Message:   "fallback: service temporarily unavailable",
					ElapsedMs: 0,
				}, nil
			case *kitexdemo.GetUserRequest:
				return &kitexdemo.GetUserResponse{
					User: &kitexdemo.User{Id: "fallback", Name: "FallbackUser", Age: 0},
				}, nil
			case *kitexdemo.ListUsersRequest:
				return &kitexdemo.ListUsersResponse{Users: nil, Total: 0}, nil
			}
			return nil, err
		}),
	)

	// ---------------------------------------------------------------
	// 4. Create the client — all governance features wired in here
	// ---------------------------------------------------------------
	//
	// Features demonstrated:
	//   • Service Discovery  — custom static resolver
	//   • Load Balancing     — interleaved weighted round-robin
	//   • RPC / Connect Timeout
	//   • Connection Pool    — long-lived idle connections
	//   • Client Middleware  — per-call logging with latency
	//   • Failure Retry      — random back-off, max 3 retries
	//   • Circuit Breaker    — per-method, trips at 50% error rate
	//   • Fallback           — synthetic response on timeout / CB-open
	//
	svc, err := userservice.NewClient("UserService",
		// [Service Discovery] Static resolver → "127.0.0.1:8080"
		client.WithResolver(discovery.NewStaticResolver([]string{"127.0.0.1:8080"})),

		// [Load Balancing] Weighted round-robin across discovered instances
		client.WithLoadBalancer(loadbalance.NewInterleavedWeightedRoundRobinBalancer()),

		// [RPC Timeout] Default 3 s; can be overridden per-call via callopt
		client.WithRPCTimeout(3*time.Second),

		// [Connect Timeout] Fail fast if the server is unreachable
		client.WithConnectTimeout(1*time.Second),

		// [Connection Pool] Reuse TCP connections; idle pool config
		client.WithLongConnection(connpool.IdleConfig{
			MinIdlePerAddress: 2,
			MaxIdlePerAddress: 10,
			MaxIdleGlobal:     100,
			MaxIdleTimeout:    30 * time.Second,
		}),

		// [Client Middleware] Logs destination, method, latency, status
		client.WithMiddleware(interceptor.ClientLoggingAndRequestID()),

		// [Failure Retry] Retry with random back-off on errors
		client.WithFailureRetry(retryPolicy),

		// [Circuit Breaker] Auto-fail when per-method error rate is too high
		client.WithCircuitBreaker(cbSuite),

		// [Fallback] Graceful degradation on timeout / CB-open
		client.WithFallback(fbPolicy),
	)
	if err != nil {
		klog.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// =================================================================
	// Demo 1 — Echo: basic unary RPC
	// =================================================================
	fmt.Println("\n=== Demo 1: Echo (basic unary) ===")
	echoResp, err := svc.Echo(ctx, &kitexdemo.EchoRequest{Message: "Hello, Kitex!"})
	printEcho(echoResp, err)

	// =================================================================
	// Demo 2 — Echo with per-call timeout override
	//          callopt.WithRPCTimeout overrides the global 3 s timeout
	//          for this single call only.
	// =================================================================
	fmt.Println("\n=== Demo 2: Echo with per-call timeout (500 ms) ===")
	echoResp, err = svc.Echo(ctx,
		&kitexdemo.EchoRequest{Message: "With 500ms timeout"},
		callopt.WithRPCTimeout(500*time.Millisecond), // [Per-call Option]
	)
	printEcho(echoResp, err)

	// =================================================================
	// Demo 3 — Echo with per-call retry policy override
	//          callopt.WithRetryPolicy replaces the global retry policy
	//          for this call: 1 retry, fixed 200 ms back-off.
	// =================================================================
	fmt.Println("\n=== Demo 3: Echo with per-call retry (1 retry, fixed 200 ms) ===")
	perCallRetry := retry.NewFailurePolicy()
	perCallRetry.WithMaxRetryTimes(1)
	perCallRetry.WithFixedBackOff(200)
	echoResp, err = svc.Echo(ctx,
		&kitexdemo.EchoRequest{Message: "With per-call retry"},
		callopt.WithRetryPolicy(retry.BuildFailurePolicy(perCallRetry)), // [Per-call Retry]
	)
	printEcho(echoResp, err)

	// =================================================================
	// Demo 4 — GetUser: successful lookup
	//          Demonstrates the full User struct with enum, map, list fields.
	// =================================================================
	fmt.Println("\n=== Demo 4: GetUser (existing user id=1) ===")
	userResp, err := svc.GetUser(ctx, &kitexdemo.GetUserRequest{UserId: "1"})
	if err != nil {
		handleError("GetUser", err)
	} else {
		u := userResp.User
		fmt.Printf("  id=%s name=%s age=%d email=%s role=%s\n",
			u.Id, u.Name, u.Age, derefStr(u.Email), derefRole(u.Role))
		fmt.Printf("  tags=%v  permissions=%v\n", u.Tags, u.Permissions)
	}

	// =================================================================
	// Demo 5 — GetUser: NotFoundException
	//          The server returns a typed Thrift exception.  The client
	//          type-asserts on it to extract structured error details.
	// =================================================================
	fmt.Println("\n=== Demo 5: GetUser (non-existent user id=999) ===")
	_, err = svc.GetUser(ctx, &kitexdemo.GetUserRequest{UserId: "999"})
	handleError("GetUser", err) // prints typed exception details

	// =================================================================
	// Demo 6 — ListUsers: filter + pagination
	// =================================================================
	fmt.Println("\n=== Demo 6a: ListUsers (all users) ===")
	listResp, err := svc.ListUsers(ctx, &kitexdemo.ListUsersRequest{})
	printUsers(listResp, err)

	fmt.Println("\n=== Demo 6b: ListUsers (filter='ali') ===")
	filter := "ali"
	listResp, err = svc.ListUsers(ctx, &kitexdemo.ListUsersRequest{Filter: &filter})
	printUsers(listResp, err)

	fmt.Println("\n=== Demo 6c: ListUsers (page=1, page_size=2) ===")
	page, pageSize := int32(1), int32(2)
	listResp, err = svc.ListUsers(ctx, &kitexdemo.ListUsersRequest{
		Page:     &page,
		PageSize: &pageSize,
	})
	printUsers(listResp, err)

	// =================================================================
	// Demo 7 — LogEvent: oneway (fire-and-forget)
	//          The client sends the request and immediately returns.
	//          No response is expected or waited on.
	// =================================================================
	fmt.Println("\n=== Demo 7: LogEvent (oneway fire-and-forget) ===")
	err = svc.LogEvent(ctx, &kitexdemo.LogEventRequest{
		Event: "user_login",
		Level: kitexdemo.LogLevel_INFO,
		ContextData: map[string]string{
			"user_id": "1",
			"ip":      "192.168.1.1",
		},
	})
	if err != nil {
		klog.Errorf("LogEvent failed: %v", err)
	} else {
		fmt.Println("  LogEvent sent (oneway — no response)")
	}

	fmt.Println("\n=== All demos completed ===")
}

// ---------------------------------------------------------------------------
// Print / Error helpers
// ---------------------------------------------------------------------------

func printEcho(resp *kitexdemo.EchoResponse, err error) {
	if err != nil {
		handleError("Echo", err)
		return
	}
	fmt.Printf("  message=%q  server_id=%s\n", resp.Message, derefStr(resp.ServerId))
}

func handleError(rpc string, err error) {
	if err == nil {
		return
	}
	// Type-assert on generated Thrift exception types so we can print
	// structured fields instead of just the generic error string.
	switch e := err.(type) {
	case *kitexdemo.NotFoundException:
		fmt.Printf("  [%s] NotFoundException — resource=%s id=%s\n", rpc, e.Resource, e.Id)
	case *kitexdemo.ServiceException:
		fmt.Printf("  [%s] ServiceException  — code=%d message=%s details=%s\n",
			rpc, e.Code, e.Message, derefStr(e.Details))
	default:
		fmt.Printf("  [%s] Error: %v\n", rpc, err)
	}
}

func printUsers(resp *kitexdemo.ListUsersResponse, err error) {
	if err != nil {
		handleError("ListUsers", err)
		return
	}
	fmt.Printf("  total=%d  returned=%d\n", resp.Total, len(resp.Users))
	for _, u := range resp.Users {
		fmt.Printf("    id=%s  name=%-10s age=%d\n", u.Id, u.Name, u.Age)
	}
}

func derefStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func derefRole(r *kitexdemo.UserRole) string {
	if r == nil {
		return "<nil>"
	}
	return r.String()
}
