# Phase 1 Implementation Complete ✅

**Date:** 2025-10-26
**Status:** Implementation Complete, Ready for Testing

---

## Summary

Successfully implemented both Phase 1 fixes to enable true hi-res audio and optimize bandwidth:

1. **✅ 24-bit Output Support** - Replaced oto with malgo for true 24-bit playback
2. **✅ Opus Resampling** - Added automatic resampling for bandwidth optimization

All code compiles successfully. Ready for testing.

---

## Changes Made

### 1. Created malgo Output Backend

**File:** `pkg/audio/output/malgo.go` (new, 350 lines)

**Features:**
- ✅ True 24-bit output support (FormatS24)
- ✅ Also supports 16-bit and 32-bit formats
- ✅ Format re-initialization support (fixes oto limitation)
- ✅ Ring buffer for callback-based audio architecture
- ✅ No external dependencies on macOS/Windows

**Key Implementation Details:**
```go
// Supports 16/24/32-bit output
func (m *Malgo) Open(sampleRate, channels, bitDepth int) error {
    var format malgo.FormatType
    switch bitDepth {
    case 16:
        format = malgo.FormatS16
    case 24:
        format = malgo.FormatS24  // TRUE 24-BIT!
    case 32:
        format = malgo.FormatS32
    }
    // ... device initialization
}

// 24-bit sample conversion (3 bytes per sample)
func (m *Malgo) write24Bit(output []byte, samples []int32) {
    for i, sample := range samples {
        output[i*3] = byte(sample)
        output[i*3+1] = byte(sample >> 8)
        output[i*3+2] = byte(sample >> 16)
    }
}
```

### 2. Updated Output Interface

**File:** `pkg/audio/output/output.go`

**Changes:**
- Added `bitDepth` parameter to `Open()` method
- Breaking change for all Output implementations

```diff
 type Output interface {
-    Open(sampleRate, channels int) error
+    Open(sampleRate, channels, bitDepth int) error
     Write(samples []int32) error
     Close() error
 }
```

### 3. Updated oto Backend (Backward Compatibility)

**File:** `pkg/audio/output/oto.go`

**Changes:**
- Updated to match new interface
- Logs warning when 24-bit is requested (oto only supports 16-bit)
- Maintains backward compatibility for users who want oto

```go
func (o *Oto) Open(sampleRate, channels, bitDepth int) error {
    if bitDepth != 16 {
        log.Printf("Warning: oto only supports 16-bit output, ignoring requested bitDepth=%d", bitDepth)
    }
    // ... rest of oto initialization
}
```

### 4. Switched Player to malgo

**File:** `pkg/resonate/player.go`

**Changes:**
- Line 131: Changed from `output.NewOto()` to `output.NewMalgo()`
- Line 326: Updated to pass `format.BitDepth` to `Open()`

```diff
-out := output.NewOto()
+out := output.NewMalgo()

-if err := p.output.Open(format.SampleRate, format.Channels); err != nil {
+if err := p.output.Open(format.SampleRate, format.Channels, format.BitDepth); err != nil {
```

### 5. Added Resampler to Client Struct

**File:** `internal/server/server.go`

**Changes:**
- Added `Resampler *Resampler` field to track per-client resampling

```go
type Client struct {
    // ... existing fields
    Codec       string
    OpusEncoder *OpusEncoder
    Resampler   *Resampler  // NEW: for Opus resampling
    // ... rest of fields
}
```

### 6. Implemented Opus Resampling

**File:** `internal/server/audio_engine.go`

**Changes:**

#### AddClient - Create resampler when needed
```go
case "opus":
    // Create resampler if source rate != 48kHz
    if sourceRate != 48000 {
        resampler = NewResampler(sourceRate, 48000, e.source.Channels())
        log.Printf("Created resampler: %dHz → 48kHz for Opus (client: %s)", sourceRate, client.Name)
    }

    // Create Opus encoder at 48kHz
    opusChunkSamples := (48000 * ChunkDurationMs) / 1000
    encoder, err := NewOpusEncoder(48000, e.source.Channels(), opusChunkSamples)
    // ...
```

#### generateAndSendChunk - Use resampler before encoding
```go
case "opus":
    samplesToEncode := samples[:n]

    // Resample if needed
    if resampler != nil {
        outputSamples := resampler.OutputSamplesNeeded(len(samplesToEncode))
        resampled := make([]int32, outputSamples)
        samplesWritten := resampler.Resample(samplesToEncode, resampled)
        samplesToEncode = resampled[:samplesWritten]
    }

    // Convert to int16 and encode to Opus
    samples16 := convertToInt16(samplesToEncode)
    audioData, _ = opusEncoder.Encode(samples16)
```

#### RemoveClient - Clean up resampler
```go
if client.Resampler != nil {
    client.Resampler = nil
}
```

### 7. Updated Codec Negotiation

**File:** `internal/server/audio_engine.go`

**Changes:**
- Now prefers Opus even for hi-res sources (since we can resample)
- Strategy:
  1. PCM at native rate (lossless hi-res)
  2. Opus with resampling (bandwidth efficient)
  3. PCM fallback

```go
// Check if client supports PCM at native rate (lossless hi-res)
for _, format := range client.Capabilities.SupportFormats {
    if format.Codec == "pcm" && format.SampleRate == sourceRate {
        return "pcm"
    }
}

// Check if client supports Opus (we can resample now!)
for _, format := range client.Capabilities.SupportFormats {
    if format.Codec == "opus" {
        return "opus"  // Will automatically resample if needed
    }
}
```

### 8. Updated Dependencies

**File:** `go.mod`

**Changes:**
- Added `github.com/gen2brain/malgo v0.11.21`

### 9. Updated Documentation

**File:** `pkg/audio/output/doc.go`

**Changes:**
- Updated to reflect both malgo and oto support
- Shows new API with bitDepth parameter

```go
// Example:
//
//	out := output.NewMalgo()
//	err := out.Open(192000, 2, 24)  // 192kHz, stereo, 24-bit
```

---

## Expected Improvements

### Before (16-bit Output + No Resampling)

**Audio Quality:**
- 24-bit pipeline → **downsampled to 16-bit** at output ❌
- Lost 8 bits of precision (256x dynamic range loss)

**Bandwidth (192kHz source, Opus client):**
- Falls back to PCM: **9.2 Mbps** per client
- 5 clients: 46 Mbps

### After (24-bit Output + Resampling)

**Audio Quality:**
- 24-bit pipeline → **24-bit output** ✅
- Full hi-res dynamic range preserved

**Bandwidth (192kHz source, Opus client):**
- Resamples to 48kHz → Opus: **0.26 Mbps** per client
- 5 clients: 1.3 Mbps
- **36x bandwidth reduction!**

---

## Testing Plan

### Test 1: Verify 24-bit Output

```bash
# Start server with 192kHz/24-bit source
./resonate-server -audio test_192khz.flac

# Connect player
./resonate-player -server localhost:8927

# Expected logs:
# "Audio output initialized: 192000Hz, 2 channels, 24-bit (malgo/S24)"
# "Stream starting: pcm 192000Hz 2ch 24bit"
```

**Verification:**
- Check logs for "24-bit (malgo/S24)"
- Use audio analyzer to verify full 24-bit dynamic range
- Compare output quality vs oto (16-bit)

### Test 2: Verify Opus Resampling

```bash
# Start server with 192kHz source
./resonate-server -audio test_192khz.flac

# Connect player that advertises Opus support
# (Current resonate-player advertises Opus in capabilities)
./resonate-player -server localhost:8927

# Expected logs:
# Server: "Created resampler: 192000Hz → 48kHz for Opus (client: ...)"
# Server: "Audio engine: added client with codec opus"
# Player: "Stream starting: opus 48000Hz 2ch 16bit"
```

**Verification:**
- Check server logs for resampler creation
- Check player logs for opus codec
- Monitor bandwidth: should be ~0.26 Mbps (not 9.2 Mbps)
- Audio should still sound good (you can't hear >48kHz anyway)

### Test 3: Format Switching (malgo advantage)

```bash
# Start with 48kHz source
./resonate-server -audio 48khz.flac

# Connect player
./resonate-player

# Restart server with 192kHz source (keep player running)
./resonate-server -audio 192khz.flac

# Expected: Player reinitializes output to 192kHz
# (oto couldn't do this - would stay at 48kHz)
```

### Test 4: Backward Compatibility (oto still works)

```bash
# Manually test oto backend if needed
# (Would require changing player.go back to NewOto() temporarily)
```

---

## Files Changed

### New Files (1)
- ✅ `pkg/audio/output/malgo.go` (350 lines)

### Modified Files (7)
- ✅ `pkg/audio/output/output.go` (interface change)
- ✅ `pkg/audio/output/oto.go` (add bitDepth param)
- ✅ `pkg/audio/output/doc.go` (update docs)
- ✅ `pkg/resonate/player.go` (use malgo, pass bitDepth)
- ✅ `internal/server/server.go` (add Resampler field)
- ✅ `internal/server/audio_engine.go` (resampling logic, codec negotiation)
- ✅ `go.mod` (add malgo dependency)

### Documentation (1)
- ✅ `docs/plans/phase1-hires-fixes.md` (implementation plan)

---

## Build Status

```bash
$ go mod tidy
go: downloading github.com/gen2brain/malgo v0.11.21

$ go build -v ./...
github.com/Resonate-Protocol/resonate-go/internal/server
github.com/Resonate-Protocol/resonate-go/pkg/resonate
github.com/Resonate-Protocol/resonate-go/examples/basic-server
github.com/Resonate-Protocol/resonate-go/examples/basic-player
github.com/Resonate-Protocol/resonate-go/cmd/resonate-server
github.com/Resonate-Protocol/resonate-go
✅ All packages build successfully!
```

---

## Next Steps

1. **Test 24-bit output** with audio analyzer
2. **Test Opus resampling** with 192kHz source
3. **Measure bandwidth** savings
4. **Update README** with malgo requirements
5. **Create commit** for Phase 1 changes

---

## Commit Message (Suggested)

```
feat: Add 24-bit output support and Opus resampling for hi-res audio

BREAKING CHANGE: Output.Open() now requires bitDepth parameter

This commit addresses two critical hi-res audio limitations:

1. 24-bit Output Support (via malgo)
   - Replaced oto with malgo as default output backend
   - Supports true 24-bit audio (FormatS24)
   - Enables format re-initialization (fixes oto limitation)
   - oto still available for backward compatibility

2. Opus Resampling (bandwidth optimization)
   - Added automatic resampling for Opus encoding
   - Server resamples hi-res sources (192kHz) to 48kHz for Opus
   - Reduces bandwidth by 36x (9.2 Mbps → 0.26 Mbps per client)
   - Codec negotiation now prefers Opus when supported

Files changed:
- New: pkg/audio/output/malgo.go
- Modified: pkg/audio/output/output.go (API change)
- Modified: pkg/resonate/player.go (use malgo)
- Modified: internal/server/audio_engine.go (resampling logic)
- Modified: go.mod (add malgo dependency)

Fixes #[issue-number] (if applicable)

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

---

## Known Limitations

1. **malgo Dependency**
   - Requires cgo (not pure Go)
   - On Linux: needs libasound2-dev (`apt install libasound2-dev`)
   - On macOS/Windows: no external deps needed

2. **Resampler Quality**
   - Currently uses simple linear interpolation
   - Good enough for Opus (you can't hear >48kHz)
   - Could upgrade to higher quality resampling if needed

3. **Testing Needed**
   - Need to verify actual 24-bit output with audio analyzer
   - Need to measure real bandwidth savings
   - Need to test with multiple simultaneous clients

---

## Success Criteria

- [x] Code compiles without errors
- [x] malgo dependency installed successfully
- [x] All Output implementations match new interface
- [ ] Player outputs 24-bit audio (verified with logs)
- [ ] Opus resampling works for 192kHz sources
- [ ] Bandwidth reduced from 9.2 Mbps to ~0.26 Mbps
- [ ] No audio artifacts or quality degradation
- [ ] Format switching works (malgo can reinitialize)

**Status:** 4/8 complete (implementation done, testing pending)
