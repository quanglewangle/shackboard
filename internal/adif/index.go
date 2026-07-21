package adif

import (
	"sort"
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
	// history is kept separate from the two maps above rather than folded
	// in, because those are read on every visible spot on every poll and
	// shouldn't change shape for a feature (contact history in the QRZ
	// popup) that's only ever queried on a single click.
	history  map[string][]QSO
	qsoCount int
	syncedAt time.Time
}

func NewIndex() *Index {
	return &Index{
		anyWorked:  map[string]struct{}{},
		bandWorked: map[string]map[string]struct{}{},
		history:    map[string][]QSO{},
	}
}

// Replace discards the current index and builds a new one from qsos.
func (idx *Index) Replace(qsos []QSO, syncedAt time.Time) {
	anyWorked := make(map[string]struct{}, len(qsos))
	bandWorked := make(map[string]map[string]struct{}, len(qsos))
	history := make(map[string][]QSO, len(qsos))

	for _, q := range qsos {
		anyWorked[q.Call] = struct{}{}
		if q.Band != "" {
			bands, ok := bandWorked[q.Call]
			if !ok {
				bands = map[string]struct{}{}
				bandWorked[q.Call] = bands
			}
			bands[q.Band] = struct{}{}
		}
		history[q.Call] = append(history[q.Call], q)
	}
	for call, qs := range history {
		sort.Slice(qs, func(i, j int) bool {
			// Date+Time are both fixed-width zero-padded digit strings
			// (YYYYMMDD / HHMM), so plain string comparison sorts them
			// correctly — no need to parse into a real time.Time.
			return qs[i].Date+qs[i].Time > qs[j].Date+qs[j].Time
		})
		history[call] = qs
	}

	idx.mu.Lock()
	idx.anyWorked = anyWorked
	idx.bandWorked = bandWorked
	idx.history = history
	idx.qsoCount = len(qsos)
	idx.syncedAt = syncedAt
	idx.mu.Unlock()
}

// Contacts returns every past QSO with call, most recent first (by
// Date+Time; records with no date sort last). Same exact-match limitation
// as WorkedAny/WorkedBand.
func (idx *Index) Contacts(call string) []QSO {
	call = strings.ToUpper(strings.TrimSpace(call))
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	qs := idx.history[call]
	out := make([]QSO, len(qs))
	copy(out, qs)
	return out
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
