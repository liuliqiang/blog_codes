package main

import (
	"html/template"
	"os"
)

type Person struct {
	UserName string
}


func main() {
	// 当前对象示例
	t := template.New("Fieldname Example")
	t, _ = t.Parse("hello {{.UserName}}!")
	p := Person{"Yetship"}
	t.Execute(os.Stdout, p)
}
