package core

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ishworgurung/vanishling/cfg"
	"github.com/ishworgurung/vanishling/ttl"
	"github.com/minio/highwayhash"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
)

type fileService struct {
	wg          sync.WaitGroup // locker
	storagePath string         // file storage path
	// file ttl cleaner log path; one that holds every file name that have ingress'ed
	// so they can be deleted even if the core has crashed.
	journalPath string
	*uploader   // uploader
	lg          zerolog.Logger
}

type uploader struct {
	peerAddr  string                 // file uploader's IP address
	fileName  string                 // uploaded file name
	u         uuid.UUID              // file name: unique uuid v5
	hh        hash.Hash              // file hash
	mu        sync.RWMutex           // file lock
	journaler *ttl.VanishlingJournal // file's ttl cleaner context. one file has one cleaner
	cleaner   *ttl.Cleaner           // file's ttl cleaner context. one file has one cleaner
}

func New(ctx context.Context, logPath string,
	storagePath string, lg zerolog.Logger, seed string) (*fileService, error) {
	//FIXME: the seed
	hhSeed, err := hex.DecodeString(seed)
	if err != nil {
		return nil, fmt.Errorf("cannot decode hex key: %v", err)
	}
	hh, err := highwayhash.New(hhSeed)
	if err != nil {
		return nil, err
	}

	cleaner := ttl.NewCleaner()
	go cleaner.Start(ctx, logPath) // read path
	journaler, err := ttl.NewJournaler(ctx, lg)
	if err != nil {
		return nil, err
	}

	return &fileService{
		wg:          sync.WaitGroup{},
		storagePath: storagePath,
		journalPath: logPath,
		uploader: &uploader{
			mu:        sync.RWMutex{},
			hh:        hh,
			journaler: journaler,
			cleaner:   cleaner,
		},
		lg: lg,
	}, nil
}

func (f *fileService) upload(w http.ResponseWriter, r *http.Request) {
	defer f.wg.Done()

	if f.cleaner.IsDiskFull() {
		log.Debug().Msg("The disk is almost full. Refusing to serve further upload requests")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// for audit purpose
	f.peerAddr = r.Header.Get("X-Real-IP")
	if len(f.peerAddr) == 0 {
		f.peerAddr = r.RemoteAddr
	}

	if err := r.ParseMultipartForm(cfg.DefaultMaxUploadByte); err != nil {
		log.Info().Msg(f.peerAddr + ":" + err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	uploadedFile, handler, err := r.FormFile("file")
	if err != nil {
		log.Info().Msg(f.peerAddr + ":" + "error retrieving the File: %s" + err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		if err = uploadedFile.Close(); err != nil {
			log.Info().Msg(f.peerAddr + ":" + err.Error())
		}
	}()

	if err = f.setFileName(handler.Filename, handler.Size); err != nil {
		log.Info().Msg(f.peerAddr + ":" + err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashedFileName, err := f.hashFile(uploadedFile)
	if err != nil {
		log.Info().Msg(f.peerAddr + ":" + err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Info().Msg(f.peerAddr + ": ok")
	w.Header().Add(cfg.DefaultFileIdHeader, hashedFileName)
	w.WriteHeader(http.StatusOK)

	uploadedFileTTL := r.Header.Get(cfg.DefaultTTLHeader)
	if len(uploadedFileTTL) != 0 {
		t, err := time.ParseDuration(uploadedFileTTL)
		if err != nil || t.Hours() > cfg.DefaultMaxTTLHours {
			t = cfg.DefaultFileTTL
		}
		f.journaler.SetFileTTL(t)
		log.Info().Err(err).Msgf(
			"setting TTL value of '%s' for file: '%s' and hashed file id: %s",
			f.journaler.GetFileTTL(), f.getFileName(), hashedFileName)

	} else {
		log.Info().Msgf(f.peerAddr+":"+
			"setting default TTL of '%s' for file: '%s' and hashed file id: %s",
			cfg.DefaultFileTTL, f.getFileName(), hashedFileName)
		f.journaler.SetFileTTL(cfg.DefaultFileTTL)
	}

	// set ttl for deletion in the log entry in case, core goes down.
	if err := f.journaler.CommitJournal(f.journaler.GetFileTTL(), cfg.DefaultStoragePath,
		hashedFileName); err != nil {
		log.Info().Err(err).Msgf(
			"could not write log entry for file '%s' with hashed file id '%s'",
			f.getFileName(), hashedFileName)
	}
}

//FIXME: Need to validate that path based attacks is not possible with the code below
func (f *fileService) setFileName(fn string, fs int64) error {
	if len(fn) == 0 {
		return errors.New(f.peerAddr + ": invalid file name")
	}
	if strings.Contains(fn, "..") || strings.Contains(fn, "/") {
		return errors.New(f.peerAddr + ": invalid file name")
	}
	if fs == 0 {
		return errors.New(f.peerAddr + ": zero byte file uploaded")
	}
	f.uploader.fileName = fn
	return nil
}

func (f *fileService) getFileName() string {
	return f.uploader.fileName
}

func (f *fileService) ensureDirWritable() error {
	ns := uuid.NewV4().String()
	p := filepath.Join(f.storagePath, ns)
	os.MkdirAll(f.storagePath, 0755)
	tmp, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(p)
	return nil
}

// Hash the file and use it as a file name.
func (f *fileService) hashFile(uploadedFile multipart.File) (string, error) {
	var err error
	if err := f.ensureDirWritable(); err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hh.Write([]byte(time.Now().String())) // mixer
	_, err = io.Copy(f.hh, uploadedFile)
	if err != nil {
		return "", err
	}
	checksum := hex.EncodeToString(f.hh.Sum(nil))
	p := filepath.Join(f.storagePath, checksum)
	tmp, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := tmp.Close(); err != nil {
			f.lg.Debug().Msgf("failed to close tmp file: %s", err)
		}
	}()

	// seek to the start of the uploaded file
	_, _ = uploadedFile.Seek(0, io.SeekStart)
	fileBytes, err := ioutil.ReadAll(uploadedFile)
	if err != nil {
		return "", err
	}
	n, err := tmp.Write(fileBytes)
	if n <= 0 || err != nil {
		return "", err
	}
	return checksum, nil
}

func (f *fileService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqMethod := strings.ToUpper(r.Method)
	if reqMethod == http.MethodPost || reqMethod == http.MethodPut {
		f.wg.Add(1)
		go f.upload(w, r)
		f.wg.Wait()
		return
	} else if reqMethod == http.MethodGet {
		f.wg.Add(1)
		go f.download(w, r)
		f.wg.Wait()
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func (f *fileService) delete(w http.ResponseWriter, r *http.Request) {
	// for audit purpose
	f.peerAddr = r.Header.Get("X-Real-IP")
	if len(f.peerAddr) == 0 {
		f.peerAddr = r.RemoteAddr
	}
}

func (f *fileService) download(w http.ResponseWriter, r *http.Request) {
	defer f.wg.Done()
	// for audit purpose
	f.peerAddr = r.Header.Get("X-Real-IP")
	if len(f.peerAddr) == 0 {
		f.peerAddr = r.RemoteAddr
	}

	fileHash := r.Header.Get(cfg.DefaultFileIdHeader)
	if len(fileHash) == 0 {
		log.Info().Msgf(f.peerAddr+": error retrieving the file '%s'", fileHash)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if strings.Contains(fileHash, "..") || strings.Contains(fileHash, "/") {
		log.Info().Msgf(f.peerAddr+": error retrieving the file '%s'", fileHash)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//FIXME: Check that path based attacks is not possible with the code below
	p := filepath.Join(f.storagePath, fileHash)
	// seek to the start of the uploaded file
	fileBytes, err := ioutil.ReadFile(p)
	if err != nil {
		log.Info().Msgf(f.peerAddr+": error while reading the file: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Info().Msgf(f.peerAddr + ": ok")
	w.WriteHeader(http.StatusOK)
	w.Write(fileBytes)
	return
}
