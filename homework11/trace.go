package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	ICMP_ECHO_REPLY    = 0
	ICMP_ECHO_REQUEST  = 8
	ICMP_TIME_EXCEEDED = 11
)

type ICMPHeader struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	SeqNum   uint16
}

type Result struct {
	Num   int
	IP    string
	Host  string
	RTT   []time.Duration
	Final bool
}

func checksum(data []byte) uint16 {
	var sum uint32
	length := len(data)

	for i := 0; i < length-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i:]))
	}

	if length%2 == 1 {
		sum += uint32(data[length-1]) << 8
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16

	return uint16(^sum)
}

func createICMPPacket(id uint16, seqNum uint16) []byte {
	packet := make([]byte, 8)
	packet[0] = ICMP_ECHO_REQUEST
	packet[1] = 0
	packet[2] = 0
	packet[3] = 0
	binary.BigEndian.PutUint16(packet[4:6], id)
	binary.BigEndian.PutUint16(packet[6:8], seqNum)

	checksumVal := checksum(packet)
	binary.BigEndian.PutUint16(packet[2:4], checksumVal)

	return packet
}

func parseICMPResponse(data []byte, expectedID uint16) (*ICMPHeader, bool) {
	if len(data) >= 20 && (data[0]>>4) == 4 {
		ipHeaderLen := int(data[0]&0x0F) * 4
		if len(data) >= ipHeaderLen {
			data = data[ipHeaderLen:]
		}
	}

	if len(data) < 8 {
		return nil, false
	}

	header := &ICMPHeader{
		Type:     data[0],
		Code:     data[1],
		Checksum: binary.BigEndian.Uint16(data[2:4]),
	}

	if header.Type == ICMP_ECHO_REPLY {
		header.ID = binary.BigEndian.Uint16(data[4:6])
		header.SeqNum = binary.BigEndian.Uint16(data[6:8])
		return header, header.ID == expectedID
	}

	if header.Type == ICMP_TIME_EXCEEDED {
		if len(data) < 36 {
			return nil, false
		}

		innerIPHeaderLen := int(data[8]&0x0F) * 4
		icmpHeaderOffset := 8 + innerIPHeaderLen

		if len(data) >= icmpHeaderOffset+8 {
			innerICMPData := data[icmpHeaderOffset:]

			if innerICMPData[0] == ICMP_ECHO_REQUEST {
				header.ID = binary.BigEndian.Uint16(innerICMPData[4:6])
				header.SeqNum = binary.BigEndian.Uint16(innerICMPData[6:8])
				return header, header.ID == expectedID
			}
		}
	}

	return header, false
}

func sendICMPPacket(conn *net.IPConn, destAddr *net.IPAddr, ttl int, id uint16, seqNum uint16) (time.Time, error) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		return time.Time{}, err
	}

	var setOptErr error
	err = syscallConn.Control(func(fd uintptr) {
		setOptErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	})
	if err != nil {
		return time.Time{}, err
	}
	if setOptErr != nil {
		return time.Time{}, setOptErr
	}

	packet := createICMPPacket(id, seqNum)
	start := time.Now()
	_, err = conn.WriteTo(packet, destAddr)
	return start, err
}

func receiveICMPResponse(conn *net.IPConn, timeout time.Duration, expectedID uint16, start time.Time) (*ICMPHeader, string, time.Duration, error) {
	buffer := make([]byte, 1500)
	endTime := time.Now().Add(timeout)

	for {
		remaining := time.Until(endTime)
		if remaining <= 0 {
			return nil, "", 0, os.ErrDeadlineExceeded
		}

		_ = conn.SetReadDeadline(endTime)
		n, addr, err := conn.ReadFrom(buffer)

		rtt := time.Since(start)

		if err != nil {
			return nil, "", 0, err
		}

		header, isOurs := parseICMPResponse(buffer[:n], expectedID)
		if isOurs {
			srcIP := ""
			if addr != nil {
				srcIP = addr.String()
			}
			return header, srcIP, rtt, nil
		}
	}
}

func lookupHost(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}

func traceRoute(destHost string, maxHops int, numProbes int, timeout time.Duration) ([]Result, error) {
	destAddr, err := net.ResolveIPAddr("ip4", destHost)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host %s: %v", destHost, err)
	}

	fmt.Printf("Tracing route to %s [%s]\n", destHost, destAddr.IP.String())
	fmt.Printf("over a maximum of %d hops:\n\n", maxHops)

	conn, err := net.ListenIP("ip4:icmp", &net.IPAddr{IP: net.IPv4zero})
	if err != nil {
		return nil, fmt.Errorf("failed to create ICMP socket: %v\n", err)
	}
	defer func() { _ = conn.Close() }()

	var results []Result
	processID := uint16(os.Getpid() & 0xFFFF)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	for ttl := 1; ttl <= maxHops; ttl++ {
		select {
		case <-sigCh:
			fmt.Println("\nTrace aborted by user.")
			return results, nil
		default:
		}

		fmt.Printf("%2d  ", ttl)

		var rtts []time.Duration
		var hopIP string
		isFinal := false
		gotResponse := false

		for probe := 0; probe < numProbes; probe++ {
			seqNum := uint16(ttl*100 + probe)

			start, err := sendICMPPacket(conn, destAddr, ttl, processID, seqNum)
			if err != nil {
				fmt.Print("! ")
				continue
			}

			header, srcIP, rtt, err := receiveICMPResponse(conn, timeout, processID, start)
			if err != nil {
				if os.IsTimeout(err) {
					fmt.Print("* ")
				} else {
					fmt.Print("? ")
				}
				continue
			}

			if header.Type == ICMP_TIME_EXCEEDED || header.Type == ICMP_ECHO_REPLY {
				if !gotResponse {
					hopIP = srcIP
					gotResponse = true
				}

				rtts = append(rtts, rtt)
				fmt.Printf("%.1fms ", float64(rtt.Microseconds())/1000.0)

				if header.Type == ICMP_ECHO_REPLY && srcIP == destAddr.IP.String() {
					isFinal = true
				}
			} else {
				fmt.Print("* ")
			}
		}

		if gotResponse {
			fmt.Printf(" %s", hopIP)
			hopResult := Result{
				Num:   ttl,
				IP:    hopIP,
				RTT:   rtts,
				Final: isFinal,
			}
			results = append(results, hopResult)
		} else {
			results = append(results, Result{Num: ttl, IP: "*", Final: false})
		}
		fmt.Println()

		if isFinal {
			break
		}
	}

	return results, nil
}

func printResults(results []Result) {
	fmt.Println("\nRoute Information:")

	var wg sync.WaitGroup
	var mu sync.Mutex
	hostnames := make(map[string]string)

	for _, hop := range results {
		if hop.IP != "*" {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				name := lookupHost(ip)
				if name != "" {
					mu.Lock()
					hostnames[ip] = name
					mu.Unlock()
				}
			}(hop.IP)
		}
	}
	wg.Wait()

	for _, hop := range results {
		if hop.IP == "*" {
			fmt.Printf("%2d  Request timed out.\n", hop.Num)
			continue
		}

		fmt.Printf("%2d  %s", hop.Num, hop.IP)
		if name, ok := hostnames[hop.IP]; ok {
			fmt.Printf(" [%s]", name)
		}

		if len(hop.RTT) > 0 {
			fmt.Print("  RTTs:")
			for _, rtt := range hop.RTT {
				fmt.Printf(" %.1fms", float64(rtt.Microseconds())/1000.0)
			}
		}

		if hop.Final {
			fmt.Print(" (destination reached)")
		}
		fmt.Println()
	}
}

func main() {
	var (
		maxHops   int
		numProbes int
		timeout   int
	)

	flag.IntVar(&maxHops, "m", 30, "Maximum number of hops")
	flag.IntVar(&numProbes, "n", 3, "Number of probes per hop")
	flag.IntVar(&timeout, "t", 500, "Timeout in milliseconds")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Printf("Usage: %s [-m max_hops] [-n num_probes] [-t timeout_ms] host\n", os.Args[0])
		fmt.Println("Example: sudo ./tracert -n 3 google.com")
		os.Exit(1)
	}

	destHost := flag.Arg(0)

	fmt.Printf("Starting trace to %s\n", destHost)
	fmt.Printf("Parameters: max_hops=%d, probes=%d, timeout=%dms\n\n", maxHops, numProbes, timeout)

	results, err := traceRoute(destHost, maxHops, numProbes, time.Duration(timeout)*time.Millisecond)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	printResults(results)
}
