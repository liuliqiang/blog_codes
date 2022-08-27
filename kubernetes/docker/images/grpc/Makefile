pwd_dir = $(shell pwd)
genproto:
	mkdir -p $(pwd_dir)/proto
	docker run --rm -v $(pwd_dir):/proto lukelau/protoc:1.0 -I/proto --go_out=plugins=gprc:/proto/proto /proto/demo.proto

build:
	go build -o grpc-server server/main.go

server-docker:
	docker build -t lukelau/grpc-server:1.1 . -f Dockerfile-server
