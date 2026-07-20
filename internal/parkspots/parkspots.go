// Package parkspots polls the POTA and SOTA activator-spot feeds and
// caches a merged, time-sorted list.
package parkspots

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"
)

// utcTimeLayouts covers the timestamp formats actually seen from the two
// feeds, which don't agree on ISO 8601 formatting: POTA omits a timezone
// suffix (but is UTC), SOTA includes a "Z" suffix, sometimes with
// fractional seconds.
var utcTimeLayouts = []string{
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.999999999Z",
	"2006-01-02T15:04:05",
}

// parseUTCTime tries each known layout in turn. A spot with an
// unparseable timestamp is still shown (Data.Spots still includes it) —
// it just sorts to the zero-time end of the list rather than being
// dropped outright.
func parseUTCTime(raw string) time.Time {
	for _, layout := range utcTimeLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// Spot is a normalized POTA or SOTA activation spot.
type Spot struct {
	Program   string    `json:"program"` // "POTA" or "SOTA"
	Activator string    `json:"activator"`
	Reference string    `json:"reference"`
	RefName   string    `json:"ref_name"`
	FreqKHz   float64   `json:"freq_khz"`
	Band      string    `json:"band"`
	Mode      string    `json:"mode"`
	Spotter   string    `json:"spotter"`
	Comment   string    `json:"comment"`
	TimeUTC   time.Time `json:"time_utc"`
}

// Data is the merged view returned to API callers.
type Data struct {
	Spots     []Spot    `json:"spots"`
	POTAOk    bool      `json:"pota_ok"`
	SOTAOk    bool      `json:"sota_ok"`
	FetchedAt time.Time `json:"fetched_at"`
}

// Cache holds the latest successfully fetched spots from each source
// independently — a failure on one source doesn't discard the other's
// data, mirroring spacewx.Cache's stale-but-valid philosophy per source.
type Cache struct {
	mu        sync.RWMutex
	potaSpots []Spot
	sotaSpots []Spot
	potaOK    bool
	sotaOK    bool
	fetchedAt time.Time
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) Get() Data {
	c.mu.RLock()
	defer c.mu.RUnlock()

	merged := make([]Spot, 0, len(c.potaSpots)+len(c.sotaSpots))
	merged = append(merged, c.potaSpots...)
	merged = append(merged, c.sotaSpots...)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].TimeUTC.After(merged[j].TimeUTC)
	})

	return Data{
		Spots:     merged,
		POTAOk:    c.potaOK,
		SOTAOk:    c.sotaOK,
		FetchedAt: c.fetchedAt,
	}
}

func (c *Cache) setPOTA(spots []Spot, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err == nil {
		c.potaSpots = spots
	}
	c.potaOK = err == nil
	c.fetchedAt = time.Now().UTC()
}

func (c *Cache) setSOTA(spots []Spot, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err == nil {
		c.sotaSpots = spots
	}
	c.sotaOK = err == nil
	c.fetchedAt = time.Now().UTC()
}

// Poll fetches both feeds immediately, then on the given interval until
// ctx is cancelled. The two fetches are independent, so one source being
// down never blocks the other's updates.
func Poll(ctx context.Context, cache *Cache, client *http.Client, potaURL, sotaURL string, interval time.Duration) {
	fetch := func() {
		spots, err := fetchPOTA(client, potaURL)
		cache.setPOTA(spots, err)

		spots, err = fetchSOTA(client, sotaURL)
		cache.setSOTA(spots, err)
	}

	fetch()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetch()
		}
	}
}
