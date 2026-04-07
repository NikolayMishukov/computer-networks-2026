package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SMTP struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"smtp"`
	User struct {
		Email string `yaml:"email"`
	} `yaml:"user"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func readResponse(reader *bufio.Reader) (string, error) {
	var response strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		response.WriteString(line)

		if len(line) < 4 || line[3] != '-' {
			break
		}
	}
	return response.String(), nil
}

func sendCommand(conn net.Conn, reader *bufio.Reader, cmd string) (string, error) {
	fmt.Printf("CLIENT: %s\n", cmd)
	_, err := fmt.Fprintf(conn, "%s\r\n", cmd)
	if err != nil {
		return "", err
	}

	resp, err := readResponse(reader)
	if err != nil {
		return "", err
	}

	fmt.Printf("SERVER: %s", resp)
	return resp, nil
}

func handleError(msg string, err error) {
	if err != nil {
		log.Fatal(msg, err)
	}
}

func main() {
	to := flag.String("to", "", "Recipient email")
	server := flag.String("server", "sandbox.smtp.mailtrap.io", "SMTP server")
	subject := flag.String("subject", "Test Email", "Subject")
	bodyPath := flag.String("body", "assets/body.txt", "Path to file with body")
	imagePath := flag.String("img", "", "Path to image")
	configPath := flag.String("config", "assets/config.yaml", "Path to config")
	flag.Parse()

	if *to == "" {
		log.Fatal("Recipient email is required")
	}
	body, err := os.ReadFile(*bodyPath)
	handleError("Failed to read body:", err)
	var imageData []byte
	if *imagePath != "" {
		imageData, err = os.ReadFile(*imagePath)
		handleError("Failed to read image:", err)
	}
	cfg, err := loadConfig(*configPath)
	handleError("Failed to load config:", err)

	conn, err := net.Dial("tcp", *server+":"+strconv.Itoa(cfg.SMTP.Port))
	handleError("Failed to create tcp connection:", err)
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	resp, err := readResponse(reader)
	handleError("Failed to read first server message:", err)
	fmt.Printf("SERVER: %s", resp)

	_, err = sendCommand(conn, reader, "EHLO localhost")
	handleError("Failed to send EHLO:", err)

	_, err = sendCommand(conn, reader, "AUTH LOGIN")
	handleError("Failed to login:", err)

	_, err = sendCommand(conn, reader,
		base64.StdEncoding.EncodeToString([]byte(cfg.SMTP.Username)))
	handleError("Failed to send username:", err)

	_, err = sendCommand(conn, reader,
		base64.StdEncoding.EncodeToString([]byte(cfg.SMTP.Password)))
	handleError("Failed to send password:", err)

	_, err = sendCommand(conn, reader,
		fmt.Sprintf("MAIL FROM:<%s>", cfg.User.Email))
	handleError("Failed to set sender:", err)

	_, err = sendCommand(conn, reader,
		fmt.Sprintf("RCPT TO:<%s>", *to))
	handleError("Failed to set recipient:", err)

	_, err = sendCommand(conn, reader, "DATA")
	handleError("Failed to set content:", err)

	boundary := "my-unique-boundary-192749812"

	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", cfg.User.Email))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", *to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", *subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	msg.WriteString("\r\n")

	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(string(body) + "\r\n")

	if imageData != nil {
		imgBase64 := base64.StdEncoding.EncodeToString(imageData)
		fileName := filepath.Base(*imagePath)
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString(fmt.Sprintf("Content-Type: image/png; name=\"%s\"\r\n", fileName))
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", fileName))
		msg.WriteString("\r\n")
		msg.WriteString(imgBase64 + "\r\n")
	}

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	msg.WriteString(".\r\n")

	fmt.Println("CLIENT: sending message body...")
	_, err = fmt.Fprintf(conn, "%s\r\n", msg.String())
	handleError("Failed to send message body:", err)

	resp, err = readResponse(reader)
	handleError("Failed to read response:", err)
	fmt.Printf("SERVER: %s", resp)

	_, err = sendCommand(conn, reader, "QUIT")
	handleError("Failed to quit:", err)

	fmt.Println("Email sent successfully!")
}
