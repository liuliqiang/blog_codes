package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var filePath string
	flag.StringVar(&filePath, "f", "", "full path of a file")
	flag.Parse()

	// todo(@liuliqiang): lstat return the symbolic info
	// stat return real file info
	fileinfo, err := os.Lstat(filePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if fileinfo.Mode()&os.ModeSymlink != 0 {
		fmt.Println(filePath, "is a symbolic link")
		realpath, err := filepath.EvalSymlinks(filePath)
		if err == nil {
			fmt.Println("Path:", realpath)
		}
	}
}
