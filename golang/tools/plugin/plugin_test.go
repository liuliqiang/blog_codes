package main

import (
	"fmt"
	"os"
	"plugin"
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
	greeter, err := reloadMod("./plugins/chinese/greet.so")
	if err != nil {
		panic(err)
	}

	for i := 0; i < b.N; i++ {
		greeter.Greet()
	}
}

type Greeter interface {
	Greet()
}

func reloadMod(modPath string) (g Greeter, err error) {
	plug, err := plugin.Open(modPath)
	if err != nil {
		panic(err)
	}

	symGreeter, err := plug.Lookup("Greeter")
	if err != nil {
		panic(err)
	}

	greeter, ok := symGreeter.(Greeter)
	if !ok {
		fmt.Println("unexpected type from module symbol")
		os.Exit(1)
	}

	return greeter, nil
}
