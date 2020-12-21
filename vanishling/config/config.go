package config

import "time"

// FIXME: This should ideally come from config file
const (
	DefaultStoragePath        = "/tmp/vanishling/uploads"
	DefaultLogPath            = "/tmp/vanishling/log"
	DefaultLogCleanerInterval = time.Second * 60
	DefaultLogFile            = "entries.log"
	DefaultHHSeed             = 0xffffa210 // FIXME
	DefaultFileTTL            = time.Duration(time.Minute * 5)
	DefaultMaxUploadByte      = 1024 * 15
	DefaultFileIdHeader       = "x-file-id"
	DefaultTTLHeader          = "x-ttl"
)
