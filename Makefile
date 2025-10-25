# ABOUTME: Build automation for Resonate Protocol server and player
# ABOUTME: Provides targets for building, testing, and cleaning binaries

.PHONY: all build player server clean test install

# Default target builds both
all: build

# Build both player and server
build: player server

# Build the player
player:
	@echo "Building resonate player..."
	go build -o resonate-go .

# Build the server
server:
	@echo "Building resonate server..."
	go build -o resonate-server ./cmd/resonate-server

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean built binaries
clean:
	@echo "Cleaning binaries..."
	rm -f resonate-go resonate-server

# Install both binaries to GOPATH/bin
install:
	@echo "Installing binaries..."
	go install .
	go install ./cmd/resonate-server
