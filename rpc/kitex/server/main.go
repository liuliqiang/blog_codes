package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/stats"
	"github.com/cloudwego/kitex/server"

	"github.com/liuliqiang/blog_codes/rpc/kitex/interceptor"
	"github.com/liuliqiang/blog_codes/rpc/kitex/kitex_gen/kitexdemo"
	"github.com/liuliqiang/blog_codes/rpc/kitex/kitex_gen/kitexdemo/userservice"
)

// ---------------------------------------------------------------------------
// stats.Tracer — lifecycle hooks (Start / Finish) that fire around every RPC.
//
// This is a separate mechanism from Middleware:
//   - Middleware wraps the handler and can inspect / modify req & resp.
//   - Tracer provides lightweight Start/Finish hooks for metrics collection.
//
// Here we use it to measure and log per-RPC duration.
// ---------------------------------------------------------------------------

// compile-time assertion that metricsTracer satisfies stats.Tracer
var _ stats.Tracer = (*metricsTracer)(nil)

type metricsTracer struct{}

type tracerCtxKey struct{}

func (t *metricsTracer) Start(ctx context.Context) context.Context {
	// Store the start time in the context so Finish() can compute duration.
	return context.WithValue(ctx, tracerCtxKey{}, time.Now())
}

func (t *metricsTracer) Finish(ctx context.Context) {
	ri := rpcinfo.GetRPCInfo(ctx)
	method := "unknown"
	if ri != nil && ri.To() != nil {
		method = ri.To().Method()
	}
	if start, ok := ctx.Value(tracerCtxKey{}).(time.Time); ok {
		klog.Infof("[Server:Tracer] method=%s duration=%v", method, time.Since(start))
	}
}

// ---------------------------------------------------------------------------
// Service Implementation
//
// UserServiceImpl holds an in-memory user store and satisfies the generated
// kitexdemo.UserService interface.  Every method signature must match what
// kitex generated from the Thrift IDL exactly.
// ---------------------------------------------------------------------------

type UserServiceImpl struct {
	mu    sync.RWMutex
	users map[string]*kitexdemo.User
}

func NewUserServiceImpl() *UserServiceImpl {
	impl := &UserServiceImpl{
		users: make(map[string]*kitexdemo.User),
	}

	// Seed demo data covering all optional/required/enum/map/list fields.
	admin := kitexdemo.UserRole_ADMIN
	user := kitexdemo.UserRole_USER
	guest := kitexdemo.UserRole_GUEST

	impl.users["1"] = &kitexdemo.User{
		Id:          "1",
		Name:        "Alice",
		Age:         30,
		Email:       strPtr("alice@example.com"),
		Role:        &admin,
		Tags:        map[string]string{"dept": "engineering", "level": "senior"},
		Permissions: []string{"read", "write", "deploy"},
	}
	impl.users["2"] = &kitexdemo.User{
		Id:          "2",
		Name:        "Bob",
		Age:         25,
		Email:       strPtr("bob@example.com"),
		Role:        &user,
		Tags:        map[string]string{"dept": "product"},
		Permissions: []string{"read", "write"},
	}
	impl.users["3"] = &kitexdemo.User{
		Id:   "3",
		Name: "Charlie",
		Age:  35,
		Role: &guest,
		Tags: map[string]string{"dept": "marketing"},
		// Email and Permissions intentionally omitted (optional fields)
	}
	return impl
}

// Echo — basic unary RPC.  Returns the message prefixed with "echo: ".
func (s *UserServiceImpl) Echo(ctx context.Context, req *kitexdemo.EchoRequest) (r *kitexdemo.EchoResponse, err error) {
	serverID := fmt.Sprintf("server-%d", os.Getpid())
	time.Sleep(time.Millisecond) // simulate a tiny bit of work

	return &kitexdemo.EchoResponse{
		Message:   fmt.Sprintf("echo: %s", req.Message),
		ElapsedMs: 1,
		ServerId:  &serverID,
	}, nil
}

// GetUser — looks up a user by ID.  Returns NotFoundException when the ID
// does not exist, demonstrating Kitex Thrift exception handling.
func (s *UserServiceImpl) GetUser(ctx context.Context, req *kitexdemo.GetUserRequest) (r *kitexdemo.GetUserResponse, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[req.UserId]
	if !ok {
		// Return a typed Thrift exception — the client can type-assert on it.
		return nil, &kitexdemo.NotFoundException{
			Resource: "User",
			Id:       req.UserId,
		}
	}
	return &kitexdemo.GetUserResponse{User: user}, nil
}

// ListUsers — returns a paginated, optionally filtered list of users.
// Demonstrates complex return types (list of structs) and pagination logic.
func (s *UserServiceImpl) ListUsers(ctx context.Context, req *kitexdemo.ListUsersRequest) (r *kitexdemo.ListUsersResponse, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect and sort for deterministic output
	var users []*kitexdemo.User
	for _, u := range s.users {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Id < users[j].Id
	})

	// Optional name filter (case-insensitive substring match)
	if req.Filter != nil && *req.Filter != "" {
		filter := strings.ToLower(*req.Filter)
		var filtered []*kitexdemo.User
		for _, u := range users {
			if strings.Contains(strings.ToLower(u.Name), filter) {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}

	total := int32(len(users))

	// Pagination defaults
	page := int32(1)
	pageSize := int32(10)
	if req.Page != nil && *req.Page > 0 {
		page = *req.Page
	}
	if req.PageSize != nil && *req.PageSize > 0 {
		pageSize = *req.PageSize
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &kitexdemo.ListUsersResponse{
		Users: users[start:end],
		Total: total,
	}, nil
}

// LogEvent — oneway (fire-and-forget) RPC.  The client does not wait for a
// response.  On the server side we simply log the event.
func (s *UserServiceImpl) LogEvent(ctx context.Context, req *kitexdemo.LogEventRequest) (err error) {
	klog.Infof("[LogEvent] level=%v event=%s context_data=%v",
		req.Level, req.Event, req.ContextData)
	return nil
}

func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	// --- Logging -----------------------------------------------------------
	klog.SetLevel(klog.LevelInfo)
	klog.SetOutput(os.Stdout)

	svc := NewUserServiceImpl()

	// --- Server creation with all governance options ----------------------
	//
	// Features demonstrated:
	//   • WithServiceAddr   — explicit listen address
	//   • WithMaxConnIdleTime — idle connection cleanup
	//   • WithExitWaitTime  — graceful-shutdown drain window
	//   • WithMiddleware    — per-RPC interceptor (logging + status)
	//   • WithTracer        — lifecycle hooks for metrics
	//
	svr := userservice.NewServer(svc,
		// [Bind Address] Listen on 0.0.0.0:8080
		server.WithServiceAddr(&net.TCPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 8080}),

		// [Idle Timeout] Close connections that have been idle > 5 min
		server.WithMaxConnIdleTime(5*time.Minute),

		// [Exit Wait] Allow up to 3 s for in-flight RPCs to finish during shutdown
		server.WithExitWaitTime(3*time.Second),

		// [Middleware] Server-side interceptor — logs method name + error status
		server.WithMiddleware(interceptor.ServerLoggingAndMetrics()),

		// [Tracer] Lifecycle hooks — measures and logs duration for every RPC
		server.WithTracer(&metricsTracer{}),
	)

	// --- Graceful shutdown via OS signals ----------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		klog.Info("Received shutdown signal, stopping server gracefully...")
		if err := svr.Stop(); err != nil {
			klog.Errorf("Error during shutdown: %v", err)
		}
		klog.Info("Server stopped")
	}()

	// --- Run (blocks) ----------------------------------------------------
	klog.Info("Starting UserService on 0.0.0.0:8080")
	if err := svr.Run(); err != nil {
		klog.Fatalf("Failed to start server: %v", err)
	}
}
