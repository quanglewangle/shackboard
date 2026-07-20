package cluster

import (
	"sync"
	"time"
)

// Spot is a single DX cluster spot.
type Spot struct {
	Spotter    string    `json:"spotter"`
	FreqKHz    float64   `json:"freq_khz"`
	Band       string    `json:"band"`
	DXCall     string    `json:"dx_call"`
	Comment    string    `json:"comment"`
	TimeUTC    string    `json:"time_utc"`
	ReceivedAt time.Time `json:"received_at"`
}

// Buffer is a concurrency-safe, fixed-capacity ring buffer of recent spots.
type Buffer struct {
	mu     sync.Mutex
	spots  []Spot
	max    int
	maxAge time.Duration
}

func NewBuffer(max int, maxAge time.Duration) *Buffer {
	return &Buffer{
		spots:  make([]Spot, 0, max),
		max:    max,
		maxAge: maxAge,
	}
}

// Add appends a spot, evicting the oldest entry once at capacity.
func (b *Buffer) Add(s Spot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.spots = append(b.spots, s)
	if over := len(b.spots) - b.max; over > 0 {
		b.spots = b.spots[over:]
	}
}

// Recent returns up to limit spots, newest first, dropping anything older
// than maxAge. Always returns a copy, never the internal slice.
func (b *Buffer) Recent(limit int) []Spot {
	b.mu.Lock()
	defer b.mu.Unlock()

	cutoff := time.Now().Add(-b.maxAge)
	out := make([]Spot, 0, limit)
	for i := len(b.spots) - 1; i >= 0; i-- {
		if len(out) >= limit {
			break
		}
		s := b.spots[i]
		if s.ReceivedAt.Before(cutoff) {
			break // spots are appended in time order, so the rest are older still
		}
		out = append(out, s)
	}
	return out
}
