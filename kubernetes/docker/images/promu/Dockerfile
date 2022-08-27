FROM golang:1.10.0

MAINTAINER "Liqiang Liu" "liqianglau@outlook.com"

WORKDIR /go/src/github.com/prometheus

RUN git clone https://github.com/prometheus/promu.git
RUN cd promu && git checkout v0.2.0 && make build
