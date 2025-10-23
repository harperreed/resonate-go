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

	// HACK: On first sync, calculate when server's event loop started in Unix time
	// This lets us match the server's loop.time() exactly
	if cs.sampleCount == 0 && serverLoopStartUnix == 0 {
		// t2 is server's loop.time() in microseconds when it received our request
		// On first sync, t4 is our monotonic time, so we need to get actual Unix time
		nowUnix := time.Now().UnixMicro()

		// Account for half RTT: when server sent its response (t3), the Unix time was approx now - rtt/2
		// But we want Unix time at t2 (when server received our request)
		// Time from t2 to t3 is (t3 - t2), so work backwards from now
		unixAtT3 := nowUnix - (rtt / 2)  // Unix time when server sent response
		unixAtT2 := unixAtT3 - (t3 - t2) // Unix time when server received request
		serverLoopStartUnix = unixAtT2 - t2

		log.Printf("HACK: Calculated server loop start at Unix time %d", serverLoopStartUnix)
		log.Printf("HACK: t2=%dμs (server_recv), t3=%dμs (server_sent), rtt=%dμs",
			t2, t3, rtt)
		log.Printf("HACK: unix_now=%dμs, unix_at_t3=%dμs, unix_at_t2=%dμs",
			nowUnix, unixAtT3, unixAtT2)

		// Set our stored offset to zero since we're now perfectly synchronized
		cs.offset = 0
		cs.sampleCount++
		cs.quality = QualityGood
		return
	}

	// Discard samples with high RTT (network congestion)
	if rtt > 100000 { // 100ms
		log.Printf("Discarding sync sample: high RTT %dμs", rtt)
		return
	}

	// After the hack is applied, the offset should be near zero
	// Apply exponential smoothing for fine-tuning
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

// ServerToLocalTime converts server timestamp to local wall clock time
func (cs *ClockSync) ServerToLocalTime(serverTime int64) time.Time {
	offset := cs.GetOffset()
	// offset = (server_time - client_time)
	// So: client_time = server_time - offset
	localMicros := serverTime - offset

	// Convert microseconds to time.Time
	return time.Unix(0, localMicros*1000)
}

// CurrentMicros returns time in microseconds that matches server's loop.time()
// HACK: The server uses loop.time() (monotonic) for everything and doesn't account
// for client clock offset. We match it by using Unix time minus when server started.
func CurrentMicros() int64 {
	if serverLoopStartUnix == 0 {
		// Before first sync, use our monotonic time as fallback
		return int64(time.Since(startTime) / time.Microsecond)
	}

	// Return time that matches server's loop.time() in microseconds
	return time.Now().UnixMicro() - serverLoopStartUnix
}

// startTime is when our process started (fallback before first sync)
var startTime = time.Now()

// serverLoopStartUnix is when the server's asyncio event loop started, in Unix microseconds
// Calculated from first time sync: unix_now - server_loop_time
var serverLoopStartUnix int64
