package ttl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ishworgurung/vanishling/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Deleter struct {
	file    string        // file name
	fileTTL time.Duration // the ttl of logFileName for deletion
	zlog    zerolog.Logger
	ctx     context.Context
}

func NewDeleter(ctx context.Context, zlog zerolog.Logger) Deleter {
	if err := os.MkdirAll(config.DefaultLogPath, 0755); err != nil {
		log.Info().Msgf("error: %s", err)
	}
	if err := os.MkdirAll(config.DefaultStoragePath, 0755); err != nil {
		log.Info().Msgf("error: %s", err)
	}
	return Deleter{
		zlog: zlog,
		ctx:  ctx,
	}
}

func (d *Deleter) GetFileTTL() time.Duration {
	return d.fileTTL
}

func (d *Deleter) WriteLogEntry(ttl time.Duration, storagePath string, f string) error {
	lsfPath := filepath.Join(config.DefaultLogPath, config.DefaultLogFile)
	if _, err := os.Stat(lsfPath); err != nil {
		if err := os.MkdirAll(config.DefaultLogPath, 0755); err != nil {
			return err
		}
	}
	journal, err := os.OpenFile(lsfPath, os.O_RDWR|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := journal.Close(); err != nil {
			d.zlog.Debug().Err(err)
		}
	}()

	fsPath := filepath.Join(storagePath, f)
	expirationUnix := time.Now().Add(ttl).Format(time.UnixDate)
	// journal entry format: `expiration-unix-date,original-ttl,uploadedFilepath`
	le := fmt.Sprintf("%v,%v,%s\n", expirationUnix, ttl, fsPath)
	_, err = journal.WriteString(le)
	return err
}

func (d *Deleter) SetFileTTL(ttl time.Duration) {
	d.fileTTL = config.DefaultFileTTL
}
