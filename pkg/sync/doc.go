// ABOUTME: Clock synchronization package
// ABOUTME: Provides NTP-style clock sync with Resonate servers
// Package sync provides clock synchronization for precise audio timing.
//
// Uses NTP-style round-trip time measurement to sync with server clocks.
//
// Example:
//
//	clock := sync.NewClock()
//	err := clock.Sync("localhost:8927")
//	serverTime := clock.ServerTime()
package sync
