# Music Assistant Hi-Res Issue

## Problem
Music Assistant shows "signal degraded" and only displays 44.1kHz/16bit and 48kHz/16bit as supported formats, even though our player advertises full hi-res capabilities (up to 192kHz/24bit).

## What We're Advertising
Our player sends comprehensive capabilities in `client/hello`:

```json
{
  "support_formats": [
    {"codec": "pcm", "channels": 2, "sample_rate": 192000, "bit_depth": 24},
    {"codec": "pcm", "channels": 2, "sample_rate": 176400, "bit_depth": 24},
    {"codec": "pcm", "channels": 2, "sample_rate": 96000, "bit_depth": 24},
    {"codec": "pcm", "channels": 2, "sample_rate": 88200, "bit_depth": 24},
    {"codec": "pcm", "channels": 2, "sample_rate": 48000, "bit_depth": 16},
    {"codec": "pcm", "channels": 2, "sample_rate": 44100, "bit_depth": 16},
    {"codec": "opus", "channels": 2, "sample_rate": 48000, "bit_depth": 16}
  ],
  "support_codecs": ["pcm", "opus"],
  "support_sample_rates": [192000, 176400, 96000, 88200, 48000, 44100],
  "support_bit_depth": [24, 16]
}
```

Both new-style (`support_formats`) and legacy-style (separate arrays) are provided for maximum compatibility.

## Verification
When connecting directly to our own resonate-go server:
- ✅ Server correctly negotiates 192kHz/24bit PCM
- ✅ Player receives and plays hi-res audio
- ✅ Format negotiation works as expected

This proves our implementation is correct.

## Possible Causes

### 1. MA's Resonate Provider Has Hardcoded Limits
Music Assistant's built-in Resonate provider might have:
- Hardcoded maximum sample rate (48kHz)
- Hardcoded bit depth (16-bit)
- Safety filters that only expose "known good" formats

### 2. MA Filters Based on What It Can Provide
MA might be filtering player capabilities based on what formats MA itself can deliver:
- If MA's audio pipeline doesn't support >48kHz, it hides those options
- If MA's source doesn't have hi-res, it might not show hi-res player caps

### 3. MA Version Limitations
Older versions of Music Assistant might not support hi-res audio at all, regardless of player capabilities.

### 4. Protocol Parsing Bug
MA's Resonate provider might have a bug in how it parses capabilities:
- Might only look at legacy arrays and misinterpret them
- Might do AND logic instead of OR (only showing common formats)
- Might not understand the newer `support_formats` array

## Debugging Steps

### 1. Check MA Logs
Look for player connection logs:
```bash
# In Music Assistant logs, search for:
- "Resonate player connected"
- "Player capabilities"
- Your player name
```

### 2. Check MA Version
- Go to Settings → Info
- Check Music Assistant version
- Check if there are Resonate provider updates

### 3. Test Direct Connection
Compare behavior:
```bash
# Direct to our server (should work)
./resonate-server -debug -no-tui
./resonate-player -server localhost:8927 -name "test"
# Should see: "Stream starting: pcm 192000Hz 2ch 24bit"

# Through MA (shows degraded)
# Connect player to MA
# Check what format MA sends
```

### 4. Check MA Settings
- Settings → Providers → Resonate
- Look for any sample rate or quality limitations
- Check if there's a "max quality" or "sample rate" setting

## Workarounds

### Option 1: Use Direct Connection
Skip Music Assistant and connect directly to our server:
```bash
./resonate-server -audio /path/to/hires.flac
./resonate-player -server <server-ip>:8927
```

### Option 2: Request MA Enhancement
File an issue with Music Assistant to:
- Support hi-res audio (>48kHz)
- Properly parse player hi-res capabilities
- Update Resonate provider

### Option 3: Custom MA Provider
Create a custom Resonate provider for MA that:
- Properly reads player capabilities
- Doesn't filter hi-res formats
- Passes through native sample rates

## Expected vs. Actual

### Expected (Direct Connection)
```
Player advertises: 192kHz/24bit capable
Server provides: 192kHz/24bit source
Negotiation: Match! Use PCM 192kHz/24bit
Result: ✅ Hi-res audio delivered
```

### Actual (Through MA)
```
Player advertises: 192kHz/24bit capable
MA sees: Only 44.1kHz/16bit, 48kHz/16bit ???
MA sends: 48kHz/16bit (downsampled)
Result: ❌ "Signal degraded" warning
```

## Conclusion

**Our player implementation is correct.** The issue is in Music Assistant's Resonate provider, which is not properly recognizing or supporting hi-res capabilities.

**Next Steps:**
1. Get MA logs to confirm where filtering happens
2. Check MA version and update if needed
3. Consider filing MA enhancement request
4. For now, use direct connection for hi-res audio

## Testing MA Compatibility
If you want to ensure our player works with MA's current limitations, we could add a "compatibility mode" that:
- Only advertises 48kHz/16bit
- Disables hi-res formats
- Forces lowest common denominator

But this defeats the purpose of building a hi-res player! Better to fix MA or use direct connections.
