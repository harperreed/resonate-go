// ABOUTME: TUI update helpers for server
// ABOUTME: Functions to send server state updates to TUI
package server

// updateTUI sends current server state to TUI
func (s *Server) updateTUI() {
	if s.tui == nil {
		return
	}

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	// Build client list
	clients := make([]ClientInfo, 0, len(s.clients))
	for _, client := range s.clients {
		client.mu.RLock()
		codec := client.Codec
		state := client.State
		client.mu.RUnlock()

		clients = append(clients, ClientInfo{
			Name:  client.Name,
			ID:    client.ID,
			Codec: codec,
			State: state,
		})
	}

	// Get audio title
	audioTitle := "Test Tone (440Hz)"
	if s.audioEngine != nil && s.audioEngine.source != nil {
		title, artist, _ := s.audioEngine.source.Metadata()
		if artist != "" {
			audioTitle = artist + " - " + title
		} else {
			audioTitle = title
		}
	}

	// Send update
	s.tui.Update(ServerStatus{
		Name:       s.config.Name,
		Port:       s.config.Port,
		Clients:    clients,
		AudioTitle: audioTitle,
	})
}
