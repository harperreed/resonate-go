// ABOUTME: TUI initialization and control
// ABOUTME: Wraps bubbletea program for player UI
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel creates a new TUI model
func NewModel() Model {
	return Model{
		volume: 100,
		state:  "idle",
	}
}

// Run starts the TUI
func Run() (*tea.Program, error) {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	return p, nil
}
