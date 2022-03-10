package app

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Encrypt Handler object holds connections for a stream of requests for encryption,
// connection to storage, and a set of available encrypt workers
type EncryptorHandler struct {
	Encrypt    chan Request
	Store      chan SignedRequest
	Encryptors chan struct{}
}

// HandleEncryptRequests forever listens for a request, and when found waits for a worker
// to be availavle and assigns the task
func (scheduler *EncryptorHandler) HandleEncryptRequests(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			request := <-scheduler.Encrypt
			<-scheduler.Encryptors
			go encryptorParent(scheduler, request)
		}
	}
}

// encryptor handles calling the encryption service and reporting the results. If successful, persist to storage
func encryptor(scheduler *EncryptorHandler, request Request) error {
	// SSL Certs seem expired for synthesias endpoint
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   1 * time.Minute,
	}

	req, err := http.NewRequest("GET", "https://hiring.api.synthesia.io/crypto/sign?message="+request.Message, nil)
	if err != nil {
		logrus.Errorf("Error forming HTTP request. Details %v", err.Error())
		return err
	}

	req.Header.Set("Authorization", "d553641c25b216da081629334a9e6fb8")

	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Error response from HTTP request. Details %v", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("Did not recieve a OK response from HTTP request for requestId: %v. Response code: %v", request.RequestId, resp.StatusCode)
		return errors.New("Did not recieve a OK response from HTTP request")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Failed to read response body for requestId: %v. Details %v", request.RequestId, err.Error())
		return errors.New("Failed to read response body.")
	}

	signature := string(body)
	SignedRequest := SignedRequest{RequestId: request.RequestId, Signature: signature, Add: true}
	scheduler.Store <- SignedRequest
	return nil
}

// encryptorParent creates a child routine to handle the encryption and monitors and handles failure(s)
func encryptorParent(scheduler *EncryptorHandler, request Request) {
	encryptorErrors := make(chan error, 1)
	go func() {
		encryptorErrors <- encryptor(scheduler, request)
	}()
	for {
		encryptorError := <-encryptorErrors
		if encryptorError != nil {
			logrus.Debug("Encryptor failed to sign request, will try again...")
			time.Sleep(1 * time.Minute)
			go func() {
				encryptorErrors <- encryptor(scheduler, request)
			}()
		} else {
			logrus.Debugf("Signature Successful for requestId: %v", request.RequestId)
			time.Sleep(1 * time.Minute)
			scheduler.Encryptors <- struct{}{}
			break
		}
	}
}

// InstantiateEncryptors starts our set of encyptors with the max number of workers
// akin to the load capability our downstream service can handle
func InstantiateEncryptors(maxNumberOfEncryptors int, encryptors chan struct{}) {
	for i := 0; i < maxNumberOfEncryptors; i++ {
		encryptors <- struct{}{}
	}
}
