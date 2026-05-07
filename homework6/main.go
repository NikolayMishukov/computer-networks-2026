package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host IP address")
	port := flag.String("port", "21", "port")
	user := flag.String("user", "TestUser", "username")
	password := flag.String("password", "", "password")
	flag.Parse()

	c, err := ftp.Dial(*host+":"+*port, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Fatal(err)
	}

	err = c.Login(*user, *password)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Успешное подключение к FTP серверу")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n===== МЕНЮ =====")
		fmt.Println("1 - Показать файлы")
		fmt.Println("2 - Загрузить файл на сервер")
		fmt.Println("3 - Скачать файл с сервера")
		fmt.Println("0 - Выход")
		fmt.Print("Выберите действие: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			listFiles(c)

		case "2":
			fmt.Print("Введите путь к локальному файлу: ")
			path, _ := reader.ReadString('\n')
			path = strings.TrimSpace(path)

			uploadFile(c, path)

		case "3":
			fmt.Print("Введите имя файла на сервере: ")
			remoteName, _ := reader.ReadString('\n')
			remoteName = strings.TrimSpace(remoteName)

			fmt.Print("Введите путь для сохранения локально: ")
			localPath, _ := reader.ReadString('\n')
			localPath = strings.TrimSpace(localPath)

			downloadFile(c, remoteName, localPath)

		case "0":
			fmt.Println("Выход...")
			_ = c.Quit()
			return

		default:
			fmt.Println("Неверный пункт меню")
		}
	}
}

func listFiles(c *ftp.ServerConn) {
	var traverse func(string, string)
	traverse = func(root, indent string) {
		entries, err := c.List(root)
		if err != nil {
			log.Println(err)
			return
		}

		for _, entry := range entries {
			if entry.Type == ftp.EntryTypeFolder {
				fmt.Printf("%s[DIR]  %s\n", indent, entry.Name)

				fullPath := root
				if root != "/" {
					fullPath += "/"
				}
				fullPath += entry.Name

				traverse(fullPath, indent+"  ")
			} else {
				fmt.Printf("%s[FILE] %s (%d bytes)\n", indent, entry.Name, entry.Size)
			}
		}
	}

	fmt.Println("\nСодержимое сервера:")
	traverse("/", "")
}

func uploadFile(c *ftp.ServerConn, localPath string) {

	file, err := os.Open(localPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() { _ = file.Close() }()

	remoteFileName := filepath.Base(localPath)

	err = c.Stor(remoteFileName, file)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Файл успешно загружен на сервер")
}

func downloadFile(c *ftp.ServerConn, remoteFileName string, localPath string) {

	response, err := c.Retr(remoteFileName)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() { _ = response.Close() }()

	file, err := os.Create(localPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, response)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Файл успешно скачан")
}
