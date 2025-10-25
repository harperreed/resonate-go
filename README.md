# Resonate Go

A complete Resonate Protocol implementation in Go, featuring both server and player components for synchronized multi-room audio streaming.

## Features

### Server
- Stream audio files (MP3, FLAC, WAV) or generate test tones
- Multi-codec support (Opus @ 256kbps, PCM)
- mDNS service advertisement for automatic discovery
- Real-time terminal UI showing connected clients
- WebSocket-based streaming with precise timestamps

### Player
- Automatic server discovery via mDNS
- Multi-codec support (Opus, FLAC, PCM)
- Precise clock synchronization for multi-room audio
- Interactive terminal UI with volume control
- Jitter buffer for smooth playback

## Installation

### Prerequisites

You'll need `pkg-config` and the Opus libraries installed:

```bash
# macOS
brew install pkg-config opus opusfile

# Ubuntu/Debian
sudo apt-get install pkg-config libopus-dev libopusfile-dev

# Fedora
sudo dnf install pkg-config opus-devel opusfile-devel
```

### Build

Build both server and player:

```bash
make
```

Or build individually:

```bash
make server  # Builds resonate-server
make player  # Builds resonate-go
```

## Usage

### Server

Start a server with the interactive TUI (default):

```bash
./resonate-server
```

Stream an audio file:

```bash
./resonate-server --audio /path/to/music.mp3
```

Run without TUI (streaming logs to stdout):

```bash
./resonate-server --no-tui
```

#### Server Options

- `--port` - WebSocket server port (default: 8927)
- `--name` - Server friendly name (default: hostname-resonate-server)
- `--audio` - Audio file to stream (MP3, FLAC, WAV). If not specified, plays 440Hz test tone
- `--log-file` - Log file path (default: resonate-server.log)
- `--debug` - Enable debug logging
- `--no-mdns` - Disable mDNS advertisement (clients must connect manually)
- `--no-tui` - Disable TUI, use streaming logs instead

#### Server TUI

The server TUI shows:
- Server name and port
- Uptime
- Currently playing audio
- Connected clients with codec and state
- Press `q` or `Ctrl+C` to quit

### Player

Start a player (auto-discovers servers via mDNS):

```bash
./resonate-go --name "Living Room"
```

Connect to a specific server manually:

```bash
./resonate-go --server ws://192.168.1.100:8927 --name "Kitchen"
```

#### Player Options

- `--server` - Manual server WebSocket address (skips mDNS discovery)
- `--port` - Port for mDNS advertisement (default: 8927)
- `--name` - Player friendly name (default: hostname-resonate-player)
- `--buffer-ms` - Jitter buffer size in milliseconds (default: 150)
- `--log-file` - Log file path (default: resonate-player.log)
- `--debug` - Enable debug logging

#### Player TUI

The player TUI shows:
- Player name
- Server connection status
- Current audio title/artist
- Codec and sample rate
- Buffer depth
- Clock sync statistics (offset, RTT, drift)
- Playback statistics (received, played, dropped)
- Volume control (Up/Down arrows or +/- keys)
- Press `m` to mute/unmute
- Press `q` or `Ctrl+C` to quit

## Architecture

### Server

The server streams audio in 20ms chunks with microsecond timestamps. Audio is buffered 500ms ahead to allow for network jitter and clock synchronization.

**Pipeline:**
1. Audio source (file decoder or test tone generator)
2. Per-client codec negotiation (Opus or PCM)
3. Timestamp generation using monotonic clock
4. WebSocket binary message streaming

### Player

The player uses a sophisticated scheduling system to ensure perfectly synchronized playback across multiple rooms.

**Pipeline:**
1. WebSocket client receives timestamped audio chunks
2. Clock sync system converts server timestamps to local time
3. Priority queue scheduler with startup buffering (200ms)
4. Persistent audio player with streaming I/O pipe
5. Software volume control and mixing

### Clock Synchronization

Both server and player use NTP-style clock synchronization:
- Continuous RTT measurement
- Drift correction
- Microsecond precision timestamps
- Handles network jitter and delays

## Example: Multi-Room Setup

Terminal 1 - Start the server:
```bash
./resonate-server --audio ~/Music/favorite-album.mp3
```

Terminal 2 - Living room player:
```bash
./resonate-go --name "Living Room"
```

Terminal 3 - Kitchen player:
```bash
./resonate-go --name "Kitchen"
```

Both players will discover the server via mDNS and start playing in perfect sync.

## Development

Run tests:
```bash
make test
```

Clean binaries:
```bash
make clean
```

Install to GOPATH/bin:
```bash
make install
```

## Protocol

Implements the [Resonate Protocol](https://github.com/Resonate-Protocol/spec) specification.
