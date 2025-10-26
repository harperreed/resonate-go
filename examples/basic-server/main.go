// ABOUTME: Basic Resonate server example
// ABOUTME: Demonstrates how to create a simple streaming server
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Resonate-Protocol/resonate-go/pkg/resonate"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 8927, "Server port")
	serverName := flag.String("name", "Basic Server", "Server name")
	sampleRate := flag.Int("rate", 192000, "Sample rate (Hz)")
	channels := flag.Int("channels", 2, "Number of channels")
	enableMDNS := flag.Bool("mdns", true, "Enable mDNS service advertisement")
	flag.Parse()

	log.Printf("Creating test tone source: %dHz, %d channels", *sampleRate, *channels)

	// Create a test tone audio source
	source := resonate.NewTestTone(*sampleRate, *channels)

	// Create server configuration
	config := resonate.ServerConfig{
		Port:       *port,
		Name:       *serverName,
		Source:     source,
		EnableMDNS: *enableMDNS,
		Debug:      false,
	}

	// Create server
	server, err := resonate.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Starting Resonate server...")
	log.Printf("  Name: %s", *serverName)
	log.Printf("  Port: %d", *port)
	log.Printf("  Audio: %dHz, %d channels, 24-bit", *sampleRate, *channels)
	if *enableMDNS {
		log.Printf("  mDNS: enabled")
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
	}()

	// Print client info periodically
	go func() {
		for {
			select {
			case <-make(chan struct{}):
			}
		}
	}()

	// Wait for interrupt signal or error
	log.Printf("\nServer running. Press Ctrl+C to stop")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Printf("Received interrupt signal, shutting down...")
	case err := <-errChan:
		log.Printf("Server error: %v", err)
	}

	// Stop server
	server.Stop()
	log.Printf("Server stopped")
}
