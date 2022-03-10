BINARY_NAME=synthesia

help: ## This is help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-z0-9A-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' ${MAKEFILE_LIST}

build: ## FBuilds binary wrt current OS
	@go build

run: ## Runs the compiled binary
	@echo "This program contains optional parameters, run 'make options-help' for more info"
	@./${BINARY_NAME}

build-and-run: build run

options-help: ## Displays application optional parameters
	@echo "-maxRequestQueueSize=<val>, type int, default 300"
	@echo "-maxSynthesiaRequestsPerMinute=<val>, type int, default 10"
	@echo "-serverPort=<val>, type string, default :8080 (*Preceding ':' required, else program will hang)"
	@echo "-logLevel=<val>, type string, default debug"

clean: ## Removes object files from package source directories and persisted state files
	@go clean
	@> ./internal/persistence/pending.json
	@> ./internal/persistence/signatures.json

fmt: ## Format all go source files
	@find . -name '*.go' | grep -v vendor/ | xargs -L1 gofmt -w -s -l

lint: ## Lint all go source files, 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2'
	@golangci-lint run

vet: ## Vet all go source files
	@go list . | grep -v vendor/ | xargs -L1 go vet

tidy-and-vendor: ## Cleans up the modules
	@go mod tidy && go mod vendor

test: ## Run all of the tests
	@go list . | grep -v vendor/ | xargs -L1 go test -count=1 -v

test-report: ## Generates a coverage report and opens browser to visualize code coverage
	@go test -count=1 ./... -coverprofile=coverage.out && go tool cover -html=coverage.out