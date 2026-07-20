package qrzlogbook

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quanglewangle/shackboard/internal/adif"
)

func TestFetchAndReplaceUnescapesADIF(t *testing.T) {
	// Real shape confirmed against the live API: ADIF is the last field,
	// and its content is HTML-entity-escaped.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(
			"RESULT=OK&COUNT=1&ADIF=&lt;call:4&gt;W1AW&lt;band:3&gt;20m&lt;mode:3&gt;SSB&lt;eor&gt;",
		))
	}))
	defer srv.Close()

	idx := adif.NewIndex()
	n, err := FetchAndReplace(http.DefaultClient, srv.URL, "testkey", idx)
	if err != nil {
		t.Fatalf("FetchAndReplace: %v", err)
	}
	if n != 1 {
		t.Fatalf("got %d QSOs, want 1", n)
	}
	if !idx.WorkedBand("W1AW", "20m") {
		t.Error("expected W1AW/20m to be worked after sync — ADIF wasn't unescaped/parsed correctly")
	}
}

func TestFetchAndReplaceEmptyLogbookIsNotAnError(t *testing.T) {
	// A genuinely empty logbook returns RESULT=OK with no ADIF field at
	// all — must not be confused with a real failure.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("RESULT=OK&COUNT=0"))
	}))
	defer srv.Close()

	idx := adif.NewIndex()
	n, err := FetchAndReplace(http.DefaultClient, srv.URL, "testkey", idx)
	if err != nil {
		t.Fatalf("FetchAndReplace: unexpected error for an empty-but-OK logbook: %v", err)
	}
	if n != 0 {
		t.Fatalf("got %d QSOs, want 0", n)
	}
}

func TestFetchAndReplaceFailureLeavesIndexAlone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("RESULT=FAIL&REASON=Invalid API Key"))
	}))
	defer srv.Close()

	idx := adif.NewIndex()
	idx.Replace([]adif.QSO{{Call: "W1AW", Band: "20m"}}, time.Now())

	_, err := FetchAndReplace(http.DefaultClient, srv.URL, "badkey", idx)
	if err == nil {
		t.Fatal("expected an error for a RESULT=FAIL response")
	}

	// The prior successful sync's data must survive a failed fetch.
	if !idx.WorkedBand("W1AW", "20m") {
		t.Error("expected prior index contents to survive a failed fetch")
	}
}
