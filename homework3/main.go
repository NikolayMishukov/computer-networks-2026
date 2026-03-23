package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "Запуск: main <port>")
		return
	}
	port := flag.Args()[0]

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка запуска сервера:", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Сервер запущен на http://localhost:%s\r\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Ошибка подключения:", err)
			return
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Ошибка чтения запроса:", err)
		return
	}

	fmt.Println("Запрос:", requestLine)

	parts := strings.Split(requestLine, " ")
	if len(parts) < 2 {
		return
	}

	if parts[1] == "/" {
		parts[1] = "/index.html"
	}
	fileName := "./assets" + parts[1]

	contentType := "text/plain; charset=utf-8"
	if strings.HasSuffix(fileName, ".html") {
		contentType = "text/html; charset=utf-8"
	}

	data, err := os.ReadFile(fileName)

	if err != nil {
		response := "HTTP/1.1 404 Not Found\r\n"
		response += "Content-Type: text/plain\r\n"
		response += "\r\n"
		response += "404 Not Found"

		conn.Write([]byte(response))
		return
	}

	response := "HTTP/1.1 200 OK\r\n"
	response += "Content-Type: " + contentType + "\r\n"
	response += fmt.Sprintf("Content-Length: %d\r\n", len(data))
	response += "\r\n"

	conn.Write([]byte(response))
	conn.Write(data)
}
