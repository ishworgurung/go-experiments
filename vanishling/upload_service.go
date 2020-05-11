package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ishworgurung/vanishling/internals"

	uuid "github.com/satori/go.uuid"

	"github.com/minio/highwayhash"
)

type fileUploader struct {
	wg sync.WaitGroup // locker
	p  string         // file storage path
	// file ttl cleaner log path; one that holds every file name that have ingress'ed
	// so they can be deleted even if the service crashed.
	lp                string
	fileUploadContext // file's upload context
}

type fileUploadContext struct {
	fn string       // file uploaded name
	u  uuid.UUID    // file name: unique uuid v5
	h  hash.Hash    // file unique hash
	l  sync.RWMutex // file lock
	// file's ttl cleaner context. one file has one cleaner
	ttlCleanerContext internals.TTLDeleteContext
	// file's ttl cleaner context. one file has one cleaner
	logTtlCleanerContext *internals.LogBasedTTLDeleteContext
}

func newFileUploaderSvc() (*fileUploader, error) {
	//FIXME: key
	seed, err := hex.DecodeString(
		"000102030405060708090A0B0C0D0E0FF0E0D0C0B0A090807060504030201000")
	if err != nil {
		return nil, fmt.Errorf("cannot decode hex key: %v", err)
	}
	hh, err := highwayhash.New(seed)
	if err != nil {
		return nil, err
	}

	logBasedTTLCleaner := internals.NewLogBasedTTLDeleterService()
	go logBasedTTLCleaner.StartLogCleanerTimerLoop(defaultLogPath) // read path

	return &fileUploader{
		wg: sync.WaitGroup{},
		p:  defaultStoragePath,
		lp: defaultLogPath,
		fileUploadContext: fileUploadContext{
			l:                    sync.RWMutex{},
			h:                    hh,
			ttlCleanerContext:    internals.NewTTLDeleterService(),
			logTtlCleanerContext: logBasedTTLCleaner,
		},
	}, nil
}

func (f *fileUploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (f *fileUploader) delete(w http.ResponseWriter, r *http.Request) {

}

func (f *fileUploader) download(w http.ResponseWriter, r *http.Request) {
	defer f.wg.Done()
	fileHash := r.Header.Get(defaultFileIdHeader)
	if len(fileHash) == 0 {
		log.Printf("error retrieving the file '%s'", fileHash)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//FIXME: Path based attacks is possible with the code below
	p := filepath.Join(f.p, fileHash)
	// seek to the start of the uploaded file
	fileBytes, err := ioutil.ReadFile(p)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fileContents := string(fileBytes)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, fileContents)
	return
}

func (f *fileUploader) upload(w http.ResponseWriter, r *http.Request) {
	defer f.wg.Done()
	if err := r.ParseMultipartForm(defaultMaxUploadByte); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	uploadedFile, handler, err := r.FormFile("file")
	if err != nil {
		log.Println("error retrieving the File")
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		if err = uploadedFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	if err = f.setFileName(handler.Filename, handler.Size); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err, fileNameAsHashId := f.hashFile(uploadedFile)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Add(defaultFileIdHeader, *fileNameAsHashId)
	w.WriteHeader(http.StatusOK)

	uploadedFileTTL := r.Header.Get("x-ttl")
	if len(uploadedFileTTL) != 0 {
		f.ttlCleanerContext.Ttl, err = time.ParseDuration(uploadedFileTTL)
		if err != nil {
			log.Printf("invalid duration: %s", err)
			f.ttlCleanerContext.Ttl = defaultFileTTL
		}
		log.Printf(
			"uploader: setting duration of %s for file deletion (ttl) for file: %s and hashed file id: %s",
			f.ttlCleanerContext.Ttl, f.fn, *fileNameAsHashId)
	} else {
		log.Printf(
			"uploader: setting default duration of %s for file deletion (ttl) for file: %s and hashed file id: %s",
			defaultFileTTL, f.fn, *fileNameAsHashId)
		f.ttlCleanerContext.Ttl = defaultFileTTL
	}

	// set ttl for deletion in the log entry in case, service goes down.
	if err := f.ttlCleanerContext.WriteWALEntry(f.ttlCleanerContext.Ttl, defaultStoragePath, *fileNameAsHashId); err != nil {
		log.Printf("could not write WAL entry for file %s with TTL: %s : %s\n", *fileNameAsHashId, f.ttlCleanerContext.Ttl, err)
	}

	// start ttl-based cleaner for the provided file
	//go f.ttlCleanerContext.SetTTLCleaner(f.lp, *fileNameAsHashId)

}

//FIXME: Path based attacks is possible with the code below
func (f *fileUploader) setFileName(fn string, fs int64) error {
	if len(fn) == 0 {
		return errors.New("invalid file name")
	}
	f.fileUploadContext.fn = fn
	if fs == 0 {
		return errors.New("zero byte file uploaded")
	}
	return nil
}

func (f *fileUploader) ensureDirWritable() error {
	ns := uuid.NewV4().String()
	p := filepath.Join(f.p, ns)
	os.MkdirAll(f.p, 0755)
	tmp, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(p)
	return nil
}

func (f *fileUploader) hashFile(uploadedFile multipart.File) (error, *string) {
	var err error
	if err := f.ensureDirWritable(); err != nil {
		return err, nil
	}
	f.l.Lock()
	defer f.l.Unlock()
	f.h.Write([]byte(time.Now().String()))
	_, err = io.Copy(f.h, uploadedFile)
	if err != nil {
		return err, nil
	}
	checksum := hex.EncodeToString(f.h.Sum(nil))
	p := filepath.Join(f.p, checksum)
	tmp, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0644)
	if err != nil {
		return err, nil
	}
	defer tmp.Close()
	// seek to the start of the uploaded file
	uploadedFile.Seek(0, io.SeekStart)
	fileBytes, err := ioutil.ReadAll(uploadedFile)
	if err != nil {
		return err, nil
	}
	_, err = tmp.Write(fileBytes)
	if err != nil {
		return err, nil
	}
	return nil, &checksum
}
