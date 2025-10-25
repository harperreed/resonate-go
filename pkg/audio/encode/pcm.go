// ABOUTME: PCM audio encoder
// ABOUTME: Encodes int32 samples to 16-bit or 24-bit PCM bytes
package encode

import (
	"encoding/binary"
	"fmt"

	"github.com/Resonate-Protocol/resonate-go/pkg/audio"
)

// PCMEncoder encodes PCM audio
type PCMEncoder struct {
	bitDepth int
}

// NewPCM creates a new PCM encoder
func NewPCM(format audio.Format) (Encoder, error) {
	if format.Codec != "pcm" {
		return nil, fmt.Errorf("invalid codec for PCM encoder: %s", format.Codec)
	}

	if format.BitDepth != 16 && format.BitDepth != 24 {
		return nil, fmt.Errorf("unsupported bit depth: %d (supported: 16, 24)", format.BitDepth)
	}

	return &PCMEncoder{
		bitDepth: format.BitDepth,
	}, nil
}

// Encode converts int32 samples to PCM bytes
func (e *PCMEncoder) Encode(samples []int32) ([]byte, error) {
	if e.bitDepth == 24 {
		// 24-bit PCM: 3 bytes per sample
		output := make([]byte, len(samples)*3)
		for i, sample := range samples {
			bytes := audio.SampleTo24Bit(sample)
			output[i*3] = bytes[0]
			output[i*3+1] = bytes[1]
			output[i*3+2] = bytes[2]
		}
		return output, nil
	} else {
		// 16-bit PCM: 2 bytes per sample
		output := make([]byte, len(samples)*2)
		for i, sample := range samples {
			sample16 := audio.SampleToInt16(sample)
			binary.LittleEndian.PutUint16(output[i*2:], uint16(sample16))
		}
		return output, nil
	}
}

// Close releases resources
func (e *PCMEncoder) Close() error {
	return nil
}
