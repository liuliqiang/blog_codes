package main

import "fmt"

func test(ch chan bool) {
	fmt.Println("In test")
	ch <- true
}

func main() {
	ch := make(chan bool)

	for i := 0; i < 10; i++ {
		go test(ch)
	}

	for i := 0; i < 10; i++ {
		<- ch
	}
	fmt.Println("All Done!")
}


