package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("Ошибка подключения:", err)
		return
	}
	defer func() { _ = conn.Close() }()

	fmt.Print("Введите команду: ")
	reader := bufio.NewReader(os.Stdin)
	command, _ := reader.ReadString('\n')
	command = strings.TrimSpace(command)

	_, err = conn.Write([]byte(command))
	if err != nil {
		fmt.Println("Ошибка отправки:", err)
		return
	}

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Ошибка чтения:", err)
		return
	}

	fmt.Println("\nРезультат выполнения:")
	fmt.Println(string(buffer[:n]))
}
