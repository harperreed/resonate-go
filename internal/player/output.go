// ABOUTME: Audio output using oto library
// ABOUTME: Handles PCM playback with software volume control
package player

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/Resonate-Protocol/resonate-go/internal/audio"
	"github.com/ebitengine/oto/v3"
)

// Output manages audio output
type Output struct {
	ctx        context.Context
	cancel     context.CancelFunc
	otoCtx     *oto.Context
	player     *oto.Player
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	format     audio.Format
	volume     int
	muted      bool
	ready      bool
}

// NewOutput creates an audio output
func NewOutput() *Output {
	ctx, cancel := context.WithCancel(context.Background())

	return &Output{
		ctx:    ctx,
		cancel: cancel,
		volume: 100,
		muted:  false,
	}
}

// Initialize sets up oto with the specified format
func (o *Output) Initialize(format audio.Format) error {
	// If already initialized with same format, reuse the existing context
	if o.otoCtx != nil && o.format.SampleRate == format.SampleRate &&
		o.format.Channels == format.Channels && o.format.BitDepth == format.BitDepth {
		log.Printf("Audio output already initialized with same format, reusing context")
		return nil
	}

	// If format changed, we can't reinitialize oto (it only allows one context per process)
	// Log a warning but continue using the existing context
	if o.otoCtx != nil {
		log.Printf("Warning: format change detected (%dHz %dch -> %dHz %dch) but oto doesn't support reinitialization. Continuing with existing context.",
			o.format.SampleRate, o.format.Channels, format.SampleRate, format.Channels)
		return nil
	}

	op := &oto.NewContextOptions{
		SampleRate:   format.SampleRate,
		ChannelCount: format.Channels,
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return fmt.Errorf("failed to create oto context: %w", err)
	}

	<-readyChan

	o.otoCtx = ctx
	o.format = format

	// Create pipe for continuous streaming
	o.pipeReader, o.pipeWriter = io.Pipe()

	// Create persistent player that reads from the pipe
	o.player = o.otoCtx.NewPlayer(o.pipeReader)
	o.player.Play()

	o.ready = true

	log.Printf("Audio output initialized: %dHz, %d channels",
		format.SampleRate, format.Channels)

	return nil
}

// Play plays an audio buffer
func (o *Output) Play(buf audio.Buffer) error {
	if !o.ready {
		return fmt.Errorf("output not initialized")
	}

	// Apply volume to samples (already in int16 format)
	samples := applyVolume(buf.Samples, o.volume, o.muted)

	// Convert int16 samples to bytes for audio output
	output := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(output[i*2:], uint16(sample))
	}

	// Write to pipe (which feeds the persistent player)
	if _, err := o.pipeWriter.Write(output); err != nil {
		return fmt.Errorf("pipe write failed: %w", err)
	}

	return nil
}

// SetVolume sets the volume (0-100)
func (o *Output) SetVolume(volume int) {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	o.volume = volume
	log.Printf("Volume set to %d", volume)
}

// SetMuted sets mute state
func (o *Output) SetMuted(muted bool) {
	o.muted = muted
	log.Printf("Muted: %v", muted)
}

// GetVolume returns current volume
func (o *Output) GetVolume() int {
	return o.volume
}

// IsMuted returns mute state
func (o *Output) IsMuted() bool {
	return o.muted
}

// Close closes the audio output
func (o *Output) Close() {
	if o.pipeWriter != nil {
		o.pipeWriter.Close()
		o.pipeWriter = nil
	}
	if o.player != nil {
		o.player.Close()
		o.player = nil
	}
	if o.pipeReader != nil {
		o.pipeReader.Close()
		o.pipeReader = nil
	}
	if o.otoCtx != nil {
		o.otoCtx.Suspend()
		o.ready = false
	}
	o.cancel()
}

// applyVolume applies volume and mute to samples
func applyVolume(samples []int16, volume int, muted bool) []int16 {
	multiplier := getVolumeMultiplier(volume, muted)

	result := make([]int16, len(samples))
	for i, sample := range samples {
		result[i] = int16(float64(sample) * multiplier)
	}

	return result
}

// getVolumeMultiplier calculates volume multiplier
func getVolumeMultiplier(volume int, muted bool) float64 {
	if muted {
		return 0.0
	}
	return float64(volume) / 100.0
}
