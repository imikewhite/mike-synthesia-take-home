package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Storer object holds connections to the store channel,
// tracking channel and maintains the set of signatures waiting retrieval
type Storer struct {
	Store      chan SignedRequest
	Track      chan PendingRequest
	Signatures map[string]string
}

// Retriever holds information about the application as well an instance of a requestId
// attempting to be retrieved and a channel to communicate a response too
type Retriever struct {
	Signatures map[string]string
	RequestId  string
	Response   chan string
}

// StoreSignedRequests forever listens to a store channel for signed requests that can be stored
func (storer *Storer) StoreSignedRequests(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			signedRequest := <-storer.Store
			if signedRequest.Add {
				storer.Signatures[signedRequest.RequestId] = signedRequest.Signature
				// Since we've stored the request, mark it as no longer pending
				pendingRequest := PendingRequest{
					Request: Request{
						RequestId: signedRequest.RequestId,
						Message:   "",
					},
					Timing: Timing{
						time.Now(),
						0.0,
					},
					Add: false,
				}
				storer.Track <- pendingRequest
			} else {
				delete(storer.Signatures, signedRequest.RequestId)
			}
		}
	}
}

// InstantiateSignatures creates a new storage for signatures and recreates previous state if applicable
func InstantiateSignatures(signaturesPersistenceLocation string) map[string]string {
	signaturesBytes, err := os.ReadFile(signaturesPersistenceLocation)
	if err != nil {
		logrus.Errorf("Was unable to read signatures file. Details: %v", err)
		return make(map[string]string)
	}
	if len(signaturesBytes) > 0 {
		var signatures map[string]string
		if err := json.Unmarshal(signaturesBytes, &signatures); err != nil {
			logrus.Errorf("Was unable to unmarshal signatures into object. Details: %v", err)
			return make(map[string]string)
		}
		return signatures
	}
	return make(map[string]string)
}

// RetrieveSignature attempts to retrieve a requestId from the store
func (retriever *Retriever) RetrieveSignature() error {
	if signature, ok := retriever.Signatures[retriever.RequestId]; ok {
		retriever.Response <- signature
		return nil
	} else {
		return errors.New("Signature not available yet.")
	}
}
