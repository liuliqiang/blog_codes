package main

import (
	"os"
	"fmt"
)

func main() {
	userFile := "/tmp/test.txt"
	f1, err := os.Open(userFile)
	if err != nil {
		fmt.Println(userFile, err)
		return
	}

	defer f1.Close()

	buf := make([]byte, 1024)
	for {
		n, _ := f1.Read(buf)
		if 0 == n {
			break
		}
		os.Stdout.Write(buf[:n])
	}
}

