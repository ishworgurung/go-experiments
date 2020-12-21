package ttl

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ishworgurung/vanishling/config"
	"github.com/rs/zerolog/log"
)

type Cleaner struct {
	logDeleteFunc      func() error // deleter to execute on the arg on timer expiration
	logFileName        string       // Log file name
	logEntryLocker     sync.RWMutex // Log lock to serialise the write of a log entry
	logCleanerInterval *time.Ticker // Log cleaning interval ticker
}

func NewCleaner() *Cleaner {
	if err := os.MkdirAll(config.DefaultLogPath, 0755); err != nil {
		log.Info().Msgf("error: %s", err)
	}
	if err := os.MkdirAll(config.DefaultStoragePath, 0755); err != nil {
		log.Info().Msgf("error: %s", err)
	}
	return &Cleaner{
		logEntryLocker:     sync.RWMutex{},
		logCleanerInterval: time.NewTicker(config.DefaultLogCleanerInterval),
		logDeleteFunc: func() error {
			lsfPath := filepath.Join(config.DefaultLogPath, config.DefaultLogFile)
			_, err := os.OpenFile(lsfPath, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0644)
			if err != nil {
				return err
			}
			byteEntries, err := ioutil.ReadFile(lsfPath)
			if err != nil {
				log.Info().Msgf("log logFileName read error: %s\n", err)
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
					// FIXME: find a way to mark logFileName as deleted in the log entry
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
					// File has expired, delete the logFileName from filesystem.
					if err := os.Remove(fp); err == nil {
						log.Info().Msgf("log: %s deleted due to ttl expiration: %s and expiration date: %v\n", fp, ttl, expirationDate)
					}
				}
			}
			return nil
		},
	}
}

func (l *Cleaner) Start(ctx context.Context, lp string) {
	for {
		select {
		case <-ctx.Done():
			l.logCleanerInterval.Stop()
			return
		case <-l.logCleanerInterval.C:
			log.Info().Msgf("log: checking log for pending deletion")
			if err := l.logDeleteFunc(); err != nil {
				log.Info().Msgf("log: '%s' could not be deleted due to error: %s", l.logFileName, err)
			}
		}
	}
}
