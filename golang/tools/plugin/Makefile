
build-ch:
	go build -buildmode=plugin -o ./plugins/chinese/greet.so ./plugins/chinese/greet.go

build-en:
	go build -buildmode=plugin -o ./plugins/english/greet.so ./plugins/english/greet.go

build-panic:
	go build -buildmode=plugin -o ./plugins/panic/greet.so ./plugins/panic/greet.go

build-exit:
	go build -buildmode=plugin -o ./plugins/exit/greet.so ./plugins/exit/greet.go

run-normal:
	go run cmd/normal/main.go

run-exit:
	go run cmd/mock-exit/main.go

bench:
	go test -bench=. | grep -v 你好宇宙