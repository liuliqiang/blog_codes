FROM golang:1.13

COPY . /go/src/github.com/liuliqiang/admin-controller

WORKDIR /go/src/github.com/liuliqiang/admin-controller
RUN go mod tidy && go mod vendor

