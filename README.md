# Synthesia Take Home

This repo serves as "a small API that re-exposes [an] unreliable endpoint in a more reliable way" for encrypting messages

See [Challenge Doc](https://www.notion.so/Synthesia-Backend-Tech-Challenge-52a82f750aed436fbefcf4d8263a97be) for more information.

## How to run
Utilizing the make file, call 'make build' to build an exectuble according to the current OS the application is being built upon.
Subsequently, use 'make run' to start the API server

Checkout 'make options-help' to check out the configurable runtime options

Note the server saves state between runs. For a completely clean slate, run 'make clean'

## API Contract
The following endpoints and responses are outline below
### Submit a message for encryption
#### Endpoint
```http
GET http://localhost<:serverPort>/crypto/sign?message=<val>
```
#### Expected responses
| Status Code | Description | Body |
| :--- | :--- |:--- |
| 200 | `OK` | `{ "Body": string, "Signature": string, "StatusCode": int}` |
| 202 | `ACCEPTED` | `{ "Body": string, "RequestId": string, "TimeEstimate": float64, "StatusCode": int}` |
| 503 | `BAD REQUEST` | `{ "Body" : string, "StatusCode" : int } `|

### Retrieve, if ready, the signature for a given request Id
#### Endpoint
```http
GET http://localhost<:serverPort>/crypto/sign/request/{requestId}
```
#### Expected responses
| Status Code | Description | Body |
| :--- | :--- | :--- |
| 200 | `OK` | `{ "Body": string, "Signature": string, "StatusCode": int}` |
| 202 | `ACCEPTED` | `{ "Body": string, "RequestId": string, "TimeEstimate": float64, "StatusCode": int}` |
| 404 | `NOT FOUND` | `{ "Body": string, "StatusCode": int}` |

### Check health of the server
#### Endpoint
```http
GET http://localhost<:serverPort>/
```
#### Expected responses
| Status Code | Description | Body |
| :--- | :--- | :--- |
| 200 | `OK` | `{ "Body": string, "Signature": string, "StatusCode": int}` |