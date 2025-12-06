# Basic Player Example

This example demonstrates how to create a simple Sendspin player that connects to a server and plays audio.

## What it does

- Connects to a Sendspin server
- Starts audio playback with automatic clock synchronization
- Displays metadata (title, artist, album) when received
- Shows playback status and statistics
- Supports volume control and mute

## Building

```bash
cd examples/basic-player
go build
```

## Running

Connect to a local server:

```bash
./basic-player
```

Connect to a specific server:

```bash
./basic-player -server 192.168.1.100:8927
```

Set custom player name and volume:

```bash
./basic-player -name "Living Room" -volume 80
```

## Command-line options

- `-server` - Server address (default: localhost:8927)
- `-name` - Player name (default: "Basic Player")
- `-volume` - Initial volume 0-100 (default: 80)

## Key features demonstrated

1. **Simple configuration** - Just specify server address and player name
2. **Callbacks** - Handle metadata, state changes, and errors
3. **Automatic sync** - Clock synchronization happens automatically
4. **Status monitoring** - Get real-time playback statistics

## Code highlights

```go
// Create player with configuration
config := sendspin.PlayerConfig{
    ServerAddr: "localhost:8927",
    PlayerName: "Living Room",
    Volume:     80,
    OnMetadata: func(meta sendspin.Metadata) {
        log.Printf("Now playing: %s - %s", meta.Artist, meta.Title)
    },
}

player, _ := sendspin.NewPlayer(config)
player.Connect()
player.Play()
```

## Next steps

- See `examples/custom-source/` for custom audio source implementation
- Check out the player CLI (root main.go) for a full-featured TUI player
