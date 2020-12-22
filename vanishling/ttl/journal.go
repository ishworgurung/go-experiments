package ttl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ishworgurung/vanishling/cfg"
	"github.com/rs/zerolog"
)

type VanishlingJournal struct {
	file    string        // file name
	fileTTL time.Duration // the ttl of logFileName for deletion
	zlog    zerolog.Logger
	ctx     context.Context
	lsfPath string
	mu      *sync.Mutex
}

func NewJournaler(ctx context.Context, zlog zerolog.Logger) (*VanishlingJournal, error) {
	if err := os.MkdirAll(cfg.DefaultLogPath, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DefaultStoragePath, 0755); err != nil {
		return nil, err
	}
	lsfPath := filepath.Join(cfg.DefaultLogPath, cfg.DefaultLogFile)
	if _, err := os.Stat(lsfPath); err != nil {
		if err := os.MkdirAll(cfg.DefaultLogPath, 0755); err != nil {
			return nil, err
		}
	}
	return &VanishlingJournal{
		zlog:    zlog,
		ctx:     ctx,
		lsfPath: lsfPath,
		mu:      &sync.Mutex{},
	}, nil
}

func (d *VanishlingJournal) CommitJournal(ttl time.Duration, storageDir string, upFile string) error {
	jrnl, err := os.OpenFile(d.lsfPath, os.O_RDWR|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer jrnl.Close()

	// FIXME: upFile needs to be sanitised.
	f := filepath.Join(storageDir, upFile)
	if len(f) == 0 {
		return errors.New("empty file name")
	}
	e := time.Now().Add(ttl).Format(time.UnixDate)
	// journal entry format: `expiry-unix,original-ttl,uploadedFilepath`
	n, err := jrnl.WriteString(fmt.Sprintf("%v,%v,%s\n", e, ttl, f))
	if n <= 0 || err != nil {
		return fmt.Errorf("less than zero bytes were written to journal: %s", err)
	}
	return nil
}

func (d *VanishlingJournal) SetFileTTL(ttl time.Duration) {
	d.fileTTL = ttl
}

func (d *VanishlingJournal) GetFileTTL() time.Duration {
	return d.fileTTL
}
