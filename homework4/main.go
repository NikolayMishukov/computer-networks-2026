package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type CacheEntry struct {
	ETag         string              `json:"etag"`
	LastModified string              `json:"last_modified"`
	Headers      map[string][]string `json:"headers"`
	Body         []byte              `json:"body"`
}

var (
	logger     *log.Logger
	cacheMap   = make(map[string]CacheEntry)
	cacheMutex sync.RWMutex
	blackList  = make([]string, 0)
)

func initLogger() {
	file, err := os.OpenFile("assets/proxy.log", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot open log file:", err)
	}
	logger = log.New(file, "", log.LstdFlags)
}

func initBlackList(filename string) {
	if filename == "" {
		return
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to read black list file %s: %v", filename, err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		blackList = append(blackList, scanner.Text())
	}
}

func getCacheKey(targetURL string) string {
	h := sha1.New()
	h.Write([]byte(targetURL))
	return hex.EncodeToString(h.Sum(nil))
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	errorHandler := func(msg string, err error) bool {
		if err != nil {
			http.Error(w, msg, http.StatusBadRequest)
			logger.Printf("%s: %s -> %v\n", msg, r.RequestURI, err)
		}
		return err != nil
	}

	targetURL := "http://" + r.RequestURI[1:]
	parsedURL, err := url.Parse(targetURL)
	if errorHandler("Invalid URL", err) {
		return
	}

	for _, pattern := range blackList {
		switch pattern {
		case parsedURL.Host:
		case r.RequestURI:
		default:
			continue
		}
		logger.Printf("Forbidden address: %s\n", r.RequestURI)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cacheKey := getCacheKey(targetURL)
	req, err := http.NewRequest(r.Method, parsedURL.String(), r.Body)
	if errorHandler("Bad request", err) {
		return
	}

	for key, values := range r.Header {
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}

	cacheMutex.RLock()
	entry, exists := cacheMap[cacheKey]
	cacheMutex.RUnlock()

	if exists {
		if entry.ETag != "" {
			req.Header.Set("If-None-Match", entry.ETag)
		}
		if entry.LastModified != "" {
			req.Header.Set("If-Modified-Since", entry.LastModified)
		}
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if errorHandler("Bad gateway", err) {
		return
	}
	defer resp.Body.Close()

	var finalBody []byte

	if resp.StatusCode == http.StatusNotModified && exists {
		logger.Printf("[CACHE HIT] %s", targetURL)
		finalBody = entry.Body
		for key, values := range entry.Headers {
			for _, v := range values {
				w.Header().Add(key, v)
			}
		}
		resp.StatusCode = http.StatusOK
	} else if resp.StatusCode == http.StatusOK {
		logger.Printf("[CACHE MISS/UPDATE] %s", targetURL)
		finalBody, _ = io.ReadAll(resp.Body)
		headers := make(map[string][]string)

		for key, values := range resp.Header {
			for _, v := range values {
				if key == "Content-Length" {
					continue
				}
				w.Header().Add(key, v)
				headers[key] = append(headers[key], v)
			}
		}

		cacheMutex.Lock()
		cacheMap[cacheKey] = CacheEntry{
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
			Headers:      headers,
			Body:         finalBody,
		}
		cacheMutex.Unlock()
	} else {
		finalBody, _ = io.ReadAll(resp.Body)
	}

	bodyStr := string(finalBody)
	host := parsedURL.Host
	bodyStr = strings.ReplaceAll(bodyStr, `href="/`, `href="/`+host+`/`)
	bodyStr = strings.ReplaceAll(bodyStr, `src="/`, `src="/`+host+`/`)

	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(bodyStr))

	logger.Printf("%s -> %d\n", r.RequestURI, resp.StatusCode)
}

func main() {
	portFlag := flag.String("p", "8080", "port to listen on")
	blackListFlag := flag.String("bl", "", "file with list of hostnames to blacklist")
	flag.Parse()
	port := *portFlag
	blackListFile := *blackListFlag

	initLogger()

	initBlackList(blackListFile)

	http.HandleFunc("/", proxyHandler)

	log.Printf("Proxy server running on :%s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
