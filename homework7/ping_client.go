package main

import (
	"flag"
	"fmt"
	"net"
	"time"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	ip := flag.String("ip", "127.0.0.1", "ip to listen on")
	flag.Parse()

	serverAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", *ip, *port))
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Ошибка подключения:", err)
		return
	}
	defer func() { _ = conn.Close() }()

	var rtts []time.Duration
	lostPackets := 0

	fmt.Println("Запуск UDP Ping клиента...")

	for i := 1; i <= 10; i++ {
		startTime := time.Now()
		message := fmt.Sprintf("Ping %d %s", i, startTime.Format("15:04:05"))

		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Ошибка отправки:", err)
			continue
		}

		err = conn.SetReadDeadline(time.Now().Add(time.Second))
		if err != nil {
			fmt.Println("Ошибка установки ограничения времени:", err)
			continue
		}

		buf := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buf)

		if err != nil {
			fmt.Printf("Запрос %d: Request timed out\n", i)
			lostPackets++
		} else {
			rtt := time.Since(startTime)
			rtts = append(rtts, rtt)
			fmt.Printf("Ответ от %s: %s | RTT: %.4f сек\n",
				serverAddr, string(buf[:n]), rtt.Seconds())
		}
	}

	printStats(rtts, lostPackets, 10)
}

func printStats(rtts []time.Duration, lost int, total int) {
	fmt.Println("\n--- Статистика Ping ---")
	if len(rtts) == 0 {
		fmt.Println("Нет данных для расчета RTT.")
	} else {
		minRTT, maxRTT := rtts[0], rtts[0]
		var sumRTT time.Duration
		for _, r := range rtts {
			if r < minRTT {
				minRTT = r
			}
			if r > maxRTT {
				maxRTT = r
			}
			sumRTT += r
		}
		avgRTT := sumRTT / time.Duration(len(rtts))

		fmt.Printf("Минимальный RTT: %.4f сек\n", minRTT.Seconds())
		fmt.Printf("Максимальный RTT: %.4f сек\n", maxRTT.Seconds())
		fmt.Printf("Средний RTT:     %.4f сек\n", avgRTT.Seconds())
	}
	lossPercent := (float64(lost) / float64(total)) * 100
	fmt.Printf("Потеря пакетов:  %.1f%%\n", lossPercent)
}
