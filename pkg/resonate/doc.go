// ABOUTME: High-level Resonate library API
// ABOUTME: Provides simple Player and Server APIs for most use cases
// Package resonate provides high-level APIs for Resonate audio streaming.
//
// This is the main entry point for most library users, providing:
//   - Player: Connect to servers and play synchronized audio
//   - Server: Serve audio to multiple clients
//   - AudioSource: Interface for custom audio sources
//
// For lower-level control, see the audio, protocol, sync, and discovery packages.
//
// Example Player:
//
//	player, err := resonate.NewPlayer(resonate.PlayerConfig{
//	    ServerAddr: "localhost:8927",
//	    PlayerName: "Living Room",
//	    Volume:     80,
//	})
//	err = player.Connect()
//	err = player.Play()
//
// Example Server:
//
//	source, err := resonate.FileSource("/path/to/audio.flac")
//	server, err := resonate.NewServer(resonate.ServerConfig{
//	    Port:   8927,
//	    Source: source,
//	})
//	err = server.Start()
package resonate
