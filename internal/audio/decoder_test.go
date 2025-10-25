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

	// Verify little-endian conversion with 24-bit scaling
	// 0x00, 0x01 -> 0x0100 = 256 (16-bit) -> 256<<8 = 65536 (24-bit)
	// 0x02, 0x03 -> 0x0302 = 770 (16-bit) -> 770<<8 = 197120 (24-bit)
	expected0 := int32(256 << 8)
	if output[0] != expected0 {
		t.Errorf("expected first sample %d, got %d", expected0, output[0])
	}
	expected1 := int32(770 << 8)
	if output[1] != expected1 {
		t.Errorf("expected second sample %d, got %d", expected1, output[1])
	}
}
