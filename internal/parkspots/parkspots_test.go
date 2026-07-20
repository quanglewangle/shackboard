package parkspots

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPOTA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{
			"activator": "W1AW",
			"frequency": "14080.0",
			"mode": "CW",
			"reference": "US-2507",
			"parkName": null,
			"name": "Some State Park",
			"spotTime": "2026-07-20T20:14:50",
			"spotter": "K1ABC",
			"comments": "QSY 20m CW"
		}]`))
	}))
	defer srv.Close()

	spots, err := fetchPOTA(http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatalf("fetchPOTA: %v", err)
	}
	if len(spots) != 1 {
		t.Fatalf("got %d spots, want 1", len(spots))
	}
	s := spots[0]
	if s.Program != "POTA" {
		t.Errorf("Program = %q, want POTA", s.Program)
	}
	if s.Activator != "W1AW" {
		t.Errorf("Activator = %q, want W1AW", s.Activator)
	}
	if s.RefName != "Some State Park" {
		t.Errorf("RefName = %q, want %q (should use 'name', not null 'parkName')", s.RefName, "Some State Park")
	}
	if s.FreqKHz != 14080.0 {
		t.Errorf("FreqKHz = %v, want 14080.0", s.FreqKHz)
	}
	if s.Band != "20m" {
		t.Errorf("Band = %q, want 20m", s.Band)
	}
	if s.TimeUTC.IsZero() {
		t.Error("TimeUTC should parse POTA's suffixless timestamp format")
	}
}

func TestFetchSOTA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{
			"callsign": "KW7MM",
			"activatorCallsign": "W7DLZ",
			"associationCode": "W6",
			"summitCode": "WH-006",
			"summitDetails": "County Line Hill, 3432m, 10 points",
			"frequency": "14.059",
			"mode": "CW",
			"timeStamp": "2026-07-20T20:04:02Z",
			"comments": "13 dB SNR"
		}]`))
	}))
	defer srv.Close()

	spots, err := fetchSOTA(http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatalf("fetchSOTA: %v", err)
	}
	if len(spots) != 1 {
		t.Fatalf("got %d spots, want 1", len(spots))
	}
	s := spots[0]
	if s.Program != "SOTA" {
		t.Errorf("Program = %q, want SOTA", s.Program)
	}
	if s.Activator != "W7DLZ" {
		t.Errorf("Activator = %q, want W7DLZ (activatorCallsign, not callsign)", s.Activator)
	}
	if s.Spotter != "KW7MM" {
		t.Errorf("Spotter = %q, want KW7MM (callsign field)", s.Spotter)
	}
	if s.Reference != "W6/WH-006" {
		t.Errorf("Reference = %q, want W6/WH-006", s.Reference)
	}
	if s.FreqKHz != 14059.0 {
		t.Errorf("FreqKHz = %v, want 14059.0 (MHz*1000)", s.FreqKHz)
	}
	if s.Band != "20m" {
		t.Errorf("Band = %q, want 20m", s.Band)
	}
	if s.TimeUTC.IsZero() {
		t.Error("TimeUTC should parse SOTA's Z-suffixed timestamp format")
	}
}

func TestFetchSOTAMalformedFrequency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"activatorCallsign":"W7DLZ","frequency":"","timeStamp":"2026-07-20T20:04:02Z"}]`))
	}))
	defer srv.Close()

	spots, err := fetchSOTA(http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatalf("fetchSOTA: %v", err)
	}
	if len(spots) != 1 {
		t.Fatalf("got %d spots, want 1 (malformed frequency shouldn't drop the spot)", len(spots))
	}
	if spots[0].Band != "" {
		t.Errorf("Band = %q, want empty for unparseable frequency", spots[0].Band)
	}
}

func TestCacheStaleOnError(t *testing.T) {
	cache := NewCache()
	good := []Spot{{Program: "POTA", Activator: "W1AW"}}

	cache.setPOTA(good, nil)
	cache.setSOTA(nil, nil)

	data := cache.Get()
	if !data.POTAOk || len(data.Spots) != 1 {
		t.Fatalf("after good POTA fetch: POTAOk=%v spots=%v", data.POTAOk, data.Spots)
	}

	// A subsequent failed POTA fetch should keep the old spots and just
	// flip POTAOk false — SOTA's independent state is untouched.
	cache.setPOTA(nil, fmt.Errorf("boom"))
	data = cache.Get()
	if data.POTAOk {
		t.Error("expected POTAOk false after failed fetch")
	}
	if len(data.Spots) != 1 || data.Spots[0].Activator != "W1AW" {
		t.Errorf("expected stale POTA spot to survive a failed fetch, got %v", data.Spots)
	}
}
