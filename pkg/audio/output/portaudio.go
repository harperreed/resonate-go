//go:build portaudio

// ABOUTME: PortAudio output implementation
// ABOUTME: Cross-platform audio output using PortAudio
package output

import (
	"fmt"

	"github.com/Resonate-Protocol/resonate-go/pkg/audio"
	"github.com/gordonklaus/portaudio"
)

// PortAudio output implementation
type PortAudio struct {
	stream *portaudio.Stream
	buffer []int16
}

// NewPortAudio creates a new PortAudio output
func NewPortAudio() Output {
	return &PortAudio{}
}

// Open initializes PortAudio
func (p *PortAudio) Open(sampleRate, channels int) error {
	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize portaudio: %w", err)
	}

	stream, err := portaudio.OpenDefaultStream(0, channels, float64(sampleRate), 0, func(out []int16) {
		copy(out, p.buffer)
	})
	if err != nil {
		portaudio.Terminate()
		return fmt.Errorf("failed to open stream: %w", err)
	}

	p.stream = stream
	return stream.Start()
}

// Write outputs audio samples
func (p *PortAudio) Write(samples []int32) error {
	if p.stream == nil {
		return fmt.Errorf("output not opened")
	}

	// Convert int32 to int16 for PortAudio
	p.buffer = make([]int16, len(samples))
	for i, sample := range samples {
		p.buffer[i] = audio.SampleToInt16(sample)
	}

	return nil
}

// Close releases resources
func (p *PortAudio) Close() error {
	if p.stream != nil {
		if err := p.stream.Stop(); err != nil {
			return err
		}
		if err := p.stream.Close(); err != nil {
			return err
		}
	}
	return portaudio.Terminate()
}
