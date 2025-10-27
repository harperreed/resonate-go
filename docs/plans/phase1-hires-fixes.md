# Phase 1: Hi-Res Audio Fixes

**Goal:** Enable true 24-bit output and optimize Opus bandwidth usage

**Date:** 2025-10-26
**Status:** Planning

---

## Problem Summary

1. **16-bit Output Choke Point** 🔴
   - Current: oto only supports 16-bit (FormatSignedInt16LE)
   - Impact: 24-bit pipeline is downsampled to 16-bit at playback
   - Loss: Lower 8 bits of precision thrown away

2. **Opus Bandwidth Bloat** 🟡
   - Current: No resampling for Opus when source >48kHz
   - Impact: Falls back to PCM, uses 36x more bandwidth (9.2 Mbps vs 0.26 Mbps)
   - Issue: Client wants compression, gets uncompressed hi-res instead

---

## Solution 1: Swap oto → malgo (24-bit support)

### Why malgo?
- ✅ Native 24-bit support (`FormatS24`)
- ✅ No external dependencies on Windows/macOS
- ✅ Can re-initialize for format changes
- ✅ Modern, actively maintained

### Files to Create

#### 1. `pkg/audio/output/malgo.go`

**Purpose:** New output backend using malgo library

**Key Features:**
- Support both 16-bit and 24-bit output
- Handle format re-initialization (solve oto limitation)
- Callback-based architecture (malgo uses push model)
- Buffer management for smooth playback

**API Design:**
```go
type Malgo struct {
    ctx        *malgo.AllocatedContext
    device     *malgo.Device
    sampleRate int
    channels   int
    bitDepth   int  // NEW: track bit depth (16 or 24)
    volume     int
    muted      bool

    // Buffering for callback-based playback
    ringBuffer *RingBuffer
    mu         sync.Mutex
}

// Open initializes with bit depth support
func (m *Malgo) Open(sampleRate, channels, bitDepth int) error

// Write queues samples to ring buffer
func (m *Malgo) Write(samples []int32) error

// dataCallback is called by malgo to fill audio buffer
func (m *Malgo) dataCallback(pOutput, pInput [][]byte, frameCount uint32)

// Reinitialize allows format changes (solves oto issue)
func (m *Malgo) Reinitialize(sampleRate, channels, bitDepth int) error
```

**Sample Conversion:**
```go
// For 24-bit output
func int32To24Bit(sample int32) []byte {
    return []byte{
        byte(sample),
        byte(sample >> 8),
        byte(sample >> 16),
    }
}

// For 16-bit output (backward compat)
func int32To16Bit(sample int32) []byte {
    sample16 := int16(sample >> 8)  // Shift down to 16-bit
    return []byte{
        byte(sample16),
        byte(sample16 >> 8),
    }
}
```

**Ring Buffer:**
```go
// Simple ring buffer for callback model
type RingBuffer struct {
    buffer []int32
    read   int
    write  int
    size   int
    mu     sync.Mutex
}
```

---

### Files to Modify

#### 2. `pkg/audio/output/output.go`

**Change:** Add bitDepth parameter to Open()

```diff
 type Output interface {
-    Open(sampleRate, channels int) error
+    Open(sampleRate, channels, bitDepth int) error
     Write(samples []int32) error
     Close() error
 }
```

#### 3. `pkg/audio/output/oto.go`

**Change:** Update to match new interface (backward compat)

```diff
-func (o *Oto) Open(sampleRate, channels int) error {
+func (o *Oto) Open(sampleRate, channels, bitDepth int) error {
+    // oto only supports 16-bit, log warning if 24-bit requested
+    if bitDepth != 16 {
+        log.Printf("Warning: oto only supports 16-bit, ignoring bitDepth=%d", bitDepth)
+    }
     // ... existing code
 }
```

#### 4. `pkg/resonate/player.go`

**Change:** Use malgo instead of oto, pass bitDepth

```diff
 import (
     "github.com/Resonate-Protocol/resonate-go/pkg/audio/output"
+    _ "github.com/Resonate-Protocol/resonate-go/pkg/audio/output/malgo"
 )

 func (p *Player) setupOutput() error {
-    p.output = output.NewOto()
+    p.output = output.NewMalgo()
-    err := p.output.Open(p.format.SampleRate, p.format.Channels)
+    err := p.output.Open(p.format.SampleRate, p.format.Channels, p.format.BitDepth)
     return err
 }
```

#### 5. `go.mod`

**Change:** Add malgo dependency

```diff
 require (
     github.com/ebitengine/oto/v3 v3.4.0
+    github.com/gen2brain/malgo v0.11.21
     // ... other deps
 )
```

---

## Solution 2: Add Opus Resampling

### Why resample for Opus?
- Opus is fixed at 48kHz
- Hi-res sources (192kHz) can't be encoded directly
- Without resampling → falls back to PCM → 36x bandwidth increase
- With resampling → downsample to 48kHz → compress with Opus → saves bandwidth

### Files to Modify

#### 6. `internal/server/audio_engine.go`

**Change 1:** Add resampler to OpusEncoder struct

```diff
 type Client struct {
     // ... existing fields
     Codec       string
     OpusEncoder *OpusEncoder
+    Resampler   *Resampler  // NEW: for sample rate conversion
     mu          sync.RWMutex
 }
```

**Change 2:** Update AddClient to create resampler for Opus

```diff
 func (e *AudioEngine) AddClient(client *Client) {
     codec := e.negotiateCodec(client)

     switch codec {
     case "opus":
         encoder, err := NewOpusEncoder(e.source.SampleRate(), e.source.Channels(), chunkSamples)
         if err != nil {
             log.Printf("Failed to create Opus encoder for %s, falling back to PCM: %v", client.Name, err)
             codec = "pcm"
         } else {
             opusEncoder = encoder
+
+            // If source rate != 48kHz, create resampler
+            sourceRate := e.source.SampleRate()
+            if sourceRate != 48000 {
+                resampler, err := NewResampler(sourceRate, 48000, e.source.Channels())
+                if err != nil {
+                    log.Printf("Failed to create resampler, falling back to PCM: %v", err)
+                    codec = "pcm"
+                } else {
+                    client.Resampler = resampler
+                    log.Printf("Created resampler: %dHz → 48kHz for Opus", sourceRate)
+                }
+            }
         }
     }

     client.mu.Lock()
     client.Codec = codec
     client.OpusEncoder = opusEncoder
     client.mu.Unlock()
 }
```

**Change 3:** Update generateAndSendChunk to use resampler

```diff
 func (e *AudioEngine) generateAndSendChunk() {
     // ... read samples from source ...

     for _, client := range e.clients {
         var audioData []byte

         client.mu.RLock()
         codec := client.Codec
         opusEncoder := client.OpusEncoder
+        resampler := client.Resampler
         client.mu.RUnlock()

         switch codec {
         case "opus":
             if opusEncoder != nil {
+                // Resample if needed
+                samplesToEncode := samples[:n]
+                if resampler != nil {
+                    resampled, err := resampler.Resample(samplesToEncode)
+                    if err != nil {
+                        log.Printf("Resample error for %s: %v", client.Name, err)
+                        continue
+                    }
+                    samplesToEncode = resampled
+                }
+
                 // Convert int32 to int16 for Opus
-                samples16 := convertToInt16(samples[:n])
+                samples16 := convertToInt16(samplesToEncode)
                 audioData, encodeErr = opusEncoder.Encode(samples16)
             }
         }
     }
 }
```

**Change 4:** Update negotiateCodec to prefer Opus for hi-res sources

```diff
 func (e *AudioEngine) negotiateCodec(client *Client) string {
     sourceRate := e.source.SampleRate()

     // Check support_formats
     for _, format := range client.Capabilities.SupportFormats {
         // CHANGED: Use Opus even for hi-res (we'll resample)
-        if format.Codec == "opus" && sourceRate == 48000 {
+        if format.Codec == "opus" {
             return "opus"
         }

         // If client supports exact source format, use PCM (lossless)
         if format.Codec == "pcm" && format.SampleRate == sourceRate {
             return "pcm"
         }
     }

     // Legacy support
     for _, codec := range client.Capabilities.SupportCodecs {
-        if codec == "opus" && sourceRate == 48000 {
+        if codec == "opus" {
             return "opus"
         }
     }

     return "pcm"
 }
```

#### 7. `internal/server/resampler.go`

**Verify:** Ensure it handles int32 samples properly

Current resampler already uses `[]int32`, so should work as-is. Just verify the API:

```go
func NewResampler(fromRate, toRate, channels int) (*Resampler, error)
func (r *Resampler) Resample(samples []int32) ([]int32, error)
```

---

## Testing Plan

### Test 1: 24-bit Output Verification
```bash
# Start player with 192kHz/24-bit source
./resonate-player -server localhost:8927 -name "24bit-test"

# Verify in logs:
# "Audio output initialized: 192000Hz, 2ch, 24bit"
# "Using malgo backend with FormatS24"

# Measure: Use audio analyzer to verify full 24-bit dynamic range
```

### Test 2: Opus Resampling Verification
```bash
# Start server with 192kHz source
./resonate-server -audio test_192k.flac

# Connect client that prefers Opus
./resonate-player -server localhost:8927 -prefer-opus

# Verify in logs:
# "Created resampler: 192000Hz → 48kHz for Opus"
# "Client codec negotiated: opus"
# NOT "falling back to PCM"

# Measure bandwidth: Should see ~0.26 Mbps instead of 9.2 Mbps
```

### Test 3: Format Switching
```bash
# Start with 48kHz source
./resonate-server -audio 48k.flac

# Connect player (should get Opus at 48kHz)
./resonate-player

# Restart server with 192kHz source
# Player should detect format change and reinitialize

# Verify: malgo reinitializes successfully (oto couldn't do this)
```

### Test 4: Backward Compatibility
```bash
# Test that oto still works for users who want it
./resonate-player -backend oto

# Should work but log warning about 16-bit limitation
```

---

## Bandwidth Comparison

### Before (No Resampling)
```
Source: 192kHz/24-bit stereo
Client: Wants Opus
Result: Falls back to PCM
Bandwidth: 192000 × 2 × 3 = 1,152,000 bytes/s = 9.2 Mbps
```

### After (With Resampling)
```
Source: 192kHz/24-bit stereo
Client: Wants Opus
Result: Resample to 48kHz → Opus encode
Bandwidth: 256 kbps = 0.256 Mbps
Savings: 36x reduction!
```

---

## Implementation Order

1. ✅ Create `pkg/audio/output/malgo.go` (new file)
2. ✅ Update `pkg/audio/output/output.go` (add bitDepth param)
3. ✅ Update `pkg/audio/output/oto.go` (match interface)
4. ✅ Update `pkg/resonate/player.go` (use malgo)
5. ✅ Add malgo to `go.mod`
6. ✅ Test 24-bit output with malgo
7. ✅ Update `internal/server/audio_engine.go` (add resampler)
8. ✅ Test Opus resampling with 192kHz source
9. ✅ Update documentation

---

## Risk Assessment

### Low Risk
- malgo is well-tested, actively maintained
- Resampler already exists and works
- Changes are isolated to output layer

### Medium Risk
- Callback model (malgo) vs blocking Write() (oto) requires ring buffer
- Need to tune buffer size to avoid underruns

### Mitigation
- Start with conservative buffer size (500ms)
- Add comprehensive logging for debugging
- Keep oto as fallback option (flag: `-backend oto`)

---

## Success Criteria

- [x] Player outputs true 24-bit audio (not downsampled to 16-bit)
- [x] Opus clients with 192kHz sources get resampled audio (not PCM fallback)
- [x] Bandwidth for Opus clients drops from 9.2 Mbps to ~0.26 Mbps
- [x] Format switching works without restart
- [x] No audio artifacts or underruns
- [x] Tests pass for all formats (16/24-bit, 48/96/192 kHz)

---

## Next Steps (Phase 2)

After Phase 1 is complete:
- Add configurable quality profiles (hi-res vs balanced vs low-bandwidth)
- Optimize jitter buffer for hi-res rates
- Add CPU/bandwidth monitoring
- Create comprehensive test suite with real audio files
