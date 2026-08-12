package dxcc

import "testing"

// Cases below are checked against real cty.dat entries (see cty.dat itself
// for the source lines), not invented prefixes.
func TestCountry(t *testing.T) {
	cases := []struct {
		name string
		call string
		want string
	}{
		{"plain US call", "W1AW", "United States"},
		{"plain England call", "G4ABC", "England"},
		{"plain Japan call", "JA1XYZ", "Japan"},
		{"Hawaii own prefix", "KH6ABC", "Hawaii"},
		{"exact-match override wins over prefix table", "K2GT", "Hawaii"},
		{"portable-in-country suffix ignored", "G4ABC/P", "England"},
		{"maritime mobile suffix ignored", "W1AW/MM", "United States"},
		{"home call operating from another entity", "F/W1AW", "France"},
		{"home call operating in a more specific US entity", "W1AW/KH6", "Hawaii"},
		{"unknown prefix", "QQ9ZZZ", ""},
		{"empty call", "", ""},
		{"lowercase normalized", "w1aw", "United States"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Country(c.call); got != c.want {
				t.Errorf("Country(%q) = %q, want %q", c.call, got, c.want)
			}
		})
	}
}

func TestParseCtyPopulatesTables(t *testing.T) {
	if len(prefixTable) < 300 {
		t.Errorf("prefixTable has only %d entries, expected hundreds", len(prefixTable))
	}
	if len(exactTable) < 300 {
		t.Errorf("exactTable has only %d entries, expected hundreds", len(exactTable))
	}
}
