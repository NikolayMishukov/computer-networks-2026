package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	address := "[::1]:8080"
	listener, err := net.Listen("tcp6", address)
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}
	defer func() { _ = listener.Close() }()

	fmt.Printf("IPv6 сервер запущен на %s\n", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Ошибка при принятии соединения: %v", err)
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	fmt.Printf("Новое подключение от: %s\n", conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Соединение закрыто клиентом %s", conn.RemoteAddr().String())
			return
		}

		cleanMsg := strings.TrimSpace(msg)
		fmt.Printf("Получено: %s\n", cleanMsg)

		upperMsg := strings.ToUpper(cleanMsg) + "\n"

		_, err = conn.Write([]byte(upperMsg))
		if err != nil {
			log.Printf("Ошибка отправки данных: %v", err)
			return
		}
	}
}
