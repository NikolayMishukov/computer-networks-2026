// server.go
package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	broadcastAddr := net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 9999,
	}

	conn, err := net.DialUDP("udp", nil, &broadcastAddr)
	if err != nil {
		log.Fatal("Failed to make UDP connection:", err)
	}
	defer func() { _ = conn.Close() }()

	fmt.Println("UDP сервер запущен...")
	for {
		currentTime := time.Now().Format("2006.01.02 15:04:05")

		_, err := conn.Write([]byte(currentTime))
		if err != nil {
			fmt.Println("Ошибка отправки:", err)
		} else {
			fmt.Println("Отправлено:", currentTime)
		}

		time.Sleep(1 * time.Second)
	}
}
