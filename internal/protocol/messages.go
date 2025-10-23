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
	// Spec fields (newer)
	SupportFormats     []AudioFormat `json:"support_formats,omitempty"`
	BufferCapacity     int           `json:"buffer_capacity,omitempty"`
	SupportedCommands  []string      `json:"supported_commands,omitempty"`

	// Legacy fields (Music Assistant compatibility - uses separate arrays)
	SupportCodecs      []string      `json:"support_codecs,omitempty"`
	SupportChannels    []int         `json:"support_channels,omitempty"`
	SupportSampleRates []int         `json:"support_sample_rates,omitempty"`
	SupportBitDepth    []int         `json:"support_bit_depth,omitempty"`
}

// AudioFormat describes a supported audio format
type AudioFormat struct {
	Codec      string `json:"codec"`
	Channels   int    `json:"channels"`
	SampleRate int    `json:"sample_rate"`
	BitDepth   int    `json:"bit_depth"`
}

// ServerHello is the server's response to client/hello
type ServerHello struct {
	ServerID string `json:"server_id"`
	Name     string `json:"name"`
	Version  int    `json:"version"`
}

// ClientState reports the player's current state (sent as player/update message)
type ClientState struct {
	State  string `json:"state"`  // "playing" or "idle"
	Volume int    `json:"volume"` // 0-100
	Muted  bool   `json:"muted"`  // All fields are required
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
	ClientTransmitted int64 `json:"client_transmitted"` // Client timestamp in microseconds
}

// ServerTime is the response to client/time
type ServerTime struct {
	ClientTransmitted  int64 `json:"client_transmitted"`  // Echoed client timestamp
	ServerReceived     int64 `json:"server_received"`     // Server receive timestamp
	ServerTransmitted  int64 `json:"server_transmitted"`  // Server send timestamp
}
