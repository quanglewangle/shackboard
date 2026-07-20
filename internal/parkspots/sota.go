package parkspots

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/quanglewangle/shackboard/internal/cluster"
)

// sotaSpotJSON mirrors the fields shackboard uses from
// https://api2.sota.org.uk/api/spots/{count}/all. Note: "callsign" is
// actually the spotter, not the activator — "activatorCallsign" is the
// one to work / look up / match against the ADIF log. Also note
// "frequency" is in MHz here, unlike POTA's kHz.
type sotaSpotJSON struct {
	Callsign          string `json:"callsign"`
	ActivatorCallsign string `json:"activatorCallsign"`
	AssociationCode   string `json:"associationCode"`
	SummitCode        string `json:"summitCode"`
	SummitDetails     string `json:"summitDetails"`
	Frequency         string `json:"frequency"` // MHz, as a string
	Mode              string `json:"mode"`
	TimeStamp         string `json:"timeStamp"`
	Comments          string `json:"comments"`
}

func fetchSOTA(client *http.Client, feedURL string) ([]Spot, error) {
	req, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("sota: build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sota: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sota: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sota: read body: %w", err)
	}

	var raw []sotaSpotJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("sota: parse json: %w", err)
	}

	spots := make([]Spot, 0, len(raw))
	for _, p := range raw {
		freqMHz, _ := strconv.ParseFloat(p.Frequency, 64)
		freqKHz := freqMHz * 1000
		spots = append(spots, Spot{
			Program:   "SOTA",
			Activator: p.ActivatorCallsign,
			Reference: p.AssociationCode + "/" + p.SummitCode,
			RefName:   p.SummitDetails,
			FreqKHz:   freqKHz,
			Band:      cluster.BandForFreqKHz(freqKHz),
			Mode:      p.Mode,
			Spotter:   p.Callsign,
			Comment:   p.Comments,
			TimeUTC:   parseUTCTime(p.TimeStamp),
		})
	}
	return spots, nil
}
