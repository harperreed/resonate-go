# Sendspin Examples

This directory contains example programs demonstrating how to use the Sendspin library.

## Available Examples

### [basic-player](basic-player/)

A simple audio player that connects to a Sendspin server and plays synchronized audio.

**Key features:**
- Simple configuration with callbacks
- Automatic clock synchronization
- Status monitoring and statistics
- Metadata display

**Run it:**
```bash
cd basic-player
go build
./basic-player -server localhost:8927
```

### [basic-server](basic-server/)

A simple streaming server that broadcasts a test tone to connected players.

**Key features:**
- Test tone generation (440 Hz)
- Multi-client support
- Automatic codec negotiation
- mDNS service advertisement

**Run it:**
```bash
cd basic-server
go build
./basic-server
```

### [custom-source](custom-source/)

Demonstrates how to implement custom `AudioSource` interfaces for generating or processing audio.

**Key features:**
- Multiple tone mixing (chords)
- Frequency sweeps (chirps)
- Custom audio generation
- AudioSource interface implementation

**Run it:**
```bash
cd custom-source
go build
./custom-source -mode chord
```

## Quick Start

### 1. Start a server

```bash
cd examples/basic-server
go build && ./basic-server
```

### 2. Connect a player

In another terminal:

```bash
cd examples/basic-player
go build && ./basic-player
```

You should hear a 440 Hz test tone playing with synchronized timing.

## Building All Examples

From the repository root:

```bash
go build ./examples/basic-player/
go build ./examples/basic-server/
go build ./examples/custom-source/
```

## Example Progression

1. **Start with basic-server** - Understand server setup and test tone streaming
2. **Try basic-player** - Learn player configuration and playback
3. **Explore custom-source** - Implement custom audio sources

## Key Concepts Demonstrated

### Player API (`pkg/sendspin`)

```go
// Create and configure player
config := sendspin.PlayerConfig{
    ServerAddr: "localhost:8927",
    PlayerName: "Living Room",
    Volume:     80,
    OnMetadata: func(meta sendspin.Metadata) {
        fmt.Printf("Now playing: %s - %s\n", meta.Artist, meta.Title)
    },
}

player, _ := sendspin.NewPlayer(config)
player.Connect()
player.Play()
```

### Server API (`pkg/sendspin`)

```go
// Create audio source
source := sendspin.NewTestTone(192000, 2)

// Create and start server
config := sendspin.ServerConfig{
    Port:   8927,
    Source: source,
}

server, _ := sendspin.NewServer(config)
server.Start()
```

### AudioSource Interface

```go
type AudioSource interface {
    Read(samples []int32) (int, error)
    SampleRate() int
    Channels() int
    Metadata() (title, artist, album string)
    Close() error
}
```

## Audio Format

All examples use hi-res audio:
- **Sample rate**: 192 kHz (configurable)
- **Bit depth**: 24-bit
- **Channels**: 2 (stereo)
- **Format**: int32 PCM samples

## Network Discovery

Servers advertise via mDNS by default. Players can discover local servers automatically.

## Next Steps

- Check out the full CLI tools: `cmd/sendspin-server/` (and root main.go for player)
- Read the API documentation: `pkg/sendspin/`
- Explore lower-level APIs: `pkg/audio/`, `pkg/protocol/`, `pkg/sync/`

## Troubleshooting

**No audio playing?**
- Check server is running: `netstat -an | grep 8927`
- Check firewall allows port 8927
- Verify audio output device is working

**Connection refused?**
- Ensure server is started before connecting player
- Check server address is correct
- Try explicit IP instead of localhost

**Audio glitches or dropouts?**
- Increase buffer size: `-buffer 1000` (player)
- Check network latency
- Monitor clock sync quality in stats

## License

See the repository's LICENSE file for details.
