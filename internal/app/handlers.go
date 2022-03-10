package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// RequestFulfilled represents a 200 response body
type RequestFulfilled struct {
	Body       string
	Signature  string
	StatusCode int
}

// RequestFulfilled represents a 202 response body
type RequestProcessing struct {
	Body         string
	RequestId    string
	TimeEstimate float64
	StatusCode   int
}

// RequestFulfilled represents a 404 response body
type RequestDenied struct {
	Body       string
	StatusCode int
}

// RequestFulfilled represents a 200 response body for health check
type HealthRequest struct {
	Body       string
	StatusCode int
}

// For mocking in tests
var generateUUID = uuid.New

// healthHandler react to calls to the /health endpoint
func (application *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("Handling Health Check")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	healthRequest := HealthRequest{Body: "Server is running", StatusCode: http.StatusOK}
	resp, err := json.Marshal(healthRequest)
	if err != nil {
		logrus.Errorf("Unable to marshal response body. Details: %v", err.Error())
	} else {
		_, err := w.Write(resp)
		if err != nil {
			writeErrorResponse(w)
		}
	}
}

// newRequestHandler handles calls to the /crypto/sign?message=<> endpoint with new encryption requests
func (application *Application) newRequestHandler(w http.ResponseWriter, r *http.Request) {
	// Generate unique id for the request
	requestId := generateUUID().String()
	// Retrieve message and submit it for encryption, if possible
	queryItems := r.URL.Query()
	message := queryItems.Get("message")
	request := Request{RequestId: requestId, Message: message}
	select {
	case application.Encrypt <- request:
		logrus.Debugf("Encryption queue accepted the request")
		timing := application.GetEncryptionTiming(time.Now())
		application.Track <- PendingRequest{Request: request, Timing: timing, Add: true}
		signature, err := retrieveSignature(application, requestId)
		if err != nil {
			logrus.Debugf("Request not processed in time, but was recieved successfully")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			requestProcessing := RequestProcessing{
				Body:         "Request Recieved. Please check back according to the time estimate (minutes).",
				RequestId:    requestId,
				TimeEstimate: timing.TimeEstimate,
				StatusCode:   http.StatusAccepted,
			}
			resp, err := json.Marshal(requestProcessing)
			if err != nil {
				logrus.Errorf("Unable to marshal response body. Details: %v", err.Error())
			} else {
				_, err := w.Write(resp)
				if err != nil {
					writeErrorResponse(w)
				}
			}
		} else {
			logrus.Debugf("Request processed in time, returning signature")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			requestFulfilled := RequestFulfilled{
				Body:       "Request processed successfully!",
				Signature:  signature,
				StatusCode: http.StatusOK,
			}
			resp, err := json.Marshal(requestFulfilled)
			if err != nil {
				logrus.Errorf("Unable to marshal response body. Details: %v", err.Error())
			} else {
				_, err := w.Write(resp)
				if err != nil {
					writeErrorResponse(w)
				}
			}
			application.Store <- SignedRequest{RequestId: requestId, Signature: signature, Add: false}
		}
	default:
		logrus.Debugf("Encryption queue at capacity, unable to process request")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		requestDenied := RequestDenied{
			Body:       "The request could not be processed, server is at capacity. Please try again shortly.",
			StatusCode: http.StatusServiceUnavailable,
		}
		resp, err := json.Marshal(requestDenied)
		if err != nil {
			logrus.Errorf("Unable to marshal response body. Details: %v", err.Error())
		} else {
			_, err := w.Write(resp)
			if err != nil {
				writeErrorResponse(w)
			}
		}
	}
}

// currentRequestHandler handles inquiries about ongoing requests to the /crypto/sign/request/{requestId} endpoint
func (application *Application) currentRequestHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve request id for the signature the user is interested in
	params := mux.Vars(r)
	requestId := params["requestId"]
	signature, ok := application.Signatures[requestId]
	if !ok {
		if request, ok := application.Requests[requestId]; ok {
			logrus.Debugf("Request still being processed")
			// See how much time is estimated to be remaining, and if past deadline set to default estimate
			timeElapsed := time.Since(request.TimeAdded)
			minutesRemaining := request.TimeEstimate - timeElapsed.Minutes()
			if minutesRemaining < 0 {
				// Naive default
				minutesRemaining = 5
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			requestProcessing := RequestProcessing{
				Body:         "Request is still being processed. Please check back according to the time estimate (minutes).",
				RequestId:    requestId,
				TimeEstimate: float64(minutesRemaining),
				StatusCode:   http.StatusAccepted,
			}
			resp, err := json.Marshal(requestProcessing)
			if err != nil {
				logrus.Errorf("Unable to marshal response body. Details: %v", err.Error())
			} else {
				_, err := w.Write(resp)
				if err != nil {
					writeErrorResponse(w)
				}
			}

		} else {
			logrus.Debugf("Request id invalid")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			requestDenied := RequestDenied{
				Body:       "The requestId is not recognized. Please use the 'crypto/sign' endpoint to generate a new request.",
				StatusCode: http.StatusNotFound,
			}
			resp, err := json.Marshal(requestDenied)
			if err != nil {
				logrus.Errorf("Unable to marshal response body. Details: %v", err.Error())
			} else {
				_, err := w.Write(resp)
				if err != nil {
					writeErrorResponse(w)
				}
			}
		}
	} else {
		logrus.Debugf("Request completed processing, returning signature")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		requestFulfilled := RequestFulfilled{
			Body:       "Request processed successfully!",
			Signature:  signature,
			StatusCode: http.StatusOK,
		}
		resp, err := json.Marshal(requestFulfilled)
		if err != nil {
			logrus.Errorf("Unable to marshal response body. Details: %v", err.Error())
		} else {
			_, err := w.Write(resp)
			if err != nil {
				writeErrorResponse(w)
			}
		}
		application.Store <- SignedRequest{RequestId: requestId, Signature: signature, Add: false}
	}

}

// writeErrorResponse is a helper function for writing a generic error response
func writeErrorResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, err := w.Write([]byte("Error writing http response. The request may need to be reprocessed entirely. Please try again"))
	if err != nil {
		logrus.Errorf("Error writing HTTP response. Closing request. Error Detail: %v", err.Error())
	}
}

// retrieveSignature attempts to retrieve the signature for a request Id for 2 seconds,
// using exponential backoff when polling the store
func retrieveSignature(application *Application, requestId string) (string, error) {
	// Create channel to listen for signature
	resp := make(chan string)
	// Create a signature retriever (using object to pass state rather than func parameters due to backoff library req.)
	retriever := Retriever{Signatures: application.Signatures, RequestId: requestId, Response: resp}
	// Create channel to see if retrieval failed
	retrievalFailed := make(chan error)
	// Create a background operations that checks for 2 seconds (SLA req) with exponential backoff for signature
	// After 2 seconds, it will notify the retrival failed channel
	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = 50 * time.Millisecond
	retry.MaxElapsedTime = 2 * time.Second
	go func() {
		retrievalFailed <- backoff.Retry(retriever.RetrieveSignature, retry)
	}()
	// Wait until retrieval time window is exhausted, or signature recieved
	select {
	case e := <-retrievalFailed:
		logrus.Debugf("Signature not found, may still be processing")
		return "", e
	case signature := <-resp:
		logrus.Debugf("Signature recieved successfully!")
		return signature, nil
	}
}
