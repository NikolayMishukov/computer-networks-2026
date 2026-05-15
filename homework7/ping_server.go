package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"strings"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	ip := flag.String("ip", "127.0.0.1", "ip to listen on")
	flag.Parse()

	addr := net.UDPAddr{
		Port: *port,
		IP:   net.ParseIP(*ip),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		return
	}
	defer func() { _ = conn.Close() }()

	fmt.Printf("UDP Ping сервер запущен на %s:%d\n", *ip, *port)

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Ошибка чтения:", err)
			continue
		}

		if rand.Intn(100) < 20 {
			fmt.Printf("Пакет от %s потерян (имитация)\n", remoteAddr)
			continue
		}

		message := string(buf[:n])
		fmt.Printf("Получено: %s от %s\n", message, remoteAddr)

		upperMessage := strings.ToUpper(message)
		_, _ = conn.WriteToUDP([]byte(upperMessage), remoteAddr)
	}
}
