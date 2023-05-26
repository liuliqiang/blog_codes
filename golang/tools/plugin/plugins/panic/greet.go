package main

type greeting string

func (g greeting) Greet() {
	panic("I'm gone.")
}

// exported
var Greeter greeting
