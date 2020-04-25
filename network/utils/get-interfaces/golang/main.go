package main

import (
	"log"
	"net"
)

func main() {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	// handle err
	for _, i := range ifaces {
		var name = i.Name
		addrs, err := i.Addrs()
		if err != nil {
			panic(err)
		}
		// handle err
		for _, addr := range addrs {
			var (
				ip net.IP
			)
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// process IP address
			log.Printf("%s:%s = %s\n", addr.Network(), name, ip)
		}
	}
}
