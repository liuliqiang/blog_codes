package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"time"
)

type Greeter interface {
	Greet()
}

func main() {
	mod := flag.String("mod", "english", "module to load")
	flag.Parse()

	greeter, err := reloadMod(filepath.Join(*mod, "greet.so"))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		newMod := r.URL.Query().Get("mod")
		if newMod == "" {
			newMod = "chinese"
		}
		newGreeter, err := reloadMod(filepath.Join(newMod, "greet.so"))
		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusOK)
			return
		}
		greeter = newGreeter
	})
	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	for {
		func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("recovered from panic: %v\n", err)
				}
			}()
			greeter.Greet()
		}()
		time.Sleep(time.Second)
	}
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
