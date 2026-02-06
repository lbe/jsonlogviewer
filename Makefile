.PHONY: test build lint clean help

# Shell to use
SHELL := /bin/bash

# Go command (uses PATH or default)
GO ?= $(shell which go 2>/dev/null || echo /usr/local/go/bin/go)

# Default target - shows help
.DEFAULT_GOAL := help

# Display help information
help:
	@echo "Available targets:"
	@echo ""
	@echo "  make test       - Run all tests (go test ./...)"
	@echo "  make build      - Build binary to ./bin/jsonlogviewer"
	@echo "  make lint       - Run linters (goimports, gofmt, golangci-lint)"
	@echo "  make clean      - Remove build artifacts (bin/)"
	@echo "  make deps       - Download and tidy dependencies"
	@echo "  make coverage   - Run tests with coverage report"
	@echo "  make race       - Run tests with race detector"
	@echo "  make all        - Run test, lint, and build"
	@echo ""
	@echo "Options:"
	@echo "  GO=/path/to/go  - Specify path to go binary (default: go)"
	@echo ""
	@echo "Examples:"
	@echo "  make test build"
	@echo "  make lint GO=/usr/local/go/bin/go"

# Run all targets
all: test lint build

# Run all tests with coverage
test:
	$(GO) test -cover ./...

# Build the binary to bin directory
build:
	@mkdir -p bin
	$(GO) build -v -o ./bin/jsonlogviewer ./cmd/jsonlogviewer

# Run linters (order: 1. imports, 2. fmt, 3. golangci-lint)
lint:
	@echo "Running goimports check..."
	@which goimports > /dev/null 2>&1 && (goimports -l . | grep -v "^vendor/" | grep -v ".pb.go" && echo "^^^ Files need import fixes" && exit 1 || echo "goimports: OK") || echo "goimports not installed, skipping"
	@echo "Running gofmt check..."
	@gofmt -l . | grep -v "^vendor/" | grep -v ".pb.go" && echo "^^^ Files need formatting" && exit 1 || echo "gofmt: OK"
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run --max-same-issues 0 ./... || echo "golangci-lint not installed, install from https://golangci-lint.run/usage/install/"

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Run tests with coverage
coverage:
	$(GO) test -cover ./...

# Run tests with race detector
race:
	$(GO) test -race ./...
