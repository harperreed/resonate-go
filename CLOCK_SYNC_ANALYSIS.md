# Clock Synchronization Problem Analysis

## The Core Issue

The aioresonate server is rejecting all audio chunks with "Audio chunk should have played already" even though our clock synchronization appears to be working. This document explains why.

## How The Server Works

### Chunk Timestamp Calculation
```python
# When streaming starts:
play_start_time_us = int(server.loop.time() * 1_000_000) + INITIAL_PLAYBACK_DELAY_US

# For each chunk:
chunk_timestamp = play_start_time_us + (sample_offset * 1_000_000 / sample_rate)
```

### Late Chunk Detection
```python
# When sending each chunk:
now = int(server.loop.time() * 1_000_000)
if chunk_timestamp - now < 0:
    logger.error("Audio chunk should have played already, skipping it")
    continue
```

**Critical Discovery:** Both `chunk_timestamp` and `now` use `server.loop.time()` - the server's monotonic clock (seconds since asyncio event loop started).

## The Clock Sync Protocol

1. Client sends `t1` (client time when request sent)
2. Server receives at `t2` (server monotonic time)
3. Server responds at `t3` (server monotonic time)
4. Client receives at `t4` (client time when response received)

The server just **echoes** t2 and t3 back. It doesn't store or use the client's offset for anything!

## Why The Server Doesn't Use Client Offset

The server timestamps chunks using **only its own loop.time()**. It never adjusts chunk timestamps based on client clock offset.

This works fine for ESPHome clients running **in the same Python process** - they share the same `loop.time()` so no offset exists.

But it breaks for **external network clients** like ours who have independent monotonic clocks starting at different times.

## Our Current Approach (BROKEN)

```go
// At connection (our process time ~0s, server loop.time ~137s):
hackOffset = calculated_offset + 500000 // 137.3s + 500ms

// Later when sending time sync:
realMicros = time.Since(startTime)  // Our process uptime
return realMicros + hackOffset      // Try to match server's loop.time
```

**Problem:** Our time sync responses correctly show we're synchronized (~500ms ahead). But this doesn't matter because **the server never uses our clock offset when timestamping or checking chunks!**

## Why Chunks Are Rejected

Chunks are timestamped and checked entirely in server time:

```
Stream starts: server.loop.time() = 137.115s
play_start_time = 137.115 + 0.5 = 137.615s
chunk[0] timestamp = 137.615s
chunk[1] timestamp = 137.615 + 0.02s = 137.635s

When sending chunk[0]:
  now = 137.620s (time has advanced while buffering)
  if 137.615 - 137.620 < 0: TRUE - REJECT!
```

The chunks are getting timestamped correctly but there's apparently processing delay causing them to be late **even in server time**.

But wait - that doesn't explain the IMMEDIATE flood of errors. Unless...

## Alternative Theory: Wrong Time Base

What if there's a mismatch in how time is calculated somewhere? Like if `play_start_time_us` is set incorrectly, or if the server is mixing different time bases?

## The Real Bug

After analyzing the aioresonate source code, I believe the issue is:

**The aioresonate library was designed for in-process clients (ESPHome) and doesn't properly handle external networked clients with independent clocks.**

Specifically:
1. ✅ Clock sync protocol exists and works
2. ❌ Server never uses client offset when timestamping chunks
3. ❌ Server checks chunk lateness using only its own clock
4. ❌ No adjustment for network transmission time or client buffering

## Proposed Solution

Since we can't fix the server, we need to perfectly match the server's `loop.time()`:

```go
// At first time sync, calculate when server's loop started in Unix time
var serverLoopStartUnix int64

func ProcessSyncResponse(t1, t2, t3, t4 int64) {
    if first_sync {
        // t2 is server's loop.time() in microseconds
        // Calculate when that loop started in Unix time
        now_unix = time.Now().UnixMicro()
        serverLoopStartUnix = now_unix - t2
    }
}

func CurrentMicros() int64 {
    // Return time that matches server's loop.time()
    return time.Now().UnixMicro() - serverLoopStartUnix
}
```

This makes our timestamps **exactly match** the server's loop.time() (in microseconds).

## Why This Should Work

- Server's loop.time() = seconds since its event loop started
- Our reported time = microseconds since that same moment (calculated)
- Both use same time base
- No drift between our clock and server's clock checks

## Risks

1. **Unix time adjustments (NTP):** If system time jumps, our reported time jumps too
   - Mitigation: Servers rarely have NTP jumps during playback

2. **Precision:** Unix microseconds might have different precision than loop.time()
   - Mitigation: Both are microsecond-resolution

3. **Calculation error:** If our estimate of serverLoopStartUnix is wrong
   - Mitigation: We calculate it from actual t2 value, should be accurate

## Next Steps

1. Implement Unix-time-based approach
2. Test with playback
3. If still failing, add detailed logging of:
   - Exact chunk timestamps we receive
   - Server's play_start_time from logs
   - Our calculated serverLoopStartUnix

## Why Previous Attempts Failed

- **Attempt 1 (no offset):** Our monotonic time was ~137s behind server's
- **Attempt 2 (with hackOffset):** Helped but not exact match due to continued drift
- **Attempt 3 (+500ms buffer):** Made us 500ms ahead, but server doesn't care about our reported offset

All attempts failed because they tried to "synchronize" our clock with the server's, but the server **never uses that synchronization information** when handling chunks!

We need to literally return the same values the server's loop.time() would return.
