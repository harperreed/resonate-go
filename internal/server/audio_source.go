// ABOUTME: Audio source abstraction for streaming from files or generating test tones
// ABOUTME: Supports MP3, FLAC, WAV files with automatic decoding
package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/go-mp3"
	"github.com/mewkiz/flac"
)

// AudioSource provides PCM audio samples
type AudioSource interface {
	// Read reads PCM samples into the buffer (int32 for 24-bit support). Returns number of samples read or error.
	Read(samples []int32) (int, error)
	// SampleRate returns the sample rate of the audio
	SampleRate() int
	// Channels returns the number of channels
	Channels() int
	// Metadata returns title, artist, album
	Metadata() (title, artist, album string)
	// Close closes the audio source
	Close() error
}

// NewAudioSource creates an audio source from a file path or HTTP URL
// If path is empty, returns a test tone generator
// Automatically resamples to 48kHz if needed for Opus compatibility
func NewAudioSource(pathOrURL string) (AudioSource, error) {
	if pathOrURL == "" {
		return NewTestToneSource(), nil
	}

	var source AudioSource
	var err error

	// Check if it's an HTTP(S) URL
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		// Check if it's an HLS stream (.m3u8)
		if strings.Contains(pathOrURL, ".m3u8") {
			log.Printf("Streaming from HLS URL: %s", pathOrURL)
			source, err = NewFFmpegSource(pathOrURL)
			if err != nil {
				return nil, err
			}
		} else {
			log.Printf("Streaming from HTTP URL: %s", pathOrURL)
			source, err = NewHTTPMP3Source(pathOrURL)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Local file
		// Check file exists
		if _, err := os.Stat(pathOrURL); os.IsNotExist(err) {
			return nil, fmt.Errorf("audio file not found: %s", pathOrURL)
		}

		// Determine file type by extension
		ext := strings.ToLower(filepath.Ext(pathOrURL))

		switch ext {
		case ".mp3":
			source, err = NewMP3Source(pathOrURL)
			if err != nil {
				return nil, err
			}
		case ".flac":
			source, err = NewFLACSource(pathOrURL)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported audio format: %s (supported: .mp3, .flac)", ext)
		}
	}

	// Note: We no longer auto-resample here. Sources are kept at native sample rate.
	// If Opus encoding is needed and source isn't 48kHz, resampling happens per-client in audio engine.
	// This allows PCM clients to receive hi-res audio at native rates!

	return source, nil
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

func (s *MP3Source) Read(samples []int32) (int, error) {
	// Read bytes from decoder (MP3 decoder outputs int16 = 2 bytes per sample)
	numBytes := len(samples) * 2
	buf := make([]byte, numBytes)

	n, err := s.decoder.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	// Convert bytes to int16, then scale to 24-bit range
	numSamples := n / 2
	for i := 0; i < numSamples; i++ {
		sample16 := int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
		// Left-shift by 8 to convert 16-bit range to 24-bit range
		// Example: 32767 (max 16-bit) << 8 = 8388352 (near max 24-bit 8388607)
		samples[i] = int32(sample16) << 8
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

// FLACSource reads from a FLAC file
type FLACSource struct {
	file       *os.File
	stream     *flac.Stream
	sampleRate int
	channels   int
	bitDepth   int
	title      string
	artist     string
	album      string
}

// NewFLACSource creates a new FLAC audio source
func NewFLACSource(filePath string) (*FLACSource, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open FLAC file: %w", err)
	}

	stream, err := flac.New(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to decode FLAC: %w", err)
	}

	// Get stream info
	info := stream.Info
	sampleRate := int(info.SampleRate)
	channels := int(info.NChannels)
	bitDepth := int(info.BitsPerSample)

	// Extract filename as title
	filename := filepath.Base(filePath)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	log.Printf("Loaded FLAC: %s (sample rate: %d Hz, channels: %d, bit depth: %d)",
		title, sampleRate, channels, bitDepth)

	return &FLACSource{
		file:       f,
		stream:     stream,
		sampleRate: sampleRate,
		channels:   channels,
		bitDepth:   bitDepth,
		title:      title,
		artist:     "Unknown Artist",
		album:      "Unknown Album",
	}, nil
}

func (s *FLACSource) Read(samples []int32) (int, error) {
	framesNeeded := len(samples) / s.channels
	samplesRead := 0

	for samplesRead < len(samples) && framesNeeded > 0 {
		// Parse next frame
		frame, err := s.stream.ParseNext()
		if err != nil {
			if err == io.EOF {
				// Loop back to start
				if _, seekErr := s.file.Seek(0, 0); seekErr != nil {
					return samplesRead, fmt.Errorf("failed to seek to start: %w", seekErr)
				}
				// Create new stream
				newStream, decErr := flac.New(s.file)
				if decErr != nil {
					return samplesRead, fmt.Errorf("failed to create new stream: %w", decErr)
				}
				s.stream = newStream
				continue
			}
			return samplesRead, err
		}

		// Convert frame samples to int32 24-bit range
		// FLAC samples are typically 16 or 24 bit, stored as int32
		for i := 0; i < int(frame.BlockSize) && samplesRead < len(samples); i++ {
			for ch := 0; ch < s.channels && samplesRead < len(samples); ch++ {
				sample := frame.Subframes[ch].Samples[i]

				// Convert to int32 24-bit range
				// FLAC stores samples as signed integers with the specified bit depth
				if s.bitDepth == 16 {
					// Convert 16-bit to 24-bit range
					samples[samplesRead] = sample << 8
				} else if s.bitDepth == 24 {
					// Already 24-bit, use directly
					samples[samplesRead] = sample
				} else {
					// For other bit depths, scale to 24-bit range
					shift := s.bitDepth - 24
					if shift > 0 {
						samples[samplesRead] = sample >> shift
					} else {
						samples[samplesRead] = sample << -shift
					}
				}
				samplesRead++
			}
		}
	}

	return samplesRead, nil
}

func (s *FLACSource) SampleRate() int { return s.sampleRate }
func (s *FLACSource) Channels() int   { return s.channels }
func (s *FLACSource) Metadata() (string, string, string) {
	return s.title, s.artist, s.album
}
func (s *FLACSource) Close() error {
	return s.file.Close()
}

// HTTPMP3Source streams MP3 from an HTTP URL
type HTTPMP3Source struct {
	url        string
	response   *http.Response
	decoder    *mp3.Decoder
	sampleRate int
	channels   int
	title      string
}

// NewHTTPMP3Source creates a new HTTP MP3 streaming source
func NewHTTPMP3Source(url string) (*HTTPMP3Source, error) {
	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTTP stream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Create MP3 decoder from response body
	decoder, err := mp3.NewDecoder(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to decode MP3 stream: %w", err)
	}

	log.Printf("Streaming MP3 from HTTP: %s (sample rate: %d Hz)", url, decoder.SampleRate())

	return &HTTPMP3Source{
		url:        url,
		response:   resp,
		decoder:    decoder,
		sampleRate: decoder.SampleRate(),
		channels:   2, // MP3 decoder outputs stereo
		title:      "HTTP Stream",
	}, nil
}

func (s *HTTPMP3Source) Read(samples []int32) (int, error) {
	// Read bytes from decoder
	numBytes := len(samples) * 2 // int16 = 2 bytes
	buf := make([]byte, numBytes)

	n, err := s.decoder.Read(buf)
	if err != nil {
		return 0, err // Don't loop HTTP streams, just end on EOF
	}

	// Convert bytes to int16, then scale to 24-bit range
	numSamples := n / 2
	for i := 0; i < numSamples; i++ {
		sample16 := int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
		// Left-shift by 8 to convert 16-bit range to 24-bit range
		samples[i] = int32(sample16) << 8
	}

	return numSamples, nil
}

func (s *HTTPMP3Source) SampleRate() int { return s.sampleRate }
func (s *HTTPMP3Source) Channels() int   { return s.channels }
func (s *HTTPMP3Source) Metadata() (string, string, string) {
	return s.title, "HTTP Stream", ""
}
func (s *HTTPMP3Source) Close() error {
	if s.response != nil {
		return s.response.Body.Close()
	}
	return nil
}

// FFmpegSource streams audio from any URL/format using ffmpeg
// Supports HLS (.m3u8), DASH, and other streaming protocols
type FFmpegSource struct {
	url        string
	cmd        *exec.Cmd
	stdout     io.ReadCloser
	reader     *bufio.Reader
	sampleRate int
	channels   int
	title      string
}

// NewFFmpegSource creates a new ffmpeg-based audio source
// Uses ffmpeg to decode HLS/m3u8 streams and output raw PCM
func NewFFmpegSource(url string) (*FFmpegSource, error) {
	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w (install with: brew install ffmpeg)", err)
	}

	// Fixed output format for consistency
	sampleRate := 48000
	channels := 2

	// Start ffmpeg to decode the stream
	// -i <url>: input URL
	// -f s16le: output format (signed 16-bit little-endian PCM)
	// -ar 48000: output sample rate
	// -ac 2: output channels (stereo)
	// -: output to stdout
	cmd := exec.Command("ffmpeg",
		"-loglevel", "error", // Only show errors
		"-i", url,
		"-f", "s16le",
		"-ar", fmt.Sprintf("%d", sampleRate),
		"-ac", fmt.Sprintf("%d", channels),
		"-")

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get ffmpeg stdout: %w", err)
	}

	// Start ffmpeg process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	log.Printf("Streaming via ffmpeg: %s (sample rate: %d Hz, channels: %d)", url, sampleRate, channels)

	return &FFmpegSource{
		url:        url,
		cmd:        cmd,
		stdout:     stdout,
		reader:     bufio.NewReader(stdout),
		sampleRate: sampleRate,
		channels:   channels,
		title:      "Live Stream",
	}, nil
}

func (s *FFmpegSource) Read(samples []int32) (int, error) {
	// Read raw PCM bytes from ffmpeg stdout
	numBytes := len(samples) * 2 // int16 = 2 bytes
	buf := make([]byte, numBytes)

	n, err := io.ReadFull(s.reader, buf)
	if err != nil {
		return 0, err
	}

	// Convert bytes to int16, then scale to 24-bit range
	numSamples := n / 2
	for i := 0; i < numSamples; i++ {
		sample16 := int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
		// Left-shift by 8 to convert 16-bit range to 24-bit range
		samples[i] = int32(sample16) << 8
	}

	return numSamples, nil
}

func (s *FFmpegSource) SampleRate() int { return s.sampleRate }
func (s *FFmpegSource) Channels() int   { return s.channels }
func (s *FFmpegSource) Metadata() (string, string, string) {
	return s.title, "Live Stream", ""
}
func (s *FFmpegSource) Close() error {
	if s.stdout != nil {
		s.stdout.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
	return nil
}

// ResampledSource wraps an AudioSource and resamples to a target sample rate
type ResampledSource struct {
	source       AudioSource
	resampler    *Resampler
	targetRate   int
	inputBuffer  []int32
	outputBuffer []int32
}

// NewResampledSource creates a resampling wrapper around an audio source
func NewResampledSource(source AudioSource, targetRate int) *ResampledSource {
	inputRate := source.SampleRate()
	channels := source.Channels()

	// Calculate buffer sizes for 100ms chunks
	inputSamples := (inputRate * channels * 100) / 1000
	outputSamples := (targetRate * channels * 100) / 1000

	return &ResampledSource{
		source:       source,
		resampler:    NewResampler(inputRate, targetRate, channels),
		targetRate:   targetRate,
		inputBuffer:  make([]int32, inputSamples),
		outputBuffer: make([]int32, outputSamples*2), // Extra space for safety
	}
}

func (r *ResampledSource) Read(samples []int32) (int, error) {
	// Calculate how many input samples we need
	neededInput := r.resampler.InputSamplesNeeded(len(samples))
	if neededInput > len(r.inputBuffer) {
		neededInput = len(r.inputBuffer)
	}

	// Read from underlying source
	n, err := r.source.Read(r.inputBuffer[:neededInput])
	if err != nil && err != io.EOF {
		return 0, err
	}

	// Resample to output
	outputSamples := r.resampler.Resample(r.inputBuffer[:n], samples)

	return outputSamples, nil
}

func (r *ResampledSource) SampleRate() int {
	return r.targetRate
}

func (r *ResampledSource) Channels() int {
	return r.source.Channels()
}

func (r *ResampledSource) Metadata() (string, string, string) {
	return r.source.Metadata()
}

func (r *ResampledSource) Close() error {
	return r.source.Close()
}
