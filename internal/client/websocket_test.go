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
