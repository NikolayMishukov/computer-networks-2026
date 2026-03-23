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
	if len(flag.Args()) < 3 {
		fmt.Fprintln(os.Stderr, "Запуск: client <server_host> <server_port> <filename>")
		return
	}
	host := flag.Args()[0]
	port := flag.Args()[1]
	filename := flag.Args()[2]

	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка подключеня: ", err)
		return
	}
	defer conn.Close()

	request := fmt.Sprintf("GET /%s HTTP/1.1\r\n", filename)
	conn.Write([]byte(request))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "Пустой ответ сервера")
		return
	}

	responseLine := scanner.Text()
	if !strings.HasPrefix(responseLine, "HTTP/1.1 200 OK") {
		fmt.Println("Ответ сервера: " + responseLine)
		return
	}

	isHeader := true
	for scanner.Scan() {
		line := scanner.Text()
		if isHeader {
			isHeader = line != ""
		} else {
			fmt.Println(line)
		}
	}
}
