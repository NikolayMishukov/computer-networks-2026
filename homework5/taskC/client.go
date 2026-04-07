// client.go
package main

import (
	"fmt"
	"net"
)

func main() {
	addr := net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 9999,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer func() { _ = conn.Close() }()

	fmt.Println("Клиент слушает...")

	buffer := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Ошибка:", err)
			continue
		}

		fmt.Printf("Получено от %s: %s\n", remoteAddr, string(buffer[:n]))
	}
}
