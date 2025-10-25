// ABOUTME: Decoder interface definition
// ABOUTME: Common interface for all audio decoders
package decode

// Decoder decodes audio in various formats to PCM int32 samples
type Decoder interface {
	// Decode converts encoded audio data to PCM samples
	Decode(data []byte) ([]int32, error)

	// Close releases decoder resources
	Close() error
}
