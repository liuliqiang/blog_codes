package main

import (
	"os"
	"fmt"
)

func main() {
	rootPath := "/tmp/golang"
	pathMode := os.FileMode(0777)
	fileMode := os.FileMode(0666)

	// 目录操作
	err := os.Mkdir(rootPath, pathMode)
	if err != nil {
		fmt.Println("mkdir root failed")
		return
	}

	rrPath := rootPath + "/test1/test2"
	err = os.MkdirAll(rrPath, pathMode)
	if err != nil {
		fmt.Println("mkdir ", rrPath, " fialed!")
		return
	}

	err = os.Remove(rootPath)
	if err != nil {
		fmt.Println(err)
	}

	err = os.RemoveAll(rootPath)
	if err != nil {
		fmt.Println(err)
	}


	// 文件操作
	userFile := "/tmp/test.txt"
	fout, err := os.OpenFile(userFile, os.O_WRONLY | os.O_APPEND | os.O_CREATE, fileMode)
	if err != nil {
		fmt.Println(userFile, err)
		return
	}

	defer fout.Close()

	for i := 0; i < 10; i++ {
		fout.WriteString("Just a test!\r\n")
		fout.Write([]byte("Just a test!\r\n"))
	}
}

