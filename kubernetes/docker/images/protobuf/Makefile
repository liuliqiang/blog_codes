all:
	docker build -t lukelau/protobuf:0.0.2 .

vendor:
	docker run --rm \
		-v $(shell pwd):/go/src/github.com/liuliqiang/protobuf/ \
		-w /go/src/github.com/liuliqiang/protobuf \
		golang:1.13 make _vendor

_vendor:
	go mod init
	go mod tidy
	go mod vendor
