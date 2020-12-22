package cfg

import "time"

// FIXME: This should ideally come from cfg file
const (
	DefaultStoragePath        = "/tmp/vanishling/uploads"
	DefaultLogPath            = "/tmp/vanishling/log"
	DefaultLogCleanerInterval = time.Second * 5
	DefaultLogFile            = "entries.journal"
	DefaultHHSeed             = "000102030405060708090A0B0C0D0E0FF0E0D0C0B0A090807060504030201000"
	DefaultFileTTL            = time.Minute * 5
	DefaultMaxUploadByte      = 1024 * 15
	DefaultFileIdHeader       = "x-file-id"
	DefaultTTLHeader          = "x-ttl"
	DefaultMaxJournalSize     = 1024 * 1024 * 1024 * 1024 // bytes
	DefaultMaxTTLHours        = 1
)
