package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"go.uber.org/ratelimit"
)

func main() {
	httpUrl := os.Args[1]

	const (
		timeout = time.Second * 30
	)
	var tcpTransporter = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
		IdleConnTimeout:       timeout,
		ResponseHeaderTimeout: timeout,
	}
	httpClient := http.Client{
		Transport: tcpTransporter,
		Timeout:   timeout,
	}

	rl := ratelimit.New(10) // per second

	prev := time.Now()
	for i := 0; ; i++ {
		now := rl.Take()
		fmt.Println(i, now.Sub(prev))
		fetchRequest(httpClient, httpUrl)
		prev = now
	}

	// Output:
	// 0 0
	// 1 10ms
	// 2 10ms
	// 3 10ms
	// 4 10ms
	// 5 10ms
	// 6 10ms
	// 7 10ms
	// 8 10ms
	// 9 10ms
}

func fetchRequest(f http.Client, url string) {
	r, err := f.Get(url)
	if err != nil {
		log.Println(err)
		os.Exit(127)
	}
	if err := r.Body.Close(); err != nil {
		log.Println(err)
	}
	f.CloseIdleConnections()
	fmt.Printf(".")
}
