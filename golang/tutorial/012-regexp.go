package main

import (
	"regexp"
	"os"
	"fmt"
)

func isIpAddr(ip string) bool {
	if m, _ := regexp.MatchString("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}$", ip); !m {
		return false
	}
	return true
}


func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: regexp [string]")
	} else if m, _ := regexp.MatchString("^[0-9]+$", os.Args[1]); m {
		fmt.Println("数字")
	} else {
		fmt.Println("不是数字")
	}
}
