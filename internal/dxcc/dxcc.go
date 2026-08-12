// Package dxcc resolves a callsign to its DXCC entity (country) name using
// the "cty.dat" prefix table published by country-files.com (AD1C), which
// is maintained specifically for embedding in ham radio software. See
// cty.dat's own comments for provenance; it's re-fetched from
// https://www.country-files.com/cty/cty.dat occasionally, not written by
// hand — don't hand-edit it, replace the whole file instead.
package dxcc

import (
	_ "embed"
	"regexp"
	"strings"
)

//go:embed cty.dat
var rawCty string

// modifierRE strips the CQ-zone/ITU-zone/lat-lon/continent/UTC-offset
// override suffixes cty.dat allows on individual aliases, e.g. the
// "(23)[42]" in "3H0(23)[42]" — irrelevant here, we only want the prefix.
var modifierRE = regexp.MustCompile(`\([^)]*\)|\[[^\]]*\]|<[^>]*>|\{[^}]*\}|~[^~]*~`)

var (
	// prefixTable maps a literal prefix (e.g. "W", "KH6") to its DXCC
	// entity name. Looked up via longest-prefix match.
	prefixTable = map[string]string{}
	// exactTable maps a full, specific callsign (cty.dat aliases prefixed
	// with "=") to its DXCC entity name — used for portable operations,
	// expeditions, etc. that don't follow the entity's normal prefix.
	exactTable = map[string]string{}
)

func init() {
	parseCty(rawCty)
}

// parseCty parses the cty.dat format: each entity is a name/metadata
// header terminated by ':', followed by a comma-separated alias list
// terminated by ';'. Since ';' only ever appears as that terminator,
// splitting the whole file on it recovers one chunk per entity; each
// chunk's header has exactly 8 colon-delimited fields (name, CQ zone,
// ITU zone, continent, lat, lon, UTC offset, primary prefix) followed by
// the alias list, so SplitN(chunk, ":", 9) isolates the alias text.
func parseCty(data string) {
	for _, chunk := range strings.Split(data, ";") {
		fields := strings.SplitN(chunk, ":", 9)
		if len(fields) != 9 {
			continue // trailing whitespace after the last entity, etc.
		}
		country := strings.TrimSpace(fields[0])
		if country == "" {
			continue
		}
		for _, alias := range strings.Split(fields[8], ",") {
			alias = modifierRE.ReplaceAllString(alias, "")
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if strings.HasPrefix(alias, "=") {
				exactTable[strings.ToUpper(alias[1:])] = country
			} else {
				prefixTable[strings.ToUpper(alias)] = country
			}
		}
	}
}

// operatingSuffixes are portable/mobile designators that appear after a
// "/" in a callsign without indicating a different DXCC entity (e.g.
// "G4ABC/P", "W1AW/MM"). Split parts matching one of these are ignored
// when guessing which side of a "/" callsign carries the location.
var operatingSuffixes = map[string]bool{
	"P": true, "M": true, "MM": true, "AM": true, "A": true,
	"QRP": true, "QRPP": true, "LGT": true, "R": true,
}

// Country returns the DXCC entity name for call, or "" if it can't be
// resolved. Handles portable-style calls (e.g. "F/W1AW", "W1AW/KH6") by
// picking whichever "/"-separated part yields the most specific prefix
// match, which is the same heuristic contest loggers use.
func Country(call string) string {
	call = strings.ToUpper(strings.TrimSpace(call))
	if call == "" {
		return ""
	}
	if country, ok := exactTable[call]; ok {
		return country
	}

	if !strings.Contains(call, "/") {
		return prefixMatch(call)
	}

	var best, bestPrefix string
	for _, part := range strings.Split(call, "/") {
		if part == "" || operatingSuffixes[part] {
			continue
		}
		if country, prefix := prefixMatchLen(part); len(prefix) > len(bestPrefix) {
			best, bestPrefix = country, prefix
		}
	}
	return best
}

func prefixMatch(call string) string {
	country, _ := prefixMatchLen(call)
	return country
}

// prefixMatchLen returns the entity for the longest prefix of call found
// in prefixTable, along with the matched prefix itself (so callers can
// compare specificity between candidates).
func prefixMatchLen(call string) (country, prefix string) {
	for i := len(call); i >= 1; i-- {
		if c, ok := prefixTable[call[:i]]; ok {
			return c, call[:i]
		}
	}
	return "", ""
}
