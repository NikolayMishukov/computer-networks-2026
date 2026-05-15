package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

const (
	PacketSize = 512
	Timeout    = 500 * time.Millisecond
	LossRate   = 0.3
)

type Packet struct {
	SeqNum   uint32
	Checksum uint16
	IsAck    bool
	Payload  []byte
}

func (p *Packet) Serialize() []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, p.SeqNum)
	_ = binary.Write(buf, binary.BigEndian, p.Checksum)
	_ = binary.Write(buf, binary.BigEndian, p.IsAck)
	buf.Write(p.Payload)
	return buf.Bytes()
}

func Deserialize(data []byte) *Packet {
	if len(data) < 7 {
		return nil
	}
	p := &Packet{}
	p.SeqNum = binary.BigEndian.Uint32(data[:4])
	p.Checksum = binary.BigEndian.Uint16(data[4:6])
	p.IsAck = data[6] != 0
	p.Payload = data[7:]
	return p
}

func ComputeChecksum(data []byte) uint16 {
	var sum uint16 = 0
	for i, b := range data {
		if i%2 == 0 {
			sum += uint16(b)
		} else {
			sum += uint16(b) << 8
		}
	}
	return 0xFFFF - sum
}

func sendWithLoss(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	if rand.Float64() > LossRate {
		_, _ = conn.WriteToUDP(data, addr)
	} else {
		fmt.Printf("  Пакет был потерян при передаче\n")
	}
}

func SendFile(conn *net.UDPConn, addr *net.UDPAddr, filename string) error {
	file, _ := os.Open(filename)
	defer func() { _ = file.Close() }()

	buffer := make([]byte, PacketSize)
	seqNum := uint32(0)
	final := false

	for !final {
		n, err := file.Read(buffer)
		var packet Packet
		if err == io.EOF {
			packet = Packet{SeqNum: seqNum, IsAck: false, Payload: []byte("EOF")}
			final = true
		} else {
			if err != nil {
				return err
			}

			payload := buffer[:n]
			packet = Packet{
				SeqNum:  seqNum,
				IsAck:   false,
				Payload: payload,
			}
		}

		packet.Checksum = ComputeChecksum(packet.Payload)
		encoded := packet.Serialize()

		for {
			fmt.Printf("Отправка пакета %d (размер %d)\n", seqNum, n)
			sendWithLoss(conn, addr, encoded)

			_ = conn.SetReadDeadline(time.Now().Add(Timeout))
			ackBuf := make([]byte, 20)
			nAck, _, err := conn.ReadFromUDP(ackBuf)

			if err != nil {
				fmt.Printf("  Истекло время ожидания ACK %d, повтор\n", seqNum)
				continue
			}

			ackPkt := Deserialize(ackBuf[:nAck])
			if ackPkt != nil && ackPkt.IsAck && ackPkt.SeqNum == seqNum {
				fmt.Printf("Получен ACK %d\n", seqNum)
				seqNum = 1 - seqNum
				break
			}
		}
	}
	fmt.Println("Файл успешно передан")
	return nil
}

func ReceiveFile(conn *net.UDPConn, saveAs string) {
	file, _ := os.Create(saveAs)
	defer func() { _ = file.Close() }()

	expectedSeq := uint32(0)

	for {
		_ = conn.SetReadDeadline(time.Time{})
		buf := make([]byte, PacketSize+20)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		pkt := Deserialize(buf[:n])
		if pkt == nil {
			continue
		}

		if !pkt.IsAck && ComputeChecksum(pkt.Payload) != pkt.Checksum {
			fmt.Printf("  Ошибка контрольной суммы пакета %d\n", pkt.SeqNum)
			continue
		}

		if string(pkt.Payload) == "EOF" {
			ack := Packet{SeqNum: pkt.SeqNum, IsAck: true}
			_, _ = conn.WriteToUDP(ack.Serialize(), addr)
			break
		}

		if pkt.SeqNum == expectedSeq {
			fmt.Printf("Получен пакет %d, записываем...\n", pkt.SeqNum)
			_, _ = file.Write(pkt.Payload)

			ack := Packet{SeqNum: expectedSeq, IsAck: true}
			sendWithLoss(conn, addr, ack.Serialize())

			expectedSeq = 1 - expectedSeq
		} else {
			fmt.Printf("Повтор пакета %d, переотправка ACK\n", pkt.SeqNum)
			ack := Packet{SeqNum: pkt.SeqNum, IsAck: true}
			sendWithLoss(conn, addr, ack.Serialize())
		}
	}
	fmt.Printf("Файл сохранен как %s\n", saveAs)
}

func main() {
	ip := flag.String("ip", "127.0.0.1", "ip to listen on")
	port := flag.Int("port", 8080, "port for forward messages")
	duplexPort := flag.Int("duplex-port", 8081, "port for backward messages")
	mode := flag.String("mode", "server", "server or client")
	fileToSend := flag.String("file", "assets/alice.txt", "file to transfer")
	flag.Parse()

	addr := net.UDPAddr{
		Port: *port,
		IP:   net.ParseIP(*ip),
	}
	duplexAddr := net.UDPAddr{
		Port: *duplexPort,
		IP:   net.ParseIP(*ip),
	}

	if *mode == "server" {
		conn, _ := net.ListenUDP("udp", &addr)
		defer func() { _ = conn.Close() }()
		fmt.Println("Сервер запущен, ожидание файла от клиента...")
		ReceiveFile(conn, "assets/received_from_client.txt")

		fmt.Println("\n--- Теперь сервер отправляет файл клиенту (Дуплекс) ---")
		err := SendFile(conn, &duplexAddr, *fileToSend)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		conn, _ := net.ListenUDP("udp", &duplexAddr)
		defer func() { _ = conn.Close() }()

		fmt.Println("Клиент отправляет файл серверу...")
		err := SendFile(conn, &addr, *fileToSend)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("\n--- Теперь клиент принимает файл от сервера (Дуплекс) ---")
		ReceiveFile(conn, "assets/received_from_server.txt")
	}
}
