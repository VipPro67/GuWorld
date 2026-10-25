package handlers

import (
	"testing"
)

func TestResolveStartingRegion(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantRegion   string
		wantX        int
		wantY        int
		wantOk       bool
	}{
		{
			name:       "empty defaults to tan_thu_coc_nam_cuong",
			input:      "",
			wantRegion: "tan_thu_coc_nam_cuong",
			wantX:      5000,
			wantY:      3000,
			wantOk:     true,
		},
		{
			name:       "direct tan_thu_coc_nam_cuong",
			input:      "tan_thu_coc_nam_cuong",
			wantRegion: "tan_thu_coc_nam_cuong",
			wantX:      5000,
			wantY:      3000,
			wantOk:     true,
		},
		{
			name:       "legacy hub_nam_cuong maps to tan_thu_coc_nam_cuong",
			input:      "hub_nam_cuong",
			wantRegion: "tan_thu_coc_nam_cuong",
			wantX:      5000,
			wantY:      3000,
			wantOk:     true,
		},
		{
			name:       "hub_thuong_gia_thanh with whitespace and uppercase",
			input:      "  HUB_THUONG_GIA_THANH  ",
			wantRegion: "tan_thu_coc_nam_cuong",
			wantX:      5000,
			wantY:      3000,
			wantOk:     true,
		},
		{
			name:       "trung chau tan thu coc",
			input:      "tan_thu_coc_trung_chau",
			wantRegion: "tan_thu_coc_trung_chau",
			wantX:      13000,
			wantY:      3000,
			wantOk:     true,
		},
		{
			name:       "legacy hub_trung_chau maps to tan_thu_coc_trung_chau",
			input:      "hub_trung_chau",
			wantRegion: "tan_thu_coc_trung_chau",
			wantX:      13000,
			wantY:      3000,
			wantOk:     true,
		},
		{
			name:       "bac nguyen tan thu coc",
			input:      "tan_thu_coc_bac_nguyen",
			wantRegion: "tan_thu_coc_bac_nguyen",
			wantX:      23000,
			wantY:      3000,
			wantOk:     true,
		},
		{
			name:       "tay mac tan thu coc",
			input:      "tan_thu_coc_tay_mac",
			wantRegion: "tan_thu_coc_tay_mac",
			wantX:      5000,
			wantY:      13000,
			wantOk:     true,
		},
		{
			name:       "dong hai tan thu coc",
			input:      "tan_thu_coc_dong_hai",
			wantRegion: "tan_thu_coc_dong_hai",
			wantX:      15000,
			wantY:      13000,
			wantOk:     true,
		},
		{
			name:       "invalid region",
			input:      "non_existent_zone",
			wantRegion: "",
			wantX:      0,
			wantY:      0,
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRegion, gotX, gotY, gotOk := resolveStartingRegion(tt.input)
			if gotOk != tt.wantOk {
				t.Fatalf("resolveStartingRegion(%q) ok = %v, want %v", tt.input, gotOk, tt.wantOk)
			}
			if gotRegion != tt.wantRegion {
				t.Errorf("resolveStartingRegion(%q) region = %q, want %q", tt.input, gotRegion, tt.wantRegion)
			}
			if gotX != tt.wantX || gotY != tt.wantY {
				t.Errorf("resolveStartingRegion(%q) coords = (%d, %d), want (%d, %d)", tt.input, gotX, gotY, tt.wantX, tt.wantY)
			}
		})
	}
}
