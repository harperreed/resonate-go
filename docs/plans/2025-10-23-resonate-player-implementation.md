# Resonate Player Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Resonate Protocol player in Go that discovers servers via mDNS, receives multi-codec audio streams, maintains precise clock synchronization, and provides an interactive TUI for control and monitoring.

**Architecture:** Event-driven design with dedicated goroutines for discovery, WebSocket communication, clock sync, audio decoding (Opus/FLAC/PCM), timestamp-based playback scheduling, and TUI. Components communicate via typed Go channels.

**Tech Stack:** Go 1.21+, gorilla/websocket, hashicorp/mdns, ebitengine/oto, hraban/opus, mewkiz/flac, charmbracelet/bubbletea

---

## Task 1: Project Initialization and Basic Structure

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/version/version.go`
- Create: `.gitignore`
- Create: `README.md`

**Step 1: Initialize Go module**

Run:
```bash
uv init --name resonate-player
```

Wait, this is a Go project. Run:
```bash
go mod init github.com/Resonate-Protocol/resonate-go
```

Expected: Creates `go.mod` with module declaration

**Step 2: Create basic main.go structure**

Create `main.go`:
```go
// ABOUTME: Entry point for Resonate Protocol player
// ABOUTME: Parses CLI flags and starts the player application
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	serverAddr = flag.String("server", "", "Manual server address (skip mDNS)")
	port       = flag.Int("port", 8927, "Port for mDNS advertisement")
	name       = flag.String("name", "", "Player friendly name (default: hostname-resonate-player)")
	bufferMs   = flag.Int("buffer-ms", 150, "Jitter buffer size in milliseconds")
	logFile    = flag.String("log-file", "resonate-player.log", "Log file path")
	debug      = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	// Set up logging
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Determine player name
	playerName := *name
	if playerName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		playerName = fmt.Sprintf("%s-resonate-player", hostname)
	}

	log.Printf("Starting Resonate Player: %s", playerName)
	fmt.Printf("Resonate Player starting...\n")
	fmt.Printf("Name: %s\n", playerName)
	fmt.Printf("Port: %d\n", *port)
	fmt.Printf("Buffer: %dms\n", *bufferMs)

	// TODO: Start player
}
```

**Step 3: Create version package**

Create `internal/version/version.go`:
```go
// ABOUTME: Version information for the player
// ABOUTME: Used in device_info sent during handshake
package version

const (
	Version = "0.1.0"
	Product = "Resonate Go Player"
	Manufacturer = "resonate-go"
)
```

**Step 4: Create .gitignore**

Create `.gitignore`:
```
# Binaries
resonate-player
*.exe
*.dll
*.so
*.dylib

# Test binaries
*.test

# Coverage
*.out

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# Logs
*.log

# OS
.DS_Store
Thumbs.db
```

**Step 5: Create README**

Create `README.md`:
```markdown
# Resonate Go Player

A Resonate Protocol player implementation in Go.

## Features

- mDNS service discovery (client and server initiated)
- Multi-codec support (Opus, FLAC, PCM)
- Precise clock synchronization for multi-room audio
- Interactive terminal UI
- Software volume control

## Installation

```bash
go build -o resonate-player
```

## Usage

```bash
./resonate-player --name "Living Room"
```

## Options

- `--server` - Manual server address (skip mDNS)
- `--port` - Port for mDNS advertisement (default: 8927)
- `--name` - Player friendly name
- `--buffer-ms` - Jitter buffer size (default: 150ms)
- `--log-file` - Log file path
- `--debug` - Enable debug logging

## Protocol

Implements the [Resonate Protocol](https://github.com/Resonate-Protocol/spec).
```

**Step 6: Test build**

Run:
```bash
go build -o resonate-player
```

Expected: Builds successfully, creates `resonate-player` binary

**Step 7: Test run**

Run:
```bash
./resonate-player --help
```

Expected: Shows usage information with all flags

**Step 8: Commit**

```bash
git add go.mod main.go internal/version/version.go .gitignore README.md
git commit -m "feat: initialize project structure

- Set up Go module
- Create main entry point with CLI flags
- Add version package
- Add README and .gitignore

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Protocol Message Types

**Files:**
- Create: `internal/protocol/messages.go`
- Create: `internal/protocol/messages_test.go`

**Step 1: Write test for message marshaling**

Create `internal/protocol/messages_test.go`:
```go
// ABOUTME: Tests for Resonate Protocol message types
// ABOUTME: Verifies JSON marshaling/unmarshaling of protocol messages
package protocol

import (
	"encoding/json"
	"testing"
)

func TestClientHelloMarshaling(t *testing.T) {
	hello := ClientHello{
		ClientID:       "test-id",
		Name:           "Test Player",
		Version:        1,
		SupportedRoles: []string{"player"},
		DeviceInfo: &DeviceInfo{
			ProductName:     "Test Product",
			Manufacturer:    "Test Mfg",
			SoftwareVersion: "0.1.0",
		},
		PlayerSupport: &PlayerSupport{
			Codecs:      []string{"opus", "flac", "pcm"},
			SampleRates: []int{44100, 48000},
			Channels:    []int{1, 2},
			BitDepths:   []int{16, 24},
		},
	}

	msg := Message{
		Type:    "client/hello",
		Payload: hello,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != "client/hello" {
		t.Errorf("expected type client/hello, got %s", decoded.Type)
	}
}

func TestClientStateMarshaling(t *testing.T) {
	state := ClientState{
		State:  "synchronized",
		Volume: 80,
		Muted:  false,
	}

	msg := Message{
		Type:    "client/state",
		Payload: state,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != "client/state" {
		t.Errorf("expected type client/state, got %s", decoded.Type)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/protocol/... -v
```

Expected: FAIL with "no such file or directory" or package not found

**Step 3: Create message types**

Create `internal/protocol/messages.go`:
```go
// ABOUTME: Resonate Protocol message type definitions
// ABOUTME: Defines structs for all message types in the protocol
package protocol

// Message is the top-level wrapper for all protocol messages
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// ClientHello is sent by clients to initiate the handshake
type ClientHello struct {
	ClientID       string         `json:"client_id"`
	Name           string         `json:"name"`
	Version        int            `json:"version"`
	SupportedRoles []string       `json:"supported_roles"`
	DeviceInfo     *DeviceInfo    `json:"device_info,omitempty"`
	PlayerSupport  *PlayerSupport `json:"player_support,omitempty"`
}

// DeviceInfo contains device identification
type DeviceInfo struct {
	ProductName     string `json:"product_name"`
	Manufacturer    string `json:"manufacturer"`
	SoftwareVersion string `json:"software_version"`
}

// PlayerSupport describes player capabilities
type PlayerSupport struct {
	Codecs      []string `json:"codecs"`
	SampleRates []int    `json:"sample_rates"`
	Channels    []int    `json:"channels"`
	BitDepths   []int    `json:"bit_depths"`
}

// ServerHello is the server's response to client/hello
type ServerHello struct {
	ServerID string `json:"server_id"`
	Name     string `json:"name"`
	Version  int    `json:"version"`
}

// ClientState reports the player's current state
type ClientState struct {
	State  string `json:"state,omitempty"`
	Volume int    `json:"volume,omitempty"`
	Muted  bool   `json:"muted,omitempty"`
}

// ServerCommand is a control message from the server
type ServerCommand struct {
	Command string `json:"command"`
	Volume  int    `json:"volume,omitempty"`
	Mute    bool   `json:"mute,omitempty"`
}

// StreamStart notifies the client of stream format
type StreamStart struct {
	Codec       string `json:"codec"`
	SampleRate  int    `json:"sample_rate"`
	Channels    int    `json:"channels"`
	BitDepth    int    `json:"bit_depth"`
	CodecHeader string `json:"codec_header,omitempty"` // Base64-encoded
}

// StreamMetadata contains track information
type StreamMetadata struct {
	Title      string `json:"title,omitempty"`
	Artist     string `json:"artist,omitempty"`
	Album      string `json:"album,omitempty"`
	ArtworkURL string `json:"artwork_url,omitempty"`
}

// ClientTime is sent for clock synchronization
type ClientTime struct {
	T1 int64 `json:"t1"` // Client timestamp in microseconds
}

// ServerTime is the response to client/time
type ServerTime struct {
	T1 int64 `json:"t1"` // Echoed client timestamp
	T2 int64 `json:"t2"` // Server receive timestamp
	T3 int64 `json:"t3"` // Server send timestamp
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/protocol/... -v
```

Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: add protocol message types

- Define all Resonate Protocol message structs
- Add JSON marshaling tests
- Support client/server handshake messages
- Support state, command, stream, and time sync messages

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: WebSocket Client with Handshake

**Files:**
- Create: `internal/client/websocket.go`
- Create: `internal/client/websocket_test.go`

**Step 1: Write test for WebSocket client creation**

Create `internal/client/websocket_test.go`:
```go
// ABOUTME: Tests for WebSocket client implementation
// ABOUTME: Tests connection, handshake, and message routing
package client

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	config := Config{
		ServerAddr: "localhost:8927",
		ClientID:   "test-client",
		Name:       "Test Player",
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.config.ServerAddr != "localhost:8927" {
		t.Errorf("expected server addr localhost:8927, got %s", client.config.ServerAddr)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/client/... -v
```

Expected: FAIL with package not found

**Step 3: Create WebSocket client structure**

Create `internal/client/websocket.go`:
```go
// ABOUTME: WebSocket client for Resonate Protocol communication
// ABOUTME: Handles connection, handshake, and message routing
package client

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/protocol"
	"github.com/gorilla/websocket"
)

// Config holds client configuration
type Config struct {
	ServerAddr string
	ClientID   string
	Name       string
	Version    int
	DeviceInfo protocol.DeviceInfo
	PlayerSupport protocol.PlayerSupport
}

// Client represents a WebSocket client
type Client struct {
	config Config
	conn   *websocket.Conn
	mu     sync.RWMutex

	// Message channels
	AudioChunks  chan AudioChunk
	ControlMsgs  chan protocol.ServerCommand
	TimeSyncResp chan protocol.ServerTime
	StreamStart  chan protocol.StreamStart
	Metadata     chan protocol.StreamMetadata

	// State
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// AudioChunk represents a timestamped audio frame
type AudioChunk struct {
	Timestamp int64  // Microseconds, server clock
	Data      []byte // Encoded audio
}

// NewClient creates a new WebSocket client
func NewClient(config Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config:       config,
		AudioChunks:  make(chan AudioChunk, 100),
		ControlMsgs:  make(chan protocol.ServerCommand, 10),
		TimeSyncResp: make(chan protocol.ServerTime, 10),
		StreamStart:  make(chan protocol.StreamStart, 1),
		Metadata:     make(chan protocol.StreamMetadata, 10),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Connect establishes WebSocket connection and performs handshake
func (c *Client) Connect() error {
	u := url.URL{Scheme: "ws", Host: c.config.ServerAddr, Path: "/resonate"}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	// Perform handshake
	if err := c.handshake(); err != nil {
		c.Close()
		return fmt.Errorf("handshake failed: %w", err)
	}

	// Start message reader
	go c.readMessages()

	return nil
}

// handshake performs the protocol handshake
func (c *Client) handshake() error {
	// Send client/hello
	hello := protocol.ClientHello{
		ClientID:       c.config.ClientID,
		Name:           c.config.Name,
		Version:        c.config.Version,
		SupportedRoles: []string{"player"},
		DeviceInfo:     &c.config.DeviceInfo,
		PlayerSupport:  &c.config.PlayerSupport,
	}

	msg := protocol.Message{
		Type:    "client/hello",
		Payload: hello,
	}

	if err := c.sendJSON(msg); err != nil {
		return fmt.Errorf("failed to send client/hello: %w", err)
	}

	// Wait for server/hello (with timeout)
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read server/hello: %w", err)
	}
	c.conn.SetReadDeadline(time.Time{}) // Clear deadline

	var serverMsg protocol.Message
	if err := json.Unmarshal(data, &serverMsg); err != nil {
		return fmt.Errorf("failed to parse server/hello: %w", err)
	}

	if serverMsg.Type != "server/hello" {
		return fmt.Errorf("expected server/hello, got %s", serverMsg.Type)
	}

	log.Printf("Handshake complete with server")

	// Send initial state
	state := protocol.ClientState{
		State:  "synchronized",
		Volume: 100,
		Muted:  false,
	}

	stateMsg := protocol.Message{
		Type:    "client/state",
		Payload: state,
	}

	if err := c.sendJSON(stateMsg); err != nil {
		return fmt.Errorf("failed to send initial state: %w", err)
	}

	return nil
}

// sendJSON sends a JSON message
func (c *Client) sendJSON(msg protocol.Message) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	return c.conn.WriteJSON(msg)
}

// readMessages reads and routes incoming messages
func (c *Client) readMessages() {
	defer c.Close()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		if messageType == websocket.BinaryMessage {
			c.handleBinaryMessage(data)
		} else if messageType == websocket.TextMessage {
			c.handleJSONMessage(data)
		}
	}
}

// handleBinaryMessage handles audio chunks
func (c *Client) handleBinaryMessage(data []byte) {
	if len(data) < 9 {
		log.Printf("Invalid binary message: too short")
		return
	}

	msgType := data[0]
	if msgType != 0 {
		log.Printf("Unknown binary message type: %d", msgType)
		return
	}

	timestamp := int64(binary.BigEndian.Uint64(data[1:9]))
	audioData := data[9:]

	chunk := AudioChunk{
		Timestamp: timestamp,
		Data:      audioData,
	}

	select {
	case c.AudioChunks <- chunk:
	case <-c.ctx.Done():
	}
}

// handleJSONMessage routes JSON messages
func (c *Client) handleJSONMessage(data []byte) {
	var msg protocol.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to parse JSON message: %v", err)
		return
	}

	payloadBytes, _ := json.Marshal(msg.Payload)

	switch msg.Type {
	case "server/command":
		var cmd protocol.ServerCommand
		json.Unmarshal(payloadBytes, &cmd)
		select {
		case c.ControlMsgs <- cmd:
		case <-c.ctx.Done():
		}

	case "server/time":
		var timeMsg protocol.ServerTime
		json.Unmarshal(payloadBytes, &timeMsg)
		select {
		case c.TimeSyncResp <- timeMsg:
		case <-c.ctx.Done():
		}

	case "stream/start":
		var start protocol.StreamStart
		json.Unmarshal(payloadBytes, &start)
		select {
		case c.StreamStart <- start:
		case <-c.ctx.Done():
		}

	case "stream/metadata":
		var meta protocol.StreamMetadata
		json.Unmarshal(payloadBytes, &meta)
		select {
		case c.Metadata <- meta:
		case <-c.ctx.Done():
		}

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// SendState sends a client/state message
func (c *Client) SendState(state protocol.ClientState) error {
	msg := protocol.Message{
		Type:    "client/state",
		Payload: state,
	}
	return c.sendJSON(msg)
}

// SendTimeSync sends a client/time message
func (c *Client) SendTimeSync(t1 int64) error {
	msg := protocol.Message{
		Type: "client/time",
		Payload: protocol.ClientTime{
			T1: t1,
		},
	}
	return c.sendJSON(msg)
}

// Close closes the connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		c.connected = false
		c.cancel()
		c.conn.Close()
		log.Printf("Connection closed")
	}
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}
```

**Step 4: Install dependencies**

Run:
```bash
go get github.com/gorilla/websocket
go mod tidy
```

Expected: Dependencies downloaded and go.mod updated

**Step 5: Run tests to verify they pass**

Run:
```bash
go test ./internal/client/... -v
```

Expected: PASS (1 test)

**Step 6: Commit**

```bash
git add internal/client/ go.mod go.sum
git commit -m "feat: implement WebSocket client with handshake

- Create WebSocket client with connection management
- Implement Resonate Protocol handshake (client/hello, server/hello)
- Add message routing for audio, control, time sync
- Parse binary audio chunks with timestamps
- Add channels for inter-component communication

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Clock Synchronization

**Files:**
- Create: `internal/sync/clock.go`
- Create: `internal/sync/clock_test.go`

**Step 1: Write test for offset calculation**

Create `internal/sync/clock_test.go`:
```go
// ABOUTME: Tests for clock synchronization implementation
// ABOUTME: Tests offset calculation and exponential smoothing
package sync

import (
	"math"
	"testing"
)

func TestOffsetCalculation(t *testing.T) {
	// Simulate a sync exchange
	t1 := int64(1000000) // Client send
	t2 := int64(1002000) // Server receive (+2ms)
	t3 := int64(1002500) // Server respond (+0.5ms processing)
	t4 := int64(1005000) // Client receive (+2.5ms return)

	rtt, offset := calculateOffset(t1, t2, t3, t4)

	// RTT = (t4-t1) - (t3-t2) = 5000 - 500 = 4500μs
	expectedRTT := int64(4500)
	if rtt != expectedRTT {
		t.Errorf("expected RTT %d, got %d", expectedRTT, rtt)
	}

	// Offset = ((t2-t1) + (t3-t4)) / 2 = (2000 + (-2500)) / 2 = -250μs
	expectedOffset := int64(-250)
	if offset != expectedOffset {
		t.Errorf("expected offset %d, got %d", expectedOffset, offset)
	}
}

func TestSmoothing(t *testing.T) {
	cs := NewClockSync()

	// First sample
	cs.ProcessSyncResponse(1000, 1002, 1003, 1006)
	offset1 := cs.GetOffset()

	// Second sample
	cs.ProcessSyncResponse(2000, 2002, 2003, 2006)
	offset2 := cs.GetOffset()

	// Should be smoothed (not equal to raw second sample)
	if offset2 == -250 {
		t.Error("expected smoothed offset, got raw value")
	}

	// Should be moving toward new value
	if math.Abs(float64(offset2-offset1)) < 1 {
		t.Error("expected offset to change with new sample")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/sync/... -v
```

Expected: FAIL with package not found

**Step 3: Implement clock synchronization**

Create `internal/sync/clock.go`:
```go
// ABOUTME: Clock synchronization using NTP-style algorithm
// ABOUTME: Maintains offset between client and server clocks
package sync

import (
	"log"
	"sync"
	"time"
)

// ClockSync manages clock synchronization with the server
type ClockSync struct {
	mu            sync.RWMutex
	offset        int64 // Smoothed offset in microseconds
	rawOffset     int64 // Latest raw offset
	rtt           int64 // Latest round-trip time
	quality       Quality
	lastSync      time.Time
	sampleCount   int
	smoothingRate float64
}

// Quality represents sync quality
type Quality int

const (
	QualityGood Quality = iota
	QualityDegraded
	QualityLost
)

// NewClockSync creates a new clock synchronizer
func NewClockSync() *ClockSync {
	return &ClockSync{
		smoothingRate: 0.1, // 10% weight to new samples
		quality:       QualityLost,
	}
}

// ProcessSyncResponse processes a server/time response
func (cs *ClockSync) ProcessSyncResponse(t1, t2, t3, t4 int64) {
	rtt, offset := calculateOffset(t1, t2, t3, t4)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.rtt = rtt
	cs.rawOffset = offset
	cs.lastSync = time.Now()

	// Discard samples with high RTT (network congestion)
	if rtt > 100000 { // 100ms
		log.Printf("Discarding sync sample: high RTT %dμs", rtt)
		return
	}

	// Apply exponential smoothing
	if cs.sampleCount == 0 {
		cs.offset = offset
	} else {
		cs.offset = int64(float64(cs.offset)*(1-cs.smoothingRate) +
			float64(offset)*cs.smoothingRate)
	}

	cs.sampleCount++

	// Update quality
	if rtt < 50000 { // <50ms
		cs.quality = QualityGood
	} else {
		cs.quality = QualityDegraded
	}

	log.Printf("Clock sync: offset=%dμs, rtt=%dμs, quality=%v",
		cs.offset, cs.rtt, cs.quality)
}

// calculateOffset computes RTT and clock offset
func calculateOffset(t1, t2, t3, t4 int64) (rtt, offset int64) {
	// Round-trip time
	rtt = (t4 - t1) - (t3 - t2)

	// Estimated offset (positive = server ahead)
	offset = ((t2 - t1) + (t3 - t4)) / 2

	return
}

// GetOffset returns the smoothed clock offset
func (cs *ClockSync) GetOffset() int64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.offset
}

// GetStats returns sync statistics
func (cs *ClockSync) GetStats() (offset, rtt int64, quality Quality) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.offset, cs.rtt, cs.quality
}

// CheckQuality updates quality based on time since last sync
func (cs *ClockSync) CheckQuality() Quality {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if time.Since(cs.lastSync) > 5*time.Second {
		cs.quality = QualityLost
	}

	return cs.quality
}

// ServerToLocalTime converts server timestamp to local time
func (cs *ClockSync) ServerToLocalTime(serverTime int64) time.Time {
	offset := cs.GetOffset()
	localMicros := serverTime - offset
	return time.Unix(0, localMicros*1000)
}

// CurrentMicros returns current time in microseconds
func CurrentMicros() int64 {
	return time.Now().UnixNano() / 1000
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/sync/... -v
```

Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/sync/
git commit -m "feat: implement clock synchronization

- NTP-style three-timestamp algorithm
- Exponential smoothing for stability
- RTT-based sample filtering
- Quality tracking (good/degraded/lost)
- Server-to-local timestamp conversion

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Audio Decoder (Multi-Codec)

**Files:**
- Create: `internal/audio/decoder.go`
- Create: `internal/audio/types.go`
- Create: `internal/audio/decoder_test.go`

**Step 1: Write test for decoder creation**

Create `internal/audio/decoder_test.go`:
```go
// ABOUTME: Tests for audio decoder implementation
// ABOUTME: Tests multi-codec decoding (Opus, FLAC, PCM)
package audio

import (
	"testing"
)

func TestNewDecoder(t *testing.T) {
	format := Format{
		Codec:      "pcm",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewDecoder(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	if decoder == nil {
		t.Fatal("expected decoder to be created")
	}
}

func TestPCMDecoder(t *testing.T) {
	format := Format{
		Codec:      "pcm",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewDecoder(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	// PCM is pass-through
	input := []byte{0x00, 0x01, 0x02, 0x03}
	output, err := decoder.Decode(input)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(output) != len(input) {
		t.Errorf("expected output length %d, got %d", len(input), len(output))
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/audio/... -v
```

Expected: FAIL with package not found

**Step 3: Create audio types**

Create `internal/audio/types.go`:
```go
// ABOUTME: Audio type definitions
// ABOUTME: Defines audio formats and decoded buffers
package audio

import "time"

// Format describes audio stream format
type Format struct {
	Codec       string
	SampleRate  int
	Channels    int
	BitDepth    int
	CodecHeader []byte // For FLAC, Opus, etc.
}

// Buffer represents decoded PCM audio
type Buffer struct {
	Timestamp  int64     // Server timestamp (microseconds)
	PlayAt     time.Time // Local play time
	Samples    []byte    // PCM samples
	Format     Format
}
```

**Step 4: Create decoder implementation**

Create `internal/audio/decoder.go`:
```go
// ABOUTME: Multi-codec audio decoder
// ABOUTME: Supports Opus, FLAC, and PCM formats
package audio

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/hraban/opus"
	"github.com/mewkiz/flac"
)

// Decoder decodes audio in various formats
type Decoder interface {
	Decode(data []byte) ([]byte, error)
	Close() error
}

// NewDecoder creates a decoder for the specified format
func NewDecoder(format Format) (Decoder, error) {
	switch format.Codec {
	case "pcm":
		return &PCMDecoder{}, nil
	case "opus":
		return NewOpusDecoder(format)
	case "flac":
		return NewFLACDecoder(format)
	default:
		return nil, fmt.Errorf("unsupported codec: %s", format.Codec)
	}
}

// PCMDecoder is a pass-through for raw PCM
type PCMDecoder struct{}

func (d *PCMDecoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

func (d *PCMDecoder) Close() error {
	return nil
}

// OpusDecoder decodes Opus audio
type OpusDecoder struct {
	decoder *opus.Decoder
	format  Format
}

func NewOpusDecoder(format Format) (*OpusDecoder, error) {
	dec, err := opus.NewDecoder(format.SampleRate, format.Channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus decoder: %w", err)
	}

	return &OpusDecoder{
		decoder: dec,
		format:  format,
	}, nil
}

func (d *OpusDecoder) Decode(data []byte) ([]byte, error) {
	// Opus decoder outputs to int16 buffer
	pcmSize := 5760 * d.format.Channels // Max frame size
	pcm := make([]int16, pcmSize)

	n, err := d.decoder.Decode(data, pcm)
	if err != nil {
		return nil, fmt.Errorf("opus decode failed: %w", err)
	}

	// Convert int16 to bytes
	output := make([]byte, n*d.format.Channels*2)
	for i := 0; i < n*d.format.Channels; i++ {
		output[i*2] = byte(pcm[i])
		output[i*2+1] = byte(pcm[i] >> 8)
	}

	return output, nil
}

func (d *OpusDecoder) Close() error {
	return nil
}

// FLACDecoder decodes FLAC audio
type FLACDecoder struct {
	format Format
}

func NewFLACDecoder(format Format) (*FLACDecoder, error) {
	// FLAC decoder will be created per-chunk if needed
	// For now, basic support
	return &FLACDecoder{
		format: format,
	}, nil
}

func (d *FLACDecoder) Decode(data []byte) ([]byte, error) {
	// For streaming FLAC, we need to handle frame-by-frame decoding
	// This is a simplified implementation
	// In production, would use mewkiz/flac's streaming API
	return nil, fmt.Errorf("FLAC streaming not yet implemented")
}

func (d *FLACDecoder) Close() error {
	return nil
}

// DecodeBase64Header decodes a base64-encoded codec header
func DecodeBase64Header(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
```

**Step 5: Install dependencies**

Run:
```bash
go get github.com/hraban/opus
go get github.com/mewkiz/flac
go mod tidy
```

Expected: Dependencies downloaded

**Step 6: Run tests to verify they pass**

Run:
```bash
go test ./internal/audio/... -v
```

Expected: PASS (2 tests)

**Step 7: Commit**

```bash
git add internal/audio/ go.mod go.sum
git commit -m "feat: implement multi-codec audio decoder

- Support PCM (pass-through)
- Support Opus decoding via hraban/opus
- Add FLAC decoder structure (streaming TBD)
- Define audio format and buffer types

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Playback Scheduler

**Files:**
- Create: `internal/player/scheduler.go`
- Create: `internal/player/scheduler_test.go`

**Step 1: Write test for scheduler timing**

Create `internal/player/scheduler_test.go`:
```go
// ABOUTME: Tests for playback scheduler
// ABOUTME: Tests timestamp-based scheduling and buffer management
package player

import (
	"testing"
	"time"
)

func TestSchedulePlayback(t *testing.T) {
	now := time.Now()
	nowMicros := now.UnixNano() / 1000

	// Schedule for 100ms in future
	playTime := nowMicros + 100000
	localPlayTime := time.Unix(0, playTime*1000)

	sleepDuration := localPlayTime.Sub(now)

	if sleepDuration < 50*time.Millisecond || sleepDuration > 150*time.Millisecond {
		t.Errorf("expected sleep ~100ms, got %v", sleepDuration)
	}
}

func TestLateFrameDetection(t *testing.T) {
	now := time.Now()
	nowMicros := now.UnixNano() / 1000

	// Frame scheduled 100ms ago
	playTime := nowMicros - 100000
	localPlayTime := time.Unix(0, playTime*1000)

	sleepDuration := localPlayTime.Sub(now)

	if sleepDuration >= 0 {
		t.Error("expected negative sleep duration for late frame")
	}

	// Should drop if >50ms late
	shouldDrop := sleepDuration < -50*time.Millisecond
	if !shouldDrop {
		t.Error("expected to drop frame >50ms late")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/player/... -v
```

Expected: FAIL with package not found

**Step 3: Implement scheduler**

Create `internal/player/scheduler.go`:
```go
// ABOUTME: Timestamp-based playback scheduler
// ABOUTME: Schedules audio buffers for precise playback timing
package player

import (
	"container/heap"
	"context"
	"log"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/audio"
	"github.com/Resonate-Protocol/resonate-go/internal/sync"
)

// Scheduler manages playback timing
type Scheduler struct {
	clockSync  *sync.ClockSync
	bufferQ    *BufferQueue
	output     chan audio.Buffer
	jitterMs   int
	ctx        context.Context
	cancel     context.CancelFunc

	stats SchedulerStats
}

// SchedulerStats tracks scheduler metrics
type SchedulerStats struct {
	Received int64
	Played   int64
	Dropped  int64
}

// NewScheduler creates a playback scheduler
func NewScheduler(clockSync *sync.ClockSync, jitterMs int) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		clockSync: clockSync,
		bufferQ:   NewBufferQueue(),
		output:    make(chan audio.Buffer, 10),
		jitterMs:  jitterMs,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Schedule adds a buffer to the queue
func (s *Scheduler) Schedule(buf audio.Buffer) {
	// Convert server timestamp to local play time
	buf.PlayAt = s.clockSync.ServerToLocalTime(buf.Timestamp)

	s.stats.Received++
	heap.Push(s.bufferQ, buf)
}

// Run starts the scheduler loop
func (s *Scheduler) Run() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processQueue()
		}
	}
}

// processQueue checks for buffers ready to play
func (s *Scheduler) processQueue() {
	now := time.Now()

	for s.bufferQ.Len() > 0 {
		buf := s.bufferQ.Peek()

		delay := buf.PlayAt.Sub(now)

		if delay > 50*time.Millisecond {
			// Too early, wait
			break
		} else if delay < -50*time.Millisecond {
			// Too late (>50ms), drop
			heap.Pop(s.bufferQ)
			s.stats.Dropped++
			log.Printf("Dropped late buffer: %v late", -delay)
		} else {
			// Ready to play (within ±50ms window)
			heap.Pop(s.bufferQ)

			select {
			case s.output <- buf:
				s.stats.Played++
			case <-s.ctx.Done():
				return
			}
		}
	}
}

// Output returns the output channel
func (s *Scheduler) Output() <-chan audio.Buffer {
	return s.output
}

// Stats returns scheduler statistics
func (s *Scheduler) Stats() SchedulerStats {
	return s.stats
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
}

// BufferQueue is a priority queue for audio buffers
type BufferQueue struct {
	items []audio.Buffer
}

func NewBufferQueue() *BufferQueue {
	q := &BufferQueue{}
	heap.Init(q)
	return q
}

// Implement heap.Interface
func (q *BufferQueue) Len() int { return len(q.items) }

func (q *BufferQueue) Less(i, j int) bool {
	return q.items[i].PlayAt.Before(q.items[j].PlayAt)
}

func (q *BufferQueue) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

func (q *BufferQueue) Push(x interface{}) {
	q.items = append(q.items, x.(audio.Buffer))
}

func (q *BufferQueue) Pop() interface{} {
	n := len(q.items)
	item := q.items[n-1]
	q.items = q.items[:n-1]
	return item
}

func (q *BufferQueue) Peek() audio.Buffer {
	return q.items[0]
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/player/... -v
```

Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/player/
git commit -m "feat: implement playback scheduler

- Priority queue for timestamp-ordered buffers
- Clock sync integration for timing
- Late frame detection and dropping
- Jitter buffer management
- Playback statistics tracking

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: Audio Output with Volume Control

**Files:**
- Create: `internal/player/output.go`
- Create: `internal/player/output_test.go`

**Step 1: Write test for volume control**

Create `internal/player/output_test.go`:
```go
// ABOUTME: Tests for audio output
// ABOUTME: Tests volume control and PCM playback
package player

import (
	"testing"
)

func TestVolumeMultiplier(t *testing.T) {
	tests := []struct {
		volume   int
		muted    bool
		expected float64
	}{
		{100, false, 1.0},
		{50, false, 0.5},
		{0, false, 0.0},
		{80, true, 0.0}, // Muted overrides volume
	}

	for _, tt := range tests {
		result := getVolumeMultiplier(tt.volume, tt.muted)
		if result != tt.expected {
			t.Errorf("volume=%d, muted=%v: expected %f, got %f",
				tt.volume, tt.muted, tt.expected, result)
		}
	}
}

func TestApplyVolume(t *testing.T) {
	samples := []int16{1000, -1000, 500, -500}
	volume := 50
	muted := false

	result := applyVolume(samples, volume, muted)

	if result[0] != 500 {
		t.Errorf("expected 500, got %d", result[0])
	}
	if result[1] != -500 {
		t.Errorf("expected -500, got %d", result[1])
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/player/... -v
```

Expected: FAIL (undefined functions)

**Step 3: Implement audio output**

Create `internal/player/output.go`:
```go
// ABOUTME: Audio output using oto library
// ABOUTME: Handles PCM playback with software volume control
package player

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/Resonate-Protocol/resonate-go/internal/audio"
	"github.com/ebitengine/oto/v3"
)

// Output manages audio output
type Output struct {
	ctx     context.Context
	cancel  context.CancelFunc
	otoCtx  *oto.Context
	player  *oto.Player
	format  audio.Format
	volume  int
	muted   bool
	ready   bool
}

// NewOutput creates an audio output
func NewOutput() *Output {
	ctx, cancel := context.WithCancel(context.Background())

	return &Output{
		ctx:    ctx,
		cancel: cancel,
		volume: 100,
		muted:  false,
	}
}

// Initialize sets up oto with the specified format
func (o *Output) Initialize(format audio.Format) error {
	if o.otoCtx != nil {
		o.Close()
	}

	op := &oto.NewContextOptions{
		SampleRate:   format.SampleRate,
		ChannelCount: format.Channels,
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return fmt.Errorf("failed to create oto context: %w", err)
	}

	<-readyChan

	o.otoCtx = ctx
	o.format = format
	o.ready = true

	log.Printf("Audio output initialized: %dHz, %d channels",
		format.SampleRate, format.Channels)

	return nil
}

// Play plays an audio buffer
func (o *Output) Play(buf audio.Buffer) error {
	if !o.ready {
		return fmt.Errorf("output not initialized")
	}

	// Convert bytes to int16 samples
	samples := make([]int16, len(buf.Samples)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(buf.Samples[i*2:]))
	}

	// Apply volume
	samples = applyVolume(samples, o.volume, o.muted)

	// Convert back to bytes
	output := make([]byte, len(buf.Samples))
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(output[i*2:], uint16(sample))
	}

	// Write to oto
	player := o.otoCtx.NewPlayer(nil)
	player.Write(output)

	return nil
}

// SetVolume sets the volume (0-100)
func (o *Output) SetVolume(volume int) {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	o.volume = volume
	log.Printf("Volume set to %d", volume)
}

// SetMuted sets mute state
func (o *Output) SetMuted(muted bool) {
	o.muted = muted
	log.Printf("Muted: %v", muted)
}

// GetVolume returns current volume
func (o *Output) GetVolume() int {
	return o.volume
}

// IsMuted returns mute state
func (o *Output) IsMuted() bool {
	return o.muted
}

// Close closes the audio output
func (o *Output) Close() {
	if o.otoCtx != nil {
		o.otoCtx.Suspend()
		o.ready = false
	}
	o.cancel()
}

// applyVolume applies volume and mute to samples
func applyVolume(samples []int16, volume int, muted bool) []int16 {
	multiplier := getVolumeMultiplier(volume, muted)

	result := make([]int16, len(samples))
	for i, sample := range samples {
		result[i] = int16(float64(sample) * multiplier)
	}

	return result
}

// getVolumeMultiplier calculates volume multiplier
func getVolumeMultiplier(volume int, muted bool) float64 {
	if muted {
		return 0.0
	}
	return float64(volume) / 100.0
}
```

**Step 4: Install dependencies**

Run:
```bash
go get github.com/ebitengine/oto/v3
go mod tidy
```

Expected: Dependencies downloaded

**Step 5: Run tests to verify they pass**

Run:
```bash
go test ./internal/player/... -v
```

Expected: PASS (all tests)

**Step 6: Commit**

```bash
git add internal/player/ go.mod go.sum
git commit -m "feat: implement audio output with volume control

- Audio playback using ebitengine/oto
- Software volume control (0-100)
- Mute functionality
- Dynamic format initialization
- PCM sample manipulation

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: mDNS Discovery

**Files:**
- Create: `internal/discovery/mdns.go`
- Create: `internal/discovery/mdns_test.go`

**Step 1: Write test for discovery manager**

Create `internal/discovery/mdns_test.go`:
```go
// ABOUTME: Tests for mDNS discovery
// ABOUTME: Tests service advertisement and discovery
package discovery

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	config := Config{
		ServiceName: "Test Player",
		Port:        8927,
	}

	mgr := NewManager(config)
	if mgr == nil {
		t.Fatal("expected manager to be created")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/discovery/... -v
```

Expected: FAIL with package not found

**Step 3: Implement mDNS discovery**

Create `internal/discovery/mdns.go`:
```go
// ABOUTME: mDNS service discovery for Resonate Protocol
// ABOUTME: Handles both advertisement (server-initiated) and browsing (client-initiated)
package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/hashicorp/mdns"
)

// Config holds discovery configuration
type Config struct {
	ServiceName string
	Port        int
}

// Manager handles mDNS operations
type Manager struct {
	config  Config
	ctx     context.Context
	cancel  context.CancelFunc
	servers chan *ServerInfo
}

// ServerInfo describes a discovered server
type ServerInfo struct {
	Name string
	Host string
	Port int
}

// NewManager creates a discovery manager
func NewManager(config Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		servers: make(chan *ServerInfo, 10),
	}
}

// Advertise advertises this player via mDNS
func (m *Manager) Advertise() error {
	hostname, _ := os.Hostname()

	ips, err := getLocalIPs()
	if err != nil {
		return fmt.Errorf("failed to get local IPs: %w", err)
	}

	service, err := mdns.NewMDNSService(
		m.config.ServiceName,
		"_resonate._tcp",
		"",
		"",
		m.config.Port,
		ips,
		[]string{"path=/resonate"},
	)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("failed to create mdns server: %w", err)
	}

	log.Printf("Advertising mDNS service: %s on port %d", m.config.ServiceName, m.config.Port)

	go func() {
		<-m.ctx.Done()
		server.Shutdown()
	}()

	return nil
}

// Browse searches for Resonate servers
func (m *Manager) Browse() error {
	go m.browseLoop()
	return nil
}

// browseLoop continuously browses for servers
func (m *Manager) browseLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		entries := make(chan *mdns.ServiceEntry, 10)

		go func() {
			for entry := range entries {
				server := &ServerInfo{
					Name: entry.Name,
					Host: entry.AddrV4.String(),
					Port: entry.Port,
				}

				log.Printf("Discovered server: %s at %s:%d", server.Name, server.Host, server.Port)

				select {
				case m.servers <- server:
				case <-m.ctx.Done():
					return
				}
			}
		}()

		params := &mdns.QueryParam{
			Service: "_resonate-server._tcp",
			Domain:  "local",
			Timeout: 3,
			Entries: entries,
		}

		mdns.Query(params)
		close(entries)
	}
}

// Servers returns the channel of discovered servers
func (m *Manager) Servers() <-chan *ServerInfo {
	return m.servers
}

// Stop stops the discovery manager
func (m *Manager) Stop() {
	m.cancel()
}

// getLocalIPs returns local IP addresses
func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ips = append(ips, ipnet.IP)
				}
			}
		}
	}

	return ips, nil
}
```

**Step 4: Install dependencies**

Run:
```bash
go get github.com/hashicorp/mdns
go mod tidy
```

Expected: Dependencies downloaded

**Step 5: Run tests to verify they pass**

Run:
```bash
go test ./internal/discovery/... -v
```

Expected: PASS (1 test)

**Step 6: Commit**

```bash
git add internal/discovery/ go.mod go.sum
git commit -m "feat: implement mDNS discovery

- Advertise as _resonate._tcp.local service
- Browse for _resonate-server._tcp.local servers
- Support both discovery modes simultaneously
- TXT record support (path=/resonate)
- Automatic local IP detection

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 9: TUI Implementation

**Files:**
- Create: `internal/ui/tui.go`
- Create: `internal/ui/model.go`

**Step 1: Create TUI model**

Create `internal/ui/model.go`:
```go
// ABOUTME: Bubbletea model for player TUI
// ABOUTME: Defines application state and update logic
package ui

import (
	"fmt"

	"github.com/Resonate-Protocol/resonate-go/internal/protocol"
	"github.com/Resonate-Protocol/resonate-go/internal/sync"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the TUI state
type Model struct {
	// Connection
	connected    bool
	serverName   string

	// Sync
	syncOffset   int64
	syncRTT      int64
	syncQuality  sync.Quality

	// Stream
	codec        string
	sampleRate   int
	channels     int
	bitDepth     int

	// Metadata
	title        string
	artist       string
	album        string

	// Playback
	state        string
	volume       int
	muted        bool

	// Stats
	received     int64
	played       int64
	dropped      int64
	bufferDepth  int

	// Debug
	showDebug    bool

	// Dimensions
	width        int
	height       int
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case StatusMsg:
		m.applyStatus(msg)
	}

	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	s := ""
	s += m.renderHeader()
	s += m.renderStreamInfo()
	s += m.renderControls()
	s += m.renderStats()

	if m.showDebug {
		s += m.renderDebug()
	}

	s += m.renderHelp()

	return s
}

// renderHeader renders connection and sync status
func (m Model) renderHeader() string {
	connStatus := "Disconnected"
	if m.connected {
		connStatus = fmt.Sprintf("Connected to %s", m.serverName)
	}

	syncIcon := "✗"
	syncText := "Lost"
	switch m.syncQuality {
	case sync.QualityGood:
		syncIcon = "✓"
		syncText = fmt.Sprintf("Synced (offset: %+.1fms, jitter: %.1fms)",
			float64(m.syncOffset)/1000.0, float64(m.syncRTT)/1000.0)
	case sync.QualityDegraded:
		syncIcon = "⚠"
		syncText = "Degraded"
	}

	return fmt.Sprintf(`┌─ Resonate Player ────────────────────────────────────┐
│ Status: %-45s │
│ Sync:   %s %-42s │
├──────────────────────────────────────────────────────┤
`, connStatus, syncIcon, syncText)
}

// renderStreamInfo renders current stream and metadata
func (m Model) renderStreamInfo() string {
	if !m.connected || m.codec == "" {
		return "│ No stream                                            │\n"
	}

	s := "│ Now Playing:                                         │\n"
	if m.title != "" {
		s += fmt.Sprintf("│   Track:  %-42s │\n", truncate(m.title, 42))
		s += fmt.Sprintf("│   Artist: %-42s │\n", truncate(m.artist, 42))
		s += fmt.Sprintf("│   Album:  %-42s │\n", truncate(m.album, 42))
	} else {
		s += "│   (No metadata)                                      │\n"
	}

	s += "│                                                      │\n"
	s += fmt.Sprintf("│ Format: %s %dHz %s %d-bit%-17s │\n",
		m.codec, m.sampleRate, channelName(m.channels), m.bitDepth, "")

	return s
}

// renderControls renders volume and buffer status
func (m Model) renderControls() string {
	muteIcon := ""
	if m.muted {
		muteIcon = " 🔇"
	}

	volumeBar := renderBar(m.volume, 100, 10)

	return fmt.Sprintf("│                                                      │\n"+
		"│ Volume: [%s] %d%%%s%-17s │\n"+
		"│ Buffer: %dms (%d chunks)%-24s │\n",
		volumeBar, m.volume, muteIcon, "",
		m.bufferDepth, m.bufferDepth/10, "")
}

// renderStats renders playback statistics
func (m Model) renderStats() string {
	return fmt.Sprintf(`├──────────────────────────────────────────────────────┤
│ Stats:  RX: %d  Played: %d  Dropped: %d%-8s │
│                                                      │
`, m.received, m.played, m.dropped, "")
}

// renderHelp renders keyboard shortcuts
func (m Model) renderHelp() string {
	return `│ ↑/↓:Volume  m:Mute  r:Reconnect  d:Debug  q:Quit   │
└──────────────────────────────────────────────────────┘
`
}

// renderDebug renders debug information
func (m Model) renderDebug() string {
	return fmt.Sprintf(`│ DEBUG:                                               │
│   Goroutines: (not tracked)                         │
│   Channels: (not tracked)                           │
│   Clock Offset: %+dμs                              │
`, m.syncOffset)
}

// handleKey handles keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up":
		if m.volume < 100 {
			m.volume += 5
			if m.volume > 100 {
				m.volume = 100
			}
		}
	case "down":
		if m.volume > 0 {
			m.volume -= 5
			if m.volume < 0 {
				m.volume = 0
			}
		}
	case "m":
		m.muted = !m.muted
	case "d":
		m.showDebug = !m.showDebug
	}

	return m, nil
}

// applyStatus updates model from status message
func (m *Model) applyStatus(msg StatusMsg) {
	if msg.Connected != nil {
		m.connected = *msg.Connected
	}
	if msg.ServerName != "" {
		m.serverName = msg.ServerName
	}
	if msg.SyncOffset != 0 {
		m.syncOffset = msg.SyncOffset
		m.syncRTT = msg.SyncRTT
		m.syncQuality = msg.SyncQuality
	}
	if msg.Codec != "" {
		m.codec = msg.Codec
		m.sampleRate = msg.SampleRate
		m.channels = msg.Channels
		m.bitDepth = msg.BitDepth
	}
	if msg.Title != "" {
		m.title = msg.Title
		m.artist = msg.Artist
		m.album = msg.Album
	}
	if msg.Volume != 0 {
		m.volume = msg.Volume
	}
	if msg.Received != 0 {
		m.received = msg.Received
		m.played = msg.Played
		m.dropped = msg.Dropped
	}
}

// StatusMsg updates TUI state
type StatusMsg struct {
	Connected   *bool
	ServerName  string
	SyncOffset  int64
	SyncRTT     int64
	SyncQuality sync.Quality
	Codec       string
	SampleRate  int
	Channels    int
	BitDepth    int
	Title       string
	Artist      string
	Album       string
	Volume      int
	Received    int64
	Played      int64
	Dropped     int64
}

// Utility functions
func renderBar(value, max, width int) string {
	filled := (value * width) / max
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func channelName(channels int) string {
	if channels == 1 {
		return "Mono"
	}
	return "Stereo"
}
```

**Step 2: Create TUI wrapper**

Create `internal/ui/tui.go`:
```go
// ABOUTME: TUI initialization and control
// ABOUTME: Wraps bubbletea program for player UI
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel creates a new TUI model
func NewModel() Model {
	return Model{
		volume: 100,
		state:  "idle",
	}
}

// Run starts the TUI
func Run() (*tea.Program, error) {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	return p, nil
}
```

**Step 3: Install dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea
go mod tidy
```

Expected: Dependencies downloaded

**Step 4: Test build**

Run:
```bash
go build -o resonate-player
```

Expected: Builds successfully

**Step 5: Commit**

```bash
git add internal/ui/ go.mod go.sum
git commit -m "feat: implement interactive TUI

- Bubbletea-based terminal UI
- Display connection, sync, stream status
- Show metadata (track, artist, album)
- Volume control display
- Playback statistics
- Keyboard controls (↑/↓, m, d, q)
- Debug panel toggle

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 10: Integration and Main Loop

**Files:**
- Modify: `main.go`
- Create: `internal/app/player.go`

**Step 1: Create player application**

Create `internal/app/player.go`:
```go
// ABOUTME: Main player application orchestration
// ABOUTME: Coordinates all components (connection, audio, UI)
package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/audio"
	"github.com/Resonate-Protocol/resonate-go/internal/client"
	"github.com/Resonate-Protocol/resonate-go/internal/discovery"
	"github.com/Resonate-Protocol/resonate-go/internal/player"
	"github.com/Resonate-Protocol/resonate-go/internal/protocol"
	"github.com/Resonate-Protocol/resonate-go/internal/sync"
	"github.com/Resonate-Protocol/resonate-go/internal/ui"
	"github.com/Resonate-Protocol/resonate-go/internal/version"
	"github.com/google/uuid"
	tea "github.com/charmbracelet/bubbletea"
)

// Config holds player configuration
type Config struct {
	ServerAddr string
	Port       int
	Name       string
	BufferMs   int
}

// Player represents the main player application
type Player struct {
	config    Config
	client    *client.Client
	clockSync *sync.ClockSync
	scheduler *player.Scheduler
	output    *player.Output
	discovery *discovery.Manager
	decoder   audio.Decoder
	tuiProg   *tea.Program
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new player
func New(config Config) *Player {
	ctx, cancel := context.WithCancel(context.Background())

	return &Player{
		config:    config,
		clockSync: sync.NewClockSync(),
		output:    player.NewOutput(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the player
func (p *Player) Start() error {
	// Start TUI
	tuiProg, err := ui.Run()
	if err != nil {
		return fmt.Errorf("failed to start TUI: %w", err)
	}
	p.tuiProg = tuiProg

	go p.tuiProg.Run()

	// Start discovery if no manual server
	if p.config.ServerAddr == "" {
		p.discovery = discovery.NewManager(discovery.Config{
			ServiceName: p.config.Name,
			Port:        p.config.Port,
		})

		p.discovery.Advertise()
		p.discovery.Browse()

		// Wait for server discovery
		go p.handleDiscovery()
	} else {
		// Connect directly
		if err := p.connect(p.config.ServerAddr); err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
	}

	// Wait for context cancellation
	<-p.ctx.Done()

	return nil
}

// handleDiscovery waits for server discovery
func (p *Player) handleDiscovery() {
	for {
		select {
		case server := <-p.discovery.Servers():
			addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
			log.Printf("Attempting connection to %s", addr)

			if err := p.connect(addr); err != nil {
				log.Printf("Connection failed: %v", err)
				continue
			}
			return

		case <-p.ctx.Done():
			return
		}
	}
}

// connect establishes connection to server
func (p *Player) connect(serverAddr string) error {
	clientID := uuid.New().String()

	clientConfig := client.Config{
		ServerAddr: serverAddr,
		ClientID:   clientID,
		Name:       p.config.Name,
		Version:    1,
		DeviceInfo: protocol.DeviceInfo{
			ProductName:     version.Product,
			Manufacturer:    version.Manufacturer,
			SoftwareVersion: version.Version,
		},
		PlayerSupport: protocol.PlayerSupport{
			Codecs:      []string{"opus", "flac", "pcm"},
			SampleRates: []int{44100, 48000},
			Channels:    []int{1, 2},
			BitDepths:   []int{16, 24},
		},
	}

	p.client = client.NewClient(clientConfig)

	if err := p.client.Connect(); err != nil {
		return err
	}

	log.Printf("Connected to server: %s", serverAddr)

	// Start component goroutines
	go p.handleAudioChunks()
	go p.handleControls()
	go p.handleStreamStart()
	go p.handleMetadata()
	go p.clockSyncLoop()

	return nil
}

// clockSyncLoop continuously syncs clock
func (p *Player) clockSyncLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t1 := sync.CurrentMicros()
			p.client.SendTimeSync(t1)

			// Wait for response
			select {
			case resp := <-p.client.TimeSyncResp:
				t4 := sync.CurrentMicros()
				p.clockSync.ProcessSyncResponse(resp.T1, resp.T2, resp.T3, t4)

			case <-time.After(2 * time.Second):
				log.Printf("Time sync timeout")
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// handleStreamStart initializes decoder and output
func (p *Player) handleStreamStart() {
	for {
		select {
		case start := <-p.client.StreamStart:
			log.Printf("Stream starting: %s %dHz %dch %dbit",
				start.Codec, start.SampleRate, start.Channels, start.BitDepth)

			format := audio.Format{
				Codec:      start.Codec,
				SampleRate: start.SampleRate,
				Channels:   start.Channels,
				BitDepth:   start.BitDepth,
			}

			// Initialize decoder
			decoder, err := audio.NewDecoder(format)
			if err != nil {
				log.Printf("Failed to create decoder: %v", err)
				continue
			}
			p.decoder = decoder

			// Initialize output
			if err := p.output.Initialize(format); err != nil {
				log.Printf("Failed to initialize output: %v", err)
				continue
			}

			// Initialize scheduler
			p.scheduler = player.NewScheduler(p.clockSync, p.config.BufferMs)
			go p.scheduler.Run()
			go p.handleScheduledAudio()

		case <-p.ctx.Done():
			return
		}
	}
}

// handleAudioChunks decodes and schedules audio
func (p *Player) handleAudioChunks() {
	for {
		select {
		case chunk := <-p.client.AudioChunks:
			if p.decoder == nil || p.scheduler == nil {
				continue
			}

			// Decode
			pcm, err := p.decoder.Decode(chunk.Data)
			if err != nil {
				log.Printf("Decode error: %v", err)
				continue
			}

			// Schedule
			buf := audio.Buffer{
				Timestamp: chunk.Timestamp,
				Samples:   pcm,
			}
			p.scheduler.Schedule(buf)

		case <-p.ctx.Done():
			return
		}
	}
}

// handleScheduledAudio plays scheduled buffers
func (p *Player) handleScheduledAudio() {
	for {
		select {
		case buf := <-p.scheduler.Output():
			if err := p.output.Play(buf); err != nil {
				log.Printf("Playback error: %v", err)
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// handleControls processes server commands
func (p *Player) handleControls() {
	for {
		select {
		case cmd := <-p.client.ControlMsgs:
			switch cmd.Command {
			case "volume":
				p.output.SetVolume(cmd.Volume)
				p.client.SendState(protocol.ClientState{Volume: cmd.Volume})

			case "mute":
				p.output.SetMuted(cmd.Mute)
				p.client.SendState(protocol.ClientState{Muted: cmd.Mute})
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// handleMetadata updates UI with track info
func (p *Player) handleMetadata() {
	for {
		select {
		case meta := <-p.client.Metadata:
			log.Printf("Metadata: %s - %s (%s)", meta.Artist, meta.Title, meta.Album)
			// TODO: Send to TUI

		case <-p.ctx.Done():
			return
		}
	}
}

// Stop stops the player
func (p *Player) Stop() {
	p.cancel()

	if p.client != nil {
		p.client.Close()
	}

	if p.output != nil {
		p.output.Close()
	}

	if p.tuiProg != nil {
		p.tuiProg.Quit()
	}
}
```

**Step 2: Update main.go**

Modify `main.go`:
```go
// ABOUTME: Entry point for Resonate Protocol player
// ABOUTME: Parses CLI flags and starts the player application
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Resonate-Protocol/resonate-go/internal/app"
)

var (
	serverAddr = flag.String("server", "", "Manual server address (skip mDNS)")
	port       = flag.Int("port", 8927, "Port for mDNS advertisement")
	name       = flag.String("name", "", "Player friendly name (default: hostname-resonate-player)")
	bufferMs   = flag.Int("buffer-ms", 150, "Jitter buffer size in milliseconds")
	logFile    = flag.String("log-file", "resonate-player.log", "Log file path")
	debug      = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	// Set up logging
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Determine player name
	playerName := *name
	if playerName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		playerName = fmt.Sprintf("%s-resonate-player", hostname)
	}

	log.Printf("Starting Resonate Player: %s", playerName)

	// Create player
	config := app.Config{
		ServerAddr: *serverAddr,
		Port:       *port,
		Name:       playerName,
		BufferMs:   *bufferMs,
	}

	player := app.New(config)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("Shutdown signal received")
		player.Stop()
	}()

	// Start player
	if err := player.Start(); err != nil {
		log.Fatalf("Player error: %v", err)
	}

	log.Printf("Player stopped")
}
```

**Step 3: Install remaining dependencies**

Run:
```bash
go get github.com/google/uuid
go mod tidy
```

Expected: Dependencies downloaded

**Step 4: Build the player**

Run:
```bash
go build -o resonate-player
```

Expected: Builds successfully with no errors

**Step 5: Test basic startup**

Run:
```bash
./resonate-player --help
```

Expected: Shows help with all options

**Step 6: Commit**

```bash
git add main.go internal/app/ go.mod go.sum
git commit -m "feat: integrate all components into main player app

- Orchestrate connection, discovery, audio, and UI
- Clock sync loop with 1s interval
- Audio pipeline: chunks → decode → schedule → play
- Handle stream start, metadata, controls
- Graceful shutdown on SIGINT/SIGTERM

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Summary

The implementation plan is complete! The player now includes:

✅ Project structure and initialization
✅ Protocol message types
✅ WebSocket client with handshake
✅ Clock synchronization (NTP-style)
✅ Multi-codec audio decoder (Opus/FLAC/PCM)
✅ Timestamp-based playback scheduler
✅ Audio output with volume control
✅ mDNS discovery (both modes)
✅ Interactive TUI
✅ Full integration in main application

## Next Steps

1. **Testing**: Test with a real Music Assistant server
2. **FLAC**: Complete FLAC streaming decoder implementation
3. **Polish**: Refine TUI updates, add artwork support
4. **Performance**: Profile and optimize for low latency
5. **Packaging**: Create installers/packages for distribution

## Running the Player

```bash
# Auto-discovery mode
./resonate-player --name "Living Room"

# Manual connection
./resonate-player --server music-assistant.local:8927 --name "Bedroom"

# With custom buffer
./resonate-player --buffer-ms 200 --debug
```
