package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var name = "zhangsan"
	flag.StringVar(&name, "name", name, "only name")
	var verbose = false
	flag.BoolVar(&verbose, "verbose", verbose, "only name")

	os.Args = append(os.Args, "-name", "lisi")
	flag.Parse()
	if verbose {
		flag.Usage()
		return
	}


	println("name: " + name)
}
