package adif

import (
	"strings"
	"sync"
	"time"
)

// Index answers "have I worked this callsign before" queries against the
// most recently synced log. Concurrency-safe: Replace swaps in a whole
// new index built off to the side, so readers never see a half-built one.
//
// Known limitation: exact-string callsign matching only, so a spot for
// "G8GDS/P" won't match a logged "G8GDS". Acceptable for v1.
type Index struct {
	mu         sync.RWMutex
	anyWorked  map[string]struct{}
	bandWorked map[string]map[string]struct{}
	qsoCount   int
	syncedAt   time.Time
}

func NewIndex() *Index {
	return &Index{
		anyWorked:  map[string]struct{}{},
		bandWorked: map[string]map[string]struct{}{},
	}
}

// Replace discards the current index and builds a new one from qsos.
func (idx *Index) Replace(qsos []QSO, syncedAt time.Time) {
	anyWorked := make(map[string]struct{}, len(qsos))
	bandWorked := make(map[string]map[string]struct{}, len(qsos))

	for _, q := range qsos {
		anyWorked[q.Call] = struct{}{}
		if q.Band == "" {
			continue
		}
		bands, ok := bandWorked[q.Call]
		if !ok {
			bands = map[string]struct{}{}
			bandWorked[q.Call] = bands
		}
		bands[q.Band] = struct{}{}
	}

	idx.mu.Lock()
	idx.anyWorked = anyWorked
	idx.bandWorked = bandWorked
	idx.qsoCount = len(qsos)
	idx.syncedAt = syncedAt
	idx.mu.Unlock()
}

func (idx *Index) WorkedAny(call string) bool {
	call = strings.ToUpper(strings.TrimSpace(call))
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, ok := idx.anyWorked[call]
	return ok
}

func (idx *Index) WorkedBand(call, band string) bool {
	band = strings.ToLower(strings.TrimSpace(band))
	if band == "" {
		return false
	}
	call = strings.ToUpper(strings.TrimSpace(call))

	idx.mu.RLock()
	defer idx.mu.RUnlock()
	bands, ok := idx.bandWorked[call]
	if !ok {
		return false
	}
	_, ok = bands[band]
	return ok
}

type Status struct {
	Loaded   bool       `json:"loaded"`
	QSOCount int        `json:"qso_count"`
	SyncedAt *time.Time `json:"synced_at,omitempty"`
}

func (idx *Index) Status() Status {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	status := Status{
		Loaded:   idx.qsoCount > 0,
		QSOCount: idx.qsoCount,
	}
	if !idx.syncedAt.IsZero() {
		status.SyncedAt = &idx.syncedAt
	}
	return status
}
