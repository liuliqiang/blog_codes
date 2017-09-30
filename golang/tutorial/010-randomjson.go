package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var s interface{}
	str := `{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`
	json.Unmarshal([]byte(str), &s)
	fmt.Println(s)
	m, ok := s.(map[string]interface{})
	if ok {
		for k, v := range m {
			switch vv := v.(type) {
			case string:
				fmt.Println(k, "is string", vv)
			case int:
				fmt.Println(k, "is int", vv)
			case float64:
				fmt.Println(k, "is float", vv)
			case []interface{}:
				fmt.Println(k, "is an array:")
				for i, u := range vv {
					fmt.Println(i, u)
				}
			default:
				fmt.Println(k, "is of a type I  don't know how to handle")
			}
		}
	}
}
