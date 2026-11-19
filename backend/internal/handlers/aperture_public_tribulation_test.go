package handlers

import (
	"sort"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
)

func TestPublicApertureTribulationDamageFraction(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		want       float64
		configured bool
	}{
		{name: "configured ratio", value: "0.25", want: 0.25, configured: true},
		{name: "full wall damage", value: "1", want: 1, configured: true},
		{name: "missing", value: "", configured: false},
		{name: "zero", value: "0", configured: false},
		{name: "over one", value: "1.01", configured: false},
		{name: "not a number", value: "nope", configured: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PUBLIC_APERTURE_TRIBULATION_HP_DAMAGE_FRACTION", tt.value)
			got, configured := publicApertureTribulationDamageFraction()
			if configured != tt.configured || got != tt.want {
				t.Fatalf("publicApertureTribulationDamageFraction() = (%v, %v), want (%v, %v)", got, configured, tt.want, tt.configured)
			}
		})
	}
}

func TestPublicApertureTribulationPathUsesDominantDaoMark(t *testing.T) {
	raw := []byte(`{"wood": 3, "fire": 9, "water": 9, "metal": 2}`)
	paths := []string{gamefx.CleanPath("fire"), gamefx.CleanPath("water")}
	sort.Strings(paths)
	if got := publicApertureTribulationPath(raw); got != paths[0] {
		t.Fatalf("publicApertureTribulationPath() = %q, want %q", got, paths[0])
	}
	if got := publicApertureTribulationPath([]byte(`{"wood": 0, "fire": -1}`)); got != "environment" {
		t.Fatalf("publicApertureTribulationPath() without positive marks = %q, want environment", got)
	}
}
