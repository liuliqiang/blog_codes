package main

import (
	"encoding/xml"
	"os"
	"fmt"
	"io/ioutil"
)

type Recurlyservers struct {
	XMLName xml.Name `xml:"servers"`
	Version string `xml:"version,attr"`
	Svs []server `xml:"server"`
	Description string `xml:",innerxml"`
}


type server struct {
	ServerName string `xml:"serverName"`
	ServerIP string `xml:"serverIP"`
}


type Servers struct {
	XMLName xml.Name `xml:"servers"`
	Version string `xml:"version,attr"`
	Svs []server `xml:"server"`
}


func main() {
	file, err := os.Open("conf/008.xml")
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	v := Recurlyservers{}
	err = xml.Unmarshal(data, &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	fmt.Println(v)


	// write out xml
	fmt.Println("=============================" )
	sv := &Servers{Version: "1"}
	sv.Svs = append(sv.Svs, server{"ABCD", "127.0.0.1"})
	sv.Svs = append(sv.Svs, server{"DCBA", "127.0.0.1"})
	output, err := xml.MarshalIndent(sv, " ", "  ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	os.Stdout.Write([]byte(xml.Header))
	os.Stdout.Write(output)
}
