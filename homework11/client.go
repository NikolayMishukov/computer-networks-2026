package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	address := "[::1]:8080"
	conn, err := net.Dial("tcp6", address)
	if err != nil {
		log.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer func() { _ = conn.Close() }()

	fmt.Printf("Успешно подключено к IPv6 серверу %s\n", address)
	fmt.Println("Введите текст:")

	consoleReader := bufio.NewReader(os.Stdin)
	socketReader := bufio.NewReader(conn)

	for {
		fmt.Print("> ")
		input, err := consoleReader.ReadString('\n')
		if err != nil {
			log.Fatalf("Ошибка чтения ввода: %v", err)
		}

		if strings.TrimSpace(input) == "" {
			continue
		}

		_, err = conn.Write([]byte(input))
		if err != nil {
			log.Fatalf("Ошибка отправки на сервер: %v", err)
		}

		response, err := socketReader.ReadString('\n')
		if err != nil {
			log.Fatalf("Ошибка получения ответа от сервера: %v", err)
		}

		fmt.Printf("Ответ сервера: %s", response)
	}
}
