package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

func main() {
	fileBasedPebble()
	fmt.Printf("======")
	time.Sleep(5 * time.Second)
	memBasedPebble()
}

func memBasedPebble() {
	var memFS = vfs.NewStrictMem()
	db, err := pebble.Open("memfs-demo", &pebble.Options{FS: memFS})
	if err != nil {
		log.Fatal(err)
	}
	defer closeDB(db)

	for i := 0; i < 1e3; i++ {
		key := []byte(fmt.Sprintf("hello %d", i))
		if err := db.Set(key, []byte("world"), pebble.NoSync); err != nil {
			log.Fatal(err)
		}
		value, closer, err := db.Get(key)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %s\n", key, value)

		if err := closer.Close(); err != nil {
			log.Fatal(err)
		}
	}

}

func closeDB(db *pebble.DB) {
	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
}

func fileBasedPebble() {
	db, err := pebble.Open("disk-demo", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer closeDB(db)
	for i := 0; i < 1e3; i++ {
		key := []byte(fmt.Sprintf("hello %d", i))
		if err := db.Set(key, []byte("world"), pebble.Sync); err != nil {
			log.Fatal(err)
		}
		value, closer, err := db.Get(key)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %s\n", key, value)
		if err := closer.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
