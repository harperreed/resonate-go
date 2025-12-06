// ABOUTME: Tests for Opus decoder
// ABOUTME: Tests Opus decoder creation and validation
package decode

import (
	"testing"

	"github.com/Sendspin/sendspin-go/pkg/audio"
)

func TestNewOpus(t *testing.T) {
	format := audio.Format{
		Codec:      "opus",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewOpus(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	if decoder == nil {
		t.Fatal("expected decoder to be created")
	}
}

func TestNewOpus_InvalidCodec(t *testing.T) {
	format := audio.Format{
		Codec:      "pcm",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewOpus(format)
	if err == nil {
		t.Fatal("expected error for invalid codec, got nil")
	}

	if decoder != nil {
		t.Fatal("expected decoder to be nil for invalid codec")
	}

	expectedError := "invalid codec for Opus decoder: pcm"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestNewOpus_MonoChannel(t *testing.T) {
	format := audio.Format{
		Codec:      "opus",
		SampleRate: 48000,
		Channels:   1,
		BitDepth:   16,
	}

	decoder, err := NewOpus(format)
	if err != nil {
		t.Fatalf("failed to create mono decoder: %v", err)
	}

	if decoder == nil {
		t.Fatal("expected decoder to be created")
	}
}

func TestNewOpus_InvalidSampleRate(t *testing.T) {
	// Opus library may reject invalid sample rates
	format := audio.Format{
		Codec:      "opus",
		SampleRate: 44100, // Opus typically uses 48000
		Channels:   2,
		BitDepth:   16,
	}

	// We expect this might fail at the opus library level
	// This test documents the behavior
	decoder, err := NewOpus(format)

	// Either it succeeds (opus lib is flexible) or fails (opus lib is strict)
	// Both are valid outcomes, we just verify proper error handling
	if err != nil && decoder != nil {
		t.Fatal("if error is returned, decoder must be nil")
	}
	if err == nil && decoder == nil {
		t.Fatal("if no error, decoder must not be nil")
	}
}

func TestOpusClose(t *testing.T) {
	format := audio.Format{
		Codec:      "opus",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewOpus(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	err = decoder.Close()
	if err != nil {
		t.Errorf("expected Close to succeed, got error: %v", err)
	}
}
