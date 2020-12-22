package core

import "time"

type JournalEntry struct {
	exp time.Time
	ttl time.Duration
	ufn string
}
