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

	"github.com/minio/highwayhash"
	uuid "github.com/satori/go.uuid"
)

type healthCheck struct{}

func (p healthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

type TTLable interface {
	download(w http.ResponseWriter, r *http.Request)
	upload(w http.ResponseWriter, r *http.Request)
	delete(w http.ResponseWriter, r *http.Request)
}

type fileUploadContext struct {
	fn string       // file uploaded name
	u  uuid.UUID    // file name: unique uuid v5
	h  hash.Hash    // file unique hash
	l  sync.RWMutex // file lock
	// file's ttl cleaner context. one file has one cleaner
	ttlCleanerContext
}

type ttlCleanerContext struct {
	tm   *time.Timer        // ttl timer
	fn   func(string) error // func to execute on the arg on timer expiration
	file string             // file name
	ttl  time.Duration      // the ttl of file for deletion
}

func newTTLCleanerService() ttlCleanerContext {
	return ttlCleanerContext{
		fn: func(f string) error {
			p := filepath.Join(defaultStoragePath, f)
			log.Printf("file: %s deleted due to ttl expiration \n", p)
			if err := os.Remove(p); err != nil {
				return err
			}
			return nil
		},
	}
}

func (e *ttlCleanerContext) run(f string) {
	e.tm = time.NewTimer(e.ttl) // duration
	for {
		select {
		case <-e.tm.C:
			e.fn(f)
		}
	}
}

type fileUploader struct {
	wg                sync.WaitGroup // locker
	p                 string         // file storage path
	fileUploadContext                // file's upload context
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

	uploadedFileTTL := r.Header.Get("x-ttl")
	if len(uploadedFileTTL) != 0 {
		f.ttl, err = time.ParseDuration(uploadedFileTTL)
		if err != nil {
			log.Printf("invalid duration: %s", err)
			f.ttl = defaultFileTTL
		}
		log.Printf(
			"setting duration of %s for file deletion (ttl) for file: %s",
			f.ttl, f.fn)
	} else {
		log.Printf(
			"setting default duration of %s for file deletion (ttl) for file: %s",
			defaultFileTTL, f.fn)
		f.ttl = defaultFileTTL
	}

	err, fileHash := f.hashFile(uploadedFile)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Add(defaultFileIdHeader, *fileHash)
	w.WriteHeader(http.StatusOK)
	// start ttl-based cleaner for the provided file
	go f.run(*fileHash)
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

const (
	defaultHHSeed        = 0xffffa210
	defaultFileTTL       = time.Duration(time.Second * 5)
	defaultStoragePath   = "/tmp/ttlFileUploads"
	defaultMaxUploadByte = 10 << 20
	defaultFileIdHeader  = "X-fileid"
)

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
	return &fileUploader{
		wg: sync.WaitGroup{},
		p:  defaultStoragePath,
		fileUploadContext: fileUploadContext{
			l:                 sync.RWMutex{},
			h:                 hh,
			ttlCleanerContext: newTTLCleanerService(),
		},
	}, nil
}

func newHealthCheckSvc() (*healthCheck, error) {
	return &healthCheck{}, nil
}

func main() {

	// add route / POST
	// o if no ttl provided use default from config or else use the provided ttl
	// o upload the file and store it in filesystem
	// o after the ttl expire, delete the file from fs
	// o return auth key

	// add route / GET
	// o if the auth key correct, fetch the file
	// o if the auth key incorrect, throw 4xxs

	fileUploaderService, err := newFileUploaderSvc()
	if err != nil {
		log.Fatal(err)
	}
	healthCheckService, err := newHealthCheckSvc()
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ping", healthCheckService)
	mux.Handle("/", fileUploaderService)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
