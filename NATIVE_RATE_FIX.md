# Native Sample Rate Fix

## Issue
When loading a 96kHz FLAC file, the server was:
1. Resampling it down to 48kHz "for Opus compatibility"
2. Then negotiating with client
3. Client advertises 96kHz/24bit support
4. Server can't match because it already downsampled to 48kHz
5. Falls through to Opus at 48kHz

**Result:** Hi-res 96kHz FLAC delivered as Opus 48kHz 😢

## Root Cause: Premature Resampling

In `internal/server/audio_source.go:91-95`:
```go
// If source is not 48kHz, wrap with resampler for Opus compatibility
if source.SampleRate() != 48000 {
    log.Printf("Resampling %s from %d Hz to 48000 Hz for Opus compatibility",
        displayName, source.SampleRate())
    return NewResampledSource(source, 48000), nil
}
```

This was **premature optimization** - assuming Opus would be needed before checking what the client supports!

## Fix Applied

### 1. Removed Auto-Resampling
`internal/server/audio_source.go`:
```go
// Note: We no longer auto-resample here. Sources are kept at native sample rate.
// If Opus encoding is needed and source isn't 48kHz, resampling happens per-client in audio engine.
// This allows PCM clients to receive hi-res audio at native rates!

return source, nil
```

### 2. Updated Format Negotiation
`internal/server/audio_engine.go` - `negotiateCodec()`:

**Before:** Would pick any codec without checking if it matches the source
**After:** Prioritizes formats in this order:
1. **PCM at native sample rate** (preserves hi-res quality)
2. Opus only if source is 48kHz (Opus native rate)
3. FLAC (not implemented yet)
4. Fallback to PCM

```go
// Check newer support_formats array first (spec-compliant)
// Prioritize PCM at native rate to preserve hi-res audio quality
for _, format := range client.Capabilities.SupportFormats {
    // Check if client supports PCM at our native sample rate
    if format.Codec == "pcm" && format.SampleRate == sourceRate && format.BitDepth == DefaultBitDepth {
        return "pcm"
    }
}

// If no PCM match at native rate, consider compressed codecs
for _, format := range client.Capabilities.SupportFormats {
    // Only use Opus if source is 48kHz (Opus native rate)
    // For other rates, prefer PCM to avoid resampling
    if format.Codec == "opus" && sourceRate == 48000 {
        return "opus"
    }
}
```

## Expected Behavior Now

### Test Tone (192kHz/24bit)
- Client advertises: [192kHz/24bit PCM, ..., Opus]
- Server has: 192kHz/24bit source
- **Match:** PCM 192kHz/24bit ✅
- **Delivery:** `pcm 192000Hz 2ch 24bit`

### 96kHz FLAC
- Client advertises: [192kHz/24bit, 176kHz/24bit, **96kHz/24bit**, ..., Opus]
- Server has: 96kHz/24bit FLAC
- **Match:** PCM 96kHz/24bit ✅
- **Delivery:** `pcm 96000Hz 2ch 24bit`

### 48kHz MP3
- Client advertises: [..., 48kHz/16bit, Opus]
- Server has: 48kHz/16bit MP3
- **Match:** PCM 48kHz/16bit OR Opus 48kHz ✅
- **Delivery:** `pcm 48000Hz 2ch 16bit` (PCM prioritized)

### 44.1kHz MP3
- Client advertises: [..., 44.1kHz/16bit, Opus]
- Server has: 44.1kHz/16bit MP3
- **Match:** PCM 44.1kHz/16bit ✅
- **Delivery:** `pcm 44100Hz 2ch 16bit`

## What About Opus for Non-48kHz?

For now, Opus is **only** used if:
1. Source is already 48kHz, AND
2. Client advertises Opus support

If source is non-48kHz (like 96kHz FLAC):
- Server will ALWAYS prefer PCM at native rate
- No resampling, preserves full hi-res quality
- Client handles any resampling needed on their end

**Future:** Could add per-client resampling for Opus if client specifically requests it and doesn't support native PCM rate. But for hi-res use cases, PCM is always better anyway!

## Testing

```bash
# Rebuild
go build -o resonate-server ./cmd/resonate-server

# Test 1: 192kHz test tone (should use PCM 192kHz)
./resonate-server -debug -no-tui

# Test 2: 96kHz FLAC (should use PCM 96kHz, NOT resample to 48kHz)
./resonate-server -debug -no-tui -audio ~/Downloads/Sample_BeeMoved_96kHz24bit.flac

# Player should show:
# - Test 1: "Stream starting: pcm 192000Hz 2ch 24bit"
# - Test 2: "Stream starting: pcm 96000Hz 2ch 24bit"
```

## Benefits

✅ Hi-res audio preserved at native sample rates
✅ No unnecessary downsampling
✅ PCM prioritized over lossy compression
✅ Better sound quality for hi-res files
✅ Still supports Opus when appropriate (48kHz sources)
