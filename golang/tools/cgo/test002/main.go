package main

//#include <stdio.h>        //  序文中可以链接标准C程序库
import "C"

func main() {
	C.puts(C.CString("Hello, Cgo\n"))
}
