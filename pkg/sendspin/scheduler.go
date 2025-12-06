// ABOUTME: Timestamp-based playback scheduler for pkg/sendspin
// ABOUTME: Schedules audio buffers for precise playback timing
package sendspin

import (
	"container/heap"
	"context"
	"log"
	"time"

	"github.com/Sendspin/sendspin-go/pkg/audio"
	"github.com/Sendspin/sendspin-go/pkg/sync"
)

// Scheduler manages playback timing
type Scheduler struct {
	clockSync    *sync.ClockSync
	bufferQ      *BufferQueue
	output       chan audio.Buffer
	jitterMs     int
	ctx          context.Context
	cancel       context.CancelFunc
	buffering    bool
	bufferTarget int // Number of chunks to buffer before starting playback

	stats SchedulerStats
}

// SchedulerStats tracks scheduler metrics
type SchedulerStats struct {
	Received int64
	Played   int64
	Dropped  int64
}

// NewScheduler creates a playback scheduler
func NewScheduler(clockSync *sync.ClockSync, jitterMs int) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// Buffer 25 chunks (500ms at 20ms/chunk) to match server's 500ms lead time
	bufferTarget := 25

	return &Scheduler{
		clockSync:    clockSync,
		bufferQ:      NewBufferQueue(),
		output:       make(chan audio.Buffer, 10),
		jitterMs:     jitterMs,
		ctx:          ctx,
		cancel:       cancel,
		buffering:    true,
		bufferTarget: bufferTarget,
	}
}

// Schedule adds a buffer to the queue
func (s *Scheduler) Schedule(buf audio.Buffer) {
	// Convert server timestamp to local play time
	buf.PlayAt = s.clockSync.ServerToLocalTime(buf.Timestamp)

	// Sanity logs for first 5 chunks showing timing
	if s.stats.Received < 5 {
		serverNow := sync.ServerMicrosNow()
		diff := buf.Timestamp - serverNow
		rtt, quality := s.clockSync.GetStats()

		log.Printf("Chunk #%d: timestamp=%dµs, serverNow=%dµs, diff=%dµs (%.1fms), rtt=%dµs, quality=%v",
			s.stats.Received, buf.Timestamp, serverNow, diff, float64(diff)/1000.0, rtt, quality)
	}

	s.stats.Received++
	heap.Push(s.bufferQ, buf)
}

// Run starts the scheduler loop
func (s *Scheduler) Run() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processQueue()
		}
	}
}

// processQueue checks for buffers ready to play
func (s *Scheduler) processQueue() {
	// Check if we're still buffering at startup
	if s.buffering {
		if s.bufferQ.Len() >= s.bufferTarget {
			log.Printf("Startup buffering complete: %d chunks ready", s.bufferQ.Len())
			s.buffering = false
		} else {
			// Still buffering, don't start playback yet
			return
		}
	}

	now := time.Now()

	for s.bufferQ.Len() > 0 {
		buf := s.bufferQ.Peek()

		delay := buf.PlayAt.Sub(now)

		if delay > 50*time.Millisecond {
			// Too early, wait
			break
		} else if delay < -50*time.Millisecond {
			// Too late (>50ms), drop
			heap.Pop(s.bufferQ)
			s.stats.Dropped++
			log.Printf("Dropped late buffer: %v late", -delay)
		} else {
			// Ready to play (within ±50ms window)
			heap.Pop(s.bufferQ)

			select {
			case s.output <- buf:
				s.stats.Played++
			case <-s.ctx.Done():
				return
			}
		}
	}
}

// Output returns the output channel
func (s *Scheduler) Output() <-chan audio.Buffer {
	return s.output
}

// Stats returns scheduler statistics
func (s *Scheduler) Stats() SchedulerStats {
	return s.stats
}

// BufferDepth returns the current buffer queue depth in milliseconds
func (s *Scheduler) BufferDepth() int {
	// Each buffer is typically 10ms (480 samples at 48kHz)
	return s.bufferQ.Len() * 10
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
}

// BufferQueue is a priority queue for audio buffers
type BufferQueue struct {
	items []audio.Buffer
}

func NewBufferQueue() *BufferQueue {
	q := &BufferQueue{}
	heap.Init(q)
	return q
}

// Implement heap.Interface
func (q *BufferQueue) Len() int { return len(q.items) }

func (q *BufferQueue) Less(i, j int) bool {
	// Bounds check to prevent crashes
	if i >= len(q.items) || j >= len(q.items) {
		return false
	}
	return q.items[i].PlayAt.Before(q.items[j].PlayAt)
}

func (q *BufferQueue) Swap(i, j int) {
	// Bounds check to prevent crashes
	if i >= len(q.items) || j >= len(q.items) {
		return
	}
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

func (q *BufferQueue) Push(x interface{}) {
	q.items = append(q.items, x.(audio.Buffer))
}

func (q *BufferQueue) Pop() interface{} {
	n := len(q.items)
	if n == 0 {
		return audio.Buffer{}
	}
	item := q.items[n-1]
	q.items = q.items[:n-1]
	return item
}

func (q *BufferQueue) Peek() audio.Buffer {
	if len(q.items) == 0 {
		return audio.Buffer{}
	}
	return q.items[0]
}
