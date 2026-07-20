package cluster

import "testing"

func TestParseSpotLine(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		wantOK  bool
		wantDX  string
		wantFrq float64
		wantBnd string
	}{
		{
			name:    "real captured line",
			line:    "DX de SP2DMB:    70154.0  EA5TT        JO92CF<>IM99 -1dB              1918Z",
			wantOK:  true,
			wantDX:  "EA5TT",
			wantFrq: 70154.0,
			wantBnd: "4m",
		},
		{
			name:    "extra whitespace",
			line:    "DX de W1AW:      14205.0    G8GDS      CQ                     1234Z",
			wantOK:  true,
			wantDX:  "G8GDS",
			wantFrq: 14205.0,
			wantBnd: "20m",
		},
		{
			name:   "lowercase spotter call",
			line:   "DX de ve7cc:     7002.0  W1AW         CW                      0100Z",
			wantOK: false, // regex requires uppercase callsigns, as clusters normally send
		},
		{
			name:    "missing comment field",
			line:    "DX de K1ABC:     21200.0  N2XYZ                                0000Z",
			wantOK:  true,
			wantDX:  "N2XYZ",
			wantFrq: 21200.0,
			wantBnd: "15m",
		},
		{
			name:   "command prompt, not a spot",
			line:   "VE7CC de VE7CC 20-Jul-2026 1918Z >",
			wantOK: false,
		},
		{
			name:   "blank line",
			line:   "",
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spot, ok := parseSpotLine(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if spot.DXCall != tc.wantDX {
				t.Errorf("DXCall = %q, want %q", spot.DXCall, tc.wantDX)
			}
			if spot.FreqKHz != tc.wantFrq {
				t.Errorf("FreqKHz = %v, want %v", spot.FreqKHz, tc.wantFrq)
			}
			if spot.Band != tc.wantBnd {
				t.Errorf("Band = %q, want %q", spot.Band, tc.wantBnd)
			}
		})
	}
}
