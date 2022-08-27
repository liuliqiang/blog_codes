FROM golang:1.11.2

ENV GO111MODULE=off
ENV http_proxy=http://10.0.0.9:8118
ENV https_proxy=http://10.0.0.9:8118

RUN go get k8s.io/code-generator; exit 0
RUN go get k8s.io/apimachinery; exit 0

ARG repo="${GOPATH}/src/github.com/liuliqiang/admin"

RUN mkdir -p $repo

WORKDIR $GOPATH/src/k8s.io/code-generator

VOLUME $repo