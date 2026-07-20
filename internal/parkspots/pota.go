package parkspots

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/quanglewangle/shackboard/internal/cluster"
)

// potaSpotJSON mirrors the fields shackboard uses from
// https://api.pota.app/spot/activator. Note: the park name is in "name",
// not "parkName" — the latter is often null in practice.
type potaSpotJSON struct {
	Activator string `json:"activator"`
	Frequency string `json:"frequency"` // kHz, as a string
	Mode      string `json:"mode"`
	Reference string `json:"reference"`
	Name      string `json:"name"`
	SpotTime  string `json:"spotTime"`
	Spotter   string `json:"spotter"`
	Comments  string `json:"comments"`
}

func fetchPOTA(client *http.Client, feedURL string) ([]Spot, error) {
	req, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("pota: build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pota: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pota: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pota: read body: %w", err)
	}

	var raw []potaSpotJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("pota: parse json: %w", err)
	}

	spots := make([]Spot, 0, len(raw))
	for _, p := range raw {
		freqKHz, _ := strconv.ParseFloat(p.Frequency, 64)
		spots = append(spots, Spot{
			Program:   "POTA",
			Activator: p.Activator,
			Reference: p.Reference,
			RefName:   p.Name,
			FreqKHz:   freqKHz,
			Band:      cluster.BandForFreqKHz(freqKHz),
			Mode:      p.Mode,
			Spotter:   p.Spotter,
			Comment:   p.Comments,
			TimeUTC:   parseUTCTime(p.SpotTime),
		})
	}
	return spots, nil
}
