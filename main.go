package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/imikewhite/synthesia/internal/app"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// Config struct holds all optional parameters for the application
type Config struct {
	MaxRequestQueueSize           int
	MaxSynthesiaRequestsPerMinute int
	ServerPort                    string
	LogLevel                      string
	SignaturesPersistenceLocation string
	PendingPersistenceLocation    string
}

// SetConfigs sets application configs using parameters passed in or default values
func SetConfigs() Config {
	// limit 300 enroute requests - feels like if someone has to wait more that 30 minutes we shouldnt accept the request
	maxRequestQueueSize := flag.Int("maxRequestQueueSize", 300, "Max requests to hold in queue")
	maxSynthesiaRequestsPerMinute := flag.Int("maxSynthesiaRequestsPerMinute", 10, "Max requests that can be made to Synthesia per minute")
	serverPort := flag.String("serverPort", ":8080", "Server port, including preceding colon (will hang otherwise)")
	logLevel := flag.String("logLevel", "debug", "Set the log level for the application; panic, fatal, error, warn, info, debug, trace")
	flag.Parse()
	conf := Config{
		MaxRequestQueueSize:           *maxRequestQueueSize,
		MaxSynthesiaRequestsPerMinute: *maxSynthesiaRequestsPerMinute,
		ServerPort:                    *serverPort,
		LogLevel:                      *logLevel,
		SignaturesPersistenceLocation: "./internal/persistence/signatures.json",
		PendingPersistenceLocation:    "./internal/persistence/pending.json",
	}
	return conf
}

// SaveState saves the state of the application to be persisted on next invokation
func SaveState(signatures map[string]string, pendingRequests map[string]app.PendingRequest, config Config) {
	signaturesBytes, err := json.MarshalIndent(signatures, "", " ")
	if err != nil {
		logrus.Errorf("Failed saving signature state during shutdown. Details: %v", err.Error())
	} else {
		_ = os.WriteFile(config.SignaturesPersistenceLocation, signaturesBytes, 0644)
	}
	pendingRequestsBytes, err := json.MarshalIndent(pendingRequests, "", " ")
	if err != nil {
		logrus.Errorf("Failed saving pending requests state during shutdown. Details: %v", err.Error())
	} else {
		_ = os.WriteFile(config.PendingPersistenceLocation, pendingRequestsBytes, 0644)
	}
}

func main() {
	// Get configs, set log level and create context for application
	config := SetConfigs()
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		level = logrus.DebugLevel
	}
	logrus.SetLevel(level)
	ctx, cancel := context.WithCancel(context.Background())

	// create channels used for application communication and state storage
	store := make(chan app.SignedRequest)
	signatures := app.InstantiateSignatures(config.SignaturesPersistenceLocation)
	encrypt := make(chan app.Request, config.MaxRequestQueueSize)
	encryptors := make(chan struct{}, config.MaxSynthesiaRequestsPerMinute)
	go app.InstantiateEncryptors(config.MaxSynthesiaRequestsPerMinute, encryptors)
	// create a channel for track
	track := make(chan app.PendingRequest)
	requests := app.InstantiateCurrentRequests(encrypt, config.PendingPersistenceLocation)

	// go routines
	// spin up tracker worker that forever listens to track queue and performs the operations onto the currentRequests
	tracker := app.Tracker{Track: track, Requests: requests}
	trackerErrors := make(chan error, 1)
	go func() {
		trackerErrors <- tracker.TrackPendingRequests(ctx)
	}()
	// spin up a Storer worker that forever listencs to store queue and performs the operation onto the store
	storer := app.Storer{Store: store, Track: track, Signatures: signatures}
	storerErrors := make(chan error, 1)
	go func() {
		storerErrors <- storer.StoreSignedRequests(ctx)
	}()
	// Sping up worker scheduler that listens to encrypt queue and schedules an encryption when available
	encryptorHandler := app.EncryptorHandler{Encrypt: encrypt, Store: store, Encryptors: encryptors}
	encryptorHandlerErrors := make(chan error, 1)
	go func() {
		encryptorHandlerErrors <- encryptorHandler.HandleEncryptRequests(ctx)
	}()
	// define application using all components, and start listening for incoming requests
	application := app.Application{Encrypt: encrypt, Store: store, Signatures: signatures, Track: track, Requests: requests, ServerPort: config.ServerPort}
	logrus.Debug("Starting API Server...")
	router := app.NewRouter(&application)
	applicationErrors := make(chan error, 1)
	go func() {
		logrus.Infof("Listening on %s...", application.ServerPort)
		applicationErrors <- http.ListenAndServe(application.ServerPort, router)
	}()
	// Listen for sig kill to send signal to store signatures and current requests
	shutdown := make(chan os.Signal, 1)
	go func() {
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	}()
	// Now that the program is running, handle recoverable errors and gracefully exit with critical error/kill signal
Program:
	for {
		select {
		case encryptorHandlerError := <-encryptorHandlerErrors:
			logrus.Errorf("Encryptor handler failed unexpectedly with the following error: %v. Creating new encryptor handler.", encryptorHandlerError.Error())
			go func() {
				encryptorHandlerErrors <- encryptorHandler.HandleEncryptRequests(ctx)
			}()
		case storerError := <-storerErrors:
			logrus.Errorf("Storer failed unexpectedly with the following error: %v. Creating new storer.", storerError.Error())
			go func() {
				storerErrors <- storer.StoreSignedRequests(ctx)
			}()
		case trackerError := <-trackerErrors:
			logrus.Errorf("Tracker failed unexpectedly with the following error: %v. Creating new tracker.", trackerError.Error())
			go func() {
				trackerErrors <- tracker.TrackPendingRequests(ctx)
			}()
		case applicationError := <-applicationErrors:
			logrus.Errorf("Application failed unexpectedly with the following error: %v. Shutting Down.", applicationError.Error())
			SaveState(storer.Signatures, tracker.Requests, config)
			cancel()
			break Program
		case <-shutdown:
			logrus.Error("Server recieved Kill command. Shutting Down.")
			SaveState(storer.Signatures, tracker.Requests, config)
			cancel()
			break Program
		}
	}
}
