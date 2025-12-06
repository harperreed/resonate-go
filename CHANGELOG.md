# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0] - 2025-10-25

### Added

**Library-First Architecture**
- Complete restructure from CLI-focused to library-first design
- Three-tier architecture: high-level convenience API, component APIs, and private implementation
- Comprehensive public API in `pkg/` for library consumers
- CLI tools reimplemented as thin wrappers using the public library

**Public API Packages**

- `pkg/sendspin` - High-level Player and Server APIs for most use cases
  - `Player` - Connect to servers and play synchronized audio with volume control, callbacks
  - `Server` - Serve audio to multiple clients with custom source support
  - `AudioSource` - Interface for custom audio sources
  - `FileSource()` - Read audio from FLAC, MP3, WAV files
  - `TestToneSource()` - Generate test tones for debugging

- `pkg/audio` - Core audio types and utilities
  - `Format` - Audio stream format descriptor (codec, sample rate, channels, bit depth)
  - `Buffer` - Decoded PCM audio with timestamp information
  - Sample conversion functions for 16-bit and 24-bit audio

- `pkg/audio/decode` - Audio decoders for multiple codecs
  - `Decoder` - Common interface for all decoders
  - PCM decoder - 16-bit and 24-bit support
  - Opus decoder - with int16 to int32 conversion
  - FLAC decoder - stub for future implementation
  - MP3 decoder - with int16 to int32 conversion

- `pkg/audio/encode` - Audio encoders
  - `Encoder` - Common interface for all encoders
  - PCM encoder - 16-bit and 24-bit support
  - Opus encoder - with int32 to int16 conversion

- `pkg/audio/resample` - Sample rate conversion
  - Linear interpolation resampler
  - Support for upsampling and downsampling
  - Multi-channel support

- `pkg/audio/output` - Audio playback interfaces
  - `Output` - Common interface for playback backends
  - PortAudio implementation for cross-platform audio

- `pkg/protocol` - Sendspin wire protocol
  - Message types for client/server communication
  - WebSocket client implementation
  - Protocol version negotiation

- `pkg/sync` - Clock synchronization
  - NTP-style clock sync with Sendspin servers
  - Round-trip time measurement
  - Quality tracking for sync accuracy

- `pkg/discovery` - mDNS service discovery
  - Discover Sendspin servers on local network
  - Advertise server availability
  - Service registration and browsing

**Examples**
- `examples/basic-player` - Simple audio player example
- `examples/basic-server` - Simple audio server example
- `examples/custom-source` - Custom audio source implementation

**Documentation**
- Comprehensive godoc comments for all public types and functions
- Package-level documentation for each public package
- Examples in README demonstrating library usage
- Complete refactoring design document

**Testing**
- 180+ tests covering all public APIs
- Integration tests for Player and Server
- Unit tests for audio processing components
- Codec-specific tests for encoders and decoders

### Changed

**Breaking Changes**
- CLI tools now use public library APIs instead of internal implementations
- Internal packages moved to `pkg/` for public consumption
- Audio processing now consistently uses int32 samples in 24-bit range

**Architecture**
- Player CLI (`sendspin-player`) - Thin wrapper around `pkg/sendspin.Player`
- Server CLI (`sendspin-server`) - Thin wrapper around `pkg/sendspin.Server`
- All audio processing moved to reusable public packages
- Clean separation between library code and CLI code

### Fixed
- Consistent sample format across all audio processing components
- Proper resource cleanup in decoders and encoders
- Thread-safe clock synchronization
- Robust error handling in all public APIs

### Technical Details

**Audio Pipeline**
- All decoders output int32 samples in 24-bit range for consistent hi-res audio
- All encoders accept int32 samples and convert as needed for codec
- Resampler uses linear interpolation for quality upsampling/downsampling
- PortAudio output converts int32 to int16 for playback

**Wire Protocol**
- WebSocket-based streaming with binary audio frames
- Clock sync messages for precise timing
- Metadata messages for track information
- Client state messages for monitoring

**Clock Synchronization**
- NTP-style round-trip time measurement
- Multiple sync samples for accuracy
- Quality tracking and bad sample rejection
- Thread-safe concurrent access

**Service Discovery**
- mDNS-based server discovery
- Automatic service registration
- Server name and capability advertising

### Migration Guide

For users of pre-1.0.0 versions, the library API is now the recommended way to use Sendspin:

**Old (Internal API):**
```go
// Not recommended - internal packages
import "github.com/Sendspin/sendspin-go/internal/player"
```

**New (Public API):**
```go
// Recommended - public library API
import "github.com/Sendspin/sendspin-go/pkg/sendspin"

player, err := sendspin.NewPlayer(sendspin.PlayerConfig{
    ServerAddr: "localhost:8927",
    PlayerName: "Living Room",
    Volume:     80,
})
```

### Version Information
- Go 1.23+
- Supports hi-res audio up to 192kHz/24-bit
- Cross-platform: macOS, Linux, Windows
- Architecture: x86_64, ARM64

---

## [0.3.0] - 2025-10-24

### Added
- Hi-res audio support up to 192kHz/24-bit
- Multi-source audio support
- Server TUI with real-time client monitoring
- Comprehensive server and player documentation

### Fixed
- Audio timing and distortion issues
- Buffer management improvements
- Clock synchronization accuracy

---

## [0.2.0] - 2025-10-20

### Added
- Basic player and server functionality
- Clock synchronization
- mDNS discovery
- TUI for player

### Changed
- Initial implementation

---

## [0.1.0] - 2025-10-15

### Added
- Initial project structure
- Basic audio streaming
- WebSocket protocol
