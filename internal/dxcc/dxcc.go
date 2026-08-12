// Package dxcc resolves a callsign to its DXCC entity (country name plus
// an approximate lat/lon) using the "cty.dat" prefix table published by
// country-files.com (AD1C), which is maintained specifically for embedding
// in ham radio software. See cty.dat's own comments for provenance; it's
// re-fetched from https://www.country-files.com/cty/cty.dat occasionally,
// not written by hand — don't hand-edit it, replace the whole file instead.
package dxcc

import (
	_ "embed"
	"regexp"
	"strconv"
	"strings"
)

//go:embed cty.dat
var rawCty string

// modifierRE strips the CQ-zone/ITU-zone/lat-lon/continent/UTC-offset
// override suffixes cty.dat allows on individual aliases, e.g. the
// "(23)[42]" in "3H0(23)[42]" — irrelevant here, we only want the prefix.
var modifierRE = regexp.MustCompile(`\([^)]*\)|\[[^\]]*\]|<[^>]*>|\{[^}]*\}|~[^~]*~`)

// Entity is a DXCC entity: a country/territory name plus its approximate
// center coordinates (the entity's average location, not any individual
// station's — cty.dat doesn't carry per-station precision).
type Entity struct {
	Name string
	Lat  float64
	Lon  float64
}

var (
	// prefixTable maps a literal prefix (e.g. "W", "KH6") to its DXCC
	// entity. Looked up via longest-prefix match.
	prefixTable = map[string]Entity{}
	// exactTable maps a full, specific callsign (cty.dat aliases prefixed
	// with "=") to its DXCC entity — used for portable operations,
	// expeditions, etc. that don't follow the entity's normal prefix.
	exactTable = map[string]Entity{}
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
//
// cty.dat's longitude is given with west positive (the opposite of the
// usual GPS/web-mapping convention), so it's negated when stored here.
func parseCty(data string) {
	for _, chunk := range strings.Split(data, ";") {
		fields := strings.SplitN(chunk, ":", 9)
		if len(fields) != 9 {
			continue // trailing whitespace after the last entity, etc.
		}
		name := strings.TrimSpace(fields[0])
		if name == "" {
			continue
		}
		lat, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
		lon, _ := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)
		entity := Entity{Name: name, Lat: lat, Lon: -lon}

		for _, alias := range strings.Split(fields[8], ",") {
			alias = modifierRE.ReplaceAllString(alias, "")
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if strings.HasPrefix(alias, "=") {
				exactTable[strings.ToUpper(alias[1:])] = entity
			} else {
				prefixTable[strings.ToUpper(alias)] = entity
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
// resolved. See Locate for the full entity including coordinates.
func Country(call string) string {
	entity, _ := Locate(call)
	return entity.Name
}

// Locate resolves call to its DXCC entity (name + approximate lat/lon),
// reporting ok=false if the callsign can't be matched. Handles
// portable-style calls (e.g. "F/W1AW", "W1AW/KH6") by picking whichever
// "/"-separated part yields the most specific prefix match, which is the
// same heuristic contest loggers use.
func Locate(call string) (entity Entity, ok bool) {
	call = strings.ToUpper(strings.TrimSpace(call))
	if call == "" {
		return Entity{}, false
	}
	if e, found := exactTable[call]; found {
		return e, true
	}

	if !strings.Contains(call, "/") {
		return prefixMatchLen(call)
	}

	var bestPrefix string
	for _, part := range strings.Split(call, "/") {
		if part == "" || operatingSuffixes[part] {
			continue
		}
		if e, prefix, found := prefixMatchLenRaw(part); found && len(prefix) > len(bestPrefix) {
			entity, ok, bestPrefix = e, true, prefix
		}
	}
	return entity, ok
}

func prefixMatchLen(call string) (Entity, bool) {
	e, _, ok := prefixMatchLenRaw(call)
	return e, ok
}

// prefixMatchLenRaw returns the entity for the longest prefix of call
// found in prefixTable, along with the matched prefix itself (so callers
// can compare specificity between candidates).
func prefixMatchLenRaw(call string) (entity Entity, prefix string, ok bool) {
	for i := len(call); i >= 1; i-- {
		if e, found := prefixTable[call[:i]]; found {
			return e, call[:i], true
		}
	}
	return Entity{}, "", false
}
