// ABOUTME: Audio output package for playing audio
// ABOUTME: Provides Output interface and PortAudio implementation
// Package output provides audio playback interfaces.
//
// Currently supports PortAudio for cross-platform audio output.
//
// Example:
//
//	out := output.NewPortAudio()
//	err := out.Open(48000, 2)
//	err = out.Write(samples)
package output
