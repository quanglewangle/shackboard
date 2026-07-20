// Package spacewx polls the N0NBH hamqsl.com feed for per-band propagation
// conditions and caches the result in memory.
package spacewx

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// BandCondition is a single band's Good/Fair/Poor rating for one time of day.
type BandCondition struct {
	Band  string `json:"band"`
	Day   string `json:"day"`
	Night string `json:"night"`
}

// Data is the cached, reshaped result of the feed.
type Data struct {
	UpdatedUTC string          `json:"updated_utc"`
	Bands      []BandCondition `json:"bands"`
	FetchedAt  time.Time       `json:"fetched_at"`
}

// solarXML mirrors only the fields we need from the hamqsl.com feed;
// encoding/xml silently ignores everything else (solarflux, kindex, etc).
type solarXML struct {
	XMLName xml.Name `xml:"solar"`
	Data    struct {
		Updated    string `xml:"updated"`
		Conditions struct {
			Band []struct {
				Name string `xml:"name,attr"`
				Time string `xml:"time,attr"`
				Cond string `xml:",chardata"`
			} `xml:"band"`
		} `xml:"calculatedconditions"`
	} `xml:"solardata"`
}

// Cache holds the latest successfully parsed Data. A failed poll keeps the
// previous good data and only records the error, so callers can serve
// stale-but-valid data instead of a hard failure.
type Cache struct {
	mu   sync.RWMutex
	data *Data
	err  error
}

func NewCache() *Cache {
	return &Cache{}
}

// Get returns the latest known-good data (nil if no poll has ever
// succeeded) and the most recent error, if any.
func (c *Cache) Get() (*Data, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data, c.err
}

func (c *Cache) set(d *Data, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d != nil {
		c.data = d
	}
	c.err = err
}

// Poll fetches the feed immediately, then on the given interval (floored at
// one hour to stay a light client of this free community feed) until ctx is
// cancelled.
func Poll(ctx context.Context, cache *Cache, client *http.Client, feedURL string, interval time.Duration) {
	if interval < time.Hour {
		interval = time.Hour
	}

	fetch := func() {
		d, err := fetchOnce(client, feedURL)
		cache.set(d, err)
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

func fetchOnce(client *http.Client, feedURL string) (d *Data, err error) {
	defer func() {
		if r := recover(); r != nil {
			d, err = nil, fmt.Errorf("spacewx: panic parsing feed: %v", r)
		}
	}()

	req, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("spacewx: build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spacewx: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spacewx: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("spacewx: read body: %w", err)
	}

	var parsed solarXML
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("spacewx: parse xml: %w", err)
	}

	byName := map[string]*BandCondition{}
	var order []string
	for _, b := range parsed.Data.Conditions.Band {
		bc, ok := byName[b.Name]
		if !ok {
			bc = &BandCondition{Band: b.Name}
			byName[b.Name] = bc
			order = append(order, b.Name)
		}
		switch b.Time {
		case "day":
			bc.Day = b.Cond
		case "night":
			bc.Night = b.Cond
		}
	}

	bands := make([]BandCondition, 0, len(order))
	for _, name := range order {
		bands = append(bands, *byName[name])
	}

	return &Data{
		UpdatedUTC: parsed.Data.Updated,
		Bands:      bands,
		FetchedAt:  time.Now().UTC(),
	}, nil
}
