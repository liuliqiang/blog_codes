// Package discovery provides a static implementation of Kitex's
// discovery.Resolver interface.
//
// In production you would swap this for an etcd / consul / nacos resolver.
// The static resolver is useful for local development, testing, and as a
// clear example of the interface contract.
package discovery

import (
	"context"
	"fmt"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
)

// staticResolver implements discovery.Resolver with a fixed endpoint list.
// Every call to Resolve() returns the same set of instances.
type staticResolver struct {
	endpoints []string // each entry is "host:port"
}

// NewStaticResolver creates a Resolver that always resolves to the given
// list of endpoints.  Assign different weights to the instances if you want
// to influence the load balancer's distribution.
func NewStaticResolver(endpoints []string) discovery.Resolver {
	return &staticResolver{endpoints: endpoints}
}

// Target returns a cache key that uniquely identifies the target service.
// Kitex calls this once when the client is created.
func (r *staticResolver) Target(_ context.Context, target rpcinfo.EndpointInfo) string {
	if target != nil {
		return target.ServiceName()
	}
	return "unknown"
}

// Resolve returns the current set of instances for the given service
// description string.  Because this resolver is static, the result never
// changes after construction.
func (r *staticResolver) Resolve(_ context.Context, desc string) (discovery.Result, error) {
	instances := make([]discovery.Instance, 0, len(r.endpoints))
	for i, ep := range r.endpoints {
		// discovery.DefaultWeight == 10; assign equal weight here.
		// For weighted load balancing, vary the weight per instance.
		instances = append(instances, discovery.NewInstance("tcp", ep, discovery.DefaultWeight, map[string]string{
			"index": fmt.Sprintf("%d", i),
		}))
	}
	return discovery.Result{
		Cacheable: true,
		CacheKey:  "static:" + desc,
		Instances: instances,
	}, nil
}

// Diff compares two resolution results and reports added/updated/removed
// instances.  We delegate to Kitex's built-in DefaultDiff helper.
func (r *staticResolver) Diff(cacheKey string, prev, next discovery.Result) (discovery.Change, bool) {
	return discovery.DefaultDiff(cacheKey, prev, next)
}

// Name returns a human-readable identifier for this resolver.
func (r *staticResolver) Name() string {
	return "static-resolver"
}
