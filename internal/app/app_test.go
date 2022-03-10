package app

import (
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApp_NewRouter(t *testing.T) {
	mockApplication := Application{
		Encrypt:    make(chan Request),
		Store:      make(chan SignedRequest),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockBodyExpected := `{"Body":"Server is running","StatusCode":200}`
	tests := []struct {
		name             string
		application      Application
		bodyExpected     string
		want             int
		isTestingFailure bool
	}{
		{
			name:             "Handles health",
			application:      mockApplication,
			bodyExpected:     mockBodyExpected,
			want:             200,
			isTestingFailure: false,
		},
		{
			name:             "NotFound undefined endpoints",
			application:      mockApplication,
			bodyExpected:     mockBodyExpected,
			want:             404,
			isTestingFailure: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(&tt.application)
			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatalf("Failed to create API request for tests. Details: %v", err)
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if status := rr.Code; status != tt.want && !tt.isTestingFailure {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.want)
			}
			if rr.Body.String() != tt.bodyExpected {
				t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), tt.bodyExpected)
			}
		})
	}
}

func TestApp_GetEncryptionTiming(t *testing.T) {
	mockApplicationEmptyEncrypt := Application{
		Encrypt:    make(chan Request, 301),
		Store:      make(chan SignedRequest),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockApplicationLargeEncrypt := Application{
		Encrypt:    make(chan Request, 301),
		Store:      make(chan SignedRequest),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockApplicationSmallEncrypt := Application{
		Encrypt:    make(chan Request, 301),
		Store:      make(chan SignedRequest),
		Signatures: make(map[string]string),
		Track:      make(chan PendingRequest),
		Requests:   make(map[string]PendingRequest),
		ServerPort: ":8080",
	}
	mockTimeNow := time.Now()
	tests := []struct {
		name        string
		application Application
		queueSize   int
		timeNow     time.Time
		want        Timing
	}{
		{
			name:        "1 minute wait time, empty Queue",
			application: mockApplicationEmptyEncrypt,
			queueSize:   0,
			timeNow:     mockTimeNow,
			want:        Timing{TimeAdded: mockTimeNow, TimeEstimate: float64(1)},
		},
		{
			name:        "60 minute wait time, large queue",
			application: mockApplicationLargeEncrypt,
			queueSize:   300,
			timeNow:     mockTimeNow,
			want:        Timing{TimeAdded: mockTimeNow, TimeEstimate: float64(60)},
		},
		{
			name:        "7 minute wait time, small queue",
			application: mockApplicationSmallEncrypt,
			queueSize:   35,
			timeNow:     mockTimeNow,
			want:        Timing{TimeAdded: mockTimeNow, TimeEstimate: float64(7)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.queueSize; i++ {
				tt.application.Encrypt <- Request{RequestId: "requestId", Message: "message"}
			}
			timing := tt.application.GetEncryptionTiming(tt.timeNow)
			if !cmp.Equal(timing, tt.want) {
				t.Errorf("Failed, wait time not as expected. Wanted: %v Got: %v", tt.want, timing)
			}
		})
	}
}
