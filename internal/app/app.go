package app

import (
	"math"
	"time"

	"github.com/gorilla/mux"
)

// Application contains the configuration settings for the core API service
type Application struct {
	Encrypt    chan Request
	Store      chan SignedRequest
	Signatures map[string]string
	Track      chan PendingRequest
	Requests   map[string]PendingRequest
	ServerPort string
}

// SignedRequest contains a unique identifier for the request, the encryption
// signature of a message, and a flag to add (if true) or remove (if false)
// from storage
type SignedRequest struct {
	RequestId string
	Signature string
	Add       bool
}

// Request contains the message canidate for encryption and a unique identifier
// for the request
type Request struct {
	RequestId string
	Message   string
}

type Timing struct {
	TimeAdded    time.Time
	TimeEstimate float64
}

// PendingRequest contains a unique identifier for the request, and a flag to
// add (if true) or remove (if false) from progress tracker
type PendingRequest struct {
	Request
	Timing
	Add bool
}

// newRouter is a private function that defines the routes for the API and the call methods
func NewRouter(application *Application) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", application.healthHandler).Methods("GET")
	router.HandleFunc("/crypto/sign", application.newRequestHandler).Methods("GET")
	router.HandleFunc("/crypto/sign/request/{requestId}", application.currentRequestHandler).Methods("GET")
	return router
}

// GetEncryptionTimeEstimate estimated time for a message to be encrypted, dependent on current encryption queue
// Default to 1 minute (case where nothing in queue, yet encryptor is having to keep retrying)
func (application *Application) GetEncryptionTiming(currentTime time.Time) Timing {
	timeEstimate := math.Max(1, math.Ceil(float64(len(application.Encrypt))/5.0))
	return Timing{TimeAdded: currentTime, TimeEstimate: timeEstimate}
}
