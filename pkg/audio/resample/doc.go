// ABOUTME: Audio resampling package using linear interpolation
// ABOUTME: Converts audio between different sample rates
// Package resample provides audio sample rate conversion.
//
// Uses linear interpolation for converting between sample rates.
// Handles both upsampling and downsampling.
//
// Example:
//
//	r := resample.New(44100, 48000, 2)
//	outputSize := r.Resample(inputSamples, outputSamples)
package resample
