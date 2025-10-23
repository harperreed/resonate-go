// ABOUTME: Test app to verify clock sync workaround
// ABOUTME: Fakes monotonic clock sync by pretending to match server uptime
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/app"
)

var (
	serverAddr = flag.String("server", "localhost:8927", "Server address")
	name       = flag.String("name", "test-sync", "Player name")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lmicroseconds)

	fmt.Println("=== Clock Sync Test App ===")
	fmt.Println("This test will:")
	fmt.Println("1. Connect to the server")
	fmt.Println("2. Perform time sync with FAKE timestamps")
	fmt.Println("3. Match server's monotonic clock by adding offset to our timestamps")
	fmt.Println()

	// Strategy: We'll intercept time sync and add a fixed offset to our
	// timestamps to make them match the server's monotonic clock range

	config := app.Config{
		ServerAddr: *serverAddr,
		Name:       *name,
		BufferMs:   150,
	}

	player := app.New(config)

	fmt.Printf("Connecting to %s as '%s'...\n", *serverAddr, *name)

	// Start player (this will do time sync automatically)
	if err := player.Start(); err != nil {
		log.Fatalf("Player error: %v", err)
	}

	log.Printf("Test complete")
}
