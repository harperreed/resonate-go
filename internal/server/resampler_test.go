// ABOUTME: Tests for audio resampler
// ABOUTME: Tests linear interpolation resampling between sample rates
package server

import (
	"testing"
)

func TestNewResampler(t *testing.T) {
	r := NewResampler(44100, 48000, 2)

	if r == nil {
		t.Fatal("expected resampler to be created")
	}

	if r.inputRate != 44100 {
		t.Errorf("expected inputRate 44100, got %d", r.inputRate)
	}

	if r.outputRate != 48000 {
		t.Errorf("expected outputRate 48000, got %d", r.outputRate)
	}

	if r.channels != 2 {
		t.Errorf("expected channels 2, got %d", r.channels)
	}
}

func TestResampleUpsampling(t *testing.T) {
	// 44100 -> 48000 (upsampling by factor of ~1.088)
	r := NewResampler(44100, 48000, 2)

	// Input: 100 stereo samples (200 int16 values)
	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 100) // Ramp signal
	}

	// Calculate expected output size
	expectedSize := int(float64(len(input)) * float64(48000) / float64(44100))
	output := make([]int32, expectedSize)

	n := r.Resample(input, output)

	// Should have produced output
	if n == 0 {
		t.Fatal("resampler produced no output")
	}

	// Should have produced approximately the expected amount
	// Allow some tolerance due to rounding
	if n < expectedSize-10 || n > expectedSize+10 {
		t.Errorf("expected ~%d samples, got %d", expectedSize, n)
	}

	// Output should have interpolated values (not exact copies)
	allZero := true
	for i := 0; i < n; i++ {
		if output[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("output contains only zeros")
	}
}

func TestResampleDownsampling(t *testing.T) {
	// 48000 -> 44100 (downsampling by factor of ~0.91875)
	r := NewResampler(48000, 44100, 2)

	// Input: 100 stereo samples
	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 100)
	}

	expectedSize := int(float64(len(input)) * float64(44100) / float64(48000))
	output := make([]int32, expectedSize)

	n := r.Resample(input, output)

	if n == 0 {
		t.Fatal("resampler produced no output")
	}

	if n < expectedSize-10 || n > expectedSize+10 {
		t.Errorf("expected ~%d samples, got %d", expectedSize, n)
	}
}

func TestResampleSameRate(t *testing.T) {
	// No resampling needed (48000 -> 48000)
	r := NewResampler(48000, 48000, 2)

	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 100)
	}

	output := make([]int32, len(input)+10) // Extra space for rounding
	n := r.Resample(input, output)

	// Should produce approximately the same number of samples
	// Allow small tolerance for floating point rounding
	if n < len(input)-5 || n > len(input)+5 {
		t.Errorf("expected ~%d samples, got %d", len(input), n)
	}

	// Values should be similar (allow for interpolation artifacts)
	for i := 0; i < n && i < len(input); i++ {
		diff := abs(int(output[i]) - int(input[i]))
		if diff > 200 { // Allow some rounding errors
			t.Errorf("sample %d: expected ~%d, got %d (diff %d)", i, input[i], output[i], diff)
		}
	}
}

func TestResampleStereo(t *testing.T) {
	// Test that stereo channels are handled correctly
	r := NewResampler(44100, 48000, 2)

	// Create input with different L/R patterns
	input := make([]int32, 20) // 10 stereo samples
	for i := 0; i < 10; i++ {
		input[i*2] = 1000    // Left channel
		input[i*2+1] = -1000 // Right channel
	}

	output := make([]int32, 30) // Space for upsampled output
	n := r.Resample(input, output)

	if n == 0 {
		t.Fatal("resampler produced no output")
	}

	// Check that L/R pattern is preserved (approximately)
	leftPositive := 0
	rightNegative := 0
	for i := 0; i < n/2; i++ {
		if output[i*2] > 0 {
			leftPositive++
		}
		if output[i*2+1] < 0 {
			rightNegative++
		}
	}

	// Most samples should maintain the pattern
	if leftPositive < n/4 {
		t.Error("left channel pattern not preserved")
	}
	if rightNegative < n/4 {
		t.Error("right channel pattern not preserved")
	}
}

func TestResampleMono(t *testing.T) {
	// Test mono resampling
	r := NewResampler(44100, 48000, 1)

	input := make([]int32, 100)
	for i := range input {
		input[i] = int32(i * 50)
	}

	expectedSize := int(float64(len(input)) * float64(48000) / float64(44100))
	output := make([]int32, expectedSize)

	n := r.Resample(input, output)

	if n == 0 {
		t.Fatal("resampler produced no output")
	}
}

func TestResampleLargeRatioUp(t *testing.T) {
	// Test large upsampling ratio (44.1k -> 192k)
	r := NewResampler(44100, 192000, 2)

	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 10)
	}

	expectedSize := int(float64(len(input)) * float64(192000) / float64(44100))
	output := make([]int32, expectedSize)

	n := r.Resample(input, output)

	if n == 0 {
		t.Fatal("resampler produced no output")
	}

	// Should have significantly more samples
	if n < len(input)*3 {
		t.Errorf("expected at least 3x upsampling, got %d from %d", n, len(input))
	}
}

func TestResampleLargeRatioDown(t *testing.T) {
	// Test large downsampling ratio (192k -> 48k)
	r := NewResampler(192000, 48000, 2)

	input := make([]int32, 200)
	for i := range input {
		input[i] = int32(i * 10)
	}

	expectedSize := int(float64(len(input)) * float64(48000) / float64(192000))
	output := make([]int32, expectedSize)

	n := r.Resample(input, output)

	if n == 0 {
		t.Fatal("resampler produced no output")
	}

	// Should have significantly fewer samples
	if n > len(input)/2 {
		t.Errorf("expected at most 1/2 samples after downsampling, got %d from %d", n, len(input))
	}
}

func TestResampleEmptyInput(t *testing.T) {
	r := NewResampler(44100, 48000, 2)

	input := []int32{}
	output := make([]int32, 100)

	n := r.Resample(input, output)

	if n != 0 {
		t.Errorf("expected 0 samples from empty input, got %d", n)
	}
}

func TestResampleSmallBuffer(t *testing.T) {
	r := NewResampler(44100, 48000, 2)

	// Small input
	input := []int32{100, -100, 200, -200}
	output := make([]int32, 10)

	n := r.Resample(input, output)

	// Should produce some output
	if n == 0 {
		t.Fatal("resampler produced no output from small buffer")
	}
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
