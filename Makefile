# Binary name
BINARY_NAME=generateK8sPatchfile

# Go related variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Go build flags
LDFLAGS=-ldflags "-w -s"

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

.PHONY: all build clean test coverage vet lint format run help

## Build:
all: clean build test run ## Clean, build, test and run the application

build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${GOBIN}/${BINARY_NAME} .
	@echo "Binary has been built to ${GOBIN}/${BINARY_NAME}"

clean: ## Remove build related files
	@echo "Cleaning up..."
	rm -rf ${GOBIN}
	rm -f ${BINARY_NAME}
	rm -f coverage.out
	go clean -testcache
	@echo "Cleanup complete"

## Test:
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## Code Quality:
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run

format: ## Run gofmt
	@echo "Running gofmt..."
	gofmt -s -w .

## Dependency Management:
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## Run:
run: ## Run the binary
	@echo "Running ${BINARY_NAME}..."
	${GOBIN}/${BINARY_NAME} generate

## Help:
help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "  \033[36m%-15s\033[0m %s\n", "target", "help"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Default target
.DEFAULT_GOAL := help