// ABOUTME: Tests for loop-origin clock synchronization
// ABOUTME: Tests RTT calculation, loop origin establishment, and time conversion
package sync

import (
	"testing"
	"time"
)

func TestRTTCalculation(t *testing.T) {
	// Simulate a sync exchange with 4.5ms RTT
	t1 := int64(1000000) // Client send (Unix µs)
	t2 := int64(2000)    // Server receive (server loop µs) - +2ms from client send
	t3 := int64(2500)    // Server send (server loop µs) - +0.5ms processing
	t4 := int64(1005000) // Client receive (Unix µs) - +5ms total

	cs := NewClockSync()
	cs.ProcessSyncResponse(t1, t2, t3, t4)

	// RTT = (t4-t1) - (t3-t2) = 5000 - 500 = 4500µs
	rtt, _ := cs.GetStats()
	expectedRTT := int64(4500)
	if rtt != expectedRTT {
		t.Errorf("expected RTT %dµs, got %dµs", expectedRTT, rtt)
	}
}

func TestLoopOriginEstablishment(t *testing.T) {
	cs := NewClockSync()

	// Before sync, should not be synced
	if cs.synced {
		t.Error("expected not synced initially")
	}

	// Process first sync response
	now := time.Now().UnixMicro()
	t1 := now - 10000    // Sent 10ms ago
	t2 := int64(1000000) // Server loop time at receive
	t3 := t2 + 100       // Server processed in 0.1ms
	t4 := now            // Received now

	cs.ProcessSyncResponse(t1, t2, t3, t4)

	// Should now be synced
	if !cs.synced {
		t.Error("expected synced after first response")
	}

	// Quality should be good (low RTT)
	_, quality := cs.GetStats()
	if quality != QualityGood {
		t.Errorf("expected QualityGood, got %v", quality)
	}

	// Should have established loop origin
	// serverLoopStartUnix = now - t2
	expectedOrigin := now - t2
	actualOrigin := cs.serverLoopStartUnix

	// Allow some tolerance (should be within 1ms)
	diff := expectedOrigin - actualOrigin
	if diff < -1000 || diff > 1000 {
		t.Errorf("loop origin off by %dµs (expected ~%d, got %d)",
			diff, expectedOrigin, actualOrigin)
	}
}

func TestServerToLocalTimeConversion(t *testing.T) {
	cs := NewClockSync()

	// Simulate sync at a known time
	clientNow := time.Now().UnixMicro()
	serverLoopTime := int64(5000000) // 5 seconds into server loop

	// Sync: client sent at (now-1000), server received at serverLoopTime
	cs.ProcessSyncResponse(
		clientNow-1000,
		serverLoopTime,
		serverLoopTime+50,
		clientNow,
	)

	// Convert server time to local time
	serverTime := serverLoopTime + 100000 // 100ms later in server loop
	localTime := cs.ServerToLocalTime(serverTime)

	// Local time should be approximately clientNow + 100000µs
	expectedLocal := time.UnixMicro(clientNow + 100000)
	diff := localTime.Sub(expectedLocal).Microseconds()

	// Should be within 10ms tolerance (accounting for processing delays)
	if diff < -10000 || diff > 10000 {
		t.Errorf("time conversion off by %dµs", diff)
	}
}

func TestServerMicrosNow(t *testing.T) {
	cs := NewClockSync()
	SetGlobalClockSync(cs)

	// Before sync, should return Unix time
	before := time.Now().UnixMicro()
	serverNow1 := ServerMicrosNow()
	after := time.Now().UnixMicro()

	if serverNow1 < before || serverNow1 > after {
		t.Error("expected ServerMicrosNow to return Unix time before sync")
	}

	// Perform sync
	clientNow := time.Now().UnixMicro()
	serverLoopTime := int64(3000000) // 3 seconds into server loop

	cs.ProcessSyncResponse(
		clientNow-1000,
		serverLoopTime,
		serverLoopTime+50,
		clientNow,
	)

	// After sync, should return server loop time
	serverNow2 := ServerMicrosNow()

	// Server loop time should be roughly serverLoopTime (allowing for passage of time)
	// The exact value depends on when we call it, but it should be in the ballpark
	if serverNow2 < serverLoopTime-100000 || serverNow2 > serverLoopTime+100000 {
		t.Errorf("ServerMicrosNow returned %d, expected around %d", serverNow2, serverLoopTime)
	}
}

func TestQualityTracking(t *testing.T) {
	cs := NewClockSync()

	// Good quality: RTT < 50ms
	cs.ProcessSyncResponse(1000000, 1000, 1100, 1025000)
	_, quality := cs.GetStats()
	if quality != QualityGood {
		t.Errorf("expected QualityGood for 25ms RTT, got %v", quality)
	}

	// Degraded quality: RTT > 50ms
	cs.ProcessSyncResponse(2000000, 2000, 2100, 2080000)
	_, quality = cs.GetStats()
	if quality != QualityDegraded {
		t.Errorf("expected QualityDegraded for 80ms RTT, got %v", quality)
	}
}

func TestQualityDegradation(t *testing.T) {
	cs := NewClockSync()

	// Establish sync
	cs.ProcessSyncResponse(1000000, 1000, 1100, 1025000)

	// Check quality immediately - should be good
	quality := cs.CheckQuality()
	if quality != QualityGood {
		t.Errorf("expected QualityGood initially, got %v", quality)
	}

	// Simulate time passing by directly manipulating lastSync
	cs.mu.Lock()
	cs.lastSync = time.Now().Add(-6 * time.Second)
	cs.mu.Unlock()

	// Check quality after 6 seconds - should be lost
	quality = cs.CheckQuality()
	if quality != QualityLost {
		t.Errorf("expected QualityLost after 6s, got %v", quality)
	}
}

func TestHighRTTRejection(t *testing.T) {
	cs := NewClockSync()

	// First sync with good RTT establishes loop origin
	cs.ProcessSyncResponse(1000000, 1000, 1100, 1025000)
	origin1 := cs.serverLoopStartUnix
	sampleCount1 := cs.sampleCount

	// Second sync with very high RTT (>100ms) should not update sample count or origin
	cs.ProcessSyncResponse(2000000, 2000, 2100, 2250000)
	origin2 := cs.serverLoopStartUnix
	sampleCount2 := cs.sampleCount

	// Loop origin should not change (high RTT sample rejected)
	if origin2 != origin1 {
		t.Errorf("expected loop origin to stay the same after high RTT sample")
	}

	// Sample count should not increase (sample was discarded)
	if sampleCount2 != sampleCount1 {
		t.Errorf("expected sample count to stay at %d, got %d", sampleCount1, sampleCount2)
	}
}

func TestConcurrentAccess(t *testing.T) {
	cs := NewClockSync()
	SetGlobalClockSync(cs)

	// Perform initial sync
	cs.ProcessSyncResponse(1000000, 1000, 1100, 1025000)

	// Spawn multiple goroutines accessing the clock sync
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				// Read operations
				cs.GetStats()
				cs.CheckQuality()
				ServerMicrosNow()
				cs.ServerToLocalTime(int64(j * 1000))

				// Write operation
				cs.ProcessSyncResponse(
					int64(1000000+j),
					int64(1000+j),
					int64(1100+j),
					int64(1025000+j),
				)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still have valid state
	rtt, quality := cs.GetStats()
	if rtt <= 0 {
		t.Error("invalid RTT after concurrent access")
	}
	if quality == QualityLost {
		t.Error("unexpected QualityLost after concurrent access")
	}
}
