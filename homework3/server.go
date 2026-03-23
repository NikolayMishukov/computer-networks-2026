package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		fmt.Fprintln(os.Stderr, "Запуск: server <port> <concurrency_level>")
		return
	}
	port := flag.Args()[0]
	concurrencyLevel, err := strconv.Atoi(flag.Args()[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Concurrency level must be an integer:", err)
		return
	}

	var mutex sync.Mutex
	cond := sync.NewCond(&mutex)
	goroutines := 0

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

		go func() {
			mutex.Lock()
			for goroutines == concurrencyLevel {
				cond.Wait()
			}
			goroutines++
			mutex.Unlock()

			handleConnection(conn)

			mutex.Lock()
			goroutines--
			cond.Signal()
			mutex.Unlock()
		}()
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
