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
	noTUI      = flag.Bool("no-tui", false, "Disable TUI, use streaming logs instead")
	streamLogs = flag.Bool("stream-logs", false, "Alias for -no-tui")
)

func main() {
	flag.Parse()

	// Determine if we should use TUI or streaming logs
	useTUI := !(*noTUI || *streamLogs)

	// Set up logging
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer func() { _ = f.Close() }()

	if useTUI {
		// TUI mode: log only to file
		log.SetOutput(f)
	} else {
		// Streaming logs mode: log to both stdout and file
		multiWriter := io.MultiWriter(os.Stdout, f)
		log.SetOutput(multiWriter)
	}

	// Determine player name
	playerName := *name
	if playerName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		playerName = fmt.Sprintf("%s-resonate-player", hostname)
	}

	if !useTUI {
		log.Printf("Starting Resonate Player: %s", playerName)
		log.Printf("TUI disabled - logging to file for debugging")
	}

	// Create player
	config := app.Config{
		ServerAddr: *serverAddr,
		Port:       *port,
		Name:       playerName,
		BufferMs:   *bufferMs,
		UseTUI:     useTUI,
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
