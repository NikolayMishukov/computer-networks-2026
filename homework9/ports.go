package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
)

func main() {
	ip := flag.String("ip", "127.0.0.1", "ip to scan on")
	from := flag.Int("from-port", 0, "port to start from")
	to := flag.Int("to-port", 0, "port to end to")
	flag.Parse()

	for port := *from; port < *to; port++ {
		address := net.JoinHostPort(*ip, strconv.Itoa(port))

		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err != nil {
			fmt.Printf("Порт %d свободен\n", port)
		} else {
			fmt.Printf("Порт %d занят\n", port)
			_ = conn.Close()
		}
	}
}
