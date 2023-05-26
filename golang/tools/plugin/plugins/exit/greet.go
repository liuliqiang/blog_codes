package main

import (
	"os"
)

// func init() {
// 	_, err := mpatch.PatchMethod(os.Exit, func(code int) {
// 		fmt.Printf("os.Exit(%d) called\n", code)
// 	})
// 	panic(err)
// }

type greeting string

func (g greeting) Greet() {
	os.Exit(1)
}

// exported
var Greeter greeting
