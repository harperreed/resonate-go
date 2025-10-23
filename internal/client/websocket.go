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
		State:  "idle",
		Volume: 100,
		Muted:  false,
	}

	stateMsg := protocol.Message{
		Type:    "player/update",
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

	log.Printf("WebSocket: Received binary chunk, timestamp=%d, size=%d", timestamp, len(audioData))

	chunk := AudioChunk{
		Timestamp: timestamp,
		Data:      audioData,
	}

	select {
	case c.AudioChunks <- chunk:
		log.Printf("WebSocket: Chunk sent to channel")
	case <-c.ctx.Done():
		log.Printf("WebSocket: Context done, dropping chunk")
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

	case "session/update":
		var update protocol.SessionUpdate
		json.Unmarshal(payloadBytes, &update)
		log.Printf("Session update: group=%s, state=%s", update.GroupID, update.PlaybackState)
		// For now, just log it - we could use this to update player state

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// SendState sends a player/update message
func (c *Client) SendState(state protocol.ClientState) error {
	msg := protocol.Message{
		Type:    "player/update",
		Payload: state,
	}
	return c.sendJSON(msg)
}

// SendTimeSync sends a client/time message
func (c *Client) SendTimeSync(t1 int64) error {
	msg := protocol.Message{
		Type: "client/time",
		Payload: protocol.ClientTime{
			ClientTransmitted: t1,
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
