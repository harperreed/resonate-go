# Library Refactor Design

**Date:** 2025-10-25
**Status:** Approved
**Approach:** Ground-Up Redesign (Approach C)

## Overview

Convert resonate-go from a CLI-focused project to a library-first architecture with layered APIs. The existing CLI tools will become thin wrappers that use the public library APIs, serving as reference implementations.

## Goals

1. **Library-first architecture** - Primary use case is embedding in Go applications
2. **Layered APIs** - High-level convenience for most users, low-level building blocks for power users
3. **Aggressive migration** - Complete restructure, not gradual wrapper approach
4. **Backward compatibility** - CLI tools use library but maintain same functionality
5. **Clean package design** - Intuitive organization, clear boundaries, good documentation

## Use Cases

### Primary Use Cases
- Embed player in Go applications (desktop music apps, smart home controllers)
- Embed server in Go applications (music streaming services, home media servers)
- Build custom audio pipelines (decoders, resamplers, encoders for custom processing)
- Enable Music Assistant and similar systems to integrate Resonate directly

### Example Use Cases
- Desktop music player with custom UI
- Multi-room audio controller
- Home media server with Resonate output
- Audio processing pipeline with hi-res support
- Music Assistant native Resonate provider

## Architecture

### Three-Layer Design

**Layer 1: High-Level Convenience (`pkg/resonate/`)**
- Simple constructors: `NewPlayer()`, `NewServer()`
- Sensible defaults for common use cases
- Hides complexity: connection management, format negotiation, error recovery
- Target: Users who want "just play audio" or "just serve audio"

**Layer 2: Component APIs (`pkg/audio/`, `pkg/protocol/`, `pkg/sync/`, `pkg/discovery/`)**
- Building blocks for custom implementations
- Each package focused on one concern
- Composable: mix and match components
- Target: Users building custom audio pipelines or integrations

**Layer 3: Internal Implementation (`internal/`)**
- CLI app logic (`internal/app/`)
- TUI implementation (`internal/ui/`)
- Implementation details not exposed as public API

## Package Structure

```
resonate-go/
├── pkg/
│   ├── resonate/           # High-level convenience API
│   │   ├── player.go       # Player API
│   │   ├── server.go       # Server API
│   │   └── source.go       # AudioSource interface + built-ins
│   ├── audio/              # Audio fundamentals
│   │   ├── format.go       # Format, Buffer types
│   │   ├── types.go        # Constants, conversion helpers
│   │   ├── decode/         # Decoders
│   │   │   ├── decoder.go  # Interface
│   │   │   ├── pcm.go      # PCM decoder
│   │   │   ├── opus.go     # Opus decoder
│   │   │   ├── flac.go     # FLAC decoder
│   │   │   └── mp3.go      # MP3 decoder
│   │   ├── encode/         # Encoders
│   │   │   ├── encoder.go  # Interface
│   │   │   ├── pcm.go      # PCM encoder
│   │   │   └── opus.go     # Opus encoder
│   │   ├── resample/       # Resampling
│   │   │   └── resampler.go
│   │   └── output/         # Audio output
│   │       ├── output.go   # Interface
│   │       └── portaudio.go
│   ├── protocol/           # Resonate wire protocol
│   │   ├── messages.go     # Protocol message types
│   │   └── client.go       # WebSocket client
│   ├── sync/               # Clock synchronization
│   │   └── clock.go
│   └── discovery/          # mDNS discovery
│       └── mdns.go
├── cmd/
│   ├── resonate-player/    # Thin CLI wrapper
│   └── resonate-server/    # Thin CLI wrapper
├── internal/
│   ├── app/                # CLI app logic
│   └── ui/                 # TUI implementation
├── examples/               # Example code
│   ├── basic-player/
│   ├── basic-server/
│   ├── custom-source/
│   ├── multi-room/
│   └── audio-pipeline/
└── docs/
    └── plans/
```

## API Design

### High-Level API (`pkg/resonate/`)

#### Player API

```go
package resonate

// PlayerConfig for creating a player
type PlayerConfig struct {
    ServerAddr string  // "localhost:8927" or discovered via mDNS
    PlayerName string  // Display name
    Volume     int     // 0-100
    DebugMode  bool    // Enable debug logging
}

// Player represents a Resonate audio player
type Player struct {
    // unexported fields
}

// NewPlayer creates a new player instance
func NewPlayer(cfg PlayerConfig) (*Player, error)

// Connect to the configured server
func (p *Player) Connect() error

// Play starts playback
func (p *Player) Play() error

// Pause pauses playback
func (p *Player) Pause() error

// Stop stops playback and disconnects
func (p *Player) Stop() error

// SetVolume adjusts volume (0-100)
func (p *Player) SetVolume(vol int) error

// Mute toggles mute
func (p *Player) Mute(muted bool) error

// Status returns current playback status
func (p *Player) Status() PlayerStatus

// Close releases resources
func (p *Player) Close() error
```

#### Server API

```go
package resonate

// ServerConfig for creating a server
type ServerConfig struct {
    Address    string      // "0.0.0.0"
    Port       int         // 8927
    Source     AudioSource // Where audio comes from
    EnablemDNS bool        // Advertise via mDNS
    DebugMode  bool
}

// Server serves audio to Resonate clients
type Server struct {
    // unexported fields
}

// NewServer creates a new server instance
func NewServer(cfg ServerConfig) (*Server, error)

// Start begins serving (blocks)
func (s *Server) Start() error

// Stop gracefully shuts down
func (s *Server) Stop() error

// Clients returns connected client info
func (s *Server) Clients() []ClientInfo
```

#### AudioSource Interface

```go
package resonate

// AudioSource provides audio samples for the server
type AudioSource interface {
    Read(samples []int32) (int, error)
    SampleRate() int
    Channels() int
    BitDepth() int
    Metadata() (title, artist, album string)
    Close() error
}

// Built-in source constructors
func FileSource(path string) (AudioSource, error)
func TestToneSource(frequency float64) AudioSource
```

### Component-Level APIs

#### `pkg/audio/` - Audio Fundamentals

```go
package audio

// Core types
type Format struct {
    Codec      string
    SampleRate int
    Channels   int
    BitDepth   int
    CodecHeader []byte
}

type Buffer struct {
    Timestamp int64
    PlayAt    time.Time
    Samples   []int32
    Format    Format
}

// Constants
const (
    Max24Bit = 8388607  // 2^23 - 1
    Min24Bit = -8388608 // -2^23
)

// Conversion helpers
func SampleToInt16(sample int32) int16
func SampleFromInt16(sample int16) int32
func SampleTo24Bit(sample int32) [3]byte
func SampleFrom24Bit(b [3]byte) int32
```

#### `pkg/audio/decode/` - Decoders

```go
package decode

// Decoder interface
type Decoder interface {
    Decode(data []byte) ([]int32, error)
    Close() error
}

// Constructors for each codec
func NewPCM(format audio.Format) (Decoder, error)
func NewOpus(format audio.Format) (Decoder, error)
func NewFLAC(format audio.Format) (Decoder, error)
func NewMP3(format audio.Format) (Decoder, error)
```

#### `pkg/audio/encode/` - Encoders

```go
package encode

type Encoder interface {
    Encode(samples []int32) ([]byte, error)
    Close() error
}

func NewPCM(format audio.Format) (Encoder, error)
func NewOpus(format audio.Format) (Encoder, error)
```

#### `pkg/audio/resample/` - Resampling

```go
package resample

type Resampler struct {
    // unexported
}

func New(inputRate, outputRate, channels int) *Resampler
func (r *Resampler) Resample(input, output []int32) int
```

#### `pkg/audio/output/` - Audio Output

```go
package output

type Output interface {
    Open(sampleRate, channels int) error
    Write(samples []int32) error
    Close() error
}

func NewPortAudio() Output
```

#### `pkg/protocol/` - Resonate Wire Protocol

```go
package protocol

// Message types
type HelloMessage struct {
    PlayerName       string
    SupportFormats   []Format
    SupportCodecs    []string
    SupportSampleRates []int
    SupportBitDepth  []int
}

type StartMessage struct {
    Format       Format
    ServerTime   int64
    StreamOffset int64
}

type ChunkMessage struct {
    Timestamp int64
    Data      []byte
}

type ControlMessage struct {
    Command string // "play", "pause", "stop"
}

// Client for low-level protocol control
type Client struct {
    // unexported
}

func NewClient(serverAddr string) (*Client, error)
func (c *Client) SendHello(hello HelloMessage) error
func (c *Client) ReadMessage() (interface{}, error)
func (c *Client) Close() error
```

#### `pkg/sync/` - Clock Synchronization

```go
package sync

type Clock struct {
    // unexported
}

func NewClock() *Clock
func (c *Clock) Sync(serverAddr string) error
func (c *Clock) ServerTime() int64
func (c *Clock) LocalTime() time.Time
func (c *Clock) Offset() int64
```

#### `pkg/discovery/` - mDNS Server Discovery

```go
package discovery

type Service struct {
    Name    string
    Address string
    Port    int
}

// Discover servers on the network
func Discover(timeout time.Duration) ([]Service, error)

// Advertise this server
func Advertise(name string, port int) error
func StopAdvertising() error
```

## CLI Migration

The existing CLI tools (`cmd/resonate-player/main.go`, `cmd/resonate-server/main.go`) will be rewritten to use the high-level `pkg/resonate/` API. They will handle only:
- Flag parsing
- TUI setup (using `internal/ui/`)
- Signal handling
- Calling library functions

### Example - New Player CLI

```go
package main

import (
    "github.com/harperreed/resonate-go/pkg/resonate"
    "github.com/harperreed/resonate-go/internal/ui"
)

func main() {
    // Parse flags
    cfg := resonate.PlayerConfig{
        ServerAddr: *serverFlag,
        PlayerName: *nameFlag,
        Volume:     *volumeFlag,
        DebugMode:  *debugFlag,
    }

    // Create player using library
    player, err := resonate.NewPlayer(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer player.Close()

    // Connect and play
    if err := player.Connect(); err != nil {
        log.Fatal(err)
    }

    // Run TUI (internal implementation)
    ui.RunPlayerUI(player)
}
```

## Examples

Create `examples/` directory with real-world usage:
- `examples/basic-player/` - Simple player implementation
- `examples/basic-server/` - Simple server implementation
- `examples/custom-source/` - Custom AudioSource implementation
- `examples/multi-room/` - Multiple synchronized players
- `examples/audio-pipeline/` - Using low-level audio components

These examples serve dual purpose:
1. Documentation for library users
2. Integration tests for the library

## Migration Strategy

### Step-by-Step Plan

1. **Create new package structure** - Set up `pkg/` directories with package stubs
2. **Move audio primitives first** - `pkg/audio/`, `pkg/audio/decode/`, etc. (foundation layer)
3. **Move protocol layer** - `pkg/protocol/`, `pkg/sync/`, `pkg/discovery/`
4. **Build high-level APIs** - `pkg/resonate/` wrapping the components
5. **Migrate CLI tools** - Rewrite to use `pkg/resonate/`
6. **Add examples** - Create `examples/` directory with working code
7. **Documentation** - README updates, godoc comments on all exported types/functions

### Testing Strategy

- Move existing tests alongside the code as packages are migrated
- Add integration tests in `examples/` (they serve dual purpose as docs)
- Ensure CLI tools work identically to current behavior
- Test both high-level and low-level APIs work independently

### Version Strategy

- This is a major refactor → tag as `v1.0.0` when complete
- Signals stable library API and commitment to compatibility
- Previous v0.0.x were "pre-library" releases for CLI tools only

### Rollout Plan

1. Work in a feature branch/worktree (`library-refactor`)
2. Validate each layer works before moving to next
3. Merge to main when:
   - All packages implemented
   - CLI tools work with library
   - Examples run successfully
   - Tests pass
4. Tag `v1.0.0` release

## Documentation Requirements

### Package Documentation
Every exported package, type, function must have godoc comments:
- Package-level doc explaining purpose and usage
- Type-level doc with example
- Function-level doc with parameters and return values

### README Updates
- Add "Using as a Library" section
- Show both high-level and low-level API examples
- Link to examples directory
- Keep CLI usage docs

### Examples
Each example should be a complete, runnable program showing real-world usage.

## Success Criteria

1. ✅ All code moved from `internal/` to `pkg/` where appropriate
2. ✅ High-level API works for simple player/server use cases
3. ✅ Low-level APIs allow building custom pipelines
4. ✅ CLI tools work using public library APIs
5. ✅ All examples run successfully
6. ✅ All tests pass
7. ✅ Documentation complete (godoc + README + examples)
8. ✅ Tagged as v1.0.0

## Timeline Estimate

- **5-7 days** total for complete migration
- Foundation (audio/protocol): 2 days
- High-level API: 1-2 days
- CLI migration: 1 day
- Examples + documentation: 1-2 days
- Testing + polish: 1 day

## Trade-offs

### Advantages
- ✅ Best API design - clean separation, intuitive naming
- ✅ Maximum flexibility - every component usable independently
- ✅ Good documentation story - clear package boundaries
- ✅ Future-proof - stable v1.0.0 API

### Disadvantages
- ❌ Most work - complete restructure vs. simple wrapper
- ❌ Higher risk - more things to potentially break
- ❌ Longer timeline - 5-7 days vs. 1-2 days for simple approach

### Mitigation
- Work in isolated worktree/branch
- Validate each layer before proceeding
- Keep existing code as reference
- Comprehensive testing at each stage

## Next Steps

1. Set up git worktree for isolated development
2. Create detailed implementation plan with tasks
3. Begin migration starting with foundation layer
