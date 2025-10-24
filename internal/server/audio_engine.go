// ABOUTME: Audio streaming engine for Resonate server
// ABOUTME: Generates test tones and streams timestamped audio to clients
package server

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"sync"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/protocol"
)

const (
	// Audio format constants
	DefaultSampleRate = 48000
	DefaultChannels   = 2
	DefaultBitDepth   = 16

	// Chunk timing
	ChunkDurationMs = 20 // 20ms chunks
	ChunkSamples    = (DefaultSampleRate * ChunkDurationMs) / 1000
	ChunkSize       = ChunkSamples * DefaultChannels * (DefaultBitDepth / 8)

	// Buffering
	BufferAheadMs = 500 // Send audio 500ms ahead
)

// AudioEngine manages audio generation and streaming
type AudioEngine struct {
	server *Server

	// Active clients
	clients   map[string]*Client
	clientsMu sync.RWMutex

	// Audio generation
	sampleIndex uint64
	frequency   float64 // Test tone frequency

	stopChan chan struct{}
	stopOnce sync.Once // Ensure Stop() is only called once
}

// NewAudioEngine creates a new audio engine
func NewAudioEngine(server *Server) *AudioEngine {
	return &AudioEngine{
		server:    server,
		clients:   make(map[string]*Client),
		frequency: 440.0, // A4 note
		stopChan:  make(chan struct{}),
	}
}

// Start starts the audio engine
func (e *AudioEngine) Start() {
	log.Printf("Audio engine starting")

	ticker := time.NewTicker(time.Duration(ChunkDurationMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.generateAndSendChunk()
		case <-e.stopChan:
			log.Printf("Audio engine stopping")
			return
		}
	}
}

// Stop stops the audio engine
func (e *AudioEngine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopChan)
	})
}

// AddClient adds a client to receive audio
func (e *AudioEngine) AddClient(client *Client) {
	e.clientsMu.Lock()
	e.clients[client.ID] = client
	e.clientsMu.Unlock()

	log.Printf("Audio engine: added client %s", client.Name)

	// Send stream/start message (use select to avoid blocking)
	streamStart := protocol.StreamStart{
		Player: &protocol.StreamStartPlayer{
			Codec:      "pcm",
			SampleRate: DefaultSampleRate,
			Channels:   DefaultChannels,
			BitDepth:   DefaultBitDepth,
		},
	}

	msg := protocol.Message{
		Type:    "stream/start",
		Payload: streamStart,
	}

	select {
	case client.sendChan <- msg:
	default:
		log.Printf("Warning: Could not send stream/start to %s (channel full)", client.Name)
	}

	// Send metadata (use select to avoid blocking)
	metadata := protocol.StreamMetadata{
		Title:  "Test Tone",
		Artist: "Resonate Server",
		Album:  "Reference Implementation",
	}

	metaMsg := protocol.Message{
		Type:    "stream/metadata",
		Payload: metadata,
	}

	select {
	case client.sendChan <- metaMsg:
	default:
		log.Printf("Warning: Could not send metadata to %s (channel full)", client.Name)
	}
}

// RemoveClient removes a client from audio streaming
func (e *AudioEngine) RemoveClient(client *Client) {
	e.clientsMu.Lock()
	defer e.clientsMu.Unlock()

	delete(e.clients, client.ID)
	log.Printf("Audio engine: removed client %s", client.Name)
}

// generateAndSendChunk generates a chunk of audio and sends it to all clients
func (e *AudioEngine) generateAndSendChunk() {
	// Get current timestamp + buffer ahead time
	currentTime := e.server.getClockMicros()
	playbackTime := currentTime + (BufferAheadMs * 1000)

	if e.server.config.Debug && e.sampleIndex%100 == 0 {
		log.Printf("[DEBUG] Generating chunk: server_time=%d, playback_time=%d, sample_index=%d",
			currentTime, playbackTime, e.sampleIndex)
	}

	// Generate audio samples (sine wave)
	samples := make([]int16, ChunkSamples*DefaultChannels)

	for i := 0; i < ChunkSamples; i++ {
		// Generate sine wave
		t := float64(e.sampleIndex+uint64(i)) / float64(DefaultSampleRate)
		sample := math.Sin(2 * math.Pi * e.frequency * t)

		// Convert to 16-bit PCM
		pcmValue := int16(sample * 32767.0 * 0.5) // 50% volume

		// Stereo (duplicate to both channels)
		samples[i*DefaultChannels] = pcmValue
		samples[i*DefaultChannels+1] = pcmValue
	}

	e.sampleIndex += ChunkSamples

	// Encode as WAV chunk (just the raw PCM data for now)
	audioData := encodePCM(samples)

	// Create binary message
	chunk := CreateAudioChunk(playbackTime, audioData)

	// Send to all clients
	e.clientsMu.RLock()
	defer e.clientsMu.RUnlock()

	for _, client := range e.clients {
		if err := e.server.sendBinary(client, chunk); err != nil {
			log.Printf("Error sending audio to %s: %v", client.Name, err)
		}
	}
}

// encodePCM encodes int16 samples as PCM bytes (little-endian)
func encodePCM(samples []int16) []byte {
	buf := new(bytes.Buffer)

	for _, sample := range samples {
		if err := binary.Write(buf, binary.LittleEndian, sample); err != nil {
			log.Printf("Error encoding PCM sample: %v", err)
			// Continue encoding remaining samples
		}
	}

	return buf.Bytes()
}
