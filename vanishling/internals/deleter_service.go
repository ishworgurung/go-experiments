package internals

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FIXME: This should ideally come from config file
const (
	defaultLogPath            = "/tmp/vanishling/log"
	defaultLogCleanerInterval = time.Second * 5
	defaultLogFile            = "entries.log"
)

type TTLDeleteContext struct {
	Tm *time.Timer // ttl timer
	//deleteFunc    func(string) error // deleter to execute on the arg on timer expiration
	//logDeleteFunc func() error       // deleter to execute on the arg on timer expiration
	file string        // file name
	Ttl  time.Duration // the ttl of file for deletion
	//logEntryLocker sync.RWMutex       // WAL lock to synchronise write of a log entry
	//logCleanerInterval *time.Ticker       // WAL cleaning interval ticker
}

type LogBasedTTLDeleteContext struct {
	logDeleteFunc      func() error // deleter to execute on the arg on timer expiration
	file               string       // file name
	logEntryLocker     sync.RWMutex // WAL lock to synchronise write of a log entry
	logCleanerInterval *time.Ticker // WAL cleaning interval ticker
	wg                 sync.WaitGroup
}

func NewLogBasedTTLDeleterService() *LogBasedTTLDeleteContext {
	return &LogBasedTTLDeleteContext{
		wg:                 sync.WaitGroup{},
		logEntryLocker:     sync.RWMutex{},
		logCleanerInterval: time.NewTicker(defaultLogCleanerInterval),
		logDeleteFunc: func() error {
			lsfPath := filepath.Join(defaultLogPath, defaultLogFile)
			_, err := os.OpenFile(lsfPath, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0644)
			if err != nil {
				return err
			}
			byteEntries, err := ioutil.ReadFile(lsfPath)
			if err != nil {
				log.Printf("log file read error: %s\n", err)
				return nil
			}
			entries := strings.SplitN(string(byteEntries), "\n", -1)
			for _, e := range entries {
				entrySlice := strings.Split(e, ",")
				if len(entrySlice) != 3 {
					return nil
				}
				fp := entrySlice[2]
				if _, err := os.Stat(fp); err != nil {
					// FIXME: find a way to mark file as deleted in the log entry
					continue
				}
				expirationDate, err := time.Parse(time.UnixDate, entrySlice[0])
				if err != nil {
					return errors.New("log: failed to parse UNIX date in log entry")
				}
				//fmt.Printf("%v\n", expirationDate)
				ttl, err := time.ParseDuration(entrySlice[1])
				if err != nil {
					return errors.New("log: invalid ttl in log entry")
				}

				if time.Now().Sub(expirationDate) > 0 {
					// File has expired, delete the file from filesystem.
					if err := os.Remove(fp); err == nil {
						log.Printf("log: %s deleted due to ttl expiration: %s and expiration date: %v\n", fp, ttl, expirationDate)
					}
				}
			}
			return nil
		},
	}
}

func NewTTLDeleterService() TTLDeleteContext {
	return TTLDeleteContext{
		//deleteFunc: func(f string) error {
		//	if f == "" {
		//		return errors.New("file name is empty")
		//	}
		//	p := filepath.Join(defaultStoragePath, f)
		//	log.Printf("file: %s deleted due to ttl expiration \n", p)
		//	if err := os.Remove(p); err != nil {
		//		return err
		//	}
		//	return nil
		//},
		//logDeleteFunc: func() error {
		//	lsfPath := filepath.Join(defaultLogPath, defaultLogFile)
		//	byteEntries, err := ioutil.ReadFile(lsfPath)
		//	if err != nil {
		//		return err
		//	}
		//	entries := strings.SplitN(string(byteEntries), "\n", -1)
		//	for _, e := range entries {
		//		entrySlice := strings.Split(e, ",")
		//		if len(entrySlice) != 3 {
		//			return nil
		//		}
		//		expirationDate, err := time.Parse(time.UnixDate, entrySlice[0])
		//		if err != nil {
		//			return errors.New("log: failed to parse UNIX date in log entry")
		//		}
		//		ttl, err := time.ParseDuration(entrySlice[1])
		//		if err != nil {
		//			return errors.New("log: invalid ttl in log entry")
		//		}
		//		fp := entrySlice[2]
		//		if time.Now().Sub(expirationDate) > 0 {
		//			// File has expired, delete the file from filesystem.
		//			if err := os.Remove(fp); err != nil {
		//				//log.Printf("log: could not delete %s: %s \n", fp, err)
		//			} else {
		//				log.Printf("log: %s deleted due to ttl expiration: %s \n", fp, ttl)
		//			}
		//		}
		//	}
		//	return nil
		//},
	}
}

// Run the cleaner. `f` is the file to delete
//func (e *TTLDeleteContext) SetTTLCleaner(lp string, f string) {
//	e.Tm = time.NewTimer(e.Ttl) // duration
//	go func() {
//		for {
//			select {
//			case <-e.Tm.C:
//				if err := e.deleteFunc(f); err != nil {
//					log.Printf("file: %s could not be deleted due to error: %s. perhaps it has been deleted by log-based TTL deleter?", f, err)
//				}
//			}
//		}
//	}()
//}

func (l *LogBasedTTLDeleteContext) StartLogCleanerTimerLoop(lp string) {
	if err := os.MkdirAll(lp, 0755); err != nil {
		log.Printf("error: %s", err)
	}

	for {
		select {
		case <-l.logCleanerInterval.C:
			log.Printf("log: checking log for pending deletion")
			if err := l.logDeleteFunc(); err != nil {
				log.Printf("log: '%s' could not be deleted due to error: %s", l.file, err)
			}
		}
	}
}

func (e *TTLDeleteContext) WriteWALEntry(ttl time.Duration, storagePath string, f string) error {
	lsfPath := filepath.Join(defaultLogPath, defaultLogFile)
	wal, err := os.OpenFile(lsfPath, os.O_RDWR|os.O_SYNC|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer wal.Close()
	fsPath := filepath.Join(storagePath, f)
	expirationUnix := time.Now().Add(ttl).Format(time.UnixDate)
	// wal entry format: `expiration-unix-date,original-ttl,uploadedFilepath`
	le := fmt.Sprintf("%v,%v,%s\n", expirationUnix, ttl, fsPath)
	_, err = wal.WriteString(le)
	return err
}
