package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"plugin"
	"time"
)

type Greeter interface {
	Greet()
}

func main() {
	// determine module to load
	lang := "english"
	flag.StringVar(&lang, "lang", lang, "language for greet.")
	flag.Parse()

	var mod string
	switch lang {
	case "english":
		mod = "./eng/eng.so"
	case "chinese":
		mod = "./chi/chi.so"
	case "swedish":
		mod = "./swe/swe.so"
	default:
		fmt.Println("don't speak that language")
		os.Exit(1)
	}

	greeter := lookupGreeter(mod)
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		mod = "./eng/eng.so"
		greeter = lookupGreeter(mod)
		writer.Write([]byte("ok"))
	})
	go func() {
		http.ListenAndServe(":8123", nil)
	}()

	for {
		// 4. use the module
		greeter.Greet()
		time.Sleep(time.Second)
	}

}

func lookupGreeter(mod string) Greeter {
	// load module
	// 1. open the so file to load the symbols
	plug, err := plugin.Open(mod)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// 2. look up a symbol (an exported function or variable)
	// in this case, variable Greeter
	symGreeter, err := plug.Lookup("Greeter")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// 3. Assert that loaded symbol is of a desired type
	// in this case interface type Greeter (defined above)
	var greeter Greeter
	greeter, ok := symGreeter.(Greeter)
	if !ok {
		fmt.Println("unexpected type from module symbol")
		os.Exit(1)
	}
	return greeter
}
