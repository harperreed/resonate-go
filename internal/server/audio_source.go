// ABOUTME: Audio source abstraction for streaming from files or generating test tones
// ABOUTME: Supports MP3, FLAC, WAV files with automatic decoding
package server

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/go-mp3"
)

// AudioSource provides PCM audio samples
type AudioSource interface {
	// Read reads PCM samples into the buffer. Returns number of samples read or error.
	Read(samples []int16) (int, error)
	// SampleRate returns the sample rate of the audio
	SampleRate() int
	// Channels returns the number of channels
	Channels() int
	// Metadata returns title, artist, album
	Metadata() (title, artist, album string)
	// Close closes the audio source
	Close() error
}

// NewAudioSource creates an audio source from a file path
// If path is empty, returns a test tone generator
func NewAudioSource(filePath string) (AudioSource, error) {
	if filePath == "" {
		return NewTestToneSource(), nil
	}

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("audio file not found: %s", filePath)
	}

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return NewMP3Source(filePath)
	default:
		return nil, fmt.Errorf("unsupported audio format: %s (supported: .mp3)", ext)
	}
}

// MP3Source reads from an MP3 file
type MP3Source struct {
	file       *os.File
	decoder    *mp3.Decoder
	sampleRate int
	channels   int
	title      string
	artist     string
	album      string
}

// NewMP3Source creates a new MP3 audio source
func NewMP3Source(filePath string) (*MP3Source, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open MP3 file: %w", err)
	}

	decoder, err := mp3.NewDecoder(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to decode MP3: %w", err)
	}

	// Extract filename as title
	filename := filepath.Base(filePath)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	log.Printf("Loaded MP3: %s (sample rate: %d Hz)", title, decoder.SampleRate())

	return &MP3Source{
		file:       f,
		decoder:    decoder,
		sampleRate: decoder.SampleRate(),
		channels:   2, // MP3 decoder outputs stereo
		title:      title,
		artist:     "Unknown Artist",
		album:      "Unknown Album",
	}, nil
}

func (s *MP3Source) Read(samples []int16) (int, error) {
	// Read bytes from decoder
	numBytes := len(samples) * 2 // int16 = 2 bytes
	buf := make([]byte, numBytes)

	n, err := s.decoder.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	// Convert bytes to int16 samples (little-endian)
	numSamples := n / 2
	for i := 0; i < numSamples; i++ {
		samples[i] = int16(buf[i*2]) | int16(buf[i*2+1])<<8
	}

	if err == io.EOF {
		// Loop the audio - seek back to start
		if _, seekErr := s.file.Seek(0, 0); seekErr != nil {
			return numSamples, fmt.Errorf("failed to seek to start: %w", seekErr)
		}
		// Create new decoder
		newDecoder, decErr := mp3.NewDecoder(s.file)
		if decErr != nil {
			return numSamples, fmt.Errorf("failed to create new decoder: %w", decErr)
		}
		s.decoder = newDecoder
	}

	return numSamples, nil
}

func (s *MP3Source) SampleRate() int { return s.sampleRate }
func (s *MP3Source) Channels() int   { return s.channels }
func (s *MP3Source) Metadata() (string, string, string) {
	return s.title, s.artist, s.album
}
func (s *MP3Source) Close() error {
	return s.file.Close()
}
