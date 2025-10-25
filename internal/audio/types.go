// ABOUTME: Audio type definitions
// ABOUTME: Defines audio formats and decoded buffers
package audio

import "time"

// Format describes audio stream format
type Format struct {
	Codec       string
	SampleRate  int
	Channels    int
	BitDepth    int
	CodecHeader []byte // For FLAC, Opus, etc.
}

// Buffer represents decoded PCM audio
type Buffer struct {
	Timestamp  int64     // Server timestamp (microseconds)
	PlayAt     time.Time // Local play time
	Samples    []int16   // PCM samples (int16 format to avoid conversions)
	Format     Format
}
