// ABOUTME: Resonate Protocol message type definitions
// ABOUTME: Defines structs for all message types in the protocol
package protocol

// Message is the top-level wrapper for all protocol messages
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// ClientHello is sent by clients to initiate the handshake
type ClientHello struct {
	ClientID       string         `json:"client_id"`
	Name           string         `json:"name"`
	Version        int            `json:"version"`
	SupportedRoles []string       `json:"supported_roles"`
	DeviceInfo     *DeviceInfo    `json:"device_info,omitempty"`
	PlayerSupport  *PlayerSupport `json:"player_support,omitempty"`
}

// DeviceInfo contains device identification
type DeviceInfo struct {
	ProductName     string `json:"product_name"`
	Manufacturer    string `json:"manufacturer"`
	SoftwareVersion string `json:"software_version"`
}

// PlayerSupport describes player capabilities
type PlayerSupport struct {
	Codecs      []string `json:"codecs"`
	SampleRates []int    `json:"sample_rates"`
	Channels    []int    `json:"channels"`
	BitDepths   []int    `json:"bit_depths"`
}

// ServerHello is the server's response to client/hello
type ServerHello struct {
	ServerID string `json:"server_id"`
	Name     string `json:"name"`
	Version  int    `json:"version"`
}

// ClientState reports the player's current state
type ClientState struct {
	State  string `json:"state,omitempty"`
	Volume int    `json:"volume,omitempty"`
	Muted  bool   `json:"muted,omitempty"`
}

// ServerCommand is a control message from the server
type ServerCommand struct {
	Command string `json:"command"`
	Volume  int    `json:"volume,omitempty"`
	Mute    bool   `json:"mute,omitempty"`
}

// StreamStart notifies the client of stream format
type StreamStart struct {
	Codec       string `json:"codec"`
	SampleRate  int    `json:"sample_rate"`
	Channels    int    `json:"channels"`
	BitDepth    int    `json:"bit_depth"`
	CodecHeader string `json:"codec_header,omitempty"` // Base64-encoded
}

// StreamMetadata contains track information
type StreamMetadata struct {
	Title      string `json:"title,omitempty"`
	Artist     string `json:"artist,omitempty"`
	Album      string `json:"album,omitempty"`
	ArtworkURL string `json:"artwork_url,omitempty"`
}

// ClientTime is sent for clock synchronization
type ClientTime struct {
	T1 int64 `json:"t1"` // Client timestamp in microseconds
}

// ServerTime is the response to client/time
type ServerTime struct {
	T1 int64 `json:"t1"` // Echoed client timestamp
	T2 int64 `json:"t2"` // Server receive timestamp
	T3 int64 `json:"t3"` // Server send timestamp
}
