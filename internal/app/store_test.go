package app

import (
	"context"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"os"
	"testing"
)

func TestStorer_StoreSignedRequests(t *testing.T) {
	tests := []struct {
		name                  string
		signedRequest         SignedRequest
		startingSignatures    map[string]string
		want                  map[string]string
		isTestingConextCancel bool
		isTestingFailure      bool
	}{
		{
			name:                  "Successful storage",
			signedRequest:         SignedRequest{RequestId: "requestId", Signature: "signature", Add: true},
			startingSignatures:    make(map[string]string),
			want:                  map[string]string{"requestId": "signature"},
			isTestingConextCancel: false,
			isTestingFailure:      false,
		},
		{
			name:                  "Successful Deletion",
			signedRequest:         SignedRequest{RequestId: "requestId", Signature: "signature", Add: false},
			startingSignatures:    map[string]string{"requestId": "signature"},
			want:                  map[string]string{},
			isTestingConextCancel: false,
			isTestingFailure:      false,
		},
		{
			name:                  "Successful Context Cancellation",
			signedRequest:         SignedRequest{RequestId: "requestId", Signature: "signature", Add: true},
			startingSignatures:    make(map[string]string),
			want:                  make(map[string]string),
			isTestingConextCancel: true,
			isTestingFailure:      false,
		},
		{
			name:                  "Failed Deletion",
			signedRequest:         SignedRequest{RequestId: "requestId", Signature: "signature", Add: false},
			startingSignatures:    map[string]string{},
			want:                  map[string]string{},
			isTestingConextCancel: false,
			isTestingFailure:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStoreChan := make(chan SignedRequest)
			mockTrackChan := make(chan PendingRequest)
			ctx, cancel := context.WithCancel(context.Background())
			mockStore := Storer{Store: mockStoreChan, Track: mockTrackChan, Signatures: tt.startingSignatures}
			mockStorerErrors := make(chan error, 1)
			go func() {
				mockStorerErrors <- mockStore.StoreSignedRequests(ctx)
			}()
			if tt.isTestingConextCancel {
				cancel()
				err := <-mockStorerErrors
				if err != nil {
					t.Errorf("Unexpected Error in test for StoreSignedRequests. Details: %v", err.Error())
				}
			} else {
				mockStoreChan <- tt.signedRequest
				select {
				case err := <-mockStorerErrors:
					if err != nil && !tt.isTestingFailure {
						t.Errorf("Unexpected Error in test for StoreSignedRequests. Details: %v", err.Error())
					} else if err == nil && tt.isTestingFailure {
						t.Error("Was expecting an error to occur but none did.")
					} else if err == nil && !cmp.Equal(mockStore.Signatures, tt.want) {
						t.Errorf("Signatures not as expected. Wanted: %v, Got: %v", tt.want, mockStore.Signatures)
					}
					cancel()
				default:
					cancel()
				}
			}
		})
	}
}

func TestInstantiateSignatures(t *testing.T) {
	populatedSignaturesBytes, err := os.ReadFile("../../testdata/populatedSignatureState.json")
	if err != nil {
		t.Errorf("Was unable to read test populated signatures file. Details: %v", err)
	}
	var populatedSignatures map[string]string
	if err := json.Unmarshal(populatedSignaturesBytes, &populatedSignatures); err != nil {
		t.Errorf("Was unable to unmarshal test populated signatures into object. Details: %v", err)
	}
	tests := []struct {
		name              string
		inputFileLocation string
		want              map[string]string
	}{
		{
			name:              "Successful Load of populated state",
			inputFileLocation: "../../testdata/populatedSignatureState.json",
			want:              populatedSignatures,
		},
		{
			name:              "Successful Load of empty state (fresh start)",
			inputFileLocation: "../../testdata/emptyState.json",
			want:              make(map[string]string),
		},
		{
			name:              "Successful Load of empty state object (restart)",
			inputFileLocation: "../../testdata/emptyStructState.json",
			want:              make(map[string]string),
		},
		{
			name:              "Bad file location",
			inputFileLocation: "../../testdata/fakeLocation.json",
			want:              make(map[string]string),
		},
		{
			name:              "Unmarshable state",
			inputFileLocation: "../../testdata/unmarshableState.json",
			want:              make(map[string]string),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signatures := InstantiateSignatures(tt.inputFileLocation)
			if !cmp.Equal(signatures, tt.want) {
				t.Errorf("Signatures was not instantiated to the desired state. Want: %v, Recieved: %v", tt.want, signatures)
			}
		})
	}
}

func TestRetriever_RetrieveSignature(t *testing.T) {
	tests := []struct {
		name             string
		retriever        Retriever
		want             string
		isTestingFailure bool
	}{
		{
			name: "Successful Retrieval",
			retriever: Retriever{
				Signatures: map[string]string{"requestId": "signature"},
				RequestId:  "requestId",
				Response:   make(chan string),
			},
			want:             "signature",
			isTestingFailure: false,
		},
		{
			name: "Failed Retrieval",
			retriever: Retriever{
				Signatures: make(map[string]string),
				RequestId:  "requestId",
				Response:   make(chan string),
			},
			want:             "",
			isTestingFailure: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrievalErrors := make(chan error, 1)
			go func() {
				retrievalErrors <- tt.retriever.RetrieveSignature()
			}()
			select {
			case err := <-retrievalErrors:
				if err != nil && !tt.isTestingFailure {
					t.Error("Failed to retrieve signature when requestId present")
				}
			case signature := <-tt.retriever.Response:
				if !tt.isTestingFailure {
					if !cmp.Equal(signature, tt.want) {
						t.Error("Failed to retrieve signature when requestId present")
					}
				} else {
					t.Error("Expected a failure but instead got a valid response")
				}
			}
		})
	}
}
