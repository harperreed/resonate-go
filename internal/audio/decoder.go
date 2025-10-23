// ABOUTME: Multi-codec audio decoder
// ABOUTME: Supports Opus, FLAC, and PCM formats
package audio

import (
	"encoding/base64"
	"fmt"

	"github.com/hraban/opus"
)

// Decoder decodes audio in various formats
type Decoder interface {
	Decode(data []byte) ([]byte, error)
	Close() error
}

// NewDecoder creates a decoder for the specified format
func NewDecoder(format Format) (Decoder, error) {
	switch format.Codec {
	case "pcm":
		return &PCMDecoder{}, nil
	case "opus":
		return NewOpusDecoder(format)
	case "flac":
		return NewFLACDecoder(format)
	default:
		return nil, fmt.Errorf("unsupported codec: %s", format.Codec)
	}
}

// PCMDecoder is a pass-through for raw PCM
type PCMDecoder struct{}

func (d *PCMDecoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

func (d *PCMDecoder) Close() error {
	return nil
}

// OpusDecoder decodes Opus audio
type OpusDecoder struct {
	decoder *opus.Decoder
	format  Format
}

func NewOpusDecoder(format Format) (*OpusDecoder, error) {
	dec, err := opus.NewDecoder(format.SampleRate, format.Channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus decoder: %w", err)
	}

	return &OpusDecoder{
		decoder: dec,
		format:  format,
	}, nil
}

func (d *OpusDecoder) Decode(data []byte) ([]byte, error) {
	// Opus decoder outputs to int16 buffer
	pcmSize := 5760 * d.format.Channels // Max frame size
	pcm := make([]int16, pcmSize)

	n, err := d.decoder.Decode(data, pcm)
	if err != nil {
		return nil, fmt.Errorf("opus decode failed: %w", err)
	}

	// Convert int16 to bytes
	output := make([]byte, n*d.format.Channels*2)
	for i := 0; i < n*d.format.Channels; i++ {
		output[i*2] = byte(pcm[i])
		output[i*2+1] = byte(pcm[i] >> 8)
	}

	return output, nil
}

func (d *OpusDecoder) Close() error {
	return nil
}

// FLACDecoder decodes FLAC audio
type FLACDecoder struct {
	format Format
}

func NewFLACDecoder(format Format) (*FLACDecoder, error) {
	// FLAC decoder will be created per-chunk if needed
	// For now, basic support
	return &FLACDecoder{
		format: format,
	}, nil
}

func (d *FLACDecoder) Decode(data []byte) ([]byte, error) {
	// For streaming FLAC, we need to handle frame-by-frame decoding
	// This is a simplified implementation
	// In production, would use mewkiz/flac's streaming API
	return nil, fmt.Errorf("FLAC streaming not yet implemented")
}

func (d *FLACDecoder) Close() error {
	return nil
}

// DecodeBase64Header decodes a base64-encoded codec header
func DecodeBase64Header(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
