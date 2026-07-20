package adif

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name        string
		data        string
		wantQSOs    []QSO
		wantSkipped int
		wantRecords int
	}{
		{
			name: "header ending in EOH then one QSO",
			data: "ADIF Export<adif_ver:5>3.1.4<programid:6>Foobar<eoh>" +
				"<call:4>W1AW<band:3>20m<mode:3>SSB<eor>",
			wantQSOs:    []QSO{{Call: "W1AW", Band: "20m", Mode: "SSB"}},
			wantRecords: 1,
		},
		{
			name:        "no header at all",
			data:        "<call:5>G8GDS<band:3>40m<mode:2>CW<eor>",
			wantQSOs:    []QSO{{Call: "G8GDS", Band: "40m", Mode: "CW"}},
			wantRecords: 1,
		},
		{
			name: "record missing CALL is skipped, parsing continues",
			data: "<band:3>20m<mode:3>SSB<eor>" +
				"<call:4>W1AW<band:3>20m<eor>",
			wantQSOs:    []QSO{{Call: "W1AW", Band: "20m"}},
			wantSkipped: 1,
			wantRecords: 2,
		},
		{
			name:        "BAND absent, derived from FREQ (MHz)",
			data:        "<call:4>W1AW<freq:6>14.080<eor>",
			wantQSOs:    []QSO{{Call: "W1AW", Band: "20m"}},
			wantRecords: 1,
		},
		{
			name:        "case-insensitive tag names",
			data:        "<Call:4>W1AW<BAND:3>20m<mode:3>SSB<EOR>",
			wantQSOs:    []QSO{{Call: "W1AW", Band: "20m", Mode: "SSB"}},
			wantRecords: 1,
		},
		{
			name: "multiple records in one file",
			data: "<call:4>W1AW<band:3>20m<eor>" +
				"<call:5>G8GDS<band:3>40m<eor>" +
				"<call:6>VK2ABC<band:3>15m<eor>",
			wantQSOs: []QSO{
				{Call: "W1AW", Band: "20m"},
				{Call: "G8GDS", Band: "40m"},
				{Call: "VK2ABC", Band: "15m"},
			},
			wantRecords: 3,
		},
		{
			name:        "unparseable FREQ leaves Band empty, QSO still included",
			data:        "<call:4>W1AW<freq:3>bad<eor>",
			wantQSOs:    []QSO{{Call: "W1AW", Band: ""}},
			wantRecords: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Parse([]byte(tc.data))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if result.RecordCount != tc.wantRecords {
				t.Errorf("RecordCount = %d, want %d", result.RecordCount, tc.wantRecords)
			}
			if result.SkippedNoCall != tc.wantSkipped {
				t.Errorf("SkippedNoCall = %d, want %d", result.SkippedNoCall, tc.wantSkipped)
			}
			if len(result.QSOs) != len(tc.wantQSOs) {
				t.Fatalf("got %d QSOs, want %d: %+v", len(result.QSOs), len(tc.wantQSOs), result.QSOs)
			}
			for i, got := range result.QSOs {
				if got != tc.wantQSOs[i] {
					t.Errorf("QSO[%d] = %+v, want %+v", i, got, tc.wantQSOs[i])
				}
			}
		})
	}
}

func TestIndexWorkedAnyAndBand(t *testing.T) {
	idx := NewIndex()
	idx.Replace([]QSO{
		{Call: "W1AW", Band: "20m"},
		{Call: "W1AW", Band: "40m"},
		{Call: "G8GDS", Band: ""}, // unknown band, shouldn't register any band match
	}, time.Now())

	if !idx.WorkedAny("w1aw") {
		t.Error("expected WorkedAny(w1aw) true (case-insensitive)")
	}
	if !idx.WorkedBand("W1AW", "20M") {
		t.Error("expected WorkedBand(W1AW, 20M) true (case-insensitive band)")
	}
	if idx.WorkedBand("W1AW", "15m") {
		t.Error("expected WorkedBand(W1AW, 15m) false — never worked on 15m")
	}
	if !idx.WorkedAny("G8GDS") {
		t.Error("expected WorkedAny(G8GDS) true")
	}
	if idx.WorkedBand("G8GDS", "20m") {
		t.Error("expected WorkedBand(G8GDS, 20m) false — logged with empty band")
	}
	if idx.WorkedAny("VK2ABC") {
		t.Error("expected WorkedAny(VK2ABC) false — never logged")
	}
}
