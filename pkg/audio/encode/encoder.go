// ABOUTME: Encoder interface definition
// ABOUTME: Common interface for all audio encoders
package encode

// Encoder encodes PCM int32 samples to various formats
type Encoder interface {
	// Encode converts PCM samples to encoded audio data
	Encode(samples []int32) ([]byte, error)

	// Close releases encoder resources
	Close() error
}
