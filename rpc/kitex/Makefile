GO     = /root/soft/go/go/bin/go
KITEX  = /root/src/golang/gopath1.22/bin/kitex
export GOPATH = /root/src/golang/gopath1.22

.PHONY: generate mod-tidy build-server build-client run-server run-client clean

# ---------------------------------------------------------------------------
# Code generation
# ---------------------------------------------------------------------------

# Regenerate kitex_gen/ from the Thrift IDL.
# Run this whenever idl/demo.thrift changes.
generate:
	$(KITEX) -module github.com/liuliqiang/blog_codes/rpc/kitex idl/demo.thrift

# Sync go.mod / go.sum with actual imports
mod-tidy:
	$(GO) mod tidy

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build-server: mod-tidy
	mkdir -p /tmp/bin
	$(GO) build -o /tmp/bin/server ./server

build-client: mod-tidy
	mkdir -p /tmp/bin
	$(GO) build -o /tmp/bin/client ./client

# ---------------------------------------------------------------------------
# Run  (start the server in one terminal, then the client in another)
# ---------------------------------------------------------------------------

run-server: build-server
	./bin/server

run-client: build-client
	./bin/client

# ---------------------------------------------------------------------------
# Housekeeping
# ---------------------------------------------------------------------------

clean:
	rm -rf bin
