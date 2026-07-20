// Package qrzlogbook syncs a shackboard adif.Index directly from a user's
// QRZ Logbook (QRZ's Logbook Data API — a separate service from the QRZ
// XML callsign-lookup API the sibling qrzlook service uses), so "worked
// before" state never requires a manual ADIF export/upload.
package qrzlogbook

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/quanglewangle/shackboard/internal/adif"
)

// FetchAndReplace fetches the whole logbook via ACTION=FETCH and replaces
// idx's contents. Returns the number of QSOs loaded.
//
// QRZ's response is a set of "&"-delimited KEY=VALUE fields with ADIF
// always last (so its content, which may itself contain "&" or "=", never
// needs escaping there) — but the ADIF value itself IS HTML-entity-escaped
// ("&lt;"/"&gt;" instead of "<"/">"), confirmed by inspecting a real
// response byte-for-byte. Must be html.UnescapeString'd before parsing, or
// adif.Parse silently finds zero records (no <eor>/<eoh> to match).
func FetchAndReplace(client *http.Client, feedURL, apiKey string, idx *adif.Index) (int, error) {
	// No OPTION means "no filter" — fetches the whole logbook. Confirmed
	// against the live API: OPTION=MAX:0 is NOT "unlimited", it means
	// "zero records" (RESULT=OK, COUNT=<real total>, but no ADIF field
	// at all) — a real gotcha that would have silently synced an empty
	// index every single time.
	form := url.Values{
		"KEY":    {apiKey},
		"ACTION": {"FETCH"},
	}

	resp, err := client.PostForm(feedURL, form)
	if err != nil {
		return 0, fmt.Errorf("qrzlogbook: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("qrzlogbook: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("qrzlogbook: read body: %w", err)
	}

	marker := []byte("ADIF=")
	i := bytes.Index(body, marker)
	if i < 0 {
		// A genuinely empty logbook returns RESULT=OK with no ADIF field
		// at all, not an error — only treat it as a failure if RESULT
		// isn't OK.
		if bytes.Contains(body, []byte("RESULT=OK")) {
			idx.Replace(nil, time.Now().UTC())
			return 0, nil
		}
		return 0, fmt.Errorf("qrzlogbook: FETCH failed: %s", strings.TrimSpace(string(body)))
	}

	adifText := html.UnescapeString(string(body[i+len(marker):]))

	result, err := adif.Parse([]byte(adifText))
	if err != nil {
		return 0, fmt.Errorf("qrzlogbook: parse adif: %w", err)
	}

	idx.Replace(result.QSOs, time.Now().UTC())
	return len(result.QSOs), nil
}

// Poll fetches immediately, then on the given interval (floored at one
// hour — this is a personal logbook, not a public feed, but there's no
// reason to hammer QRZ's API for data that only changes when new QSOs are
// logged) until ctx is cancelled. A no-op if apiKey is empty, so callers
// can unconditionally launch this as a goroutine.
func Poll(ctx context.Context, idx *adif.Index, client *http.Client, feedURL, apiKey string, interval time.Duration) {
	if apiKey == "" {
		return
	}
	if interval < time.Hour {
		interval = time.Hour
	}

	fetch := func() {
		n, err := FetchAndReplace(client, feedURL, apiKey, idx)
		if err != nil {
			log.Printf("qrzlogbook: %v", err)
			return
		}
		log.Printf("qrzlogbook: synced %d QSOs", n)
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
