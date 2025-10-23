// ABOUTME: Entry point for Resonate Protocol player
// ABOUTME: Parses CLI flags and starts the player application
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

	// Set up logging
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

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
	fmt.Printf("Resonate Player starting...\n")
	fmt.Printf("Name: %s\n", playerName)
	fmt.Printf("Port: %d\n", *port)
	fmt.Printf("Buffer: %dms\n", *bufferMs)

	// TODO: Start player
}
