package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	for idx, arg := range os.Args {
		fmt.Printf("args[%d]: %s\n", idx, arg)
	}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GO") {
			fmt.Println(env)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println("cwd: " + cwd)
}
