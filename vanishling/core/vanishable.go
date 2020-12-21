package core

import "net/http"

// A service that implements `Vanishable` interface
// is a service that supports download, upload or delete
// operations.
type Vanishable interface {
	download(w http.ResponseWriter, r *http.Request)
	upload(w http.ResponseWriter, r *http.Request)
	delete(w http.ResponseWriter, r *http.Request)
}
