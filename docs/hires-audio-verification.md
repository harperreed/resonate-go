# Hi-Res Audio Verification Report

**Status:** 🟡 Needs Testing
**Version:** v0.9.0
**Date:** 2025-10-26

## Executive Summary

resonate-go implements **full hi-res audio support** with 24-bit depth and sample rates up to 192kHz. However, **compatibility with Music Assistant and other Resonate servers needs verification** due to potential codec negotiation differences.

**Supported Hi-Res Formats:**
- ✅ 192kHz/24-bit PCM (lossless)
- ✅ 176.4kHz/24-bit PCM (lossless)
- ✅ 96kHz/24-bit PCM (lossless)
- ✅ 88.2kHz/24-bit PCM (lossless)
- ⚠️  Opus encoding requires 48kHz (no automatic resampling)

---

## Current Implementation Analysis

### 1. Audio Pipeline Capabilities

**Sample Rate Support** (`pkg/resonate/player.go:170-185`):
```go
SupportFormats: []protocol.AudioFormat{
    {Codec: "pcm", Channels: 2, SampleRate: 192000, BitDepth: 24},  // Hi-res
    {Codec: "pcm", Channels: 2, SampleRate: 176400, BitDepth: 24},  // Hi-res
    {Codec: "pcm", Channels: 2, SampleRate: 96000, BitDepth: 24},   // Hi-res
    {Codec: "pcm", Channels: 2, SampleRate: 88200, BitDepth: 24},   // Hi-res
    {Codec: "pcm", Channels: 2, SampleRate: 48000, BitDepth: 16},   // CD quality
    {Codec: "pcm", Channels: 2, SampleRate: 44100, BitDepth: 16},   // CD quality
    {Codec: "opus", Channels: 2, SampleRate: 48000, BitDepth: 16},  // Compressed
},
```

**Bit Depth Pipeline:**
- Server default: 24-bit (`pkg/resonate/server.go:32`)
- Player supports: 16-bit and 24-bit (`pkg/audio/decode/pcm.go:23-24`)
- Internal representation: int32 for full 24-bit range (`pkg/audio/types.go:11-12`)
- Audio output: Converts to 16-bit for device (`pkg/audio/output/oto.go:100-103`)

**Sample Rate Pipeline:**
- Source: Any rate (192kHz default for TestTone)
- Encoder: Opus requires 48kHz, PCM supports any rate
- Decoder: PCM supports any rate, Opus fixed at 48kHz
- Output: oto library accepts any sample rate

---

## 2. Codec Negotiation

### Server-Side Logic (`internal/server/audio_engine.go:200-230`)

```go
func (e *AudioEngine) negotiateCodec(client *Client) string {
    sourceRate := e.source.SampleRate()

    // Check if client advertised support for exact source format
    for _, format := range caps.SupportFormats {
        // If source is 48kHz PCM and client wants Opus, prefer Opus
        if format.Codec == "opus" && sourceRate == 48000 {
            return "opus"
        }

        // If client supports exact source format, use PCM
        if format.Codec == "pcm" && format.SampleRate == sourceRate && format.BitDepth == DefaultBitDepth {
            return "pcm"
        }
    }

    // FALLBACK: If no exact match, try legacy fields
    // This is for backward compatibility with Music Assistant
    if codec == "opus" && sourceRate == 48000 {
        return "opus"
    }

    // Final fallback: PCM at source rate
    return "pcm"
}
```

### Issues Identified

**🔴 Critical: No Automatic Resampling for Opus**

When source is 192kHz and client supports Opus:
- Current behavior: Falls back to PCM at 192kHz
- Expected behavior (Music Assistant?): Resample 192kHz → 48kHz, encode to Opus
- Impact: Client receives uncompressed PCM instead of Opus (4x bandwidth increase)

**Test Case:**
```
Source: 192kHz/24-bit test tone
Client: Advertises Opus support
Expected: Server resamples to 48kHz and sends Opus
Actual: Server sends 192kHz PCM (no resampling)
```

**🟡 Warning: Oto Output Limitation**

The audio output uses oto library which:
- Only supports 16-bit output (`oto.FormatSignedInt16LE`)
- Accepts any sample rate but resamples internally to device native rate
- **24-bit samples are downsampled to 16-bit** (`pkg/audio/output/oto.go:100-103`)

This means the full hi-res pipeline (24-bit) is preserved until final output, but playback is 16-bit.

**🟡 Warning: Oto Context Cannot Be Reinitialized**

From `pkg/audio/output/oto.go:51-56`:
```go
// If format changed, we can't reinitialize oto (it only allows one context per process)
// Log a warning but continue using the existing context
if o.otoCtx != nil {
    log.Printf("Warning: format change detected (%dHz %dch -> %dHz %dch) but oto doesn't support reinitialization. Continuing with existing context.")
    return nil
}
```

Impact: If stream format changes mid-session, player continues with original format.

---

## 3. Bandwidth Analysis

### PCM Bandwidth (192kHz/24-bit stereo):

```
Sample rate:   192,000 Hz
Channels:      2
Bit depth:     24 bits = 3 bytes
Bytes/second:  192000 × 2 × 3 = 1,152,000 bytes/s = 9.216 Mbps
```

### Opus Bandwidth (48kHz/16-bit stereo @ 256kbps):

```
Bitrate:       256 kbps (configurable, set in opus_encoder.go:31)
Bytes/second:  32,000 bytes/s = 0.256 Mbps
Compression:   36x smaller than 192kHz PCM!
```

### Impact of Missing Resampling

If Music Assistant sends 192kHz source and expects Opus:
- **Without resampling:** 9.216 Mbps per client (current)
- **With resampling:** 0.256 Mbps per client (ideal)
- **Difference:** 36x higher bandwidth usage!

For 5 simultaneous clients:
- PCM: 46 Mbps
- Opus: 1.3 Mbps

---

## 4. Compatibility Testing Plan

### Test Matrix

| Source Format | Client Codec | Expected Behavior | Status |
|--------------|--------------|-------------------|--------|
| 192kHz/24-bit PCM | PCM 192kHz | Direct PCM stream | ✅ Should work |
| 192kHz/24-bit PCM | Opus 48kHz | Resample + Opus | ⚠️  **Falls back to PCM** |
| 96kHz/24-bit PCM | PCM 96kHz | Direct PCM stream | ✅ Should work |
| 96kHz/24-bit PCM | Opus 48kHz | Resample + Opus | ⚠️  **Falls back to PCM** |
| 48kHz/16-bit PCM | Opus 48kHz | Direct Opus encode | ✅ Works |
| 48kHz/16-bit PCM | PCM 48kHz | Direct PCM stream | ✅ Works |

### Required Tests

**Test 1: Music Assistant Compatibility**
```bash
# Start resonate-go server with 192kHz source
./resonate-server --audio test_192khz_24bit.flac

# Connect Music Assistant player
# Expected: MA requests Opus, server resamples and encodes
# Actual: ???
```

**Test 2: Multi-Room Sync at Hi-Res**
```bash
# Start server with 192kHz PCM
./resonate-server --audio hires_test.flac

# Start 5 players
for i in {1..5}; do
  ./resonate-player --name "Player-$i" &
done

# Verify:
# - All players receive 192kHz PCM
# - Sync stays within 10ms
# - No dropped frames
# - Network bandwidth is acceptable
```

**Test 3: Sample Rate Switching**
```bash
# Start server with 48kHz source
./resonate-server --audio 48khz.flac

# Connect player (should get Opus)
./resonate-player --name "Test"

# Switch to 192kHz source on server
# (Would require server restart currently)

# Expected: Player handles format change gracefully
# Actual: Oto can't reinitialize - player continues with 48kHz
```

**Test 4: Bit Depth Verification**
```bash
# Generate 24-bit test tone with known frequency spectrum
./resonate-server --audio 24bit_sweep.wav

# Record output from player
# Analyze frequency spectrum
# Verify: Full 24-bit dynamic range preserved until output stage
```

---

## 5. Known Issues & Limitations

### Issue 1: No Automatic Resampling for Opus 🔴

**Location:** `internal/server/audio_engine.go:209, 219`

**Current Code:**
```go
if format.Codec == "opus" && sourceRate == 48000 {
    return "opus"
}
```

**Problem:** Only uses Opus if source is already 48kHz. Doesn't resample hi-res sources.

**Fix Required:**
```go
if format.Codec == "opus" {
    // Resample to 48kHz if needed
    if sourceRate != 48000 {
        // Create resampler from sourceRate to 48kHz
        encoder.resampler = NewResampler(sourceRate, 48000, channels)
    }
    return "opus"
}
```

**Impact:** High bandwidth usage for Opus clients with hi-res sources.

---

### Issue 2: 16-bit Output Limitation 🟡

**Location:** `pkg/audio/output/oto.go:62`

**Current Code:**
```go
Format: oto.FormatSignedInt16LE,
```

**Problem:** oto library only supports 16-bit output format. Full 24-bit resolution is lost at playback.

**Potential Fix:**
- Switch to cpal library which supports 24-bit (i32 format)
- Or accept 16-bit as "good enough" for most audio devices

**Impact:** Hi-res audio benefits (dynamic range) not fully realized at speaker output.

---

### Issue 3: Format Switching Limitation 🟡

**Location:** `pkg/audio/output/oto.go:53-56`

**Problem:** oto context can only be initialized once per process. Format changes require restart.

**Workaround:** Document that format changes require player restart.

**Impact:** Poor UX if server changes stream format mid-session.

---

## 6. Recommendations

### Immediate Actions (v0.9.x)

1. **🔴 Priority 1: Add Resampling to Opus Path**
   - Implement automatic resampling in `audio_engine.go`
   - Use existing `pkg/audio/resample.Resampler`
   - Test with 192kHz → 48kHz Opus encoding
   - Verify bandwidth reduction

2. **🟡 Priority 2: Test with Music Assistant**
   - Deploy resonate-go server with MA
   - Verify codec negotiation compatibility
   - Test hi-res audio file playback
   - Document any protocol differences

3. **🟡 Priority 3: Document Bit Depth Behavior**
   - Add to README: "24-bit pipeline, 16-bit output"
   - Explain why (oto library limitation)
   - Note that most audio devices are 16-bit anyway

4. **🟢 Priority 3: Add Integration Tests**
   - Create test suite for hi-res formats
   - Verify sample rate handling
   - Check codec negotiation logic
   - Measure bandwidth usage

### Future Enhancements (v1.0+)

1. **Consider cpal for 24-bit Output**
   - Replace oto with cpal
   - Support true 24-bit playback
   - More control over audio device selection

2. **Dynamic Format Switching**
   - Handle format changes without restart
   - Reinitialize audio output when needed
   - Smooth transitions between formats

3. **Configurable Quality Profiles**
   - "Low bandwidth" → Force Opus with resampling
   - "Balanced" → Opus for >48kHz, PCM for ≤48kHz
   - "Hi-Res" → Always use PCM at source rate

---

## 7. Verification Checklist

Before marking v1.0.0 as ready:

- [ ] Verify 192kHz/24-bit PCM playback (end-to-end)
- [ ] Verify 96kHz/24-bit PCM playback
- [ ] Test automatic resampling for Opus (after implementing)
- [ ] Test with Music Assistant server
- [ ] Test with other Resonate protocol implementations
- [ ] Measure multi-room sync accuracy at hi-res
- [ ] Document bandwidth requirements
- [ ] Add warning about 16-bit output limitation
- [ ] Create hi-res test audio files repository
- [ ] Profile CPU usage with hi-res streams
- [ ] Test with 5+ simultaneous hi-res clients
- [ ] Verify no audio artifacts at high sample rates
- [ ] Check for buffer underruns at 192kHz
- [ ] Test format negotiation with all supported rates

---

## 8. Test Resources Needed

**Audio Test Files:**
- `test_192khz_24bit.flac` - Full hi-res test
- `test_96khz_24bit.flac` - Mid-tier hi-res
- `test_48khz_24bit.flac` - Baseline quality
- `sweep_24bit.wav` - Frequency sweep for bit depth verification
- `dynamic_range_test.wav` - Test 24-bit dynamic range

**Test Equipment:**
- Music Assistant server instance
- Multiple player instances (5+)
- Network bandwidth monitor
- Audio spectrum analyzer
- Sync measurement tools

---

## 9. Conclusion

resonate-go has **excellent foundational support for hi-res audio** with a clean 24-bit pipeline and support for sample rates up to 192kHz. However, **critical compatibility testing is required** to ensure:

1. **Codec negotiation works with Music Assistant** (especially Opus fallback)
2. **Bandwidth optimization** through automatic resampling to Opus
3. **Multi-room sync accuracy** maintained at hi-res rates
4. **No audio quality degradation** in the pipeline

The biggest concern is the **lack of automatic resampling for Opus encoding** which could cause 36x higher bandwidth usage when Music Assistant expects Opus but receives PCM.

**Status: Ready for Testing** 🧪
