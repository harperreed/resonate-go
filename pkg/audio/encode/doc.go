// ABOUTME: Audio encoder package for encoding PCM to various formats
// ABOUTME: Provides Encoder interface and implementations for PCM, Opus
// Package encode provides audio encoders for various codecs.
//
// Supports: PCM (16-bit and 24-bit), Opus
//
// All encoders accept int32 samples in 24-bit range and encode
// to wire format.
//
// Example:
//
//	encoder, err := encode.NewPCM(format)
//	data, err := encoder.Encode(samples)
package encode
