# Custom Audio Source Example

This example demonstrates how to implement the `AudioSource` interface to create custom audio generators for streaming.

## What it does

This example shows three different custom audio source implementations:

1. **MultiToneSource** - Mixes multiple sine waves (creates chords)
2. **SweepSource** - Generates frequency sweeps (chirps)
3. **Built-in TestTone** - Single frequency tone (for comparison)

Each demonstrates different aspects of the `AudioSource` interface.

## Building

```bash
cd examples/custom-source
go build
```

## Running

Generate an A major chord (440, 554, 659 Hz):

```bash
./custom-source -mode chord
```

Generate a frequency sweep from 220 to 880 Hz:

```bash
./custom-source -mode sweep
```

Generate a single 440 Hz tone:

```bash
./custom-source -mode single
```

## Command-line options

- `-port` - Server port (default: 8927)
- `-mode` - Source mode: chord, sweep, single (default: chord)
- `-rate` - Sample rate in Hz (default: 192000)
- `-channels` - Number of channels (default: 2)

## AudioSource Interface

To create a custom audio source, implement this interface:

```go
type AudioSource interface {
    // Read fills the buffer with PCM samples (int32 for 24-bit audio)
    Read(samples []int32) (int, error)

    // SampleRate returns the sample rate
    SampleRate() int

    // Channels returns the number of channels
    Channels() int

    // Metadata returns title, artist, album
    Metadata() (title, artist, album string)

    // Close closes the audio source
    Close() error
}
```

## Implementation details

### MultiToneSource

Generates multiple sine waves and mixes them together:

```go
type MultiToneSource struct {
    frequencies []float64
    sampleRate  int
    channels    int
    sampleIndex uint64
    mu          sync.Mutex
}

func (s *MultiToneSource) Read(samples []int32) (int, error) {
    // For each sample frame:
    for i := 0; i < numFrames; i++ {
        t := float64(s.sampleIndex + i) / float64(s.sampleRate)

        // Mix all frequencies
        var mixed float64
        for _, freq := range s.frequencies {
            mixed += math.Sin(2 * math.Pi * freq * t)
        }
        mixed /= float64(len(s.frequencies))

        // Convert to 24-bit PCM (int32)
        pcmValue := int32(mixed * 8388607 * 0.5)

        // Write to all channels
        for ch := 0; ch < s.channels; ch++ {
            samples[i*s.channels + ch] = pcmValue
        }
    }
    return len(samples), nil
}
```

### SweepSource

Generates a time-varying frequency sweep:

```go
func (s *SweepSource) Read(samples []int32) (int, error) {
    for i := 0; i < numFrames; i++ {
        t := float64(s.sampleIndex + i) / float64(s.sampleRate)

        // Calculate current frequency based on progress
        progress := math.Mod(t, s.duration) / s.duration
        freq := s.startFreq + (s.endFreq - s.startFreq) * progress

        // Generate sine wave at current frequency
        sample := math.Sin(2 * math.Pi * freq * t)
        pcmValue := int32(sample * 8388607 * 0.5)

        // Write to all channels
        for ch := 0; ch < s.channels; ch++ {
            samples[i*s.channels + ch] = pcmValue
        }
    }
    return len(samples), nil
}
```

## Key concepts

1. **Sample format**: Use `int32` for 24-bit audio (range: -8388608 to 8388607)
2. **Interleaving**: For stereo (2 channels), samples are stored as [L, R, L, R, ...]
3. **Thread safety**: Use mutex if internal state is accessed from multiple goroutines
4. **Continuous playback**: Track `sampleIndex` to maintain phase continuity

## Use cases for custom sources

- **File streaming**: Read from MP3, FLAC, WAV files
- **Live input**: Capture from microphone or line-in
- **Synthesizers**: Generate music programmatically
- **Network streams**: Proxy audio from internet radio
- **Audio processing**: Apply effects, mixing, filtering
- **Testing**: Generate test signals for debugging

## Next steps

- Implement a file-based source using `pkg/audio/decode`
- Add audio effects (reverb, echo, filters)
- Create a source that mixes multiple input sources
- Implement a source that reads from an audio device
