package handlers

import "testing"

func TestRealtimeMapIDPrefersDetailedChildRegion(t *testing.T) {
	tests := []struct {
		name, regionID, zoneType, assigned, want string
	}{
		{"KHU_VUC overrides stale outer binding", "thanh_mao_son", "KHU_VUC", "world_map", "thanh_mao_son"},
		{"SAFE_ZONE overrides stale outer binding", "hub_dong_hai", "SAFE_ZONE", "world_map", "hub_dong_hai"},
		{"NGU_VUC keeps outer map", "dai_vuc_dong_hai", "NGU_VUC", "world_map", "world_map"},
		{"special region keeps explicit assignment", "secret_realm", "SECRET_REALM", "realm_map", "realm_map"},
		{"missing assignment falls back to world map", "", "", "", "world_map"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := realtimeMapID(test.regionID, test.zoneType, test.assigned); got != test.want {
				t.Fatalf("realtimeMapID()=%q, want %q", got, test.want)
			}
		})
	}
}
