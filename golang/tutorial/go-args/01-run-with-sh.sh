#!/bin/sh

go build -o sh-c sh-c.go
sh -c $(pwd)/sh-c arg1 arg2
