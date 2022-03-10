package app

import (
	"context"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"os"
	"testing"
	"time"
)

func TestTracker_TrackPendingRequests(t *testing.T) {
	pendingRequestAddDefault := PendingRequest{
		Request: Request{
			RequestId: "requestId",
			Message:   "message",
		},
		Timing: Timing{
			time.Now(),
			0.0,
		},
		Add: true,
	}
	pendingRequestDeleteDefault := PendingRequest{
		Request: Request{
			RequestId: "requestId",
			Message:   "message",
		},
		Timing: Timing{
			time.Now(),
			0.0,
		},
		Add: false,
	}
	tests := []struct {
		name                  string
		pendingRequest        PendingRequest
		startingRequests      map[string]PendingRequest
		want                  map[string]PendingRequest
		isTestingConextCancel bool
		isTestingFailure      bool
	}{
		{
			name:                  "Successful Addition",
			pendingRequest:        pendingRequestAddDefault,
			startingRequests:      make(map[string]PendingRequest),
			want:                  map[string]PendingRequest{"requestId": pendingRequestAddDefault},
			isTestingConextCancel: false,
			isTestingFailure:      false,
		},
		{
			name:                  "Successful Deletion",
			pendingRequest:        pendingRequestDeleteDefault,
			startingRequests:      map[string]PendingRequest{"requestId": pendingRequestDeleteDefault},
			want:                  make(map[string]PendingRequest),
			isTestingConextCancel: false,
			isTestingFailure:      false,
		},
		{
			name:                  "Successful Context Cancellation",
			pendingRequest:        pendingRequestAddDefault,
			startingRequests:      make(map[string]PendingRequest),
			want:                  make(map[string]PendingRequest),
			isTestingConextCancel: true,
			isTestingFailure:      false,
		},
		{
			name:                  "Failed Deletion",
			pendingRequest:        pendingRequestDeleteDefault,
			startingRequests:      make(map[string]PendingRequest),
			want:                  make(map[string]PendingRequest),
			isTestingConextCancel: false,
			isTestingFailure:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTrackChan := make(chan PendingRequest)
			ctx, cancel := context.WithCancel(context.Background())
			mockTrack := Tracker{Track: mockTrackChan, Requests: tt.startingRequests}
			mockTrackerErrors := make(chan error, 1)
			go func() {
				mockTrackerErrors <- mockTrack.TrackPendingRequests(ctx)
			}()
			if tt.isTestingConextCancel {
				cancel()
				err := <-mockTrackerErrors
				if err != nil {
					t.Errorf("Unexpected Error in test for TrackPendingRequests. Details: %v", err.Error())
				}
			} else {
				mockTrackChan <- tt.pendingRequest
				select {
				case err := <-mockTrackerErrors:
					if err != nil && !tt.isTestingFailure {
						t.Errorf("Unexpected Error in test for TrackPendingRequests. Details: %v", err.Error())
					} else if err == nil && tt.isTestingFailure {
						t.Error("Was expecting an error to occur but none did")
					} else if err == nil && !cmp.Equal(mockTrack.Requests, tt.want) {
						t.Errorf("Requests not as expected. Wanted: %v, Got: %v", mockTrack.Requests, tt.want)
					}
					cancel()
				default:
					cancel()
				}
			}
		})
	}
}

func TestInstantiateCurrentRequests(t *testing.T) {
	populatedPendingBytes, err := os.ReadFile("../../testdata/populatedPendingState.json")
	if err != nil {
		t.Errorf("Was unable to read test populated signatures file. Details: %v", err)
	}
	var populatedPending map[string]PendingRequest
	if err := json.Unmarshal(populatedPendingBytes, &populatedPending); err != nil {
		t.Errorf("Was unable to unmarshal test populated signatures into object. Details: %v", err)
	}
	tests := []struct {
		name              string
		inputFileLocation string
		want              map[string]PendingRequest
	}{
		{
			name:              "Successful Load of populated state",
			inputFileLocation: "../../testdata/populatedPendingState.json",
			want:              populatedPending,
		},
		{
			name:              "Successful Load of empty state (fresh start)",
			inputFileLocation: "../../testdata/emptyState.json",
			want:              make(map[string]PendingRequest),
		},
		{
			name:              "Successful Load of empty state object (restart)",
			inputFileLocation: "../../testdata/emptyStructState.json",
			want:              make(map[string]PendingRequest),
		},
		{
			name:              "Bad file location",
			inputFileLocation: "../../testdata/fakeLocation.json",
			want:              make(map[string]PendingRequest),
		},
		{
			name:              "Unmarshable state",
			inputFileLocation: "../../testdata/unmarshableState.json",
			want:              make(map[string]PendingRequest),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEncryptionChan := make(chan Request)
			mockInstantiationResults := make(chan map[string]PendingRequest, 1)
			go func() {
				mockInstantiationResults <- InstantiateCurrentRequests(mockEncryptionChan, tt.inputFileLocation)
			}()
			select {
			case <-mockEncryptionChan:
			case pendingRequests := <-mockInstantiationResults:
				if !cmp.Equal(pendingRequests, tt.want) {
					t.Errorf("Requests was not instantiated to the desired state. Want: %v, Recieved: %v", tt.want, pendingRequests)
				}
			}
		})
	}
}
