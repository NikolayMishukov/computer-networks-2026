package main

import (
	"fmt"
	"net"
	"os/exec"

	"golang.org/x/text/encoding/charmap"
)

func handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Ошибка чтения:", err)
		return
	}

	command := string(buffer[:n])
	fmt.Println("Получена команда:", command)

	cmd := exec.Command("cmd", "/C", command)

	output, err := cmd.CombinedOutput()
	decoder := charmap.CodePage866.NewDecoder()
	utf8Output, err := decoder.Bytes(output)
	if err != nil {
		output = append(output, []byte("\nОшибка: "+err.Error())...)
	}
	_, _ = conn.Write(utf8Output)
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Println("Сервер запущен на порту 8080...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка подключения:", err)
			continue
		}

		go handleConnection(conn)
	}
}
