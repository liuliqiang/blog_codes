package main

import (
	"flag"
	"fmt"
)

// github.com/liuliqiang/blog-demos/pracitce/go-args/simple-00.go
// made by https://liqiang.io
func main() {
	var name = "zhangsan"
	flag.StringVar(&name, "name", name, "only name")
	flag.Parse()

	fmt.Printf("name: %s\n", name)
}
