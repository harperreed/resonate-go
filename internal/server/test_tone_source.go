// ABOUTME: Test tone generator for audio source
// ABOUTME: Generates 440Hz sine wave for testing
package server

import (
	"math"
	"sync"
)

// TestToneSource generates a 440Hz test tone
type TestToneSource struct {
	sampleIndex uint64
	sampleMu    sync.Mutex
	frequency   float64
}

// NewTestToneSource creates a new test tone generator
func NewTestToneSource() *TestToneSource {
	return &TestToneSource{
		frequency: 440.0, // A4 note
	}
}

func (s *TestToneSource) Read(samples []int16) (int, error) {
	s.sampleMu.Lock()
	defer s.sampleMu.Unlock()

	numSamples := len(samples) / 2 // Stereo

	for i := 0; i < numSamples; i++ {
		// Generate sine wave
		t := float64(s.sampleIndex+uint64(i)) / float64(DefaultSampleRate)
		sample := math.Sin(2 * math.Pi * s.frequency * t)

		// Convert to 16-bit PCM
		pcmValue := int16(sample * 32767.0 * 0.5) // 50% volume

		// Stereo (duplicate to both channels)
		samples[i*2] = pcmValue
		samples[i*2+1] = pcmValue
	}

	s.sampleIndex += uint64(numSamples)

	return len(samples), nil
}

func (s *TestToneSource) SampleRate() int { return DefaultSampleRate }
func (s *TestToneSource) Channels() int   { return DefaultChannels }
func (s *TestToneSource) Metadata() (string, string, string) {
	return "Test Tone", "Resonate Server", "Reference Implementation"
}
func (s *TestToneSource) Close() error { return nil }
