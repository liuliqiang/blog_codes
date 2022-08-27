package main

import "fmt"

// created by: https://liqiang.io
func main() {
	a := []int{1, 2, 3}
	sliceModify(a)
	fmt.Printf("a: %v\n", a)

	sliceAppend(a)
	fmt.Printf("a: %v\n", a)
}

func sliceModify(a []int) {
	a[1] = 3
}

func sliceAppend(a []int) {
	a = append(a, 1)
}

func twoSlice(b [][]int) {
	fmt.Printf("in before: %p\n", &b)
	b = append(b, []int{1, 2, 3})
	fmt.Printf("in after: %p\n", &b)
}
func twoSlicePointer(b *[][]int) {
	*b = append(*b, []int{1, 2, 3})
}
