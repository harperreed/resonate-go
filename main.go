// ABOUTME: Entry point for Resonate Protocol player
// ABOUTME: Parses CLI flags and starts the player application
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Resonate-Protocol/resonate-go/internal/app"
)

var (
	serverAddr = flag.String("server", "", "Manual server address (skip mDNS)")
	port       = flag.Int("port", 8927, "Port for mDNS advertisement")
	name       = flag.String("name", "", "Player friendly name (default: hostname-resonate-player)")
	bufferMs   = flag.Int("buffer-ms", 150, "Jitter buffer size in milliseconds")
	logFile    = flag.String("log-file", "resonate-player.log", "Log file path")
	debug      = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	// Set up logging to both console and file
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()

	// Log to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)

	// Determine player name
	playerName := *name
	if playerName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		playerName = fmt.Sprintf("%s-resonate-player", hostname)
	}

	log.Printf("Starting Resonate Player: %s", playerName)

	// Create player
	config := app.Config{
		ServerAddr: *serverAddr,
		Port:       *port,
		Name:       playerName,
		BufferMs:   *bufferMs,
	}

	player := app.New(config)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("Shutdown signal received")
		player.Stop()
	}()

	// Start player
	if err := player.Start(); err != nil {
		log.Fatalf("Player error: %v", err)
	}

	log.Printf("Player stopped")
}
