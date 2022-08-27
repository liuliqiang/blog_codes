docker:
	docker build -t admin-controller .

attach:
	docker run -w /go/src/github.com/liuliqiang/admin-controller \
	    -v $(shell pwd):/go/src/github.com/liuliqiang/admin-controller \
	     --rm -it admin-controller:latest /bin/sh

gen:
	hack/update-codegen.sh

run:
	go run main.go controller.go --kubeconfig /etc/rancher/k3s/k3s.yaml

.PHONY: docker attach gen
