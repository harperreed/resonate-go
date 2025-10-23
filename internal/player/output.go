// ABOUTME: Audio output using oto library
// ABOUTME: Handles PCM playback with software volume control
package player

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/Resonate-Protocol/resonate-go/internal/audio"
	"github.com/ebitengine/oto/v3"
)

// Output manages audio output
type Output struct {
	ctx     context.Context
	cancel  context.CancelFunc
	otoCtx  *oto.Context
	player  *oto.Player
	format  audio.Format
	volume  int
	muted   bool
	ready   bool
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
	if o.otoCtx != nil {
		o.Close()
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

	// Convert bytes to int16 samples
	samples := make([]int16, len(buf.Samples)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(buf.Samples[i*2:]))
	}

	// Apply volume
	samples = applyVolume(samples, o.volume, o.muted)

	// Convert back to bytes
	output := make([]byte, len(buf.Samples))
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(output[i*2:], uint16(sample))
	}

	// Write to oto
	reader := bytes.NewReader(output)
	player := o.otoCtx.NewPlayer(reader)
	player.Play()

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
