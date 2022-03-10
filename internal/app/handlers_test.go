package app

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApp_newRequestHandler(t *testing.T) {
	generateUUID = func() uuid.UUID {
		requestId, err := uuid.FromBytes([]byte("requestId-16byte"))
		if err != nil {
			t.Fatalf("Unable to generate mock UUID. Details: %v", err.Error())
		}
		return requestId
	}
	mockSuccessApplication := Application{
		Encrypt:    make(chan Request, 1),
		Store:      make(chan SignedRequest, 1),
		Signatures: map[string]string{generateUUID().String(): "signature"},
		Track:      make(chan PendingRequest, 1),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockCapacityApplication := Application{
		Encrypt:    make(chan Request),
		Store:      make(chan SignedRequest, 1),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest, 1),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockAcceptedApplication := Application{
		Encrypt:    make(chan Request, 1),
		Store:      make(chan SignedRequest, 1),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest, 1),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockCapacityBody := `{"Body":"The request could not be processed, server is at capacity. Please try again shortly.","StatusCode":503}`
	mockSuccessBody := `{"Body":"Request processed successfully!","Signature":"signature","StatusCode":200}`
	mockAcceptedBody := fmt.Sprintf(`{"Body":"Request Recieved. Please check back according to the time estimate (minutes).","RequestId":"%v","TimeEstimate":1,"StatusCode":202}`, generateUUID().String())
	tests := []struct {
		name         string
		application  Application
		mockFunc     func()
		bodyExpected string
		statusCode   int
	}{
		{
			name:        "Success new request, processed",
			application: mockSuccessApplication,
			mockFunc: func() {
				generateUUID = generateUUID //nolint
			},
			bodyExpected: mockSuccessBody,
			statusCode:   200,
		},
		{
			name:        "Success new request, accepted",
			application: mockAcceptedApplication,
			mockFunc: func() {
				generateUUID = generateUUID //nolint
			},
			bodyExpected: mockAcceptedBody,
			statusCode:   202,
		},
		{
			name:        "Denied new request, at capacity",
			application: mockCapacityApplication,
			mockFunc: func() {
				generateUUID = generateUUID //nolint
			},
			bodyExpected: mockCapacityBody,
			statusCode:   503,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			router := NewRouter(&tt.application)

			req, err := http.NewRequest("GET", "/crypto/sign", nil)
			if err != nil {
				t.Fatalf("Failed to create API request for tests. Details: %v", err)
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if status := rr.Code; !cmp.Equal(status, tt.statusCode) {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.statusCode)
			}
			if !cmp.Equal(rr.Body.String(), tt.bodyExpected) {
				t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), tt.bodyExpected)
			}
		})
	}
}

func TestApp_currentRequestHandler(t *testing.T) {
	mockSuccessApplication := Application{
		Encrypt:    make(chan Request, 1),
		Store:      make(chan SignedRequest, 1),
		Signatures: map[string]string{generateUUID().String(): "signature"},
		Track:      make(chan PendingRequest, 1),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockNotFoundApplication := Application{
		Encrypt:    make(chan Request),
		Store:      make(chan SignedRequest, 1),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest, 1),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockAcceptedApplication := Application{
		Encrypt:    make(chan Request, 1),
		Store:      make(chan SignedRequest, 1),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest, 1),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	pr := PendingRequest{
		Request: Request{
			RequestId: generateUUID().String(),
			Message:   "message",
		},
		Timing: Timing{
			time.Now(),
			0.0,
		},
		Add: true,
	}
	mockAcceptedApplication.Requests[generateUUID().String()] = pr
	mockNotFoundBody := `{"Body":"The requestId is not recognized. Please use the 'crypto/sign' endpoint to generate a new request.","StatusCode":404}`
	mockSuccessBody := `{"Body":"Request processed successfully!","Signature":"signature","StatusCode":200}`
	mockAcceptedBody := fmt.Sprintf(`{"Body":"Request is still being processed. Please check back according to the time estimate (minutes).","RequestId":"%v","TimeEstimate":5,"StatusCode":202}`, generateUUID().String())
	tests := []struct {
		name         string
		application  Application
		mockFunc     func()
		bodyExpected string
		statusCode   int
	}{
		{
			name:        "Request was fulfilled, returning signature",
			application: mockSuccessApplication,
			mockFunc: func() {
				generateUUID = generateUUID //nolint
			},
			bodyExpected: mockSuccessBody,
			statusCode:   200,
		},
		{
			name:        "Request known, but still processing",
			application: mockAcceptedApplication,
			mockFunc: func() {
				generateUUID = generateUUID //nolint
			},
			bodyExpected: mockAcceptedBody,
			statusCode:   202,
		},
		{
			name:        "Request not found",
			application: mockNotFoundApplication,
			mockFunc: func() {
				generateUUID = generateUUID //nolint
			},
			bodyExpected: mockNotFoundBody,
			statusCode:   404,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			router := NewRouter(&tt.application)

			req, err := http.NewRequest("GET", fmt.Sprintf("/crypto/sign/request/%v", generateUUID().String()), nil)
			if err != nil {
				t.Fatalf("Failed to create API request for tests. Details: %v", err)
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if status := rr.Code; !cmp.Equal(status, tt.statusCode) {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.statusCode)
			}
			if !cmp.Equal(rr.Body.String(), tt.bodyExpected) {
				t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), tt.bodyExpected)
			}
		})
	}
}
