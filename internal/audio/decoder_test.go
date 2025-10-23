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
