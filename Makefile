# ABOUTME: Build automation for Sendspin Protocol server and player
# ABOUTME: Provides targets for building, testing, and cleaning binaries

.PHONY: all build player server test test-verbose test-coverage lint clean install \
	build-all build-linux build-darwin help

# Version from git tag or default
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/Sendspin/sendspin-go/internal/version.Version=$(VERSION)

# Default target
all: build

# Build both player and server
build: player server

# Build the player
player:
	@echo "Building sendspin-player..."
	go build -ldflags "$(LDFLAGS)" -o sendspin-player .

# Build the server
server:
	@echo "Building sendspin-server..."
	go build -ldflags "$(LDFLAGS)" -o sendspin-server ./cmd/sendspin-server

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run --timeout=5m

# Clean built binaries and artifacts
clean:
	@echo "Cleaning binaries and artifacts..."
	rm -f sendspin-player sendspin-server resonate-player resonate-server
	rm -rf bin/ dist/
	rm -f coverage.out coverage.html

# Install both binaries to GOPATH/bin
install:
	@echo "Installing binaries..."
	go install -ldflags "$(LDFLAGS)" .
	go install -ldflags "$(LDFLAGS)" ./cmd/sendspin-server

# Build all platforms (like CI)
build-all: build-linux build-darwin

# Build Linux binaries
build-linux:
	@echo "Building Linux binaries..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-player-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-player-linux-arm64 .
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-server-linux-amd64 ./cmd/sendspin-server
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-server-linux-arm64 ./cmd/sendspin-server

# Build macOS binaries
build-darwin:
	@echo "Building macOS binaries..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-player-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-player-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-server-darwin-amd64 ./cmd/sendspin-server
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/sendspin-server-darwin-arm64 ./cmd/sendspin-server

# Show help
help:
	@echo "Sendspin Protocol - Build Targets"
	@echo ""
	@echo "  make              - Build player and server"
	@echo "  make player       - Build sendspin-player"
	@echo "  make server       - Build sendspin-server"
	@echo "  make test         - Run tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-coverage- Run tests with coverage report"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make clean        - Remove built binaries"
	@echo "  make install      - Install to GOPATH/bin"
	@echo "  make build-all    - Build all platforms"
	@echo "  make build-linux  - Build Linux binaries"
	@echo "  make build-darwin - Build macOS binaries"
	@echo ""
	@echo "Version: $(VERSION)"
