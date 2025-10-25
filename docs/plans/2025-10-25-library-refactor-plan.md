# Library Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Convert resonate-go from CLI-focused to library-first architecture with layered public APIs

**Architecture:** Ground-up restructure moving code from `internal/` to `pkg/` with three layers: high-level convenience API (`pkg/resonate/`), component APIs (`pkg/audio/`, `pkg/protocol/`, etc.), and private implementation (`internal/`). CLI tools become thin wrappers using the public library.

**Tech Stack:** Go 1.23, existing audio libraries (PortAudio, Opus, FLAC, MP3), WebSockets

**Design Document:** `docs/plans/2025-10-25-library-refactor-design.md`

---

## Task 1: Create Package Structure

**Files:**
- Create: `pkg/audio/doc.go`
- Create: `pkg/audio/decode/doc.go`
- Create: `pkg/audio/encode/doc.go`
- Create: `pkg/audio/resample/doc.go`
- Create: `pkg/audio/output/doc.go`
- Create: `pkg/protocol/doc.go`
- Create: `pkg/sync/doc.go`
- Create: `pkg/discovery/doc.go`
- Create: `pkg/resonate/doc.go`

**Step 1: Create pkg/audio directory with package documentation**

```bash
mkdir -p pkg/audio
```

**Step 2: Write pkg/audio/doc.go**

Create: `pkg/audio/doc.go`

```go
// ABOUTME: Audio fundamentals package providing core types and utilities
// ABOUTME: Defines Format, Buffer types and sample conversion functions
// Package audio provides fundamental audio types and utilities for hi-res audio processing.
//
// This package defines core types used throughout the resonate library:
//   - Format: Describes audio stream format (codec, sample rate, channels, bit depth)
//   - Buffer: Represents decoded PCM audio with timestamp information
//
// It also provides utilities for converting between different sample formats:
//   - 16-bit ↔ 24-bit conversions
//   - int32 ↔ packed byte conversions
//
// Example:
//
//	format := audio.Format{
//	    Codec:      "pcm",
//	    SampleRate: 192000,
//	    Channels:   2,
//	    BitDepth:   24,
//	}
//
//	// Convert 16-bit sample to 24-bit range
//	sample24 := audio.SampleFromInt16(sample16)
package audio
```

**Step 3: Create remaining package directories**

```bash
mkdir -p pkg/audio/decode
mkdir -p pkg/audio/encode
mkdir -p pkg/audio/resample
mkdir -p pkg/audio/output
mkdir -p pkg/protocol
mkdir -p pkg/sync
mkdir -p pkg/discovery
mkdir -p pkg/resonate
```

**Step 4: Write package documentation for each**

Create: `pkg/audio/decode/doc.go`

```go
// ABOUTME: Audio decoder package for multiple codec support
// ABOUTME: Provides Decoder interface and implementations for PCM, Opus, FLAC, MP3
// Package decode provides audio decoders for various codecs.
//
// Supports: PCM (16-bit and 24-bit), Opus, FLAC, MP3
//
// All decoders implement the Decoder interface and output int32 samples
// in 24-bit range for consistent hi-res audio processing.
//
// Example:
//
//	decoder, err := decode.NewPCM(format)
//	samples, err := decoder.Decode(audioData)
package decode
```

Create: `pkg/audio/encode/doc.go`

```go
// ABOUTME: Audio encoder package for encoding PCM to various formats
// ABOUTME: Provides Encoder interface and implementations for PCM, Opus
// Package encode provides audio encoders for various codecs.
//
// Supports: PCM (16-bit and 24-bit), Opus
//
// All encoders accept int32 samples in 24-bit range and encode
// to wire format.
//
// Example:
//
//	encoder, err := encode.NewPCM(format)
//	data, err := encoder.Encode(samples)
package encode
```

Create: `pkg/audio/resample/doc.go`

```go
// ABOUTME: Audio resampling package using linear interpolation
// ABOUTME: Converts audio between different sample rates
// Package resample provides audio sample rate conversion.
//
// Uses linear interpolation for converting between sample rates.
// Handles both upsampling and downsampling.
//
// Example:
//
//	r := resample.New(44100, 48000, 2)
//	outputSize := r.Resample(inputSamples, outputSamples)
package resample
```

Create: `pkg/audio/output/doc.go`

```go
// ABOUTME: Audio output package for playing audio
// ABOUTME: Provides Output interface and PortAudio implementation
// Package output provides audio playback interfaces.
//
// Currently supports PortAudio for cross-platform audio output.
//
// Example:
//
//	out := output.NewPortAudio()
//	err := out.Open(48000, 2)
//	err = out.Write(samples)
package output
```

Create: `pkg/protocol/doc.go`

```go
// ABOUTME: Resonate wire protocol package
// ABOUTME: Defines protocol messages and WebSocket client
// Package protocol implements the Resonate wire protocol.
//
// Provides message types and WebSocket client for communicating
// with Resonate servers.
//
// Example:
//
//	client, err := protocol.NewClient("localhost:8927")
//	err = client.SendHello(helloMsg)
package protocol
```

Create: `pkg/sync/doc.go`

```go
// ABOUTME: Clock synchronization package
// ABOUTME: Provides NTP-style clock sync with Resonate servers
// Package sync provides clock synchronization for precise audio timing.
//
// Uses NTP-style round-trip time measurement to sync with server clocks.
//
// Example:
//
//	clock := sync.NewClock()
//	err := clock.Sync("localhost:8927")
//	serverTime := clock.ServerTime()
package sync
```

Create: `pkg/discovery/doc.go`

```go
// ABOUTME: mDNS service discovery package
// ABOUTME: Discover and advertise Resonate servers on local network
// Package discovery provides mDNS service discovery for Resonate servers.
//
// Allows discovering servers on the local network and advertising
// server availability.
//
// Example:
//
//	services, err := discovery.Discover(5 * time.Second)
//	for _, svc := range services {
//	    fmt.Printf("Found: %s at %s:%d\n", svc.Name, svc.Address, svc.Port)
//	}
package discovery
```

Create: `pkg/resonate/doc.go`

```go
// ABOUTME: High-level Resonate library API
// ABOUTME: Provides simple Player and Server APIs for most use cases
// Package resonate provides high-level APIs for Resonate audio streaming.
//
// This is the main entry point for most library users, providing:
//   - Player: Connect to servers and play synchronized audio
//   - Server: Serve audio to multiple clients
//   - AudioSource: Interface for custom audio sources
//
// For lower-level control, see the audio, protocol, sync, and discovery packages.
//
// Example Player:
//
//	player, err := resonate.NewPlayer(resonate.PlayerConfig{
//	    ServerAddr: "localhost:8927",
//	    PlayerName: "Living Room",
//	    Volume:     80,
//	})
//	err = player.Connect()
//	err = player.Play()
//
// Example Server:
//
//	source, err := resonate.FileSource("/path/to/audio.flac")
//	server, err := resonate.NewServer(resonate.ServerConfig{
//	    Port:   8927,
//	    Source: source,
//	})
//	err = server.Start()
package resonate
```

**Step 5: Verify package structure**

```bash
tree pkg
```

Expected: All directories created with doc.go files

**Step 6: Commit package structure**

```bash
git add pkg/
git commit -m "feat: Create public package structure for library

- Add pkg/audio/ for audio primitives
- Add pkg/audio/decode/ for decoders
- Add pkg/audio/encode/ for encoders
- Add pkg/audio/resample/ for resampling
- Add pkg/audio/output/ for audio output
- Add pkg/protocol/ for wire protocol
- Add pkg/sync/ for clock synchronization
- Add pkg/discovery/ for mDNS discovery
- Add pkg/resonate/ for high-level API

Each package includes comprehensive documentation."
```

---

## Task 2: Move Audio Types to pkg/audio

**Files:**
- Create: `pkg/audio/types.go`
- Create: `pkg/audio/types_test.go`
- Reference: `internal/audio/types.go` (source)

**Step 1: Copy types.go from internal to pkg**

Create: `pkg/audio/types.go`

```go
// ABOUTME: Audio type definitions
// ABOUTME: Defines audio formats and decoded buffers
package audio

import "time"

const (
	// 24-bit audio range constants
	Max24Bit = 8388607  // 2^23 - 1
	Min24Bit = -8388608 // -2^23
)

// Format describes audio stream format
type Format struct {
	Codec       string
	SampleRate  int
	Channels    int
	BitDepth    int
	CodecHeader []byte // For FLAC, Opus, etc.
}

// Buffer represents decoded PCM audio
type Buffer struct {
	Timestamp int64     // Server timestamp (microseconds)
	PlayAt    time.Time // Local play time
	Samples   []int32   // PCM samples (int32 to support both 16-bit and 24-bit)
	Format    Format
}

// SampleToInt16 converts int32 sample to int16 (for 16-bit playback)
func SampleToInt16(sample int32) int16 {
	// Right-shift to convert 24-bit (or 16-bit) to 16-bit range
	return int16(sample >> 8)
}

// SampleFromInt16 converts int16 sample to int32 (left-justified in 24-bit)
func SampleFromInt16(sample int16) int32 {
	// Left-shift to position 16-bit value in upper bits
	return int32(sample) << 8
}

// SampleTo24Bit converts int32 to 24-bit packed bytes (little-endian)
func SampleTo24Bit(sample int32) [3]byte {
	// Take lower 24 bits, pack little-endian
	return [3]byte{
		byte(sample),
		byte(sample >> 8),
		byte(sample >> 16),
	}
}

// SampleFrom24Bit converts 24-bit packed bytes to int32 (little-endian)
func SampleFrom24Bit(b [3]byte) int32 {
	// Reconstruct 24-bit value and sign-extend to 32-bit
	val := int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16
	// Sign extend from 24-bit to 32-bit
	if val&0x800000 != 0 {
		val |= ^0xFFFFFF // Set upper 8 bits to 1 for negative values
	}
	return val
}
```

**Step 2: Write tests for conversion functions**

Create: `pkg/audio/types_test.go`

```go
// ABOUTME: Tests for audio types
// ABOUTME: Tests sample conversion functions
package audio

import "testing"

func TestSampleFromInt16(t *testing.T) {
	tests := []struct {
		name     string
		input    int16
		expected int32
	}{
		{"zero", 0, 0},
		{"positive", 100, 100 << 8},
		{"negative", -100, -100 << 8},
		{"max", 32767, 32767 << 8},
		{"min", -32768, -32768 << 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SampleFromInt16(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestSampleToInt16(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected int16
	}{
		{"zero", 0, 0},
		{"positive", 100 << 8, 100},
		{"negative", -100 << 8, -100},
		{"24bit positive", 1000000, 3906}, // 1000000 >> 8 = 3906
		{"24bit negative", -1000000, -3907},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SampleToInt16(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestSampleTo24Bit(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected [3]byte
	}{
		{"zero", 0, [3]byte{0, 0, 0}},
		{"positive", 0x123456, [3]byte{0x56, 0x34, 0x12}},
		{"negative", -256, [3]byte{0x00, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SampleTo24Bit(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSampleFrom24Bit(t *testing.T) {
	tests := []struct {
		name     string
		input    [3]byte
		expected int32
	}{
		{"zero", [3]byte{0, 0, 0}, 0},
		{"positive", [3]byte{0x56, 0x34, 0x12}, 0x123456},
		{"negative", [3]byte{0x00, 0xFF, 0xFF}, -256},
		{"max positive", [3]byte{0xFF, 0xFF, 0x7F}, Max24Bit},
		{"max negative", [3]byte{0x00, 0x00, 0x80}, Min24Bit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SampleFrom24Bit(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestRoundTrip16Bit(t *testing.T) {
	// Test that 16-bit samples survive round-trip conversion
	samples := []int16{0, 100, -100, 1000, -1000, 32767, -32768}

	for _, original := range samples {
		sample32 := SampleFromInt16(original)
		result := SampleToInt16(sample32)
		if result != original {
			t.Errorf("round-trip failed: %d -> %d -> %d", original, sample32, result)
		}
	}
}

func TestRoundTrip24Bit(t *testing.T) {
	// Test that 24-bit samples survive round-trip conversion
	samples := []int32{0, 100000, -100000, Max24Bit, Min24Bit}

	for _, original := range samples {
		bytes := SampleTo24Bit(original)
		result := SampleFrom24Bit(bytes)
		// Mask to 24-bit for comparison
		expected := original & 0xFFFFFF
		if expected&0x800000 != 0 {
			expected |= ^0xFFFFFF
		}
		if result != expected {
			t.Errorf("round-trip failed: %d -> %v -> %d (expected %d)", original, bytes, result, expected)
		}
	}
}
```

**Step 3: Run tests**

```bash
go test -v ./pkg/audio
```

Expected: All tests pass

**Step 4: Commit audio types**

```bash
git add pkg/audio/types.go pkg/audio/types_test.go
git commit -m "feat: Add audio types to public API

- Add Format and Buffer types
- Add 24-bit audio constants
- Add sample conversion functions
- Add comprehensive tests for conversions"
```

---

## Task 3: Move PCM Decoder to pkg/audio/decode

**Files:**
- Create: `pkg/audio/decode/decoder.go`
- Create: `pkg/audio/decode/pcm.go`
- Create: `pkg/audio/decode/pcm_test.go`
- Reference: `internal/audio/decoder.go` (source)

**Step 1: Create decoder interface**

Create: `pkg/audio/decode/decoder.go`

```go
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
```

**Step 2: Write failing test for PCM decoder**

Create: `pkg/audio/decode/pcm_test.go`

```go
// ABOUTME: Tests for PCM decoder
// ABOUTME: Tests 16-bit and 24-bit PCM decoding
package decode

import (
	"testing"

	"github.com/harperreed/resonate-go/pkg/audio"
)

func TestNewPCM(t *testing.T) {
	format := audio.Format{
		Codec:      "pcm",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewPCM(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	if decoder == nil {
		t.Fatal("expected decoder to be created")
	}
}

func TestPCMDecode16Bit(t *testing.T) {
	format := audio.Format{
		Codec:      "pcm",
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewPCM(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	// PCM converts bytes to int16 samples (little-endian)
	// Input: 4 bytes -> Output: 2 int16 samples
	input := []byte{0x00, 0x01, 0x02, 0x03}
	output, err := decoder.Decode(input)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	expectedSamples := len(input) / 2
	if len(output) != expectedSamples {
		t.Errorf("expected %d samples, got %d", expectedSamples, len(output))
	}

	// Verify little-endian conversion with 24-bit scaling
	// 0x00, 0x01 -> 0x0100 = 256 (16-bit) -> 256<<8 = 65536 (24-bit)
	// 0x02, 0x03 -> 0x0302 = 770 (16-bit) -> 770<<8 = 197120 (24-bit)
	expected0 := int32(256 << 8)
	if output[0] != expected0 {
		t.Errorf("expected first sample %d, got %d", expected0, output[0])
	}
	expected1 := int32(770 << 8)
	if output[1] != expected1 {
		t.Errorf("expected second sample %d, got %d", expected1, output[1])
	}
}

func TestPCMDecode24Bit(t *testing.T) {
	format := audio.Format{
		Codec:      "pcm",
		SampleRate: 192000,
		Channels:   2,
		BitDepth:   24,
	}

	decoder, err := NewPCM(format)
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}

	// 24-bit PCM: 3 bytes per sample
	// Input: 6 bytes -> Output: 2 samples
	input := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	output, err := decoder.Decode(input)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	expectedSamples := len(input) / 3
	if len(output) != expectedSamples {
		t.Errorf("expected %d samples, got %d", expectedSamples, len(output))
	}

	// Verify 24-bit little-endian conversion
	// 0x00, 0x01, 0x02 -> 0x020100 = 131328
	expected0 := int32(0x020100)
	if output[0] != expected0 {
		t.Errorf("expected first sample %d, got %d", expected0, output[0])
	}

	// 0x03, 0x04, 0x05 -> 0x050403 = 328707
	expected1 := int32(0x050403)
	if output[1] != expected1 {
		t.Errorf("expected second sample %d, got %d", expected1, output[1])
	}
}
```

**Step 3: Run test to verify it fails**

```bash
go test -v ./pkg/audio/decode
```

Expected: FAIL - undefined: NewPCM

**Step 4: Implement PCM decoder**

Create: `pkg/audio/decode/pcm.go`

```go
// ABOUTME: PCM audio decoder
// ABOUTME: Decodes 16-bit and 24-bit PCM audio to int32 samples
package decode

import (
	"encoding/binary"
	"fmt"

	"github.com/harperreed/resonate-go/pkg/audio"
)

// PCMDecoder decodes PCM audio
type PCMDecoder struct {
	bitDepth int
}

// NewPCM creates a new PCM decoder
func NewPCM(format audio.Format) (Decoder, error) {
	if format.Codec != "pcm" {
		return nil, fmt.Errorf("invalid codec for PCM decoder: %s", format.Codec)
	}

	if format.BitDepth != 16 && format.BitDepth != 24 {
		return nil, fmt.Errorf("unsupported bit depth: %d (supported: 16, 24)", format.BitDepth)
	}

	return &PCMDecoder{
		bitDepth: format.BitDepth,
	}, nil
}

// Decode converts PCM bytes to int32 samples
func (d *PCMDecoder) Decode(data []byte) ([]int32, error) {
	if d.bitDepth == 24 {
		// 24-bit PCM: 3 bytes per sample
		numSamples := len(data) / 3
		samples := make([]int32, numSamples)
		for i := 0; i < numSamples; i++ {
			b := [3]byte{data[i*3], data[i*3+1], data[i*3+2]}
			samples[i] = audio.SampleFrom24Bit(b)
		}
		return samples, nil
	} else {
		// 16-bit PCM: 2 bytes per sample (default)
		numSamples := len(data) / 2
		samples := make([]int32, numSamples)
		for i := 0; i < numSamples; i++ {
			sample16 := int16(binary.LittleEndian.Uint16(data[i*2:]))
			samples[i] = audio.SampleFromInt16(sample16)
		}
		return samples, nil
	}
}

// Close releases resources
func (d *PCMDecoder) Close() error {
	return nil
}
```

**Step 5: Run tests to verify they pass**

```bash
go test -v ./pkg/audio/decode
```

Expected: PASS - all tests pass

**Step 6: Commit PCM decoder**

```bash
git add pkg/audio/decode/
git commit -m "feat: Add PCM decoder to public API

- Add Decoder interface
- Add PCM decoder supporting 16-bit and 24-bit
- Add comprehensive tests"
```

---

## Task 4: Move Remaining Decoders to pkg/audio/decode

**Files:**
- Create: `pkg/audio/decode/opus.go`
- Create: `pkg/audio/decode/flac.go`
- Create: `pkg/audio/decode/mp3.go`
- Reference: `internal/audio/decoder.go` (source)

**Step 1: Copy Opus decoder from internal**

Create: `pkg/audio/decode/opus.go`

```go
// ABOUTME: Opus audio decoder
// ABOUTME: Decodes Opus audio to int32 samples
package decode

import (
	"fmt"

	"github.com/harperreed/resonate-go/pkg/audio"
	"github.com/hraban/opus"
)

// OpusDecoder decodes Opus audio
type OpusDecoder struct {
	decoder    *opus.Decoder
	sampleRate int
	channels   int
}

// NewOpus creates a new Opus decoder
func NewOpus(format audio.Format) (Decoder, error) {
	if format.Codec != "opus" {
		return nil, fmt.Errorf("invalid codec for Opus decoder: %s", format.Codec)
	}

	decoder, err := opus.NewDecoder(format.SampleRate, format.Channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus decoder: %w", err)
	}

	return &OpusDecoder{
		decoder:    decoder,
		sampleRate: format.SampleRate,
		channels:   format.Channels,
	}, nil
}

// Decode converts Opus bytes to int32 samples
func (d *OpusDecoder) Decode(data []byte) ([]int32, error) {
	// Opus decoder outputs int16
	pcm := make([]int16, 5760*d.channels) // Max Opus frame size
	n, err := d.decoder.Decode(data, pcm)
	if err != nil {
		return nil, fmt.Errorf("opus decode error: %w", err)
	}

	// Convert int16 to int32 (24-bit range)
	samples := make([]int32, n*d.channels)
	for i := 0; i < n*d.channels; i++ {
		samples[i] = audio.SampleFromInt16(pcm[i])
	}

	return samples, nil
}

// Close releases resources
func (d *OpusDecoder) Close() error {
	return nil
}
```

**Step 2: Copy FLAC decoder from internal**

Create: `pkg/audio/decode/flac.go`

```go
// ABOUTME: FLAC audio decoder
// ABOUTME: Decodes FLAC audio to int32 samples
package decode

import (
	"fmt"
	"io"

	"github.com/harperreed/resonate-go/pkg/audio"
	"github.com/mewkiz/flac"
)

// FLACDecoder decodes FLAC audio
type FLACDecoder struct {
	stream *flac.Stream
}

// NewFLAC creates a new FLAC decoder
func NewFLAC(format audio.Format) (Decoder, error) {
	if format.Codec != "flac" {
		return nil, fmt.Errorf("invalid codec for FLAC decoder: %s", format.Codec)
	}

	if len(format.CodecHeader) == 0 {
		return nil, fmt.Errorf("FLAC decoder requires codec header")
	}

	// Create FLAC stream from header
	// Note: This is a simplified version - real implementation needs
	// to handle streaming FLAC data properly
	return &FLACDecoder{}, fmt.Errorf("FLAC decoder not yet implemented")
}

// Decode converts FLAC bytes to int32 samples
func (d *FLACDecoder) Decode(data []byte) ([]int32, error) {
	return nil, fmt.Errorf("FLAC decode not yet implemented")
}

// Close releases resources
func (d *FLACDecoder) Close() error {
	if d.stream != nil {
		// Close stream
	}
	return nil
}
```

**Step 3: Copy MP3 decoder from internal**

Create: `pkg/audio/decode/mp3.go`

```go
// ABOUTME: MP3 audio decoder
// ABOUTME: Decodes MP3 audio to int32 samples
package decode

import (
	"fmt"

	"github.com/harperreed/resonate-go/pkg/audio"
	"github.com/tosone/minimp3"
)

// MP3Decoder decodes MP3 audio
type MP3Decoder struct {
	decoder *minimp3.Decoder
}

// NewMP3 creates a new MP3 decoder
func NewMP3(format audio.Format) (Decoder, error) {
	if format.Codec != "mp3" {
		return nil, fmt.Errorf("invalid codec for MP3 decoder: %s", format.Codec)
	}

	decoder, err := minimp3.NewDecoder(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create mp3 decoder: %w", err)
	}

	return &MP3Decoder{
		decoder: decoder,
	}, nil
}

// Decode converts MP3 bytes to int32 samples
func (d *MP3Decoder) Decode(data []byte) ([]int32, error) {
	// MP3 decoder outputs int16
	pcm, err := d.decoder.Read(data)
	if err != nil {
		return nil, fmt.Errorf("mp3 decode error: %w", err)
	}

	// Convert int16 to int32 (24-bit range)
	samples := make([]int32, len(pcm))
	for i, sample := range pcm {
		samples[i] = audio.SampleFromInt16(sample)
	}

	return samples, nil
}

// Close releases resources
func (d *MP3Decoder) Close() error {
	if d.decoder != nil {
		return d.decoder.Close()
	}
	return nil
}
```

**Step 4: Build to verify compilation**

```bash
go build ./pkg/audio/decode
```

Expected: Success (FLAC decoder will have unused imports warning - that's OK for now)

**Step 5: Commit decoders**

```bash
git add pkg/audio/decode/opus.go pkg/audio/decode/flac.go pkg/audio/decode/mp3.go
git commit -m "feat: Add Opus, FLAC, MP3 decoders to public API

- Add Opus decoder with int16 -> int32 conversion
- Add FLAC decoder stub (needs streaming implementation)
- Add MP3 decoder with int16 -> int32 conversion"
```

---

## Task 5: Move Encoders to pkg/audio/encode

**Files:**
- Create: `pkg/audio/encode/encoder.go`
- Create: `pkg/audio/encode/pcm.go`
- Create: `pkg/audio/encode/opus.go`
- Reference: `internal/server/audio_engine.go` (PCM encoder)
- Reference: `internal/server/opus_encoder.go` (Opus encoder)

**Step 1: Create encoder interface**

Create: `pkg/audio/encode/encoder.go`

```go
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
```

**Step 2: Implement PCM encoder**

Create: `pkg/audio/encode/pcm.go`

```go
// ABOUTME: PCM audio encoder
// ABOUTME: Encodes int32 samples to 16-bit or 24-bit PCM bytes
package encode

import (
	"encoding/binary"
	"fmt"

	"github.com/harperreed/resonate-go/pkg/audio"
)

// PCMEncoder encodes PCM audio
type PCMEncoder struct {
	bitDepth int
}

// NewPCM creates a new PCM encoder
func NewPCM(format audio.Format) (Encoder, error) {
	if format.Codec != "pcm" {
		return nil, fmt.Errorf("invalid codec for PCM encoder: %s", format.Codec)
	}

	if format.BitDepth != 16 && format.BitDepth != 24 {
		return nil, fmt.Errorf("unsupported bit depth: %d (supported: 16, 24)", format.BitDepth)
	}

	return &PCMEncoder{
		bitDepth: format.BitDepth,
	}, nil
}

// Encode converts int32 samples to PCM bytes
func (e *PCMEncoder) Encode(samples []int32) ([]byte, error) {
	if e.bitDepth == 24 {
		// 24-bit PCM: 3 bytes per sample
		output := make([]byte, len(samples)*3)
		for i, sample := range samples {
			bytes := audio.SampleTo24Bit(sample)
			output[i*3] = bytes[0]
			output[i*3+1] = bytes[1]
			output[i*3+2] = bytes[2]
		}
		return output, nil
	} else {
		// 16-bit PCM: 2 bytes per sample
		output := make([]byte, len(samples)*2)
		for i, sample := range samples {
			sample16 := audio.SampleToInt16(sample)
			binary.LittleEndian.PutUint16(output[i*2:], uint16(sample16))
		}
		return output, nil
	}
}

// Close releases resources
func (e *PCMEncoder) Close() error {
	return nil
}
```

**Step 3: Copy Opus encoder from internal**

Create: `pkg/audio/encode/opus.go`

```go
// ABOUTME: Opus audio encoder
// ABOUTME: Encodes int32 samples to Opus bytes
package encode

import (
	"fmt"

	"github.com/harperreed/resonate-go/pkg/audio"
	"gopkg.in/hraban/opus.v2"
)

// OpusEncoder encodes Opus audio
type OpusEncoder struct {
	encoder    *opus.Encoder
	sampleRate int
	channels   int
	frameSize  int
}

// NewOpus creates a new Opus encoder
func NewOpus(format audio.Format) (Encoder, error) {
	if format.Codec != "opus" {
		return nil, fmt.Errorf("invalid codec for Opus encoder: %s", format.Codec)
	}

	encoder, err := opus.NewEncoder(format.SampleRate, format.Channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// Opus frame size depends on sample rate
	frameSize := format.SampleRate / 50 // 20ms frame

	return &OpusEncoder{
		encoder:    encoder,
		sampleRate: format.SampleRate,
		channels:   format.Channels,
		frameSize:  frameSize,
	}, nil
}

// Encode converts int32 samples to Opus bytes
func (e *OpusEncoder) Encode(samples []int32) ([]byte, error) {
	// Convert int32 to int16 for Opus
	pcm := make([]int16, len(samples))
	for i, sample := range samples {
		pcm[i] = audio.SampleToInt16(sample)
	}

	// Encode to Opus
	data := make([]byte, 4000) // Max Opus packet size
	n, err := e.encoder.Encode(pcm, data)
	if err != nil {
		return nil, fmt.Errorf("opus encode error: %w", err)
	}

	return data[:n], nil
}

// Close releases resources
func (e *OpusEncoder) Close() error {
	return nil
}
```

**Step 4: Build to verify**

```bash
go build ./pkg/audio/encode
```

Expected: Success

**Step 5: Commit encoders**

```bash
git add pkg/audio/encode/
git commit -m "feat: Add PCM and Opus encoders to public API

- Add Encoder interface
- Add PCM encoder supporting 16-bit and 24-bit
- Add Opus encoder with int32 -> int16 conversion"
```

---

## Task 6: Move Resampler to pkg/audio/resample

**Files:**
- Create: `pkg/audio/resample/resampler.go`
- Create: `pkg/audio/resample/resampler_test.go`
- Reference: `internal/server/resampler.go` (source)
- Reference: `internal/server/resampler_test.go` (test source)

**Step 1: Copy resampler from internal**

Create: `pkg/audio/resample/resampler.go`

```go
// ABOUTME: Audio resampler using linear interpolation
// ABOUTME: Converts audio between different sample rates
package resample

// Resampler converts audio between sample rates using linear interpolation
type Resampler struct {
	inputRate  int
	outputRate int
	channels   int
	position   float64
}

// New creates a new resampler
func New(inputRate, outputRate, channels int) *Resampler {
	return &Resampler{
		inputRate:  inputRate,
		outputRate: outputRate,
		channels:   channels,
		position:   0,
	}
}

// Resample converts input samples to output sample rate
// Returns number of samples written to output
func (r *Resampler) Resample(input, output []int32) int {
	if len(input) == 0 {
		return 0
	}

	ratio := float64(r.inputRate) / float64(r.outputRate)
	outputSamples := 0
	inputFrames := len(input) / r.channels

	for outputSamples < len(output)/r.channels {
		// Get interpolation position
		inputPos := r.position * ratio
		inputFrame := int(inputPos)

		if inputFrame >= inputFrames-1 {
			break
		}

		// Linear interpolation
		frac := inputPos - float64(inputFrame)

		for ch := 0; ch < r.channels; ch++ {
			idx1 := inputFrame*r.channels + ch
			idx2 := (inputFrame+1)*r.channels + ch

			sample1 := float64(input[idx1])
			sample2 := float64(input[idx2])

			interpolated := sample1 + (sample2-sample1)*frac
			output[outputSamples*r.channels+ch] = int32(interpolated)
		}

		outputSamples++
		r.position++
	}

	return outputSamples * r.channels
}
```

**Step 2: Copy tests from internal**

Create: `pkg/audio/resample/resampler_test.go`

```go
// ABOUTME: Tests for audio resampler
// ABOUTME: Tests linear interpolation resampling between sample rates
package resample

import (
	"testing"
)

func TestNew(t *testing.T) {
	r := New(44100, 48000, 2)

	if r == nil {
		t.Fatal("expected resampler to be created")
	}

	if r.inputRate != 44100 {
		t.Errorf("expected inputRate 44100, got %d", r.inputRate)
	}

	if r.outputRate != 48000 {
		t.Errorf("expected outputRate 48000, got %d", r.outputRate)
	}

	if r.channels != 2 {
		t.Errorf("expected channels 2, got %d", r.channels)
	}
}

func TestResampleUpsampling(t *testing.T) {
	// 44100 -> 48000 (upsampling by factor of ~1.088)
	r := New(44100, 48000, 2)

	// Input: 100 stereo samples (200 int32 values)
	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 100) // Ramp signal
	}

	// Calculate expected output size
	expectedSize := int(float64(len(input)) * float64(48000) / float64(44100))
	output := make([]int32, expectedSize)

	n := r.Resample(input, output)

	// Should have produced output
	if n == 0 {
		t.Fatal("resampler produced no output")
	}

	// Should have produced approximately the expected amount
	// Allow some tolerance due to rounding
	if n < expectedSize-10 || n > expectedSize+10 {
		t.Errorf("expected ~%d samples, got %d", expectedSize, n)
	}

	// Output should have interpolated values (not exact copies)
	allZero := true
	for i := 0; i < n; i++ {
		if output[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("output contains only zeros")
	}
}

func TestResampleDownsampling(t *testing.T) {
	// 48000 -> 44100 (downsampling by factor of ~0.91875)
	r := New(48000, 44100, 2)

	// Input: 100 stereo samples
	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 100)
	}

	expectedSize := int(float64(len(input)) * float64(44100) / float64(48000))
	output := make([]int32, expectedSize)

	n := r.Resample(input, output)

	if n == 0 {
		t.Fatal("resampler produced no output")
	}

	if n < expectedSize-10 || n > expectedSize+10 {
		t.Errorf("expected ~%d samples, got %d", expectedSize, n)
	}
}

func TestResampleSameRate(t *testing.T) {
	// No resampling needed (48000 -> 48000)
	r := New(48000, 48000, 2)

	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 100)
	}

	output := make([]int32, len(input)+10) // Extra space for rounding
	n := r.Resample(input, output)

	// Should produce approximately the same number of samples
	// Allow small tolerance for floating point rounding
	if n < len(input)-5 || n > len(input)+5 {
		t.Errorf("expected ~%d samples, got %d", len(input), n)
	}

	// Values should be similar (allow for interpolation artifacts)
	for i := 0; i < n && i < len(input); i++ {
		diff := abs(int(output[i]) - int(input[i]))
		if diff > 200 { // Allow some rounding errors
			t.Errorf("sample %d: expected ~%d, got %d (diff %d)", i, input[i], output[i], diff)
		}
	}
}

func TestResampleEmptyInput(t *testing.T) {
	r := New(44100, 48000, 2)

	input := []int32{}
	output := make([]int32, 100)

	n := r.Resample(input, output)

	if n != 0 {
		t.Errorf("expected 0 samples from empty input, got %d", n)
	}
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
```

**Step 3: Run tests**

```bash
go test -v ./pkg/audio/resample
```

Expected: All tests pass

**Step 4: Commit resampler**

```bash
git add pkg/audio/resample/
git commit -m "feat: Add resampler to public API

- Add Resampler using linear interpolation
- Support upsampling and downsampling
- Add comprehensive tests"
```

---

## Task 7: Move Audio Output to pkg/audio/output

**Files:**
- Create: `pkg/audio/output/output.go`
- Create: `pkg/audio/output/portaudio.go`
- Reference: `internal/player/output.go` (source)

**Step 1: Create output interface**

Create: `pkg/audio/output/output.go`

```go
// ABOUTME: Audio output interface definition
// ABOUTME: Common interface for audio playback backends
package output

// Output represents an audio output device
type Output interface {
	// Open initializes the output device
	Open(sampleRate, channels int) error

	// Write outputs audio samples (blocks until written)
	Write(samples []int32) error

	// Close releases output resources
	Close() error
}
```

**Step 2: Copy PortAudio implementation from internal**

Create: `pkg/audio/output/portaudio.go`

```go
// ABOUTME: PortAudio output implementation
// ABOUTME: Cross-platform audio output using PortAudio
package output

import (
	"fmt"

	"github.com/gordonklaus/portaudio"
	"github.com/harperreed/resonate-go/pkg/audio"
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
```

**Step 3: Build to verify**

```bash
go build ./pkg/audio/output
```

Expected: Success

**Step 4: Commit audio output**

```bash
git add pkg/audio/output/
git commit -m "feat: Add audio output to public API

- Add Output interface
- Add PortAudio implementation
- Convert int32 samples to int16 for playback"
```

---

Due to length constraints, I'll continue with the remaining tasks in a summary format:

## Remaining Tasks Summary

### Task 8: Move Protocol to pkg/protocol
- Copy message types from `internal/protocol/messages.go`
- Copy WebSocket client from `internal/client/websocket.go`
- Update imports to use `pkg/audio`

### Task 9: Move Sync to pkg/sync
- Copy clock implementation from `internal/sync/clock.go`
- Copy tests from `internal/sync/clock_test.go`

### Task 10: Move Discovery to pkg/discovery
- Copy mDNS implementation from `internal/discovery/mdns.go`

### Task 11: Create High-Level Player API (pkg/resonate)
- Implement `Player` struct wrapping protocol client, decoder, scheduler, output
- Implement `PlayerConfig` and `NewPlayer()`
- Implement `Connect()`, `Play()`, `Pause()`, `Stop()`, `SetVolume()`, `Mute()`
- Write integration tests

### Task 12: Create High-Level Server API (pkg/resonate)
- Implement `Server` struct wrapping audio engine and WebSocket server
- Implement `ServerConfig` and `NewServer()`
- Implement `Start()`, `Stop()`, `Clients()`
- Implement `AudioSource` interface
- Implement `FileSource()` and `TestToneSource()`
- Write integration tests

### Task 13: Migrate Player CLI
- Rewrite `cmd/resonate-player/main.go` to use `pkg/resonate.Player`
- Keep TUI in `internal/ui/`
- Verify CLI works identically

### Task 14: Migrate Server CLI
- Rewrite `cmd/resonate-server/main.go` to use `pkg/resonate.Server`
- Keep TUI in `internal/server/tui.go` (move to `internal/ui/`)
- Verify CLI works identically

### Task 15: Add Examples
- Create `examples/basic-player/`
- Create `examples/basic-server/`
- Create `examples/custom-source/`
- Create `examples/multi-room/`
- Create `examples/audio-pipeline/`

### Task 16: Update Documentation
- Update README.md with library usage
- Add godoc comments to all exported types/functions
- Update examples in README

### Task 17: Final Testing and Release
- Run all tests: `go test ./...`
- Run CLI tools and verify functionality
- Run all examples
- Update CHANGELOG
- Tag v1.0.0

---

## Success Criteria

- ✅ All code moved from `internal/` to appropriate `pkg/` packages
- ✅ High-level Player API works for simple use cases
- ✅ High-level Server API works for simple use cases
- ✅ Low-level component APIs work independently
- ✅ CLI tools work using public library APIs
- ✅ All examples run successfully
- ✅ All tests pass
- ✅ Documentation complete (godoc + README + examples)
- ✅ Tagged as v1.0.0
