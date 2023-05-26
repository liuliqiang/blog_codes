package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"time"

	"github.com/undefinedlabs/go-mpatch"
)

var (
	mpatchFunc *mpatch.Patch
)

type Greeter interface {
	Greet()
}

func main() {
	mod := flag.String("mod", "english", "module to load")
	flag.Parse()

	greeter, err := reloadMod(filepath.Join("plugins", *mod, "greet.so"))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		newMod := r.URL.Query().Get("mod")
		if newMod == "" {
			newMod = "chinese"
		}
		newGreeter, err := reloadMod(filepath.Join("plugins", newMod, "greet.so"))
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

	i := 0
	for {
		func() {
			// patchFunc, err = mpatch.PatchMethod(os.Exit, func(code int) {
			// 	fmt.Printf("os.Exit(%d) called\n", code)
			// })
			// if err != nil {
			// 	panic(err)
			// }

			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("recovered from panic: %v\n", err)
				}
				// patchFunc.Unpatch()
			}()
			greeter.Greet()
			i++
			fmt.Printf("i = %d\n", i)
			if i > 5 {
				fmt.Println("I'm ready to exit")
				os.Exit(2)
			}
		}()
		time.Sleep(time.Second)
	}
}

func reloadMod(modPath string) (g Greeter, err error) {
	if mpatchFunc != nil {
		mpatchFunc.Unpatch()
		mpatchFunc = nil
	}

	mpatchFunc, err = mpatch.PatchMethod(os.Exit, func(code int) {
		fmt.Printf("os.Exit(%d) called\n", code)
	})
	if err != nil {
		return nil, fmt.Errorf("patch os.Exit failed: %w", err)
	}

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
