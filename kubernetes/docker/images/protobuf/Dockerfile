FROM golang:1.13

ADD . /go/src/github.com/liuliqiang/protobuf/
WORKDIR /go/src/github.com/liuliqiang/protobuf/
RUN apt-get update && \
    apt-get install unzip && \
    curl -L -o /tmp/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.9.2/protoc-3.9.2-linux-x86_64.zip && \
    unzip /tmp/protoc.zip -d /usr/local
RUN go install ./vendor/github.com/golang/protobuf/protoc-gen-go
RUN go install ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
RUN go install ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

CMD ["/usr/local/bin/protoc"]
