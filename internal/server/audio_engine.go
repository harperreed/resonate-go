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
	// Audio format constants - Hi-Res Audio (192kHz/24-bit)
	DefaultSampleRate = 192000
	DefaultChannels   = 2
	DefaultBitDepth   = 24

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

	log.Printf("Audio engine: added client %s with codec %s (format: %dHz/%dbit/%dch)",
		client.Name, codec, e.source.SampleRate(), DefaultBitDepth, e.source.Channels())

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
// Prioritizes PCM at native sample rate for hi-res audio
func (e *AudioEngine) negotiateCodec(client *Client) string {
	// If client has no capabilities, default to PCM
	if client.Capabilities == nil {
		return "pcm"
	}

	sourceRate := e.source.SampleRate()

	// Check newer support_formats array first (spec-compliant)
	// Prioritize PCM at native rate to preserve hi-res audio quality
	for _, format := range client.Capabilities.SupportFormats {
		// Check if client supports PCM at our native sample rate
		if format.Codec == "pcm" && format.SampleRate == sourceRate && format.BitDepth == DefaultBitDepth {
			return "pcm"
		}
	}

	// If no PCM match at native rate, consider compressed codecs
	for _, format := range client.Capabilities.SupportFormats {
		// Only use Opus if source is 48kHz (Opus native rate)
		// For other rates, prefer PCM to avoid resampling
		if format.Codec == "opus" && sourceRate == 48000 {
			return "opus"
		}
		if format.Codec == "flac" {
			return "flac"
		}
	}

	// Check legacy support_codecs array (Music Assistant compatibility)
	for _, codec := range client.Capabilities.SupportCodecs {
		if codec == "opus" && sourceRate == 48000 {
			return "opus"
		}
		if codec == "flac" {
			return "flac"
		}
	}

	// Default to PCM even if sample rate doesn't match perfectly
	// Client will handle resampling on their end if needed
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

	// Read audio samples from source (int32 for 24-bit support)
	samples := make([]int32, totalSamples)
	n, err := e.source.Read(samples)
	if err != nil {
		log.Printf("Error reading audio source: %v", err)
		return
	}

	// Note: Chunk generation happens every 20ms (50/sec), logging disabled to avoid spam
	// Debug info: server generates chunks of size=7680 samples (20ms @ 192kHz stereo)

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
				// Convert int32 to int16 for Opus (Opus only supports 16-bit)
				samples16 := convertToInt16(samples[:n])
				audioData, encodeErr = opusEncoder.Encode(samples16)
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

// convertToInt16 converts int32 samples to int16 (for Opus encoding)
func convertToInt16(samples []int32) []int16 {
	result := make([]int16, len(samples))
	for i, s := range samples {
		// Right-shift 8 bits to convert from 24-bit to 16-bit range
		result[i] = int16(s >> 8)
	}
	return result
}

// encodePCM encodes int32 samples as 24-bit PCM bytes (little-endian, 3 bytes per sample)
func encodePCM(samples []int32) []byte {
	// 24-bit PCM: 3 bytes per sample
	output := make([]byte, len(samples)*3)
	for i, sample := range samples {
		// Pack 24-bit value (little-endian)
		output[i*3] = byte(sample)
		output[i*3+1] = byte(sample >> 8)
		output[i*3+2] = byte(sample >> 16)
	}
	return output
}
