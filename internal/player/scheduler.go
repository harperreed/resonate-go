// ABOUTME: Timestamp-based playback scheduler
// ABOUTME: Schedules audio buffers for precise playback timing
package player

import (
	"container/heap"
	"context"
	"log"
	"time"

	"github.com/Resonate-Protocol/resonate-go/internal/audio"
	"github.com/Resonate-Protocol/resonate-go/internal/sync"
)

// Scheduler manages playback timing
type Scheduler struct {
	clockSync  *sync.ClockSync
	bufferQ    *BufferQueue
	output     chan audio.Buffer
	jitterMs   int
	ctx        context.Context
	cancel     context.CancelFunc

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

	return &Scheduler{
		clockSync: clockSync,
		bufferQ:   NewBufferQueue(),
		output:    make(chan audio.Buffer, 10),
		jitterMs:  jitterMs,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Schedule adds a buffer to the queue
func (s *Scheduler) Schedule(buf audio.Buffer) {
	// Convert server timestamp to local play time
	buf.PlayAt = s.clockSync.ServerToLocalTime(buf.Timestamp)

	// Log first few buffers and problematic ones
	if s.stats.Received < 5 {
		now := time.Now()
		delay := buf.PlayAt.Sub(now)
		offset, rtt, _ := s.clockSync.GetStats()

		log.Printf("Scheduled buffer #%d: timestamp=%d, delay=%v, offset=%dμs, rtt=%dμs",
			s.stats.Received, buf.Timestamp, delay, offset, rtt)
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
	return q.items[i].PlayAt.Before(q.items[j].PlayAt)
}

func (q *BufferQueue) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

func (q *BufferQueue) Push(x interface{}) {
	q.items = append(q.items, x.(audio.Buffer))
}

func (q *BufferQueue) Pop() interface{} {
	n := len(q.items)
	item := q.items[n-1]
	q.items = q.items[:n-1]
	return item
}

func (q *BufferQueue) Peek() audio.Buffer {
	return q.items[0]
}
