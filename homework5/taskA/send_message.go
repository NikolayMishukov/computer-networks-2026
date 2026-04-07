package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/gomail.v2"
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

func main() {
	to := flag.String("to", "", "Recipient email")
	format := flag.String("format", "txt", "txt or html")
	subject := flag.String("subject", "Test Email", "Subject")
	bodyPath := flag.String("body", "assets/body.txt", "Path to file with body")
	configPath := flag.String("config", "assets/config.yaml", "Path to config")
	flag.Parse()

	if *to == "" {
		log.Fatal("Recipient email is required")
	}
	body, err := os.ReadFile(*bodyPath)
	if err != nil {
		log.Fatal("Failed to read body:", err)
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.User.Email)
	m.SetHeader("To", *to)
	m.SetHeader("Subject", *subject)
	if *format == "html" {
		m.SetBody("text/html", string(body))
	} else {
		m.SetBody("text/plain", string(body))
	}

	d := gomail.NewDialer(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
	)

	if err := d.DialAndSend(m); err != nil {
		log.Fatal("Send error:", err)
	}
	fmt.Println("Email sent successfully!")
}
