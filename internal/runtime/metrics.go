package runtime

import (
	"sort"
	"sync"
	"time"
)

const timingSampleLimit = 4096

// TimingSummary is a bounded distribution summary. Samples is the number of
// retained observations; Count includes observations overwritten by the ring.
type TimingSummary struct {
	Count, Samples uint64
	P50, P95, P99  time.Duration
}

// Metrics is optional event-loop instrumentation. A nil *Metrics keeps the
// production loop on its normal counter-free path.
type Metrics struct {
	mu              sync.Mutex
	events          uint64
	frameCount      uint64
	eventFrameCount uint64
	eventLatencies  []time.Duration
	frameTimes      []time.Duration
	eventCursor     int
	frameCursor     int
}

// NewMetrics allocates fixed-capacity, bounded timing rings.
func NewMetrics() *Metrics {
	return &Metrics{
		eventLatencies: make([]time.Duration, 0, timingSampleLimit),
		frameTimes:     make([]time.Duration, 0, timingSampleLimit),
	}
}

func (metrics *Metrics) recordEvent() {
	metrics.mu.Lock()
	metrics.events++
	metrics.mu.Unlock()
}

func (metrics *Metrics) recordFrame(frameTime time.Duration, eventLatency *time.Duration) {
	metrics.mu.Lock()
	metrics.frameCount++
	metrics.frameCursor = appendRing(metrics.frameTimes, metrics.frameCursor, frameTime)
	if len(metrics.frameTimes) < timingSampleLimit {
		metrics.frameTimes = append(metrics.frameTimes, frameTime)
	}
	if eventLatency != nil {
		metrics.eventFrameCount++
		metrics.eventCursor = appendRing(metrics.eventLatencies, metrics.eventCursor, *eventLatency)
		if len(metrics.eventLatencies) < timingSampleLimit {
			metrics.eventLatencies = append(metrics.eventLatencies, *eventLatency)
		}
	}
	metrics.mu.Unlock()
}

func appendRing(values []time.Duration, cursor int, value time.Duration) int {
	if len(values) < timingSampleLimit {
		return cursor
	}
	values[cursor] = value
	return (cursor + 1) % timingSampleLimit
}

// Snapshot returns race-safe cumulative counts and percentile summaries.
func (metrics *Metrics) Snapshot() (uint64, TimingSummary, TimingSummary) {
	if metrics == nil {
		return 0, TimingSummary{}, TimingSummary{}
	}
	metrics.mu.Lock()
	events := metrics.events
	frames := metrics.frameCount
	eventFrames := metrics.eventFrameCount
	eventValues := append([]time.Duration(nil), metrics.eventLatencies...)
	frameValues := append([]time.Duration(nil), metrics.frameTimes...)
	metrics.mu.Unlock()
	return events, summarize(eventValues, eventFrames), summarize(frameValues, frames)
}

func summarize(values []time.Duration, count uint64) TimingSummary {
	result := TimingSummary{Count: count, Samples: uint64(len(values))}
	if len(values) == 0 {
		return result
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	result.P50 = percentile(values, 50)
	result.P95 = percentile(values, 95)
	result.P99 = percentile(values, 99)
	return result
}

func percentile(values []time.Duration, percent int) time.Duration {
	index := (len(values)*percent + 99) / 100
	if index < 1 {
		index = 1
	}
	return values[index-1]
}
