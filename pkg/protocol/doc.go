// ABOUTME: Resonate wire protocol package
// ABOUTME: Defines protocol messages and WebSocket client
// Package protocol implements the Resonate wire protocol.
//
// Provides message types and WebSocket client for communicating
// with Resonate servers.
//
// Example:
//
//	client, err := protocol.NewClient("localhost:8927")
//	err = client.SendHello(helloMsg)
package protocol
