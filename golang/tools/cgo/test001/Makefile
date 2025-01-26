.PHONY: all
all:
	docker run -it \
		--rm \
		--name cgo-build \
		-v $(shell pwd):/home/liqiang \
		-w /home/liqiang \
		golang:1.22 go build -x main.go

