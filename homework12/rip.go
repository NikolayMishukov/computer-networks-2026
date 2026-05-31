package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"time"
)

type Route struct {
	Destination string
	NextHop     string
	Metric      int
}

type RouterConfig struct {
	IP        string   `json:"ip"`
	Port      int      `json:"port"`
	Neighbors []string `json:"neighbors"`
}

type RIPMessage struct {
	SenderIP string         `json:"sender_ip"`
	Routes   map[string]int `json:"routes"`
}

type Router struct {
	IP         string
	Port       int
	Neighbors  []string
	RoutingTbl map[string]Route
	IPToPort   map[string]int

	UDPConn    *net.UDPConn
	CmdChan    chan string
	ResultChan chan bool
	LogChan    chan string
}

func NewRouter(cfg RouterConfig, ipToPort map[string]int) *Router {
	r := &Router{
		IP:         cfg.IP,
		Port:       cfg.Port,
		Neighbors:  cfg.Neighbors,
		RoutingTbl: make(map[string]Route),
		IPToPort:   ipToPort,
		CmdChan:    make(chan string),
		ResultChan: make(chan bool),
		LogChan:    make(chan string),
	}
	r.RoutingTbl[r.IP] = Route{Destination: r.IP, NextHop: r.IP, Metric: 0}
	return r
}

func (r *Router) Run() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", r.Port))
	if err != nil {
		log.Fatalf("Ошибка адреса UDP для %s: %v", r.IP, err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Ошибка открытия порта для %s: %v", r.IP, err)
	}
	r.UDPConn = conn
	defer func() { _ = conn.Close() }()

	for cmd := range r.CmdChan {
		switch cmd {
		case "SEND":
			r.broadcastRoutes()
			r.ResultChan <- true

		case "PROCESS":
			changed := r.processInbox()
			r.ResultChan <- changed

		case "LOG_STEP":
			r.LogChan <- r.formatTable(fmt.Sprintf("Simulation step %%d of router %s", r.IP))

		case "LOG_FINAL":
			r.LogChan <- r.formatTable(fmt.Sprintf("Final state of router %s table:", r.IP))

		case "STOP":
			return
		}
	}
}

func (r *Router) broadcastRoutes() {
	msg := RIPMessage{
		SenderIP: r.IP,
		Routes:   make(map[string]int),
	}
	for dest, route := range r.RoutingTbl {
		msg.Routes[dest] = route.Metric
	}
	msgBytes, _ := json.Marshal(msg)

	for _, neighbor := range r.Neighbors {
		port := r.IPToPort[neighbor]
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))
		_, _ = r.UDPConn.WriteToUDP(msgBytes, addr)
	}
}

func (r *Router) processInbox() bool {
	changed := false
	_ = r.UDPConn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	buf := make([]byte, 4096)

	for {
		n, _, err := r.UDPConn.ReadFromUDP(buf)
		if err != nil {
			break
		}

		var rcvMsg RIPMessage
		_ = json.Unmarshal(buf[:n], &rcvMsg)
		neighborIP := rcvMsg.SenderIP

		for dest, metric := range rcvMsg.Routes {
			newMetric := metric + 1
			if newMetric >= 16 {
				continue
			}

			existingRoute, exists := r.RoutingTbl[dest]
			if !exists || newMetric < existingRoute.Metric {
				r.RoutingTbl[dest] = Route{
					Destination: dest,
					NextHop:     neighborIP,
					Metric:      newMetric,
				}
				changed = true
			}
		}
	}
	return changed
}

func (r *Router) formatTable(titleTemplate string) string {
	res := titleTemplate + "\n"
	res += fmt.Sprintf("%-16s %-19s %-16s %8s\n", "[Source IP]", "[Destination IP]", "[Next Hop]", "[Metric]")

	var destinations []string
	for dest := range r.RoutingTbl {
		if dest != r.IP {
			destinations = append(destinations, dest)
		}
	}
	sort.Strings(destinations)

	for _, dest := range destinations {
		route := r.RoutingTbl[dest]
		res += fmt.Sprintf("%-16s %-19s %-16s %8d\n", r.IP, dest, route.NextHop, route.Metric)
	}
	return res
}

func main() {
	createDefaultConfigIfMissing("assets/network.json")
	file, err := os.ReadFile("assets/network.json")
	if err != nil {
		log.Fatalf("Ошибка чтения network.json: %v", err)
	}

	var configs []RouterConfig
	_ = json.Unmarshal(file, &configs)

	ipToPort := make(map[string]int)
	for _, cfg := range configs {
		ipToPort[cfg.IP] = cfg.Port
	}

	var routers []*Router
	for _, cfg := range configs {
		r := NewRouter(cfg, ipToPort)
		routers = append(routers, r)
		go r.Run()
	}

	time.Sleep(100 * time.Millisecond)

	step := 1
	for {
		for _, r := range routers {
			r.CmdChan <- "SEND"
		}
		for _, r := range routers {
			<-r.ResultChan
		}

		time.Sleep(50 * time.Millisecond)

		anyChanged := false
		for _, r := range routers {
			r.CmdChan <- "PROCESS"
		}
		for _, r := range routers {
			changed := <-r.ResultChan
			if changed {
				anyChanged = true
			}
		}

		if anyChanged {
			for _, r := range routers {
				r.CmdChan <- "LOG_STEP"
				logMsg := <-r.LogChan
				fmt.Printf(logMsg, step)
			}
			fmt.Println("--------------------------------------------------")
			step++
		} else {
			break
		}
	}

	fmt.Println("\n=== CONVERGENCE REACHED ===")
	for _, r := range routers {
		r.CmdChan <- "LOG_FINAL"
		fmt.Print(<-r.LogChan)
		r.CmdChan <- "STOP"
	}
}

func createDefaultConfigIfMissing(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		defaultConfig := `[
			{"ip": "198.71.243.61", "port": 10001, "neighbors": ["42.162.54.248"]},
			{"ip": "42.162.54.248", "port": 10002, "neighbors": ["198.71.243.61", "157.105.66.180", "229.28.61.15"]},
			{"ip": "157.105.66.180", "port": 10003, "neighbors": ["42.162.54.248", "122.136.243.149"]},
			{"ip": "229.28.61.15", "port": 10004, "neighbors": ["42.162.54.248"]},
			{"ip": "122.136.243.149", "port": 10005, "neighbors": ["157.105.66.180"]}
		]`
		_ = os.WriteFile(filename, []byte(defaultConfig), 0644)
	}
}
