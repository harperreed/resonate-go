// ABOUTME: Tests for audio output
// ABOUTME: Tests volume control and PCM playback
package player

import (
	"testing"
)

func TestVolumeMultiplier(t *testing.T) {
	tests := []struct {
		volume   int
		muted    bool
		expected float64
	}{
		{100, false, 1.0},
		{50, false, 0.5},
		{0, false, 0.0},
		{80, true, 0.0}, // Muted overrides volume
	}

	for _, tt := range tests {
		result := getVolumeMultiplier(tt.volume, tt.muted)
		if result != tt.expected {
			t.Errorf("volume=%d, muted=%v: expected %f, got %f",
				tt.volume, tt.muted, tt.expected, result)
		}
	}
}

func TestApplyVolume(t *testing.T) {
	// Use int32 samples in 24-bit range (left-shifted from 16-bit)
	samples := []int32{1000 << 8, -1000 << 8, 500 << 8, -500 << 8}
	volume := 50
	muted := false

	result := applyVolume(samples, volume, muted)

	expected0 := int32(500 << 8)
	if result[0] != expected0 {
		t.Errorf("expected %d, got %d", expected0, result[0])
	}
	expected1 := int32(-500 << 8)
	if result[1] != expected1 {
		t.Errorf("expected %d, got %d", expected1, result[1])
	}
}
