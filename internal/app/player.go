// ABOUTME: Main player application orchestration
// ABOUTME: Coordinates all components (connection, audio, UI)
package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/audio"
	"github.com/Resonate-Protocol/resonate-go/internal/client"
	"github.com/Resonate-Protocol/resonate-go/internal/discovery"
	"github.com/Resonate-Protocol/resonate-go/internal/player"
	"github.com/Resonate-Protocol/resonate-go/internal/protocol"
	"github.com/Resonate-Protocol/resonate-go/internal/sync"
	"github.com/Resonate-Protocol/resonate-go/internal/version"
	"github.com/google/uuid"
)

// Config holds player configuration
type Config struct {
	ServerAddr string
	Port       int
	Name       string
	BufferMs   int
}

// Player represents the main player application
type Player struct {
	config    Config
	client    *client.Client
	clockSync *sync.ClockSync
	scheduler *player.Scheduler
	output    *player.Output
	discovery *discovery.Manager
	decoder   audio.Decoder
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new player
func New(config Config) *Player {
	ctx, cancel := context.WithCancel(context.Background())

	clockSync := sync.NewClockSync()
	sync.SetGlobalClockSync(clockSync) // Make it globally accessible for CurrentMicros()

	return &Player{
		config:    config,
		clockSync: clockSync,
		output:    player.NewOutput(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the player
func (p *Player) Start() error {
	// TUI temporarily disabled for debugging
	// tuiProg, err := ui.Run()
	// if err != nil {
	// 	return fmt.Errorf("failed to start TUI: %w", err)
	// }
	// p.tuiProg = tuiProg
	// go p.tuiProg.Run()

	log.Printf("TUI disabled - logging to file for debugging")

	// Start discovery if no manual server
	if p.config.ServerAddr == "" {
		p.discovery = discovery.NewManager(discovery.Config{
			ServiceName: p.config.Name,
			Port:        p.config.Port,
		})

		p.discovery.Advertise()
		p.discovery.Browse()

		// Wait for server discovery
		go p.handleDiscovery()
	} else {
		// Connect directly
		if err := p.connect(p.config.ServerAddr); err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
	}

	// Wait for context cancellation
	<-p.ctx.Done()

	return nil
}

// handleDiscovery waits for server discovery
func (p *Player) handleDiscovery() {
	for {
		select {
		case server := <-p.discovery.Servers():
			addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
			log.Printf("Attempting connection to %s", addr)

			if err := p.connect(addr); err != nil {
				log.Printf("Connection failed: %v", err)
				continue
			}
			return

		case <-p.ctx.Done():
			return
		}
	}
}

// connect establishes connection to server
func (p *Player) connect(serverAddr string) error {
	clientID := uuid.New().String()

	clientConfig := client.Config{
		ServerAddr: serverAddr,
		ClientID:   clientID,
		Name:       p.config.Name,
		Version:    1,
		DeviceInfo: protocol.DeviceInfo{
			ProductName:     version.Product,
			Manufacturer:    version.Manufacturer,
			SoftwareVersion: version.Version,
		},
		PlayerSupport: protocol.PlayerSupport{
			// New spec format
			SupportFormats: []protocol.AudioFormat{
				{Codec: "opus", Channels: 2, SampleRate: 48000, BitDepth: 16},
				{Codec: "flac", Channels: 2, SampleRate: 48000, BitDepth: 16},
				{Codec: "pcm", Channels: 2, SampleRate: 48000, BitDepth: 16},
			},
			BufferCapacity:    1048576,
			SupportedCommands: []string{"volume", "mute"},
			// Legacy format (Music Assistant compatibility - separate arrays)
			SupportCodecs:      []string{"opus", "flac", "pcm"},
			SupportChannels:    []int{1, 2},
			SupportSampleRates: []int{44100, 48000},
			SupportBitDepth:    []int{16, 24},
		},
	}

	p.client = client.NewClient(clientConfig)

	if err := p.client.Connect(); err != nil {
		return err
	}

	log.Printf("Connected to server: %s", serverAddr)

	// Perform initial clock sync before starting audio handlers
	if err := p.performInitialSync(); err != nil {
		log.Printf("Initial clock sync failed: %v", err)
	}

	// Start component goroutines
	go p.handleAudioChunks()
	go p.handleControls()
	go p.handleStreamStart()
	go p.handleMetadata()
	go p.clockSyncLoop()

	return nil
}

// performInitialSync does multiple sync rounds before audio starts
func (p *Player) performInitialSync() error {
	log.Printf("Performing initial clock synchronization...")

	// Do 5 quick sync rounds to establish offset
	for i := 0; i < 5; i++ {
		t1 := sync.ClientMicros() // Use raw client time for sync
		p.client.SendTimeSync(t1)

		// Wait for response with timeout
		select {
		case resp := <-p.client.TimeSyncResp:
			t4 := sync.ClientMicros() // Use raw client time for sync
			p.clockSync.ProcessSyncResponse(resp.ClientTransmitted, resp.ServerReceived, resp.ServerTransmitted, t4)

		case <-time.After(500 * time.Millisecond):
			log.Printf("Initial sync round %d timeout", i+1)
		}

		// Brief pause between syncs
		time.Sleep(100 * time.Millisecond)
	}

	offset, rtt, quality := p.clockSync.GetStats()
	log.Printf("Initial clock sync complete: offset=%dμs, rtt=%dμs, quality=%v", offset, rtt, quality)

	return nil
}

// clockSyncLoop continuously syncs clock
func (p *Player) clockSyncLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Drain any stale responses before sending new request
			for {
				select {
				case <-p.client.TimeSyncResp:
					log.Printf("Discarded stale time sync response")
				default:
					goto sendRequest
				}
			}

		sendRequest:
			t1 := sync.ClientMicros() // Use raw client time for sync
			p.client.SendTimeSync(t1)

		case resp := <-p.client.TimeSyncResp:
			// Process response asynchronously when it arrives
			t4 := sync.ClientMicros() // Use raw client time for sync
			p.clockSync.ProcessSyncResponse(resp.ClientTransmitted, resp.ServerReceived, resp.ServerTransmitted, t4)

		case <-p.ctx.Done():
			return
		}
	}
}

// handleStreamStart initializes decoder and output
func (p *Player) handleStreamStart() {
	for {
		select {
		case start := <-p.client.StreamStart:
			// Check if player field exists
			if start.Player == nil {
				log.Printf("Received stream/start with no player info")
				continue
			}

			log.Printf("Stream starting: %s %dHz %dch %dbit",
				start.Player.Codec, start.Player.SampleRate, start.Player.Channels, start.Player.BitDepth)

			format := audio.Format{
				Codec:      start.Player.Codec,
				SampleRate: start.Player.SampleRate,
				Channels:   start.Player.Channels,
				BitDepth:   start.Player.BitDepth,
			}

			// Initialize decoder
			decoder, err := audio.NewDecoder(format)
			if err != nil {
				log.Printf("Failed to create decoder: %v", err)
				continue
			}
			p.decoder = decoder

			// Initialize output
			if err := p.output.Initialize(format); err != nil {
				log.Printf("Failed to initialize output: %v", err)
				continue
			}

			// Initialize scheduler
			p.scheduler = player.NewScheduler(p.clockSync, p.config.BufferMs)
			go p.scheduler.Run()
			go p.handleScheduledAudio()

		case <-p.ctx.Done():
			return
		}
	}
}

// handleAudioChunks decodes and schedules audio
func (p *Player) handleAudioChunks() {
	chunkCount := 0
	for {
		select {
		case chunk := <-p.client.AudioChunks:
			chunkCount++

			// Detailed logging for first few chunks
			if chunkCount <= 5 {
				ourTime := sync.CurrentMicros()
				timeDiff := chunk.Timestamp - ourTime
				log.Printf("Received audio chunk #%d: chunk_ts=%d, our_time=%d, diff=%dμs",
					chunkCount, chunk.Timestamp, ourTime, timeDiff)
			}

			if p.decoder == nil || p.scheduler == nil {
				log.Printf("Skipping chunk: decoder=%v, scheduler=%v", p.decoder != nil, p.scheduler != nil)
				continue
			}

			// Decode
			pcm, err := p.decoder.Decode(chunk.Data)
			if err != nil {
				log.Printf("Decode error: %v", err)
				continue
			}

			// Schedule
			buf := audio.Buffer{
				Timestamp: chunk.Timestamp,
				Samples:   pcm,
			}
			p.scheduler.Schedule(buf)

		case <-p.ctx.Done():
			return
		}
	}
}

// handleScheduledAudio plays scheduled buffers
func (p *Player) handleScheduledAudio() {
	for {
		select {
		case buf := <-p.scheduler.Output():
			if err := p.output.Play(buf); err != nil {
				log.Printf("Playback error: %v", err)
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// handleControls processes server commands
func (p *Player) handleControls() {
	for {
		select {
		case cmd := <-p.client.ControlMsgs:
			switch cmd.Command {
			case "volume":
				p.output.SetVolume(cmd.Volume)
				p.client.SendState(protocol.ClientState{Volume: cmd.Volume})

			case "mute":
				p.output.SetMuted(cmd.Mute)
				p.client.SendState(protocol.ClientState{Muted: cmd.Mute})
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// handleMetadata updates UI with track info
func (p *Player) handleMetadata() {
	for {
		select {
		case meta := <-p.client.Metadata:
			log.Printf("Metadata: %s - %s (%s)", meta.Artist, meta.Title, meta.Album)
			// TODO: Send to TUI

		case <-p.ctx.Done():
			return
		}
	}
}

// Stop stops the player
func (p *Player) Stop() {
	p.cancel()

	if p.client != nil {
		p.client.Close()
	}

	if p.output != nil {
		p.output.Close()
	}
}
