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
	samples := []int16{1000, -1000, 500, -500}
	volume := 50
	muted := false

	result := applyVolume(samples, volume, muted)

	if result[0] != 500 {
		t.Errorf("expected 500, got %d", result[0])
	}
	if result[1] != -500 {
		t.Errorf("expected -500, got %d", result[1])
	}
}
