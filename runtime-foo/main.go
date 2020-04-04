package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

func main() {
	mem := make(map[int]string, 10240)
	t := time.NewTicker(time.Duration(5) * time.Second)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case _ = <-t.C:
				for e := 0; e < len(mem); e++ {
					mem[rand.Intn(e)] = "hello"
				}
				fmt.Printf("done")
			default:
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			runtime.GC()
		}
	}()

	wg.Wait()

	//
	//runtime.LockOSThread()
	//
	//defer runtime.UnlockOSThread()
}
