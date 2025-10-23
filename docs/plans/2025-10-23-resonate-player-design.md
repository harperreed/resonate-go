# Resonate Go Player Design

**Date:** 2025-10-23
**Status:** Design Approved
**Target:** Implement a Resonate Protocol player in Go

## Overview

A multi-room synchronized audio player implementing the Resonate Protocol specification. The player can discover and connect to Music Assistant servers, receive and decode multiple audio formats, maintain precise clock synchronization for multi-room playback, and provide a rich terminal UI for monitoring and control.

## Requirements

### Functional Requirements
- Register as a player with Resonate servers using mDNS discovery (both server-initiated and client-initiated modes)
- Accept and decode audio streams in three formats: Opus, FLAC, PCM
- Maintain sub-millisecond clock synchronization with server for multi-room audio
- Control playback volume via software mixing
- Display stream metadata (title, artist, album)
- Provide interactive TUI for monitoring and control
- Handle stream format changes dynamically
- Report player state back to server

### Non-Functional Requirements
- Cross-platform (macOS, Linux, Windows)
- Low latency playback (<200ms buffer)
- Robust reconnection on network failures
- Minimal CPU usage during playback

## Architecture

### Component Overview

Event-driven architecture using Go channels and goroutines for clean separation of concerns.

**Core Components:**

1. **Discovery Manager** - mDNS service advertisement and discovery
2. **WebSocket Client** - Protocol communication with server
3. **Clock Sync** - NTP-style time synchronization
4. **Audio Decoder** - Multi-format audio decoding
5. **Playback Scheduler** - Timestamp-based playback scheduling
6. **Audio Output** - PCM playback with volume control
7. **TUI Controller** - User interface and monitoring

**Data Flow:**
```
Network → WebSocket → Decoder → Scheduler → Audio Output → Speaker
              ↓
        Control/Time Sync → Clock/State Management
              ↓
           TUI Display
```

### Key Data Structures

```go
// Channels for inter-goroutine communication
audioChunks chan AudioChunk      // Raw encoded chunks from network
decodedAudio chan PCMBuffer      // Decoded PCM ready for playback
controlMsgs chan ControlMessage  // Volume, mute, state changes
timeSyncReq chan TimeMessage     // Clock sync requests
timeSyncResp chan TimeMessage    // Clock sync responses
uiEvents chan UIEvent            // User input from TUI

// Core types
type AudioChunk struct {
    Timestamp int64  // Microseconds, server clock
    Data      []byte // Encoded audio frame
}

type PCMBuffer struct {
    Timestamp int64    // When to play (server clock)
    Samples   []int16  // PCM samples
    Format    AudioFormat
}

type AudioFormat struct {
    Codec      string // "opus", "flac", "pcm"
    SampleRate int    // 44100, 48000, etc.
    Channels   int    // 1=mono, 2=stereo
    BitDepth   int    // 16, 24
}
```

## Component Designs

### 1. Discovery Manager

**mDNS Service Configuration:**
- **Advertise (Server-initiated):**
  - Service: `_resonate._tcp.local.`
  - Port: 8927 (default, configurable)
  - TXT record: `path=/resonate`
- **Discover (Client-initiated):**
  - Browse for: `_resonate-server._tcp.local.`
  - Port: 8927
  - TXT record: `path=/resonate`

**Connection Strategy:**
- Run both modes simultaneously on startup
- First successful connection wins
- Exponential backoff on connection failures (1s, 2s, 4s, 8s, max 30s)
- Support manual server override via CLI flag

**Library:** `github.com/hashicorp/mdns`

### 2. WebSocket Client

**Connection Lifecycle:**

1. Connect to `ws://[host]:[port]/resonate`
2. Send `client/hello`:
```json
{
  "type": "client/hello",
  "payload": {
    "client_id": "[UUID]",
    "name": "[hostname]-resonate-player",
    "version": 1,
    "supported_roles": ["player"],
    "device_info": {
      "product_name": "Resonate Go Player",
      "manufacturer": "resonate-go",
      "software_version": "0.1.0"
    },
    "player_support": {
      "codecs": ["opus", "flac", "pcm"],
      "sample_rates": [44100, 48000],
      "channels": [1, 2],
      "bit_depths": [16, 24]
    }
  }
}
```
3. Wait for `server/hello` (block other messages)
4. Send initial `client/state`:
```json
{
  "type": "client/state",
  "payload": {
    "state": "synchronized",
    "volume": 100,
    "muted": false
  }
}
```
5. Begin clock sync loop
6. Ready for streaming

**Message Routing:**
- JSON messages → parse type, route to appropriate channel
- Binary messages → byte 0 = type (0 for audio), bytes 1-8 = timestamp (big-endian int64), rest = audio data

**Library:** `github.com/gorilla/websocket`

### 3. Clock Synchronization

**Protocol:** Three-timestamp NTP-style synchronization

**Flow:**
1. Client records `t1` (local time μs)
2. Send `client/time` with `t1`
3. Server receives at `t2`, responds immediately
4. Server sends `server/time` with `{t1, t2, t3}`
5. Client receives at `t4`

**Offset Calculation:**
```go
rtt := (t4 - t1) - (t3 - t2)
offset := ((t2 - t1) + (t3 - t4)) / 2

// Exponential moving average for stability
smoothedOffset = (oldOffset * 0.9) + (offset * 0.1)
```

**Timing:**
- Send `client/time` every 1 second
- Discard samples if RTT > 100ms (network congestion)
- Track jitter via rolling window of recent offsets
- Flag sync as degraded if no response in 5 seconds

**Usage:**
```go
localPlayTime := serverTimestamp - clockOffset
```

### 4. Audio Decoder

**Decoder per Format:**

- **Opus:** `github.com/hraban/opus` (CGO-based, stable)
- **FLAC:** `github.com/mewkiz/flac` (pure Go)
- **PCM:** Direct copy (no decoding)

**Process Flow:**
1. Wait for `stream/start` message with format specification
2. Initialize appropriate decoder with codec header if provided
3. Read from `audioChunks` channel
4. Decode each chunk to PCM
5. Write to `decodedAudio` channel with original timestamp preserved

**Format Changes:**
- On new `stream/start`, tear down old decoder
- Clear buffers
- Initialize new decoder
- Resume playback

**Error Handling:**
- Log decode errors but continue (skip bad frames)
- Report error stats to TUI

### 5. Playback Scheduler

**Buffering Strategy:**
- Maintain priority queue ordered by timestamp
- Target buffer depth: 100-150ms (configurable)
- Jitter buffer to absorb network variance

**Scheduling Algorithm:**
```go
for chunk := range decodedAudio {
    // Calculate local play time
    localPlayTime := chunk.Timestamp - clockOffset

    // Wait until precise moment
    sleepDuration := localPlayTime - time.Now()

    if sleepDuration > 0 {
        time.Sleep(sleepDuration)
        sendToOutput(chunk)
    } else if sleepDuration > -50*time.Millisecond {
        // Slightly late but playable
        sendToOutput(chunk)
    } else {
        // Too late, drop
        logDroppedFrame(chunk)
    }
}
```

**Buffer Management:**
- Monitor queue depth
- Report underruns/overruns to TUI
- Adjust jitter buffer dynamically if frequent issues

### 6. Audio Output

**Library:** `github.com/ebitengine/oto/v3` (pure Go, cross-platform)

**Initialization:**
```go
ctx, ready, err := oto.NewContext(
    sampleRate,
    channels,
    bitDepth/8,  // bytes per sample
)
player := ctx.NewPlayer(pcmReader)
```

**Volume Control:**
Software mixing applied before output:
```go
volumeMultiplier := float64(volume) / 100.0

for i := range samples {
    if muted {
        samples[i] = 0
    } else {
        samples[i] = int16(float64(samples[i]) * volumeMultiplier)
    }
}
```

**Error Handling:**
- Detect audio output failures
- Attempt re-initialization
- Report to TUI and server state

### 7. TUI Controller

**Library:** `github.com/charmbracelet/bubbletea`

**Display Layout:**
```
┌─ Resonate Player ────────────────────────────────────┐
│ Status: Connected to music-assistant.local           │
│ Sync:   ✓ Synced (offset: +2.3ms, jitter: 0.8ms)   │
├──────────────────────────────────────────────────────┤
│ Now Playing:                                         │
│   Track:  The Less I Know the Better                │
│   Artist: Tame Impala                                │
│   Album:  Currents                                   │
│                                                       │
│ Format: Opus 48kHz Stereo 16-bit                    │
│                                                       │
│ Volume: [████████░░] 80%                            │
│ Buffer: 150ms (50 chunks)                           │
├──────────────────────────────────────────────────────┤
│ Stats:  RX: 12,450  Played: 12,380  Dropped: 2      │
│                                                       │
│ ↑/↓:Volume  m:Mute  r:Reconnect  d:Debug  q:Quit   │
└──────────────────────────────────────────────────────┘
```

**Keyboard Controls:**
- `↑/↓`: Volume ±5%
- `m`: Toggle mute
- `r`: Force reconnect
- `d`: Toggle debug panel (goroutine stats, channel depths)
- `q`: Quit

**State Updates:**
- Subscribe to state changes via channels
- Debounce rapid updates (max 30 FPS)
- Send `client/state` to server when volume/mute changes

**Debug Panel (toggled):**
```
Goroutines: 8
Channels:
  audioChunks:   15/100
  decodedAudio:  8/50
  controlMsgs:   0/10
Clock Offset: +2345μs ±800μs
```

## Control Flow

### Server Command Handling

**Incoming `server/command`:**
```json
{
  "type": "server/command",
  "payload": {
    "command": "volume",
    "volume": 75
  }
}
```

**Processing:**
1. Parse command type
2. Apply change to internal state
3. Update audio output (volume multiplier)
4. Send `client/state` with changed fields:
```json
{
  "type": "client/state",
  "payload": {
    "volume": 75
  }
}
```
5. Update TUI display

### Metadata Updates

**Incoming `stream/metadata`:**
```json
{
  "type": "stream/metadata",
  "payload": {
    "title": "Song Title",
    "artist": "Artist Name",
    "album": "Album Name",
    "artwork_url": "https://..."
  }
}
```

**Processing:**
1. Parse metadata
2. Update TUI display immediately
3. Optionally cache artwork for display

## Error Handling

### Network Errors
- WebSocket disconnect → Show in TUI, start reconnect backoff
- Discovery failure → Continue trying both modes indefinitely
- Timeout on handshake → Reconnect with fresh connection

### Audio Errors
- Decode failure → Log, skip frame, continue
- Output device failure → Report error state to server, attempt re-init
- Buffer underrun → Report stats, continue (audio glitch acceptable)
- Late frames → Drop if >50ms late

### Clock Sync Errors
- No sync response in 5s → Mark degraded, continue using last offset
- Excessive jitter → Increase buffer size dynamically
- Complete sync loss → Continue playing but warn in TUI

## Dependencies

```
github.com/hashicorp/mdns           # mDNS discovery
github.com/gorilla/websocket        # WebSocket client
github.com/ebitengine/oto/v3        # Audio output
github.com/hraban/opus              # Opus decoder (CGO)
github.com/mewkiz/flac              # FLAC decoder
github.com/charmbracelet/bubbletea  # TUI framework
github.com/google/uuid              # Client ID generation
```

## Configuration

**CLI Flags:**
```
--server string       Manual server address (skip mDNS)
--port int           Port for mDNS advertisement (default: 8927)
--name string        Player friendly name (default: hostname-resonate-player)
--buffer-ms int      Jitter buffer size (default: 150ms)
--log-file string    Log file path (default: ./resonate-player.log)
--debug              Enable debug logging
```

**Config File (optional):**
```yaml
# ~/.config/resonate-player/config.yaml
server: "music-assistant.local:8927"
name: "Living Room Speaker"
buffer_ms: 200
log_level: "info"
```

## Testing Strategy

### Unit Tests
- Clock offset calculation
- Audio format conversions
- Volume mixing algorithm
- Message parsing/serialization

### Integration Tests
- Mock WebSocket server for protocol testing
- Synthetic audio stream playback
- Reconnection scenarios
- Format switching

### Manual Testing
- Multi-room sync with multiple instances
- Network disruption (wifi drops)
- Long-running stability (24h+)
- Various audio formats from Music Assistant

## Future Enhancements

- Visualizer support (spectrum analyzer in TUI)
- Metadata with artwork display (kitty/sixel protocols)
- Hardware volume control integration (ALSA/PulseAudio/CoreAudio)
- Web-based control UI
- Playlist/queue display
- Audio effects (EQ, crossfade)

## References

- Resonate Protocol Spec: https://github.com/Resonate-Protocol/spec
- Music Assistant: https://music-assistant.io/
