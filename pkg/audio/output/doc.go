// ABOUTME: Audio output package for playing audio
// ABOUTME: Provides Output interface with malgo (24-bit) and oto (16-bit) implementations
// Package output provides audio playback interfaces.
//
// Currently supports:
//   - malgo (miniaudio): 16/24/32-bit output, format re-initialization supported
//   - oto: 16-bit output only, legacy support
//
// Example:
//
//	out := output.NewMalgo()
//	err := out.Open(192000, 2, 24)  // 192kHz, stereo, 24-bit
//	err = out.Write(samples)
package output
