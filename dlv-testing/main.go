package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	// What happens to `_` at runtime?
	// Does it even get a value?
	// Can you read from it?
	// Does it have a value associated with it?
	// Answer: step through it in dlv
	_, err := os.Stat("/tmp/file")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("finished")
}
