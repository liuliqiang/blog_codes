.PHONY: test client server


test:
	docker run --rm -it \
		-v $(PWD):/work \
		-w /work \
		-p 10000:10000 \
		docker.io/envoyproxy/envoy:v1.32.1 \
		/usr/local/bin/envoy -c /work/envoy.yaml
	
client:
	go run client/main.go

client2:
	go run client/main2.go