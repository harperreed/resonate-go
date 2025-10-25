// ABOUTME: Audio output interface tests
// ABOUTME: Verifies Output interface implementation
package output

import (
	"testing"
)

func TestPortAudioImplementsOutput(t *testing.T) {
	var _ Output = (*PortAudio)(nil)
}

func TestNewPortAudio(t *testing.T) {
	out := NewPortAudio()
	if out == nil {
		t.Fatal("NewPortAudio returned nil")
	}
}
