# Code Review Fixes - 24-bit Audio Implementation

## Date: 2025-10-25

After implementing the 24-bit audio pipeline, I performed a careful code review with "fresh eyes" and found several issues that have been fixed.

## Issues Found and Fixed

### 1. ✅ CRITICAL: Integer Overflow in Volume Control
**File:** `internal/player/output.go`
**Location:** `applyVolume()` function

**Problem:**
```go
result[i] = int32(float64(sample) * multiplier)
```
When applying volume to samples, there was no clipping protection. Samples at max 24-bit range (±8,388,607) multiplied by volume could overflow int32.

**Fix:**
```go
scaled := int64(float64(sample) * multiplier)

// Clamp to 24-bit range to prevent overflow
if scaled > audio.Max24Bit {
    scaled = audio.Max24Bit
} else if scaled < audio.Min24Bit {
    scaled = audio.Min24Bit
}

result[i] = int32(scaled)
```

**Impact:** Prevents audio distortion and crashes from integer overflow when volume is applied.

---

### 2. ✅ Improved Code Clarity: Better Comments
**File:** `internal/server/audio_source.go`
**Locations:** MP3Source, HTTPMP3Source, FFmpegSource Read methods

**Problem:**
Comments were confusing, referring to "int16 = 2 bytes" next to code dealing with int32 arrays.

**Before:**
```go
numBytes := len(samples) * 2 // int16 = 2 bytes
```

**After:**
```go
// Read bytes from decoder (MP3 decoder outputs int16 = 2 bytes per sample)
numBytes := len(samples) * 2
```

**Also added detailed comments for the conversion:**
```go
// Left-shift by 8 to convert 16-bit range to 24-bit range
// Example: 32767 (max 16-bit) << 8 = 8388352 (near max 24-bit 8388607)
samples[i] = int32(sample16) << 8
```

**Impact:** Makes code intention clearer for future maintainers.

---

### 3. ✅ Code Organization: Added 24-bit Constants
**File:** `internal/audio/types.go`

**Problem:**
24-bit range limits were hardcoded as magic numbers throughout the codebase.

**Fix:**
```go
const (
    // 24-bit audio range constants
    Max24Bit = 8388607   // 2^23 - 1
    Min24Bit = -8388608  // -2^23
)
```

**Updated usages:**
- `internal/player/output.go` - Volume clipping now uses `audio.Max24Bit` and `audio.Min24Bit`
- `internal/server/test_tone_source.go` - Test tone generation uses `max24bit` constant

**Impact:** Single source of truth for 24-bit range, easier to maintain.

---

### 4. ✅ Improved Test Tone Comments
**File:** `internal/server/test_tone_source.go`

**Before:**
```go
pcmValue := int32(sample * 8388607.0 * 0.5) // 50% volume
```

**After:**
```go
// Convert to 24-bit PCM (using int32)
// Scale to 24-bit range and apply 50% volume to avoid clipping
const max24bit = 8388607 // 2^23 - 1
pcmValue := int32(sample * max24bit * 0.5)
```

**Impact:** Clarifies why 50% volume is used (to prevent clipping on sine wave peaks).

---

## Issues Considered But Not Changed

### 1. ResampledSource Error Handling
**File:** `internal/server/audio_source.go:511`

**Code:**
```go
outputSamples := r.resampler.Resample(r.inputBuffer[:n], samples)
return outputSamples, nil
```

**Analysis:**
The resampler returns the number of samples actually written. While we don't explicitly check if it filled the requested amount, this is actually correct behavior - the caller receives the actual count and can handle partial fills. No change needed.

### 2. FLAC Bit Depth Conversion
**File:** `internal/server/audio_source.go:266-280`

**Code:**
```go
if s.bitDepth == 16 {
    samples[samplesRead] = sample << 8
} else if s.bitDepth == 24 {
    samples[samplesRead] = sample
} else {
    // For other bit depths, scale to 24-bit range
    shift := s.bitDepth - 24
    if shift > 0 {
        samples[samplesRead] = sample >> shift
    } else {
        samples[samplesRead] = sample << -shift
    }
}
```

**Analysis:**
The FLAC library returns samples as int32 in the native bit depth range. For 16-bit FLAC, the value is already correctly ranged ±32,768, so left-shifting by 8 is correct. For 24-bit, the value is already in the right range. The generic case handles other bit depths (8-bit, 20-bit, etc.). This logic is sound.

---

## Testing

After fixes:
- ✅ Server compiles successfully
- ✅ Player compiles successfully
- ✅ No new warnings or errors

## Summary

**Total Issues Fixed:** 4
- 1 Critical (integer overflow protection)
- 3 Code quality improvements (comments, constants)

**Lines Changed:** ~30 lines across 4 files

All fixes maintain backward compatibility and improve code robustness without changing the core 24-bit pipeline functionality.
