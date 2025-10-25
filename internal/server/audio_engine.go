// ABOUTME: Audio streaming engine for Resonate server
// ABOUTME: Generates test tones and streams timestamped audio to clients
package server

import (
	"fmt"
	"log"
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

	// Buffering
	BufferAheadMs = 500 // Send audio 500ms ahead
)

// AudioEngine manages audio generation and streaming
type AudioEngine struct {
	server *Server

	// Active clients
	clients   map[string]*Client
	clientsMu sync.RWMutex

	// Audio source (file or test tone)
	source AudioSource

	stopChan chan struct{}
	stopOnce sync.Once // Ensure Stop() is only called once
}

// NewAudioEngine creates a new audio engine
func NewAudioEngine(server *Server) (*AudioEngine, error) {
	// Create audio source
	source, err := NewAudioSource(server.config.AudioFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio source: %w", err)
	}

	return &AudioEngine{
		server:   server,
		clients:  make(map[string]*Client),
		source:   source,
		stopChan: make(chan struct{}),
	}, nil
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
		if e.source != nil {
			if err := e.source.Close(); err != nil {
				log.Printf("Error closing audio source: %v", err)
			}
		}
	})
}

// AddClient adds a client to receive audio
func (e *AudioEngine) AddClient(client *Client) {
	e.clientsMu.Lock()
	defer e.clientsMu.Unlock()

	// Negotiate codec based on client capabilities
	codec := e.negotiateCodec(client)

	// Create encoder if needed
	var opusEncoder *OpusEncoder
	chunkSamples := (e.source.SampleRate() * ChunkDurationMs) / 1000

	switch codec {
	case "opus":
		encoder, err := NewOpusEncoder(e.source.SampleRate(), e.source.Channels(), chunkSamples)
		if err != nil {
			log.Printf("Failed to create Opus encoder for %s, falling back to PCM: %v", client.Name, err)
			codec = "pcm"
		} else {
			opusEncoder = encoder
		}
	case "flac":
		// FLAC is a file format, not a streaming codec
		// It requires headers at the start and can't be split into independent chunks
		// Fall back to PCM for lossless streaming
		log.Printf("FLAC streaming not supported for %s, using PCM for lossless audio", client.Name)
		codec = "pcm"
	}

	// Set codec and encoder atomically with client lock
	client.mu.Lock()
	client.Codec = codec
	client.OpusEncoder = opusEncoder
	client.mu.Unlock()

	e.clients[client.ID] = client

	log.Printf("Audio engine: added client %s with codec %s", client.Name, codec)

	// Send stream/start message (use select to avoid blocking)
	streamStart := protocol.StreamStart{
		Player: &protocol.StreamStartPlayer{
			Codec:      codec,
			SampleRate: e.source.SampleRate(),
			Channels:   e.source.Channels(),
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
	title, artist, album := e.source.Metadata()
	metadata := protocol.StreamMetadata{
		Title:  title,
		Artist: artist,
		Album:  album,
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

	// Clean up encoder if it exists (with lock held)
	client.mu.Lock()
	if client.OpusEncoder != nil {
		client.OpusEncoder.Close()
		client.OpusEncoder = nil
	}
	client.mu.Unlock()

	delete(e.clients, client.ID)
	log.Printf("Audio engine: removed client %s", client.Name)
}

// negotiateCodec selects the best codec based on client capabilities
func (e *AudioEngine) negotiateCodec(client *Client) string {
	// If client has no capabilities, default to PCM
	if client.Capabilities == nil {
		return "pcm"
	}

	// Check legacy support_codecs array (Music Assistant compatibility)
	for _, codec := range client.Capabilities.SupportCodecs {
		if codec == "opus" {
			return "opus"
		}
		if codec == "flac" {
			return "flac"
		}
	}

	// Check newer support_formats array (spec-compliant)
	for _, format := range client.Capabilities.SupportFormats {
		if format.Codec == "opus" {
			return "opus"
		}
		if format.Codec == "flac" {
			return "flac"
		}
	}

	// Default to PCM if no compressed codec supported
	return "pcm"
}

// generateAndSendChunk generates a chunk of audio and sends it to all clients
func (e *AudioEngine) generateAndSendChunk() {
	// Get current timestamp + buffer ahead time
	currentTime := e.server.getClockMicros()
	playbackTime := currentTime + (BufferAheadMs * 1000)

	// Calculate chunk size based on source sample rate
	chunkSamples := (e.source.SampleRate() * ChunkDurationMs) / 1000
	totalSamples := chunkSamples * e.source.Channels()

	// Read audio samples from source
	samples := make([]int16, totalSamples)
	n, err := e.source.Read(samples)
	if err != nil {
		log.Printf("Error reading audio source: %v", err)
		return
	}

	if e.server.config.Debug && n > 0 {
		log.Printf("[DEBUG] Generating chunk: server_time=%d, playback_time=%d, samples_read=%d",
			currentTime, playbackTime, n)
	}

	// Send to all clients (encode per-client based on codec)
	e.clientsMu.RLock()
	defer e.clientsMu.RUnlock()

	for _, client := range e.clients {
		var audioData []byte
		var encodeErr error

		// Read codec and encoder atomically
		client.mu.RLock()
		codec := client.Codec
		opusEncoder := client.OpusEncoder
		client.mu.RUnlock()

		// Encode based on client's negotiated codec
		switch codec {
		case "opus":
			if opusEncoder != nil {
				audioData, encodeErr = opusEncoder.Encode(samples[:n])
				if encodeErr != nil {
					log.Printf("Opus encode error for %s: %v", client.Name, encodeErr)
					continue
				}
			} else {
				log.Printf("Warning: Client %s has opus codec but no encoder", client.Name)
				continue
			}
		case "pcm":
			audioData = encodePCM(samples[:n])
		default:
			// Unknown codec, fall back to PCM
			log.Printf("Warning: Unknown codec %s for client %s, using PCM", codec, client.Name)
			audioData = encodePCM(samples[:n])
		}

		// Create binary message
		chunk := CreateAudioChunk(playbackTime, audioData)

		if err := e.server.sendBinary(client, chunk); err != nil {
			log.Printf("Error sending audio to %s: %v", client.Name, err)
		}
	}
}

// encodePCM encodes int16 samples as PCM bytes (little-endian)
func encodePCM(samples []int16) []byte {
	// Directly convert int16 slice to bytes (little-endian)
	output := make([]byte, len(samples)*2)
	for i, sample := range samples {
		output[i*2] = byte(sample)       // Low byte
		output[i*2+1] = byte(sample >> 8) // High byte
	}
	return output
}
