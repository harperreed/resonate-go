// ABOUTME: Tests for FLAC decoder
// ABOUTME: Tests FLAC decoder creation and not-yet-implemented status
package decode

import (
	"testing"

	"github.com/Resonate-Protocol/resonate-go/pkg/audio"
)

func TestNewFLAC(t *testing.T) {
	format := audio.Format{
		Codec:      "flac",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   24,
	}

	decoder, err := NewFLAC(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	if decoder == nil {
		t.Fatal("expected decoder to be created")
	}
}

func TestNewFLAC_InvalidCodec(t *testing.T) {
	format := audio.Format{
		Codec:      "opus",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   24,
	}

	decoder, err := NewFLAC(format)
	if err == nil {
		t.Fatal("expected error for invalid codec, got nil")
	}

	if decoder != nil {
		t.Fatal("expected decoder to be nil for invalid codec")
	}

	expectedError := "invalid codec for FLAC decoder: opus"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestFLACDecode_NotImplemented(t *testing.T) {
	format := audio.Format{
		Codec:      "flac",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   24,
	}

	decoder, err := NewFLAC(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	// Attempt to decode - should return not implemented error
	samples, err := decoder.Decode([]byte{0x00, 0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("expected not implemented error, got nil")
	}

	if samples != nil {
		t.Fatal("expected nil samples for not implemented decoder")
	}

	expectedError := "FLAC streaming not yet implemented"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestFLACClose(t *testing.T) {
	format := audio.Format{
		Codec:      "flac",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   24,
	}

	decoder, err := NewFLAC(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	err = decoder.Close()
	if err != nil {
		t.Errorf("expected Close to succeed, got error: %v", err)
	}
}
