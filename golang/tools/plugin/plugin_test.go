package main

import (
	"fmt"
	"testing"
)

type greeter struct {
}

func (g *greeter) Greet() {
	fmt.Println("你好宇宙")
}

func BenchmarkNormalFunction(b *testing.B) {
	normalGreeter := &greeter{}
	for i := 0; i < b.N; i++ {
		normalGreeter.Greet()
	}
}

func BenchmarkPlugin(b *testing.B) {
	greeter, err := reloadMod("chinese/greet.so")
	if err != nil {
		panic(err)
	}

	for i := 0; i < b.N; i++ {
		greeter.Greet()
	}
}
