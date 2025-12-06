// ABOUTME: Tests for Sendspin Protocol message types
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
		SupportedRoles: []string{"player@v1"},
		DeviceInfo: &DeviceInfo{
			ProductName:     "Test Product",
			Manufacturer:    "Test Mfg",
			SoftwareVersion: "0.1.0",
		},
		PlayerV1Support: &PlayerV1Support{
			SupportedFormats: []AudioFormat{
				{Codec: "opus", Channels: 2, SampleRate: 48000, BitDepth: 16},
				{Codec: "flac", Channels: 2, SampleRate: 48000, BitDepth: 16},
				{Codec: "pcm", Channels: 2, SampleRate: 48000, BitDepth: 16},
			},
			BufferCapacity:    1048576,
			SupportedCommands: []string{"volume", "mute"},
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
	state := ClientStateMessage{
		Player: &PlayerState{
			State:  "synchronized",
			Volume: 80,
			Muted:  false,
		},
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

func TestServerHelloMarshaling(t *testing.T) {
	hello := ServerHello{
		ServerID:         "server-123",
		Name:             "Test Server",
		Version:          1,
		ActiveRoles:      []string{"player@v1", "metadata@v1"},
		ConnectionReason: "playback",
	}

	msg := Message{
		Type:    "server/hello",
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

	if decoded.Type != "server/hello" {
		t.Errorf("expected type server/hello, got %s", decoded.Type)
	}
}
