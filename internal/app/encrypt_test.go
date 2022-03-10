package app

import (
	"context"
	"testing"
)

func TestInstantiateEncryptors(t *testing.T) {
	tests := []struct {
		name       string
		encryptors chan struct{}
		want       int
	}{
		{
			name:       "Instantiate Encryptors with desired size",
			encryptors: make(chan struct{}, 10),
			want:       10,
		},
		{
			name:       "Instantiate Encryptors with desired size",
			encryptors: make(chan struct{}, 300),
			want:       300,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InstantiateEncryptors(tt.want, tt.encryptors)
			if len(tt.encryptors) != tt.want {
				t.Errorf("Expected channel to be instantiated with the correct amount of wokers. Got: %v, Wanted: %v", len(tt.encryptors), tt.want)
			}
		})
	}
}

func TestEncryptorHandler_HandleEncryptRequests(t *testing.T) {
	tests := []struct {
		name      string
		scheduler EncryptorHandler
		want      error
	}{
		{
			name:      "Successful Context Cancellation",
			scheduler: EncryptorHandler{Encrypt: make(chan Request), Store: make(chan SignedRequest), Encryptors: make(chan struct{})},
			want:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			mockEncryptionErrors := make(chan error, 1)
			go func() {
				mockEncryptionErrors <- tt.scheduler.HandleEncryptRequests(ctx)
			}()
			cancel()
			err := <-mockEncryptionErrors
			if err != nil {
				t.Errorf("Unexpected Error in test for encryption context cancellation. Details: %v", err.Error())
			}
		})
	}
}
