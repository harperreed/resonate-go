// ABOUTME: Clock synchronization with drift compensation
// ABOUTME: Tracks both offset AND drift to handle clock frequency differences
package sync

import (
	"log"
	"sync"
	"time"
)

// ClockSync manages clock synchronization with drift compensation
type ClockSync struct {
	mu              sync.RWMutex
	offset          int64     // Current offset in microseconds (server - client)
	drift           float64   // Clock drift rate (dimensionless: μs/μs)
	rawOffset       int64     // Latest raw offset measurement
	rtt             int64     // Latest round-trip time
	quality         Quality
	lastSync        time.Time
	lastSyncMicros  int64     // Client time (μs) when offset/drift were last updated
	sampleCount     int
	smoothingRate   float64
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
		drift:         0.0, // Start assuming no drift
	}
}

// ProcessSyncResponse processes a server/time response with drift compensation
func (cs *ClockSync) ProcessSyncResponse(t1, t2, t3, t4 int64) {
	rtt, measuredOffset := calculateOffset(t1, t2, t3, t4)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.rtt = rtt
	cs.rawOffset = measuredOffset
	cs.lastSync = time.Now()

	// Debug: Log raw timestamp values for first few syncs
	if cs.sampleCount < 3 {
		log.Printf("Raw sync timestamps: t1=%d, t2=%d, t3=%d, t4=%d",
			t1, t2, t3, t4)
		log.Printf("Calculated: rtt=%dμs, measured_offset=%dμs", rtt, measuredOffset)
	}

	// Discard samples with high RTT (network congestion)
	if rtt > 100000 { // 100ms
		log.Printf("Discarding sync sample: high RTT %dμs", rtt)
		return
	}

	// First sync: initialize offset, no drift yet
	if cs.sampleCount == 0 {
		cs.offset = measuredOffset
		cs.lastSyncMicros = t4 // Remember client time at this measurement
		cs.sampleCount++
		cs.quality = QualityGood
		log.Printf("Initial sync: offset=%dμs, rtt=%dμs", cs.offset, rtt)
		return
	}

	// Second sync: calculate initial drift
	if cs.sampleCount == 1 {
		dt := float64(t4 - cs.lastSyncMicros) // Time elapsed in client microseconds
		if dt > 0 {
			// Drift = change in offset over time
			cs.drift = float64(measuredOffset-cs.offset) / dt
			log.Printf("Drift initialized: drift=%.9f μs/μs over Δt=%.0fμs", cs.drift, dt)
		}
		cs.offset = measuredOffset
		cs.lastSyncMicros = t4
		cs.sampleCount++
		cs.quality = QualityGood
		log.Printf("Second sync: offset=%dμs, drift=%.9f, rtt=%dμs", cs.offset, cs.drift, rtt)
		return
	}

	// Subsequent syncs: predict offset using drift, then update both
	dt := float64(t4 - cs.lastSyncMicros)
	if dt <= 0 {
		log.Printf("Discarding sync sample: non-monotonic time")
		return
	}

	// Predict what offset should be based on drift
	predictedOffset := cs.offset + int64(cs.drift*dt)

	// Residual = how much our prediction was off
	residual := measuredOffset - predictedOffset

	// Reject outliers (residual > 50ms suggests network issue or clock jump)
	if residual > 50000 || residual < -50000 {
		log.Printf("Discarding sync sample: large residual %dμs (possible clock jump)", residual)
		return
	}

	// Update offset from PREDICTED offset plus gain * residual
	// This is the Kalman filter update formula (simplified with fixed gain)
	cs.offset = predictedOffset + int64(cs.smoothingRate*float64(residual))

	// Update drift: drift correction is residual / dt
	// This estimates how much the drift rate needs to change
	driftCorrection := float64(residual) / dt
	cs.drift = cs.drift + cs.smoothingRate*driftCorrection

	cs.lastSyncMicros = t4
	cs.sampleCount++

	// Update quality
	if rtt < 50000 { // <50ms
		cs.quality = QualityGood
	} else {
		cs.quality = QualityDegraded
	}

	if cs.sampleCount < 10 {
		log.Printf("Sync #%d: offset=%dμs, drift=%.9f, residual=%dμs, rtt=%dμs",
			cs.sampleCount, cs.offset, cs.drift, residual, rtt)
	}
}

// calculateOffset computes RTT and clock offset
func calculateOffset(t1, t2, t3, t4 int64) (rtt, offset int64) {
	// Round-trip time
	rtt = (t4 - t1) - (t3 - t2)

	// Estimated offset (positive = server ahead of client)
	offset = ((t2 - t1) + (t3 - t4)) / 2

	return
}

// GetOffset returns the current offset
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
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// If we haven't synced yet, assume server time = client time
	if cs.sampleCount == 0 {
		return time.Unix(0, serverTime*1000)
	}

	// Inverse of the forward transform:
	// server_time = client_time + offset + drift * (client_time - last_sync)
	// Rearranging: server_time = client_time * (1 + drift) + offset - drift * last_sync
	// Solving: client_time = (server_time - offset + drift * last_sync) / (1 + drift)

	numerator := float64(serverTime) - float64(cs.offset) + cs.drift*float64(cs.lastSyncMicros)
	denominator := 1.0 + cs.drift
	clientMicros := int64(numerator / denominator)

	// Convert microseconds to time.Time
	return time.Unix(0, clientMicros*1000)
}

// CurrentMicros returns current time in server's reference frame
// This accounts for both offset AND drift over time
func CurrentMicros() int64 {
	cs := globalClockSync
	if cs == nil {
		// Before sync initialized, use raw client time
		return ClientMicros()
	}

	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Get current raw client time
	clientNow := ClientMicros()

	// If we haven't synced yet, return client time
	if cs.sampleCount == 0 {
		return clientNow
	}

	// Apply offset and drift: server_time = client_time + offset + drift * (client_time - last_sync)
	dt := clientNow - cs.lastSyncMicros
	serverTime := clientNow + cs.offset + int64(cs.drift*float64(dt))

	return serverTime
}

// SetGlobalClockSync sets the global clock sync instance
func SetGlobalClockSync(cs *ClockSync) {
	globalClockSync = cs
}

// ClientMicros returns raw client Unix epoch time in microseconds
// This is ONLY for use in time synchronization - use CurrentMicros() for timestamps
func ClientMicros() int64 {
	return time.Now().UnixMicro()
}

// globalClockSync is the global clock synchronization instance
var globalClockSync *ClockSync
