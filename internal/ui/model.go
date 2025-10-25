// ABOUTME: Bubbletea model for player TUI
// ABOUTME: Defines application state and update logic
package ui

import (
	"fmt"
	"strings"

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
	artworkPath  string

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
	goroutines   int
	memAlloc     uint64
	memSys       uint64

	// Dimensions
	width        int
	height       int

	// Volume control channel
	volumeCtrl   *VolumeControl
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

	// Use terminal width for responsive layout
	width := m.width
	if width < 60 {
		width = 60 // Minimum width
	}
	innerWidth := width - 4 // Account for borders

	titleWidth := width - 20 // Space for "┌─ Resonate Player " prefix
	title := "┌─ Resonate Player " + repeatString("─", titleWidth) + "┐\n"

	statusLine := fmt.Sprintf("│ Status: %-*s │\n", innerWidth-9, truncate(connStatus, innerWidth-9))
	syncLine := fmt.Sprintf("│ Sync:   %s %-*s │\n", syncIcon, innerWidth-11, truncate(syncText, innerWidth-11))
	separator := "├" + repeatString("─", width-2) + "┤\n"

	return title + statusLine + syncLine + separator
}

// renderStreamInfo renders current stream and metadata
func (m Model) renderStreamInfo() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	innerWidth := width - 4

	if !m.connected || m.codec == "" {
		return fmt.Sprintf("│ %-*s │\n", innerWidth, "No stream")
	}

	s := fmt.Sprintf("│ %-*s │\n", innerWidth, "Now Playing:")
	if m.title != "" {
		metaWidth := innerWidth - 10 // Account for "  Track:  " prefix
		s += fmt.Sprintf("│   Track:  %-*s │\n", innerWidth-10, truncate(m.title, metaWidth))
		s += fmt.Sprintf("│   Artist: %-*s │\n", innerWidth-10, truncate(m.artist, metaWidth))
		s += fmt.Sprintf("│   Album:  %-*s │\n", innerWidth-10, truncate(m.album, metaWidth))
		if m.artworkPath != "" {
			s += fmt.Sprintf("│   Art:    %-*s │\n", innerWidth-10, truncate(m.artworkPath, metaWidth))
		}
	} else {
		s += fmt.Sprintf("│   %-*s │\n", innerWidth-3, "(No metadata)")
	}

	s += fmt.Sprintf("│ %-*s │\n", innerWidth, "")
	formatStr := fmt.Sprintf("Format: %s %dHz %s %d-bit",
		m.codec, m.sampleRate, channelName(m.channels), m.bitDepth)
	s += fmt.Sprintf("│ %-*s │\n", innerWidth, formatStr)

	return s
}

// renderControls renders volume and buffer status
func (m Model) renderControls() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	innerWidth := width - 4

	muteIcon := ""
	if m.muted {
		muteIcon = " 🔇"
	}

	volumeBar := renderBar(m.volume, 100, 10)

	s := fmt.Sprintf("│ %-*s │\n", innerWidth, "")
	volumeStr := fmt.Sprintf("Volume: [%s] %d%%%s", volumeBar, m.volume, muteIcon)
	s += fmt.Sprintf("│ %-*s │\n", innerWidth, volumeStr)

	bufferStr := fmt.Sprintf("Buffer: %dms (%d chunks)", m.bufferDepth, m.bufferDepth/10)
	s += fmt.Sprintf("│ %-*s │\n", innerWidth, bufferStr)

	return s
}

// renderStats renders playback statistics
func (m Model) renderStats() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	innerWidth := width - 4

	separator := "├" + repeatString("─", width-2) + "┤\n"
	statsStr := fmt.Sprintf("Stats:  RX: %d  Played: %d  Dropped: %d", m.received, m.played, m.dropped)
	statsLine := fmt.Sprintf("│ %-*s │\n", innerWidth, statsStr)
	emptyLine := fmt.Sprintf("│ %-*s │\n", innerWidth, "")

	return separator + statsLine + emptyLine
}

// renderHelp renders keyboard shortcuts
func (m Model) renderHelp() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	innerWidth := width - 4

	helpStr := "↑/↓:Volume  m:Mute  r:Reconnect  d:Debug  q:Quit"
	helpLine := fmt.Sprintf("│ %-*s │\n", innerWidth, helpStr)
	bottom := "└" + repeatString("─", width-2) + "┘\n"

	return helpLine + bottom
}

// renderDebug renders debug information
func (m Model) renderDebug() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	innerWidth := width - 4

	memAllocMB := float64(m.memAlloc) / 1024 / 1024
	memSysMB := float64(m.memSys) / 1024 / 1024

	debugTitle := fmt.Sprintf("│ %-*s │\n", innerWidth, "DEBUG:")
	goroutineStr := fmt.Sprintf("  Goroutines: %d", m.goroutines)
	goroutineLine := fmt.Sprintf("│ %-*s │\n", innerWidth, goroutineStr)
	memStr := fmt.Sprintf("  Memory: %.1f MB / %.1f MB", memAllocMB, memSysMB)
	memLine := fmt.Sprintf("│ %-*s │\n", innerWidth, memStr)
	clockStr := fmt.Sprintf("  Clock Offset: %+dμs", m.syncOffset)
	clockLine := fmt.Sprintf("│ %-*s │\n", innerWidth, clockStr)

	return debugTitle + goroutineLine + memLine + clockLine
}

// handleKey handles keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		// Send quit signal to player
		if m.volumeCtrl != nil {
			select {
			case m.volumeCtrl.Quit <- QuitMsg{}:
			default:
				// Channel full, skip
			}
		}
		return m, tea.Quit
	case "up":
		if m.volume < 100 {
			m.volume += 5
			if m.volume > 100 {
				m.volume = 100
			}
			// Send volume change to player via channel
			if m.volumeCtrl != nil {
				select {
				case m.volumeCtrl.Changes <- VolumeChangeMsg{Volume: m.volume, Muted: m.muted}:
				default:
					// Channel full, skip
				}
			}
		}
	case "down":
		if m.volume > 0 {
			m.volume -= 5
			if m.volume < 0 {
				m.volume = 0
			}
			// Send volume change to player via channel
			if m.volumeCtrl != nil {
				select {
				case m.volumeCtrl.Changes <- VolumeChangeMsg{Volume: m.volume, Muted: m.muted}:
				default:
					// Channel full, skip
				}
			}
		}
	case "m":
		m.muted = !m.muted
		// Send volume change to player via channel
		if m.volumeCtrl != nil {
			select {
			case m.volumeCtrl.Changes <- VolumeChangeMsg{Volume: m.volume, Muted: m.muted}:
			default:
				// Channel full, skip
			}
		}
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
	// Sync stats are always applied when sent (offset can be 0 for perfect sync)
	if msg.SyncOffset != 0 || msg.SyncRTT != 0 {
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
	if msg.ArtworkPath != "" {
		m.artworkPath = msg.ArtworkPath
	}
	// Volume is always applied when explicitly sent (can be 0 for silent)
	// We rely on caller not sending Volume=0 in messages unless it's intentional
	if msg.Volume != 0 {
		m.volume = msg.Volume
	}
	// Always apply stats - they can legitimately be zero
	m.received = msg.Received
	m.played = msg.Played
	m.dropped = msg.Dropped
	m.bufferDepth = msg.BufferDepth
	m.goroutines = msg.Goroutines
	m.memAlloc = msg.MemAlloc
	m.memSys = msg.MemSys
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
	ArtworkPath string
	Volume      int
	Received    int64
	Played      int64
	Dropped     int64
	BufferDepth int
	Goroutines  int
	MemAlloc    uint64
	MemSys      uint64
}

// VolumeChangeMsg requests a volume change
type VolumeChangeMsg struct {
	Volume int
	Muted  bool
}

// QuitMsg signals the player should quit
type QuitMsg struct{}

// Utility functions
func renderBar(value, max, width int) string {
	filled := (value * width) / max
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
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

func repeatString(s string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(s, count)
}
