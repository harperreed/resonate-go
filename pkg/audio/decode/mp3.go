// ABOUTME: MP3 audio decoder
// ABOUTME: Decodes MP3 audio to int32 samples
package decode

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/Sendspin/sendspin-go/pkg/audio"
	"github.com/hajimehoshi/go-mp3"
)

// MP3Decoder decodes MP3 audio
type MP3Decoder struct {
	decoder *mp3.Decoder
}

// NewMP3 creates a new MP3 decoder
func NewMP3(format audio.Format) (Decoder, error) {
	if format.Codec != "mp3" {
		return nil, fmt.Errorf("invalid codec for MP3 decoder: %s", format.Codec)
	}

	// Note: This creates a decoder but we'll need the actual MP3 data stream
	// This is a simplified implementation for frame-based decoding
	return &MP3Decoder{}, fmt.Errorf("MP3 streaming decoder not yet fully implemented")
}

// Decode converts MP3 bytes to int32 samples
func (d *MP3Decoder) Decode(data []byte) ([]int32, error) {
	if d.decoder == nil {
		// Create decoder from data bytes
		reader := bytes.NewReader(data)
		decoder, err := mp3.NewDecoder(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create mp3 decoder: %w", err)
		}
		d.decoder = decoder
	}

	// Read decoded PCM data (int16 as bytes)
	buf := make([]byte, 8192)
	n, err := d.decoder.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("mp3 decode error: %w", err)
	}

	// Convert bytes to int16 then to int32
	numSamples := n / 2 // 2 bytes per int16 sample
	samples := make([]int32, numSamples)
	for i := 0; i < numSamples; i++ {
		sample16 := int16(binary.LittleEndian.Uint16(buf[i*2:]))
		samples[i] = audio.SampleFromInt16(sample16)
	}

	return samples, nil
}

// Close releases decoder resources
func (d *MP3Decoder) Close() error {
	return nil
}
