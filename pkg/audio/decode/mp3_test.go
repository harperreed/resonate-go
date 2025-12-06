// ABOUTME: Tests for MP3 decoder
// ABOUTME: Tests MP3 decoder creation and not-yet-implemented status
package decode

import (
	"testing"

	"github.com/Sendspin/sendspin-go/pkg/audio"
)

func TestNewMP3(t *testing.T) {
	format := audio.Format{
		Codec:      "mp3",
		SampleRate: 44100,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewMP3(format)

	// NewMP3 returns "not yet fully implemented" error
	if err == nil {
		t.Fatal("expected not implemented error, got nil")
	}

	// Note: Current implementation returns a non-nil decoder with error
	// This documents the actual behavior, though ideally should return nil
	if decoder == nil {
		t.Fatal("expected decoder to be returned (current implementation)")
	}

	expectedError := "MP3 streaming decoder not yet fully implemented"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestNewMP3_InvalidCodec(t *testing.T) {
	format := audio.Format{
		Codec:      "opus",
		SampleRate: 44100,
		Channels:   2,
		BitDepth:   16,
	}

	decoder, err := NewMP3(format)
	if err == nil {
		t.Fatal("expected error for invalid codec, got nil")
	}

	if decoder != nil {
		t.Fatal("expected decoder to be nil for invalid codec")
	}

	expectedError := "invalid codec for MP3 decoder: opus"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}
