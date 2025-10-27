# Format Negotiation Fix

## Issue
User reported seeing "Format: pcm 48000Hz Stereo 16-bit" instead of the expected "Format: pcm 192000Hz Stereo 24-bit" when connecting to the server.

## Root Cause
The server's `negotiateCodec()` function was only checking for codec type (pcm/opus/flac) but **not matching the sample rate and bit depth** from the client's advertised `SupportFormats` array.

### Previous Behavior
```go
// Only checked codec type, ignored format details
for _, format := range client.Capabilities.SupportFormats {
    if format.Codec == "opus" {
        return "opus"
    }
    // ...
}
```

This meant:
- Server would pick "pcm" codec
- But then use its own `DefaultSampleRate` and `DefaultBitDepth`
- Never checked if client actually supports those values
- Could result in format mismatch

## Fix Applied
Updated `negotiateCodec()` to check **full format specification**:

```go
// Check newer support_formats array first (spec-compliant)
// Pick first format that matches what we can provide
for _, format := range client.Capabilities.SupportFormats {
    // Check if we can provide this format
    if format.Codec == "pcm" && format.SampleRate == e.source.SampleRate() && format.BitDepth == DefaultBitDepth {
        return "pcm"
    }
    // ... opus/flac checks
}
```

### How It Works Now
1. Server iterates through client's `SupportFormats` (in order of preference)
2. For PCM, checks if server can provide the exact sample rate AND bit depth
3. Returns "pcm" codec only if there's a full format match
4. Falls back to opus/flac if no PCM match
5. Final fallback to "pcm" if no codecs match

## Client Advertisement (Already Correct)
The player was already advertising correctly:
```go
SupportFormats: []protocol.AudioFormat{
    {Codec: "pcm", Channels: 2, SampleRate: 192000, BitDepth: 24},  // First!
    {Codec: "pcm", Channels: 2, SampleRate: 176400, BitDepth: 24},
    {Codec: "pcm", Channels: 2, SampleRate: 96000, BitDepth: 24},
    {Codec: "pcm", Channels: 2, SampleRate: 88200, BitDepth: 24},
    {Codec: "pcm", Channels: 2, SampleRate: 48000, BitDepth: 16},
    {Codec: "pcm", Channels: 2, SampleRate: 44100, BitDepth: 16},
    {Codec: "opus", Channels: 2, SampleRate: 48000, BitDepth: 16},
}
```

## Expected Result
With test tone source (192kHz/24-bit):
- Client advertises: [192kHz/24bit, 176.4kHz/24bit, 96kHz/24bit, ..., 48kHz/16bit]
- Server has: 192kHz/24bit source
- Match found: First format (192kHz/24bit PCM)
- Client receives: `stream/start` with 192kHz/24bit

## Improved Logging
Added detailed format logging:
```
Audio engine: added client Mac with codec pcm (format: 192000Hz/24bit/2ch)
```

## Testing
To verify fix works:
1. Stop any running server/player processes
2. Rebuild server: `go build -o resonate-server ./cmd/resonate-server`
3. Start server: `./resonate-server -debug`
4. Start player: `./resonate-player -stream-logs`
5. Check player log for: `Stream starting: pcm 192000Hz 2ch 24bit`
6. Check server log for: `Audio engine: added client ... (format: 192000Hz/24bit/2ch)`

## Potential Issues Debugged
If still seeing 48kHz/16-bit:
- Check you're using the NEW binaries (rebuild both)
- Check server logs show "192000Hz/24bit" in format line
- If connected to Music Assistant instead of our server, it may only support 48kHz/16bit
- Check player capabilities are being sent correctly (enable debug logging)
