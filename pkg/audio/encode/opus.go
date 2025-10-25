// ABOUTME: Opus audio encoder
// ABOUTME: Encodes int32 samples to Opus bytes
package encode

import (
	"fmt"

	"github.com/Resonate-Protocol/resonate-go/pkg/audio"
	"gopkg.in/hraban/opus.v2"
)

// OpusEncoder encodes Opus audio
type OpusEncoder struct {
	encoder    *opus.Encoder
	sampleRate int
	channels   int
	frameSize  int
}

// NewOpus creates a new Opus encoder
func NewOpus(format audio.Format) (Encoder, error) {
	if format.Codec != "opus" {
		return nil, fmt.Errorf("invalid codec for Opus encoder: %s", format.Codec)
	}

	encoder, err := opus.NewEncoder(format.SampleRate, format.Channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// Opus frame size depends on sample rate
	frameSize := format.SampleRate / 50 // 20ms frame

	return &OpusEncoder{
		encoder:    encoder,
		sampleRate: format.SampleRate,
		channels:   format.Channels,
		frameSize:  frameSize,
	}, nil
}

// Encode converts int32 samples to Opus bytes
func (e *OpusEncoder) Encode(samples []int32) ([]byte, error) {
	// Convert int32 to int16 for Opus
	pcm := make([]int16, len(samples))
	for i, sample := range samples {
		pcm[i] = audio.SampleToInt16(sample)
	}

	// Encode to Opus
	data := make([]byte, 4000) // Max Opus packet size
	n, err := e.encoder.Encode(pcm, data)
	if err != nil {
		return nil, fmt.Errorf("opus encode error: %w", err)
	}

	return data[:n], nil
}

// Close releases resources
func (e *OpusEncoder) Close() error {
	return nil
}
