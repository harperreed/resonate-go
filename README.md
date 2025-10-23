# Resonate Go Player

A Resonate Protocol player implementation in Go.

## Features

- mDNS service discovery (client and server initiated)
- Multi-codec support (Opus, FLAC, PCM)
- Precise clock synchronization for multi-room audio
- Interactive terminal UI
- Software volume control

## Installation

```bash
go build -o resonate-player
```

## Usage

```bash
./resonate-player --name "Living Room"
```

## Options

- `--server` - Manual server address (skip mDNS)
- `--port` - Port for mDNS advertisement (default: 8927)
- `--name` - Player friendly name
- `--buffer-ms` - Jitter buffer size (default: 150ms)
- `--log-file` - Log file path
- `--debug` - Enable debug logging

## Protocol

Implements the [Resonate Protocol](https://github.com/Resonate-Protocol/spec).
