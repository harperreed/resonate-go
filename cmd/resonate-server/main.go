// ABOUTME: Entry point for Resonate Protocol server
// ABOUTME: Parses CLI flags and starts the server application
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Resonate-Protocol/resonate-go/internal/server"
)

var (
	port      = flag.Int("port", 8927, "WebSocket server port")
	name      = flag.String("name", "", "Server friendly name (default: hostname-resonate-server)")
	logFile   = flag.String("log-file", "resonate-server.log", "Log file path")
	debug     = flag.Bool("debug", false, "Enable debug logging")
	noMDNS    = flag.Bool("no-mdns", false, "Disable mDNS advertisement")
	audioFile = flag.String("audio", "", "Audio file to stream (MP3, FLAC, WAV). If not specified, plays test tone")
)

func main() {
	flag.Parse()

	// Set up logging (both file and console)
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()

	// Log to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)

	// Determine server name
	serverName := *name
	if serverName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		serverName = fmt.Sprintf("%s-resonate-server", hostname)
	}

	log.Printf("Starting Resonate Server: %s on port %d", serverName, *port)
	if *debug {
		log.Printf("Debug logging enabled")
	}
	log.Printf("Logging to: %s", *logFile)
	log.Printf("Press Ctrl-C to stop")

	// Create server
	config := server.Config{
		Port:       *port,
		Name:       serverName,
		EnableMDNS: !*noMDNS,
		Debug:      *debug,
		AudioFile:  *audioFile,
	}

	srv := server.New(config)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("\nReceived %v signal, shutting down gracefully...", sig)
		srv.Stop()
	}()

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	log.Printf("Server stopped")
}
