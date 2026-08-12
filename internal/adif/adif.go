// Package adif parses ADIF log files (the standard amateur radio logging
// interchange format) into a lightweight worked-before index.
package adif

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/quanglewangle/shackboard/internal/cluster"
)

// QSO is the subset of an ADIF record this package cares about.
type QSO struct {
	Call string // uppercased, trimmed
	Band string // lowercase, e.g. "20m" — matches cluster.BandForFreqKHz's convention
	Mode string // display only, not used for worked-before matching
	Date string // YYYYMMDD from QSO_DATE, "" if absent — not QSO_DATE_OFF or
	// QRZ's own qrzcom_qso_download_date (when QRZ ingested the record,
	// not when the contact happened — confirmed against a real QRZ
	// Logbook export, both fields are present and easy to mix up).
	Time string // HHMM from TIME_ON, truncated if the source gives HHMMSS
}

type ParseResult struct {
	QSOs          []QSO
	RecordCount   int
	SkippedNoCall int
}

var (
	eohRe = regexp.MustCompile(`(?i)<eoh>`)
	eorRe = regexp.MustCompile(`(?i)<eor>`)
	tagRe = regexp.MustCompile(`(?i)<([A-Za-z0-9_]+):(\d+)(?::[A-Za-z]+)?>`)
)

// Parse extracts QSOs from raw ADIF file bytes. The header (if any) is
// everything up to and including the first <EOH> tag; a file with no
// header at all is also valid per the ADIF spec. Records missing CALL are
// skipped rather than failing the whole parse, since one bad record in a
// multi-thousand-QSO log shouldn't discard the rest.
func Parse(data []byte) (result ParseResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("adif: panic parsing file: %v", r)
		}
	}()

	body := data
	if loc := eohRe.FindIndex(data); loc != nil {
		body = data[loc[1]:]
	}

	records := splitRecords(body)
	result.RecordCount = len(records)

	for _, rec := range records {
		fields := parseFields(rec)

		call := strings.ToUpper(strings.TrimSpace(fields["CALL"]))
		if call == "" {
			result.SkippedNoCall++
			continue
		}

		band := strings.ToLower(strings.TrimSpace(fields["BAND"]))
		if band == "" {
			if freqMHz, ferr := strconv.ParseFloat(strings.TrimSpace(fields["FREQ"]), 64); ferr == nil {
				band = cluster.BandForFreqKHz(freqMHz * 1000)
			}
		}

		timeOn := strings.TrimSpace(fields["TIME_ON"])
		if len(timeOn) > 4 {
			timeOn = timeOn[:4]
		}

		result.QSOs = append(result.QSOs, QSO{
			Call: call,
			Band: band,
			Mode: strings.ToUpper(strings.TrimSpace(fields["MODE"])),
			Date: strings.TrimSpace(fields["QSO_DATE"]),
			Time: timeOn,
		})
	}

	return result, nil
}

// splitRecords splits body on <eor> markers. Any trailing content after
// the last one (an incomplete/dangling record) is discarded.
func splitRecords(body []byte) [][]byte {
	locs := eorRe.FindAllIndex(body, -1)
	if locs == nil {
		return nil
	}
	records := make([][]byte, 0, len(locs))
	start := 0
	for _, loc := range locs {
		records = append(records, body[start:loc[0]])
		start = loc[1]
	}
	return records
}

// parseFields scans a single record for <TAG:LEN[:TYPE]>VALUE tokens. This
// offset-driven scan (read exactly LEN bytes after each tag) tolerates
// arbitrary whitespace between tokens without needing line-based parsing.
func parseFields(rec []byte) map[string]string {
	fields := map[string]string{}
	for _, m := range tagRe.FindAllSubmatchIndex(rec, -1) {
		name := strings.ToUpper(string(rec[m[2]:m[3]]))
		length, err := strconv.Atoi(string(rec[m[4]:m[5]]))
		if err != nil || length < 0 {
			continue
		}
		valStart := m[1] // end of the whole tag match, i.e. right after '>'
		valEnd := valStart + length
		if valEnd > len(rec) {
			// Defensive guard against a corrupt/truncated length field.
			valEnd = len(rec)
		}
		fields[name] = string(rec[valStart:valEnd])
	}
	return fields
}
