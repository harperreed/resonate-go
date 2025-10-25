//go:build !portaudio

// ABOUTME: PortAudio stub when library not available
// ABOUTME: Provides compile-time placeholder when PortAudio not installed
package output

import (
	"fmt"
)

// PortAudio output implementation (stub)
type PortAudio struct{}

// NewPortAudio creates a new PortAudio output
func NewPortAudio() Output {
	return &PortAudio{}
}

// Open initializes PortAudio
func (p *PortAudio) Open(sampleRate, channels int) error {
	return fmt.Errorf("PortAudio support not enabled (build with -tags portaudio)")
}

// Write outputs audio samples
func (p *PortAudio) Write(samples []int32) error {
	return fmt.Errorf("PortAudio support not enabled (build with -tags portaudio)")
}

// Close releases resources
func (p *PortAudio) Close() error {
	return fmt.Errorf("PortAudio support not enabled (build with -tags portaudio)")
}
