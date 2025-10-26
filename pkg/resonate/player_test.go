// ABOUTME: Integration tests for Player API
// ABOUTME: Tests player creation, configuration, and basic operations
package resonate

import (
	"testing"
	"time"
)

func TestNewPlayer(t *testing.T) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
		Volume:     80,
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	if player == nil {
		t.Fatal("Expected player to be created")
	}

	// Check defaults were applied
	if player.config.BufferMs != 500 {
		t.Errorf("Expected BufferMs=500, got %d", player.config.BufferMs)
	}

	if player.config.DeviceInfo.ProductName == "" {
		t.Error("Expected default ProductName to be set")
	}

	// Check initial state
	state := player.Status()
	if state.State != "idle" {
		t.Errorf("Expected initial state='idle', got '%s'", state.State)
	}
	if state.Volume != 80 {
		t.Errorf("Expected volume=80, got %d", state.Volume)
	}
	if state.Connected {
		t.Error("Expected connected=false initially")
	}

	// Clean up
	player.Close()
}

func TestNewPlayerDefaults(t *testing.T) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Check volume default
	if player.config.Volume != 100 {
		t.Errorf("Expected default volume=100, got %d", player.config.Volume)
	}

	// Check buffer default
	if player.config.BufferMs != 500 {
		t.Errorf("Expected default BufferMs=500, got %d", player.config.BufferMs)
	}

	// Check device info defaults
	if player.config.DeviceInfo.ProductName == "" {
		t.Error("Expected default ProductName")
	}
	if player.config.DeviceInfo.Manufacturer == "" {
		t.Error("Expected default Manufacturer")
	}
	if player.config.DeviceInfo.SoftwareVersion == "" {
		t.Error("Expected default SoftwareVersion")
	}
}

func TestPlayerSetVolume(t *testing.T) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Test setting volume
	err = player.SetVolume(50)
	if err != nil {
		t.Errorf("SetVolume failed: %v", err)
	}

	state := player.Status()
	if state.Volume != 50 {
		t.Errorf("Expected volume=50, got %d", state.Volume)
	}

	// Test volume clamping - too high
	err = player.SetVolume(150)
	if err != nil {
		t.Errorf("SetVolume failed: %v", err)
	}

	state = player.Status()
	if state.Volume != 100 {
		t.Errorf("Expected volume clamped to 100, got %d", state.Volume)
	}

	// Test volume clamping - too low
	err = player.SetVolume(-10)
	if err != nil {
		t.Errorf("SetVolume failed: %v", err)
	}

	state = player.Status()
	if state.Volume != 0 {
		t.Errorf("Expected volume clamped to 0, got %d", state.Volume)
	}
}

func TestPlayerMute(t *testing.T) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Test mute
	err = player.Mute(true)
	if err != nil {
		t.Errorf("Mute failed: %v", err)
	}

	state := player.Status()
	if !state.Muted {
		t.Error("Expected muted=true")
	}

	// Test unmute
	err = player.Mute(false)
	if err != nil {
		t.Errorf("Mute failed: %v", err)
	}

	state = player.Status()
	if state.Muted {
		t.Error("Expected muted=false")
	}
}

func TestPlayerCallbacks(t *testing.T) {
	metadataCalled := false
	stateChangeCalled := false
	errorCalled := false

	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
		OnMetadata: func(m Metadata) {
			metadataCalled = true
		},
		OnStateChange: func(s PlayerState) {
			stateChangeCalled = true
		},
		OnError: func(err error) {
			errorCalled = true
		},
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Trigger state change
	player.SetVolume(50)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	if !stateChangeCalled {
		t.Error("Expected OnStateChange to be called")
	}

	// Note: OnMetadata and OnError require actual connection/errors to test
	// Those are tested in integration tests with real server
	_ = metadataCalled
	_ = errorCalled
}

func TestPlayerStats(t *testing.T) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	stats := player.Stats()

	// Before connection, should have zero stats
	if stats.Received != 0 {
		t.Errorf("Expected Received=0, got %d", stats.Received)
	}
	if stats.Played != 0 {
		t.Errorf("Expected Played=0, got %d", stats.Played)
	}
	if stats.BufferDepth != 0 {
		t.Errorf("Expected BufferDepth=0, got %d", stats.BufferDepth)
	}
}

func TestPlayerClose(t *testing.T) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	err = player.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify state after close
	state := player.Status()
	if state.Connected {
		t.Error("Expected connected=false after close")
	}
	if state.State != "idle" {
		t.Errorf("Expected state='idle' after close, got '%s'", state.State)
	}
}

func TestPlayerStateManagement(t *testing.T) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Test Player",
	}

	player, err := NewPlayer(config)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Test Play without connection (should fail)
	err = player.Play()
	if err == nil {
		t.Error("Expected Play to fail when not connected")
	}

	// Test Pause without connection (should fail)
	err = player.Pause()
	if err == nil {
		t.Error("Expected Pause to fail when not connected")
	}

	// Test Stop without connection (should fail)
	err = player.Stop()
	if err == nil {
		t.Error("Expected Stop to fail when not connected")
	}
}

// Benchmark player creation
func BenchmarkNewPlayer(b *testing.B) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Bench Player",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player, err := NewPlayer(config)
		if err != nil {
			b.Fatalf("Failed to create player: %v", err)
		}
		player.Close()
	}
}

// Benchmark SetVolume
func BenchmarkSetVolume(b *testing.B) {
	config := PlayerConfig{
		ServerAddr: "localhost:8927",
		PlayerName: "Bench Player",
	}

	player, err := NewPlayer(config)
	if err != nil {
		b.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player.SetVolume(i % 100)
	}
}
