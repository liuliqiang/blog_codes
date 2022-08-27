FROM golang:1.12.5

COPY . /go/src/github.com/liuliqiang/rest-docker

WORKDIR /go/src/github.com/liuliqiang/rest-docker

RUN CGO_ENABLED=0 GOOS=linux go build -o /rest-server main.go

FROM alpine:3.10.0

COPY --from=0 /rest-server /rest-server

USER nobody
EXPOSE 80
ENTRYPOINT ["/rest-server"]
