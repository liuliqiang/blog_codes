package main

import (
	"encoding/json"
	"fmt"
)

type Server struct {
	Servername string `json:"server_name"`
	ServerIP string `json:"server_ip"`
}

type Serverslice struct {
	Servers []Server `json:"servers"`
}

func main() {
	var s Serverslice
	str := `{"servers":[{"server_name":"Shanghai_VPN","server_ip":"127.0.0.1"},
						{"server_name":"Beijing_VPN","server_ip":"127.0.0.2"}]}`
	json.Unmarshal([]byte(str), &s)
	fmt.Println(s)

	serA := Server{"ABCD", "127.0.0.1"}
	serB := Server{"DCBA", "127.0.0.1"}
	serS := Serverslice{[]Server{serA, serB}}
	b, err := json.Marshal(serS)
	if err != nil {
		fmt.Println("json err: ", err)
		return
	}
	fmt.Println(string(b))
}
