package main

import (
	"fmt"
	"time"
)

type DeleterUploadEvent struct {
	tm    *time.Timer
	fn    func() error
	file  string
	finch chan bool
}

func newDeleterService(f string) *DeleterUploadEvent {
	return &DeleterUploadEvent{
		file:  f,
		finch: make(chan bool),
		fn: func() error {
			fmt.Printf("%s deleted\n", f)
			return nil
		},
	}
}

func (e *DeleterUploadEvent) start() {
	e.tm = time.NewTimer(2 * time.Second) // duration
	ttlFileDeleter(e)
}

func ttlFileDeleter(e *DeleterUploadEvent) {
	for {
		select {
		case <-e.tm.C:
			e.fn()
			fmt.Printf("fn done for file: %s\n", e.file)
			e.finch <- true
		}
	}
}

func main() {
	e := newDeleterService("/tmp/abcd")
	go e.start()
	<-e.finch
	fmt.Println("Timer stopped")

}
