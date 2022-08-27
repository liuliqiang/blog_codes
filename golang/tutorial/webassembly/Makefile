docker-build:
	docker run --rm -v $(shell pwd):/go/src/github.com/liuliqiang/webassembly \
	-w /go/src/github.com/liuliqiang/webassembly golang:1.13 make build

build:
	GOARCH=wasm GOOS=js go build -o main.wasm assem/main.go

run:
	docker run --rm -it -p 8080:8080 -v $(shell pwd):/go/src/github.com/liuliqiang/webassembly \
	-w /go/src/github.com/liuliqiang/webassembly \
	golang:1.13 go run main.go
.PHONY: docker-build build run
