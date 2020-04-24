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

type pingHandler struct{}

func (p pingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

type Uploadable interface {
	download(w http.ResponseWriter, r *http.Request)
	upload(w http.ResponseWriter, r *http.Request)
	delete(w http.ResponseWriter, r *http.Request)
}

type fileUploadContext struct {
	fn string       // file uploaded name
	u  uuid.UUID    // file name: unique uuid v5
	h  hash.Hash    // file unique hash
	l  sync.RWMutex // file lock
}

type fileUploader struct {
	wg  sync.WaitGroup
	ttl time.Duration
	p   string
	fileUploadContext
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
	//FIXME: Path based attacks with the code below
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
	uploadedFile, handler, err := r.FormFile("f")
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

	if err = f.validateUploadedFile(handler.Filename, handler.Size); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err, fileHash := f.contentAddressableHashFile(uploadedFile)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Add(defaultFileIdHeader, *fileHash)
	w.WriteHeader(http.StatusOK)
}

//FIXME: Path based attacks with the code below
func (f *fileUploader) validateUploadedFile(fn string, fs int64) error {
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

func (f *fileUploader) contentAddressableHashFile(uploadedFile multipart.File) (error, *string) {
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
	defaultFileTTL       = time.Duration(time.Hour * time.Duration(1))
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
		wg:  sync.WaitGroup{},
		ttl: defaultFileTTL,
		p:   defaultStoragePath,
		fileUploadContext: fileUploadContext{
			l: sync.RWMutex{},
			h: hh,
		},
	}, nil
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
	mux := http.NewServeMux()
	mux.Handle("/ping", pingHandler{})
	mux.Handle("/", fileUploaderService)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
