# Basic Server Example

This example demonstrates how to create a simple Resonate streaming server that broadcasts a test tone to connected players.

## What it does

- Creates a Resonate server that streams a 440Hz test tone
- Listens for WebSocket connections from players
- Broadcasts synchronized audio to all connected clients
- Supports mDNS service advertisement for easy discovery
- Handles multiple clients with different codec preferences

## Building

```bash
cd examples/basic-server
go build
```

## Running

Start server with defaults (192kHz/24-bit stereo):

```bash
./basic-server
```

Start on a different port:

```bash
./basic-server -port 9000
```

Configure sample rate and channels:

```bash
./basic-server -rate 48000 -channels 2
```

Disable mDNS:

```bash
./basic-server -mdns=false
```

## Command-line options

- `-port` - Server port (default: 8927)
- `-name` - Server name (default: "Basic Server")
- `-rate` - Sample rate in Hz (default: 192000)
- `-channels` - Number of channels (default: 2)
- `-mdns` - Enable mDNS advertisement (default: true)

## Key features demonstrated

1. **Simple setup** - Just create a source and start the server
2. **Test tone generation** - Built-in test tone for easy testing
3. **Automatic client handling** - Clients are managed automatically
4. **Multi-codec support** - Automatically negotiates best codec per client
5. **mDNS discovery** - Clients can find the server automatically

## Code highlights

```go
// Create audio source (440Hz test tone)
source := resonate.NewTestTone(192000, 2)

// Create and start server
config := resonate.ServerConfig{
    Port:       8927,
    Name:       "My Server",
    Source:     source,
    EnableMDNS: true,
}

server, _ := resonate.NewServer(config)
server.Start()
```

## Testing

Start the server:

```bash
./basic-server
```

In another terminal, connect a player:

```bash
cd ../../cmd/resonate-player
go run . -server localhost:8927
```

You should hear a 440Hz tone (A4 note) playing.

## Next steps

- See `examples/custom-source/` for implementing custom audio sources (files, streams, etc.)
- Check out the server CLI (`cmd/resonate-server/`) for a full-featured server with TUI
- Modify the test tone frequency or add multiple frequencies for testing
