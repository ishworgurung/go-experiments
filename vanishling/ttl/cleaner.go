package ttl

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ishworgurung/vanishling/cfg"
	"github.com/rs/zerolog/log"
)

type Cleaner struct {
	logFileName        string       // Log file name
	logEntryLocker     sync.RWMutex // Log lock to serialise the write of a log entry
	logCleanerInterval *time.Ticker // Log cleaning interval ticker
}

func NewCleaner() *Cleaner {
	if err := os.MkdirAll(cfg.DefaultLogPath, 0755); err != nil {
		log.Info().Msgf("error: %s", err)
	}
	if err := os.MkdirAll(cfg.DefaultStoragePath, 0755); err != nil {
		log.Info().Msgf("error: %s", err)
	}
	return &Cleaner{
		logFileName:        filepath.Join(cfg.DefaultLogPath, cfg.DefaultLogFile),
		logEntryLocker:     sync.RWMutex{},
		logCleanerInterval: time.NewTicker(cfg.DefaultLogCleanerInterval),
	}
}

func (l *Cleaner) Start(ctx context.Context, lp string) {
	for {
		select {
		case <-ctx.Done():
			l.logCleanerInterval.Stop()
			return
		case <-l.logCleanerInterval.C:
			log.Info().Msgf("Checking log for files with expired TTL")
			if err := l.deleteFunc(); err != nil {
				log.Info().Msgf("log: '%s' could not be deleted due to error: %s", l.logFileName, err)
			}
		}
	}
}

func (l *Cleaner) deleteFunc() error {
	lsfFile, err := os.OpenFile(l.logFileName, os.O_RDONLY|os.O_CREATE|os.O_SYNC, 0644)
	if err != nil {
		return err
	}

	byteEntries, err := ioutil.ReadFile(l.logFileName)
	if err != nil {
		return err
	}
	entries := strings.SplitN(string(byteEntries), "\n", -1)
	for _, e := range entries {
		jrnlEntry := strings.Split(e, ",")
		if len(jrnlEntry) != 3 {
			continue
		}
		fp := jrnlEntry[2]
		if _, err := os.Stat(fp); err != nil {
			// FIXME: need to find a way to mark logFileName as deleted in the log entry
			continue
		}
		expirationDate, err := time.Parse(time.UnixDate, jrnlEntry[0])
		if err != nil {
			return fmt.Errorf("failed to parse UNIX date in log entry: %s", err)
		}
		ttl, err := time.ParseDuration(jrnlEntry[1])
		if err != nil {
			return fmt.Errorf("invalid ttl in log entry: %s", err)
		}

		if time.Now().Sub(expirationDate) > 0 {
			// TTL of the file has expired.
			if err := os.Remove(fp); err != nil {
				return fmt.Errorf("failed deletion: %s", err)
			}
			log.Info().Msgf("log: %s deleted due to ttl expiration: %s and expiration date: %v\n", fp, ttl, expirationDate)
		}
	}

	err = lsfFile.Close()
	if err != nil {
		log.Error().Msgf("could not close journal file: %s", err)
		return err
	}

	s, err := os.Stat(l.logFileName)
	if err != nil {
		log.Error().Msgf("could not stat journal file: %s", err)
		return err
	}
	if s.Size() > cfg.DefaultMaxJournalSize {
		if err = os.Truncate(l.logFileName, 0); err != nil {
			log.Error().Msgf("could not truncate journal file to zero byte: %s", err)
			return err
		}
	}

	return nil
}

func (l *Cleaner) IsDiskFull() bool {
	var stat syscall.Statfs_t
	syscall.Statfs(cfg.DefaultLogPath, &stat)
	all := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := all - free
	percentageUtilized := float64(used) / float64(all) * float64(100)
	if percentageUtilized > 90 {
		return true
	}
	return false
}
