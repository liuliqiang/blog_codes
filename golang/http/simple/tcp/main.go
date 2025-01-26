package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func main() {
	// 创建一个 TCP 监听器
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("HTTP Server listening on port 8080...")

	for {
		// 接受客户端连接
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		fmt.Println("Accepted connection from:", conn.RemoteAddr())
		// 处理客户端连接
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 使用 bufio 读取请求
	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	// 解析 HTTP 请求
	fmt.Printf("Received request: %s", requestLine)
	method, path, _ := strings.Fields(requestLine)[0], strings.Fields(requestLine)[1], strings.Fields(requestLine)[2]

	// 读取并丢弃请求头
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
		fmt.Printf("Discarding header: %s", line)
	}

	// 构造简单的响应
	response := ""
	if method == "GET" && path == "/" {
		response = "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain\r\n" +
			"Content-Length: 13\r\n" +
			"\r\n" +
			"Hello, World!"
	} else {
		response = "HTTP/1.1 404 Not Found\r\n" +
			"Content-Type: text/plain\r\n" +
			"Content-Length: 9\r\n" +
			"\r\n" +
			"Not Found"
	}

	// 发送响应
	conn.Write([]byte(response))
}
