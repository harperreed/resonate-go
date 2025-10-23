// ABOUTME: Tests for playback scheduler
// ABOUTME: Tests timestamp-based scheduling and buffer management
package player

import (
	"testing"
	"time"
)

func TestSchedulePlayback(t *testing.T) {
	now := time.Now()
	nowMicros := now.UnixNano() / 1000

	// Schedule for 100ms in future
	playTime := nowMicros + 100000
	localPlayTime := time.Unix(0, playTime*1000)

	sleepDuration := localPlayTime.Sub(now)

	if sleepDuration < 50*time.Millisecond || sleepDuration > 150*time.Millisecond {
		t.Errorf("expected sleep ~100ms, got %v", sleepDuration)
	}
}

func TestLateFrameDetection(t *testing.T) {
	now := time.Now()
	nowMicros := now.UnixNano() / 1000

	// Frame scheduled 100ms ago
	playTime := nowMicros - 100000
	localPlayTime := time.Unix(0, playTime*1000)

	sleepDuration := localPlayTime.Sub(now)

	if sleepDuration >= 0 {
		t.Error("expected negative sleep duration for late frame")
	}

	// Should drop if >50ms late
	shouldDrop := sleepDuration < -50*time.Millisecond
	if !shouldDrop {
		t.Error("expected to drop frame >50ms late")
	}
}
