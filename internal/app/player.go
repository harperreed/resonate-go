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
	"github.com/Resonate-Protocol/resonate-go/internal/ui"
	"github.com/Resonate-Protocol/resonate-go/internal/version"
	"github.com/google/uuid"
	tea "github.com/charmbracelet/bubbletea"
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
	tuiProg   *tea.Program
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new player
func New(config Config) *Player {
	ctx, cancel := context.WithCancel(context.Background())

	return &Player{
		config:    config,
		clockSync: sync.NewClockSync(),
		output:    player.NewOutput(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the player
func (p *Player) Start() error {
	// Start TUI
	tuiProg, err := ui.Run()
	if err != nil {
		return fmt.Errorf("failed to start TUI: %w", err)
	}
	p.tuiProg = tuiProg

	go p.tuiProg.Run()

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
			SupportFormats: []protocol.AudioFormat{
				{Codec: "opus", Channels: 2, SampleRate: 48000, BitDepth: 16},
				{Codec: "flac", Channels: 2, SampleRate: 48000, BitDepth: 16},
				{Codec: "pcm", Channels: 2, SampleRate: 48000, BitDepth: 16},
			},
			BufferCapacity:    1048576,
			SupportedCommands: []string{"volume", "mute"},
		},
	}

	p.client = client.NewClient(clientConfig)

	if err := p.client.Connect(); err != nil {
		return err
	}

	log.Printf("Connected to server: %s", serverAddr)

	// Start component goroutines
	go p.handleAudioChunks()
	go p.handleControls()
	go p.handleStreamStart()
	go p.handleMetadata()
	go p.clockSyncLoop()

	return nil
}

// clockSyncLoop continuously syncs clock
func (p *Player) clockSyncLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t1 := sync.CurrentMicros()
			p.client.SendTimeSync(t1)

			// Wait for response
			select {
			case resp := <-p.client.TimeSyncResp:
				t4 := sync.CurrentMicros()
				p.clockSync.ProcessSyncResponse(resp.T1, resp.T2, resp.T3, t4)

			case <-time.After(2 * time.Second):
				log.Printf("Time sync timeout")
			}

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
			log.Printf("Stream starting: %s %dHz %dch %dbit",
				start.Codec, start.SampleRate, start.Channels, start.BitDepth)

			format := audio.Format{
				Codec:      start.Codec,
				SampleRate: start.SampleRate,
				Channels:   start.Channels,
				BitDepth:   start.BitDepth,
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
	for {
		select {
		case chunk := <-p.client.AudioChunks:
			if p.decoder == nil || p.scheduler == nil {
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

	if p.tuiProg != nil {
		p.tuiProg.Quit()
	}
}
