// ABOUTME: Integration tests for Server API
// ABOUTME: Tests server creation, startup, client connections, and streaming
package resonate

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/protocol"
	"github.com/gorilla/websocket"
)

func TestNewServer(t *testing.T) {
	source := NewTestTone(48000, 2)

	tests := []struct {
		name      string
		config    ServerConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Port:   8928,
				Name:   "Test Server",
				Source: source,
			},
			expectErr: false,
		},
		{
			name: "missing source",
			config: ServerConfig{
				Port: 8928,
				Name: "Test Server",
			},
			expectErr: true,
		},
		{
			name: "default port",
			config: ServerConfig{
				Name:   "Test Server",
				Source: source,
			},
			expectErr: false,
		},
		{
			name: "default name",
			config: ServerConfig{
				Port:   8928,
				Source: source,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if server == nil {
				t.Fatal("expected server to be created")
			}

			// Verify defaults
			if server.config.Port == 0 {
				t.Error("port should have been set to default")
			}
			if server.config.Name == "" {
				t.Error("name should have been set to default")
			}
		})
	}
}

func TestServerStartStop(t *testing.T) {
	source := NewTestTone(48000, 2)

	server, err := NewServer(ServerConfig{
		Port:   8929,
		Name:   "Test Server",
		Source: source,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	server.Stop()

	// Wait for server to stop
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("server did not stop within timeout")
	}
}

func TestServerClientConnection(t *testing.T) {
	source := NewTestTone(48000, 2)

	server, err := NewServer(ServerConfig{
		Port:   8930,
		Name:   "Test Server",
		Source: source,
		Debug:  true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Connect as client
	wsURL := "ws://localhost:8930/resonate"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Send client/hello
	hello := protocol.Message{
		Type: "client/hello",
		Payload: protocol.ClientHello{
			ClientID:       "test-client-1",
			Name:           "Test Client",
			Version:        1,
			SupportedRoles: []string{"player"},
			PlayerSupport: &protocol.PlayerSupport{
				SupportFormats: []protocol.AudioFormat{
					{
						Codec:      "pcm",
						Channels:   2,
						SampleRate: 48000,
						BitDepth:   24,
					},
				},
			},
		},
	}

	if err := conn.WriteJSON(hello); err != nil {
		t.Fatalf("failed to send hello: %v", err)
	}

	// Read server/hello response
	var msg protocol.Message
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read server hello: %v", err)
	}

	if msg.Type != "server/hello" {
		t.Errorf("expected server/hello, got %s", msg.Type)
	}

	// Parse server hello
	helloData, _ := json.Marshal(msg.Payload)
	var serverHello protocol.ServerHello
	if err := json.Unmarshal(helloData, &serverHello); err != nil {
		t.Fatalf("failed to unmarshal server hello: %v", err)
	}

	if serverHello.Name != "Test Server" {
		t.Errorf("expected server name 'Test Server', got %s", serverHello.Name)
	}

	// Read stream/start
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read stream/start: %v", err)
	}

	if msg.Type != "stream/start" {
		t.Errorf("expected stream/start, got %s", msg.Type)
	}

	// Read stream/metadata
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read stream/metadata: %v", err)
	}

	if msg.Type != "stream/metadata" {
		t.Errorf("expected stream/metadata, got %s", msg.Type)
	}

	// Read audio chunk (binary message)
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read audio chunk: %v", err)
	}

	if msgType != websocket.BinaryMessage {
		t.Errorf("expected binary message, got type %d", msgType)
	}

	// Verify chunk format: [type:1][timestamp:8][audio_data:N]
	if len(data) < 9 {
		t.Errorf("audio chunk too small: %d bytes", len(data))
	}

	if data[0] != AudioChunkMessageType {
		t.Errorf("expected message type %d, got %d", AudioChunkMessageType, data[0])
	}

	// Check that clients list includes our client
	clients := server.Clients()
	if len(clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(clients))
	}

	if clients[0].ID != "test-client-1" {
		t.Errorf("expected client ID 'test-client-1', got %s", clients[0].ID)
	}

	// Close connection
	conn.Close()

	// Give server time to handle disconnect
	time.Sleep(100 * time.Millisecond)

	// Verify client was removed
	clients = server.Clients()
	if len(clients) != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", len(clients))
	}

	// Stop server
	server.Stop()

	select {
	case <-errChan:
	case <-time.After(5 * time.Second):
		t.Error("server did not stop within timeout")
	}
}

func TestServerMultipleClients(t *testing.T) {
	source := NewTestTone(48000, 2)

	server, err := NewServer(ServerConfig{
		Port:   8931,
		Name:   "Test Server",
		Source: source,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start server
	go server.Start()
	time.Sleep(200 * time.Millisecond)

	// Connect multiple clients
	clients := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		wsURL := "ws://localhost:8931/resonate"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		clients[i] = conn

		// Send hello
		hello := protocol.Message{
			Type: "client/hello",
			Payload: protocol.ClientHello{
				ClientID:       fmt.Sprintf("test-client-%d", i),
				Name:           fmt.Sprintf("Test Client %d", i),
				Version:        1,
				SupportedRoles: []string{"player"},
				PlayerSupport: &protocol.PlayerSupport{
					SupportFormats: []protocol.AudioFormat{
						{
							Codec:      "pcm",
							Channels:   2,
							SampleRate: 48000,
							BitDepth:   24,
						},
					},
				},
			},
		}

		if err := conn.WriteJSON(hello); err != nil {
			t.Fatalf("failed to send hello from client %d: %v", i, err)
		}

		// Read server/hello
		var msg protocol.Message
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("failed to read server hello for client %d: %v", i, err)
		}
	}

	// Give server time to register all clients
	time.Sleep(100 * time.Millisecond)

	// Check that all clients are registered
	serverClients := server.Clients()
	if len(serverClients) != 3 {
		t.Errorf("expected 3 clients, got %d", len(serverClients))
	}

	// Close all connections
	for i, conn := range clients {
		if err := conn.Close(); err != nil {
			t.Errorf("failed to close client %d: %v", i, err)
		}
	}

	// Give server time to handle disconnects
	time.Sleep(100 * time.Millisecond)

	// Verify all clients were removed
	serverClients = server.Clients()
	if len(serverClients) != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", len(serverClients))
	}

	// Stop server
	server.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestServerDuplicateClientID(t *testing.T) {
	source := NewTestTone(48000, 2)

	server, err := NewServer(ServerConfig{
		Port:   8932,
		Name:   "Test Server",
		Source: source,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start server
	go server.Start()
	time.Sleep(200 * time.Millisecond)

	// Connect first client
	wsURL := "ws://localhost:8932/resonate"
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect first client: %v", err)
	}
	defer conn1.Close()

	hello := protocol.Message{
		Type: "client/hello",
		Payload: protocol.ClientHello{
			ClientID:       "duplicate-id",
			Name:           "First Client",
			Version:        1,
			SupportedRoles: []string{"player"},
		},
	}

	if err := conn1.WriteJSON(hello); err != nil {
		t.Fatalf("failed to send hello: %v", err)
	}

	// Read server/hello
	var msg protocol.Message
	if err := conn1.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read server hello: %v", err)
	}

	// Try to connect second client with same ID
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect second client: %v", err)
	}
	defer conn2.Close()

	if err := conn2.WriteJSON(hello); err != nil {
		t.Fatalf("failed to send hello from second client: %v", err)
	}

	// Second client should be rejected - connection should close
	// Try to read a message - should get an error
	conn2.SetReadDeadline(time.Now().Add(1 * time.Second))
	err = conn2.ReadJSON(&msg)
	if err == nil {
		// If we got a message, it should be an error message
		if msg.Type == "server/error" {
			// This is expected
		} else {
			t.Errorf("expected error or connection close for duplicate ID, got message type: %s", msg.Type)
		}
	}

	// Verify only one client is registered
	clients := server.Clients()
	if len(clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(clients))
	}

	// Stop server
	server.Stop()
	time.Sleep(100 * time.Millisecond)
}
