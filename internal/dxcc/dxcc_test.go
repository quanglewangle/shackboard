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

func TestLocate(t *testing.T) {
	// cty.dat's own header line for Japan is
	// "Japan:  25:  45:  AS:  36.40: -138.38:  -9.0:  JA:" — cty.dat gives
	// longitude west-positive, so the standard (east-positive) value is
	// the negation, 138.38.
	entity, ok := Locate("JA1XYZ")
	if !ok {
		t.Fatal("Locate(JA1XYZ) not found")
	}
	if entity.Name != "Japan" || entity.Lat != 36.40 || entity.Lon != 138.38 {
		t.Errorf("Locate(JA1XYZ) = %+v, want {Japan 36.40 138.38}", entity)
	}

	if _, ok := Locate("QQ9ZZZ"); ok {
		t.Error("Locate(QQ9ZZZ) should not resolve")
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
