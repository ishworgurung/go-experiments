package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
)

type bucketable interface {
	process(sync.WaitGroup)
	refill()
	debug(uint64)
}

type bucketState struct {
	// dynamic capacity of the bucket
	dynamicCapacity int
	// R/O
	maxCapacity int
	// number of elements leak R/O
	leak int
	// leak interval duration
	leakInterval time.Duration
	// elements is ready to be leaked
	ready chan int
	// protect access to shared resource via mutex
	m sync.Mutex
	// atomic counter
	counter uint64
	// counter channel
	counterCh chan uint64
	// enableDebug/not
	enableDebug bool
}

func newLeakyBucket(
	dynamicCapacity int,
	leak int,
	leakInterval time.Duration,
	debug bool) bucketState {
	if leak > dynamicCapacity {
		log.Fatal("can't leak more than the dynamic capacity of the bucket")
	}
	return bucketState{
		dynamicCapacity: dynamicCapacity, // this changes
		maxCapacity:     dynamicCapacity, // this does not
		leak:            leak,
		leakInterval:    leakInterval,
		m:               sync.Mutex{},
		ready:           make(chan int),
		counter:         0,
		counterCh:       make(chan uint64),
		enableDebug:     debug,
	}
}

func (b *bucketState) process(wg sync.WaitGroup) {
	defer wg.Done()
	leakTicker := time.NewTicker(b.leakInterval)
	go func() {
		for {
			select {
			case _, ok := <-leakTicker.C:
				if !ok {
					return
				}
				b.m.Lock()
				// if dynamicCapacity reaches zero or below, refill it to max
				if b.dynamicCapacity <= 0 {
					b.refill()
				}
				// decrease the dynamic capacity by leak element
				// Send the number of "ready to be leaked" slots
				// via the ready channel.
				b.dynamicCapacity = b.dynamicCapacity - b.leak
				b.ready <- b.leak
				b.m.Unlock()
			default:
			}
		}
	}()

}

func (b *bucketState) refill() {
	b.dynamicCapacity = b.maxCapacity
}

func (b *bucketState) rateLimitedHttpClient(
	wg sync.WaitGroup,
	httpClient http.Client,
	url string,
) {
	defer wg.Done()
	for {
		select {
		case slots, ok := <-b.ready:
			if !ok {
				return
			}
			for i := slots; i >= 1; i-- {
				// access protected resource
				fetchRequest(httpClient, url)
				// internal atomic counter
				b.counterCh <- atomic.AddUint64(&b.counter, 1)
			}
		default:
		}
	}
}

func (b *bucketState) debug(at uint64) {
	for {
		select {
		case c, ok := <-b.counterCh:
			if !ok {
				return
			}
			if c%at == 0 && b.enableDebug {
				println()
				spew.Dump(b)
			}
		default:
		}
	}
}

///////////////////////////////////////////////////////////////////////////

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

func main() {
	const (
		timeout        = time.Second * 30
		httpUrl string = "http://freenas.ssi.local"
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
	// Bucket with dynamic cap of ten elements and leak four
	// element every leak interval duration.
	leakInterval := time.Millisecond * time.Duration(500)
	rl := newLeakyBucket(10, 4, leakInterval, true)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go rl.rateLimitedHttpClient(wg, httpClient, httpUrl)
	go rl.process(wg)
	go rl.debug(1024)
	wg.Wait()
}
