// ABOUTME: Audio output interface definition
// ABOUTME: Common interface for audio playback backends
package output

// Output represents an audio output device
type Output interface {
	// Open initializes the output device
	Open(sampleRate, channels int) error

	// Write outputs audio samples (blocks until written)
	Write(samples []int32) error

	// Close releases output resources
	Close() error
}
