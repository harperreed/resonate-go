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
			SupportFormats: []AudioFormat{
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
