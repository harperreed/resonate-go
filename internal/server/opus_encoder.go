// ABOUTME: Opus audio encoder for bandwidth-efficient streaming
// ABOUTME: Wraps libopus to encode PCM audio to Opus format
package server

import (
	"fmt"
	"log"

	"gopkg.in/hraban/opus.v2"
)

// OpusEncoder wraps the Opus encoder
type OpusEncoder struct {
	encoder    *opus.Encoder
	sampleRate int
	channels   int
	frameSize  int // samples per channel per frame
}

// NewOpusEncoder creates a new Opus encoder
// frameSize is in samples per channel (e.g., 960 for 20ms at 48kHz)
func NewOpusEncoder(sampleRate, channels, frameSize int) (*OpusEncoder, error) {
	// Create encoder with AppAudio mode for music
	encoder, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// Set bitrate (128 kbps for stereo, 64 kbps for mono)
	bitrate := 64000 * channels
	if err := encoder.SetBitrate(bitrate); err != nil {
		log.Printf("Warning: Failed to set Opus bitrate: %v", err)
	}

	return &OpusEncoder{
		encoder:    encoder,
		sampleRate: sampleRate,
		channels:   channels,
		frameSize:  frameSize,
	}, nil
}

// Encode encodes PCM samples to Opus
// Input: []int16 PCM samples (interleaved if stereo)
// Output: []byte Opus packet
func (e *OpusEncoder) Encode(pcm []int16) ([]byte, error) {
	// Allocate output buffer (Opus can't exceed 4000 bytes per packet)
	output := make([]byte, 4000)

	// Encode
	n, err := e.encoder.Encode(pcm, output)
	if err != nil {
		return nil, fmt.Errorf("opus encode failed: %w", err)
	}

	return output[:n], nil
}

// Close closes the encoder
func (e *OpusEncoder) Close() error {
	// opus.Encoder doesn't have a Close method, nothing to do
	return nil
}
