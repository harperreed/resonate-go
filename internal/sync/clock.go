// ABOUTME: Clock synchronization using NTP-style algorithm
// ABOUTME: Maintains offset between client and server clocks
package sync

import (
	"log"
	"sync"
	"time"
)

// ClockSync manages clock synchronization with the server
type ClockSync struct {
	mu            sync.RWMutex
	offset        int64 // Smoothed offset in microseconds
	rawOffset     int64 // Latest raw offset
	rtt           int64 // Latest round-trip time
	quality       Quality
	lastSync      time.Time
	sampleCount   int
	smoothingRate float64
}

// Quality represents sync quality
type Quality int

const (
	QualityGood Quality = iota
	QualityDegraded
	QualityLost
)

// NewClockSync creates a new clock synchronizer
func NewClockSync() *ClockSync {
	return &ClockSync{
		smoothingRate: 0.1, // 10% weight to new samples
		quality:       QualityLost,
	}
}

// ProcessSyncResponse processes a server/time response
func (cs *ClockSync) ProcessSyncResponse(t1, t2, t3, t4 int64) {
	rtt, offset := calculateOffset(t1, t2, t3, t4)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.rtt = rtt
	cs.rawOffset = offset
	cs.lastSync = time.Now()

	// Debug: Log raw timestamp values for first few syncs
	if cs.sampleCount < 3 {
		log.Printf("Raw sync timestamps: t1(client_sent)=%d, t2(server_recv)=%d, t3(server_sent)=%d, t4(client_recv)=%d",
			t1, t2, t3, t4)
		log.Printf("Calculated: rtt=%dμs, raw_offset=%dμs", rtt, offset)
	}

	// Discard samples with high RTT (network congestion)
	if rtt > 100000 { // 100ms
		log.Printf("Discarding sync sample: high RTT %dμs", rtt)
		return
	}

	// Apply exponential smoothing
	if cs.sampleCount == 0 {
		cs.offset = offset
	} else {
		cs.offset = int64(float64(cs.offset)*(1-cs.smoothingRate) +
			float64(offset)*cs.smoothingRate)
	}

	cs.sampleCount++

	// Update quality
	if rtt < 50000 { // <50ms
		cs.quality = QualityGood
	} else {
		cs.quality = QualityDegraded
	}

	log.Printf("Clock sync: offset=%dμs, rtt=%dμs, quality=%v",
		cs.offset, cs.rtt, cs.quality)
}

// calculateOffset computes RTT and clock offset
func calculateOffset(t1, t2, t3, t4 int64) (rtt, offset int64) {
	// Round-trip time
	rtt = (t4 - t1) - (t3 - t2)

	// Estimated offset (positive = server ahead)
	offset = ((t2 - t1) + (t3 - t4)) / 2

	return
}

// GetOffset returns the smoothed clock offset
func (cs *ClockSync) GetOffset() int64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.offset
}

// GetStats returns sync statistics
func (cs *ClockSync) GetStats() (offset, rtt int64, quality Quality) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.offset, cs.rtt, cs.quality
}

// CheckQuality updates quality based on time since last sync
func (cs *ClockSync) CheckQuality() Quality {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if time.Since(cs.lastSync) > 5*time.Second {
		cs.quality = QualityLost
	}

	return cs.quality
}

// ServerToLocalTime converts server timestamp to local time
func (cs *ClockSync) ServerToLocalTime(serverTime int64) time.Time {
	offset := cs.GetOffset()
	// If server is ahead (positive offset), we need to subtract to get our local time
	// If server is behind (negative offset), we need to add to get our local time
	// offset = (server - client), so client = server - offset
	localMicros := serverTime - offset
	return time.Unix(0, localMicros*1000)
}

// CurrentMicros returns current time in microseconds
func CurrentMicros() int64 {
	return time.Now().UnixNano() / 1000
}
