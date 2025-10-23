// ABOUTME: Tests for clock synchronization implementation
// ABOUTME: Tests offset calculation and exponential smoothing
package sync

import (
	"math"
	"testing"
)

func TestOffsetCalculation(t *testing.T) {
	// Simulate a sync exchange
	t1 := int64(1000000) // Client send
	t2 := int64(1002000) // Server receive (+2ms)
	t3 := int64(1002500) // Server respond (+0.5ms processing)
	t4 := int64(1005000) // Client receive (+2.5ms return)

	rtt, offset := calculateOffset(t1, t2, t3, t4)

	// RTT = (t4-t1) - (t3-t2) = 5000 - 500 = 4500μs
	expectedRTT := int64(4500)
	if rtt != expectedRTT {
		t.Errorf("expected RTT %d, got %d", expectedRTT, rtt)
	}

	// Offset = ((t2-t1) + (t3-t4)) / 2 = (2000 + (-2500)) / 2 = -250μs
	expectedOffset := int64(-250)
	if offset != expectedOffset {
		t.Errorf("expected offset %d, got %d", expectedOffset, offset)
	}
}

func TestSmoothing(t *testing.T) {
	cs := NewClockSync()

	// First sample: offset = ((1002000-1000000) + (1003000-1006000)) / 2 = (2000 - 3000) / 2 = -500μs
	cs.ProcessSyncResponse(1000000, 1002000, 1003000, 1006000)
	offset1 := cs.GetOffset()

	expectedOffset1 := int64(-500)
	if offset1 != expectedOffset1 {
		t.Errorf("expected first offset %d, got %d", expectedOffset1, offset1)
	}

	// Second sample: offset = ((2003000-2000000) + (2003500-2007000)) / 2 = (3000 - 3500) / 2 = -250μs
	cs.ProcessSyncResponse(2000000, 2003000, 2003500, 2007000)
	offset2 := cs.GetOffset()

	// Calculate expected smoothed value
	// First offset is -500 (from first sample)
	// Second raw offset is -250
	// Smoothed = -500 * 0.9 + (-250) * 0.1 = -450 + (-25) = -475μs
	expectedSmoothed := int64(-475)

	// Should be smoothed (not equal to raw second sample)
	if offset2 == -250 {
		t.Error("expected smoothed offset, got raw value")
	}

	// Should be the expected smoothed value
	if offset2 != expectedSmoothed {
		t.Errorf("expected smoothed offset %d, got %d", expectedSmoothed, offset2)
	}

	// Should have changed from first sample (moving toward -250 from -500)
	if math.Abs(float64(offset2-offset1)) < 1 {
		t.Error("expected offset to change with new sample")
	}
}
