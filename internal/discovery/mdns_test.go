// ABOUTME: Tests for mDNS discovery
// ABOUTME: Tests service advertisement and discovery
package discovery

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	config := Config{
		ServiceName: "Test Player",
		Port:        8927,
	}

	mgr := NewManager(config)
	if mgr == nil {
		t.Fatal("expected manager to be created")
	}
}
