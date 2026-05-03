package properties

import "testing"

func TestDisplayEnergyClass(t *testing.T) {
	tests := map[string]string{
		"E13_G": "G2013",
		"e18_a": "A2018",
		"D2007": "D2007",
		"Energialuokka: E2018, Energiatodistuksen Voimassaoloaika: 20.04.2032": "E2018",
	}
	for input, want := range tests {
		if got := displayEnergyClass(input); got != want {
			t.Fatalf("displayEnergyClass(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDisplayCondition(t *testing.T) {
	tests := map[string]string{
		"GOOD":         "Good",
		"SATISFACTORY": "Satisfactory",
		"hyvä":         "Good",
	}
	for input, want := range tests {
		if got := displayCondition(input); got != want {
			t.Fatalf("displayCondition(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDisplayPropertyType(t *testing.T) {
	tests := map[string]string{
		"APARTMENT_HOUSE": "Apartment house",
		"row_house":       "Row house",
		"DETACHED_HOUSE":  "Detached house",
	}
	for input, want := range tests {
		if got := displayPropertyType(input); got != want {
			t.Fatalf("displayPropertyType(%q) = %q, want %q", input, got, want)
		}
	}
}
