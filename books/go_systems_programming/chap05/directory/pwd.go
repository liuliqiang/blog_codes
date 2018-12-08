package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	var realPath bool
	flag.BoolVar(&realPath, "P", false, "real path for symbolic file")
	flag.Parse()

	pwd, err := os.Getwd()
	if err == nil {
		fmt.Println(pwd)
	} else {
		fmt.Println("Error:", err)
	}

	if realPath {
		if fileinfo, err := os.Lstat(pwd); err != nil {
			log.Fatal(err)
		} else {
			if fileinfo.Mode()&os.ModeSymlink != 0 {
				realpath, err := filepath.EvalSymlinks(pwd)
				if err == nil {
					fmt.Println(realpath)
				}
			}
		}
	}
}
