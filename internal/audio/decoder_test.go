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

	// PCM converts bytes to int16 samples (little-endian)
	// Input: 4 bytes -> Output: 2 int16 samples
	input := []byte{0x00, 0x01, 0x02, 0x03}
	output, err := decoder.Decode(input)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	expectedSamples := len(input) / 2
	if len(output) != expectedSamples {
		t.Errorf("expected %d samples, got %d", expectedSamples, len(output))
	}

	// Verify little-endian conversion
	// 0x00, 0x01 -> 0x0100 = 256
	// 0x02, 0x03 -> 0x0302 = 770
	if output[0] != 256 {
		t.Errorf("expected first sample 256, got %d", output[0])
	}
	if output[1] != 770 {
		t.Errorf("expected second sample 770, got %d", output[1])
	}
}
