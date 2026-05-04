package properties

import (
	"testing"

	"koditon/internal/db"
)

func TestShortcutAdBuildingEnergyClass(t *testing.T) {
	tests := map[string]struct {
		rowEnergy string
		payload   rawMap
		want      string
	}{
		"normalizes row provider code": {rowEnergy: "E13_G", want: "G2013"},
		"uses property fallback":       {payload: rawMap{"property": map[string]any{"energyClass": "E18_A"}}, want: "A2018"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			row := db.GetShortcutAdUnifiedDetailRow{}
			if tt.rowEnergy != "" {
				row.ShortcutAdEnergyClass = &tt.rowEnergy
			}
			if got := shortcutAdBuilding(row, tt.payload, Location{}).EnergyClass; got != tt.want {
				t.Fatalf("EnergyClass = %q, want %q", got, tt.want)
			}
		})
	}
}
