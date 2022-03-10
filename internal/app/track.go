package app

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os"
)

// Tracker object holds a connection the track channel and maintains
// a set of pending requests
type Tracker struct {
	Track    chan PendingRequest
	Requests map[string]PendingRequest
}

// TrackPendingRequests forever listens to track channel for requests to store
func (tracker *Tracker) TrackPendingRequests(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			pendingRequest := <-tracker.Track
			if pendingRequest.Add {
				tracker.Requests[pendingRequest.RequestId] = pendingRequest
			} else {
				delete(tracker.Requests, pendingRequest.RequestId)
			}
		}
	}
}

// InstantiateCurrentRequests creates a new store for pending requests and recreates previous state if applicable
func InstantiateCurrentRequests(encrypt chan Request, pendingPersistenceLocation string) map[string]PendingRequest {
	pendingBytes, err := os.ReadFile(pendingPersistenceLocation)
	if err != nil {
		logrus.Errorf("Was unable to read pending file. Details: %v", err)
		return make(map[string]PendingRequest)
	}
	if len(pendingBytes) > 0 {
		var pending map[string]PendingRequest
		if err := json.Unmarshal(pendingBytes, &pending); err != nil {
			logrus.Errorf("Was unable to unmarshal pending into object. Details: %v", err)
			return make(map[string]PendingRequest)
		}
		for requestId, pendingRequest := range pending {
			encrypt <- Request{RequestId: requestId, Message: pendingRequest.Message}
		}
		return pending
	}
	return make(map[string]PendingRequest)
}
