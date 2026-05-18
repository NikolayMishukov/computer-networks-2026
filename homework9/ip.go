package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Ошибка при получении интерфейсов:", err)
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					fmt.Printf("Интерфейс: %s\n", iface.Name)
					fmt.Printf("IP-адрес: %s\n", ipnet.IP.String())
					fmt.Printf("Маска сети: %s\n", net.IP(ipnet.Mask).String())
					fmt.Println("---------------------------")
				}
			}
		}
	}
}
