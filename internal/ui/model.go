// ABOUTME: Bubbletea model for player TUI
// ABOUTME: Defines application state and update logic
package ui

import (
	"fmt"

	"github.com/Resonate-Protocol/resonate-go/internal/protocol"
	"github.com/Resonate-Protocol/resonate-go/internal/sync"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the TUI state
type Model struct {
	// Connection
	connected    bool
	serverName   string

	// Sync
	syncOffset   int64
	syncRTT      int64
	syncQuality  sync.Quality

	// Stream
	codec        string
	sampleRate   int
	channels     int
	bitDepth     int

	// Metadata
	title        string
	artist       string
	album        string

	// Playback
	state        string
	volume       int
	muted        bool

	// Stats
	received     int64
	played       int64
	dropped      int64
	bufferDepth  int

	// Debug
	showDebug    bool

	// Dimensions
	width        int
	height       int
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case StatusMsg:
		m.applyStatus(msg)
	}

	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	s := ""
	s += m.renderHeader()
	s += m.renderStreamInfo()
	s += m.renderControls()
	s += m.renderStats()

	if m.showDebug {
		s += m.renderDebug()
	}

	s += m.renderHelp()

	return s
}

// renderHeader renders connection and sync status
func (m Model) renderHeader() string {
	connStatus := "Disconnected"
	if m.connected {
		connStatus = fmt.Sprintf("Connected to %s", m.serverName)
	}

	syncIcon := "✗"
	syncText := "Lost"
	switch m.syncQuality {
	case sync.QualityGood:
		syncIcon = "✓"
		syncText = fmt.Sprintf("Synced (offset: %+.1fms, jitter: %.1fms)",
			float64(m.syncOffset)/1000.0, float64(m.syncRTT)/1000.0)
	case sync.QualityDegraded:
		syncIcon = "⚠"
		syncText = "Degraded"
	}

	return fmt.Sprintf(`┌─ Resonate Player ────────────────────────────────────┐
│ Status: %-45s │
│ Sync:   %s %-42s │
├──────────────────────────────────────────────────────┤
`, connStatus, syncIcon, syncText)
}

// renderStreamInfo renders current stream and metadata
func (m Model) renderStreamInfo() string {
	if !m.connected || m.codec == "" {
		return "│ No stream                                            │\n"
	}

	s := "│ Now Playing:                                         │\n"
	if m.title != "" {
		s += fmt.Sprintf("│   Track:  %-42s │\n", truncate(m.title, 42))
		s += fmt.Sprintf("│   Artist: %-42s │\n", truncate(m.artist, 42))
		s += fmt.Sprintf("│   Album:  %-42s │\n", truncate(m.album, 42))
	} else {
		s += "│   (No metadata)                                      │\n"
	}

	s += "│                                                      │\n"
	s += fmt.Sprintf("│ Format: %s %dHz %s %d-bit%-17s │\n",
		m.codec, m.sampleRate, channelName(m.channels), m.bitDepth, "")

	return s
}

// renderControls renders volume and buffer status
func (m Model) renderControls() string {
	muteIcon := ""
	if m.muted {
		muteIcon = " 🔇"
	}

	volumeBar := renderBar(m.volume, 100, 10)

	return fmt.Sprintf("│                                                      │\n"+
		"│ Volume: [%s] %d%%%s%-17s │\n"+
		"│ Buffer: %dms (%d chunks)%-24s │\n",
		volumeBar, m.volume, muteIcon, "",
		m.bufferDepth, m.bufferDepth/10, "")
}

// renderStats renders playback statistics
func (m Model) renderStats() string {
	return fmt.Sprintf(`├──────────────────────────────────────────────────────┤
│ Stats:  RX: %d  Played: %d  Dropped: %d%-8s │
│                                                      │
`, m.received, m.played, m.dropped, "")
}

// renderHelp renders keyboard shortcuts
func (m Model) renderHelp() string {
	return `│ ↑/↓:Volume  m:Mute  r:Reconnect  d:Debug  q:Quit   │
└──────────────────────────────────────────────────────┘
`
}

// renderDebug renders debug information
func (m Model) renderDebug() string {
	return fmt.Sprintf(`│ DEBUG:                                               │
│   Goroutines: (not tracked)                         │
│   Channels: (not tracked)                           │
│   Clock Offset: %+dμs                              │
`, m.syncOffset)
}

// handleKey handles keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up":
		if m.volume < 100 {
			m.volume += 5
			if m.volume > 100 {
				m.volume = 100
			}
		}
	case "down":
		if m.volume > 0 {
			m.volume -= 5
			if m.volume < 0 {
				m.volume = 0
			}
		}
	case "m":
		m.muted = !m.muted
	case "d":
		m.showDebug = !m.showDebug
	}

	return m, nil
}

// applyStatus updates model from status message
func (m *Model) applyStatus(msg StatusMsg) {
	if msg.Connected != nil {
		m.connected = *msg.Connected
	}
	if msg.ServerName != "" {
		m.serverName = msg.ServerName
	}
	if msg.SyncOffset != 0 {
		m.syncOffset = msg.SyncOffset
		m.syncRTT = msg.SyncRTT
		m.syncQuality = msg.SyncQuality
	}
	if msg.Codec != "" {
		m.codec = msg.Codec
		m.sampleRate = msg.SampleRate
		m.channels = msg.Channels
		m.bitDepth = msg.BitDepth
	}
	if msg.Title != "" {
		m.title = msg.Title
		m.artist = msg.Artist
		m.album = msg.Album
	}
	if msg.Volume != 0 {
		m.volume = msg.Volume
	}
	if msg.Received != 0 {
		m.received = msg.Received
		m.played = msg.Played
		m.dropped = msg.Dropped
	}
}

// StatusMsg updates TUI state
type StatusMsg struct {
	Connected   *bool
	ServerName  string
	SyncOffset  int64
	SyncRTT     int64
	SyncQuality sync.Quality
	Codec       string
	SampleRate  int
	Channels    int
	BitDepth    int
	Title       string
	Artist      string
	Album       string
	Volume      int
	Received    int64
	Played      int64
	Dropped     int64
}

// Utility functions
func renderBar(value, max, width int) string {
	filled := (value * width) / max
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func channelName(channels int) string {
	if channels == 1 {
		return "Mono"
	}
	return "Stereo"
}
