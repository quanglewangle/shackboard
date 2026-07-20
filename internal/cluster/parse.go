package cluster

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// spotRe matches a standard DX cluster spot line, e.g.:
//
//	DX de SP2DMB:    70154.0  EA5TT        JO92CF<>IM99 -1dB              1918Z
var spotRe = regexp.MustCompile(`^DX de ([A-Z0-9\-/#]+):\s+([\d.]+)\s+([A-Z0-9\-/#]+)\s+(.*?)\s*(\d{4})Z\s*$`)

// parseSpotLine parses a single line of cluster output. Most lines aren't
// spots (command prompts, WWV announcements, propagation tables, blanks) —
// that's the expected common case, and parseSpotLine just returns false for
// those rather than treating them as an error.
func parseSpotLine(line string) (Spot, bool) {
	m := spotRe.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
	if m == nil {
		return Spot{}, false
	}

	freq, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return Spot{}, false
	}

	return Spot{
		Spotter:    m[1],
		FreqKHz:    freq,
		Band:       BandForFreqKHz(freq),
		DXCall:     m[3],
		Comment:    strings.TrimSpace(m[4]),
		TimeUTC:    m[5] + "Z",
		ReceivedAt: time.Now().UTC(),
	}, true
}

type bandRange struct {
	loKHz, hiKHz float64
	name         string
}

// bandRanges is a simple static amateur-band table — good enough for a UI
// badge, no external band-plan dependency needed.
var bandRanges = []bandRange{
	{1800, 2000, "160m"},
	{3500, 4000, "80m"},
	{5330, 5410, "60m"},
	{7000, 7300, "40m"},
	{10100, 10150, "30m"},
	{14000, 14350, "20m"},
	{18068, 18168, "17m"},
	{21000, 21450, "15m"},
	{24890, 24990, "12m"},
	{28000, 29700, "10m"},
	{50000, 54000, "6m"},
	{70000, 70500, "4m"},
	{144000, 148000, "2m"},
	{420000, 450000, "70cm"},
}

// BandForFreqKHz maps a frequency in kHz to an amateur band name (e.g.
// "20m"), or "" if it doesn't fall in any known band. Shared with the adif
// and parkspots packages so there's one band table, not three.
func BandForFreqKHz(khz float64) string {
	for _, r := range bandRanges {
		if khz >= r.loKHz && khz <= r.hiKHz {
			return r.name
		}
	}
	return ""
}
