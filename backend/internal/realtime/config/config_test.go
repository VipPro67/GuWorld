package config

import "testing"

func TestLoadEnablesPublicWorldByDefault(t *testing.T) {
	cfg, err := LoadFromLookup(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("LoadFromLookup returned error: %v", err)
	}
	if !cfg.Enabled || !cfg.PhaserClientEnabled {
		t.Fatal("realtime world and Phaser client must be enabled by default")
	}
	if !cfg.AllowsJoin("player-1", "zone-1") || !cfg.AllowsJoin("player-2", "zone-2") {
		t.Fatal("all signed-in players and public zones should pass the rollout gate")
	}
	if cfg.AllowsJoin("", "zone-1") || cfg.AllowsJoin("player-1", "") {
		t.Fatal("empty player or zone IDs must be rejected")
	}
	if !cfg.AllowsWorldJoin("player-1", "zone-1", DefaultMapVersion, DefaultContentVersion) {
		t.Fatal("matching shipped versions should be admitted")
	}
	if cfg.AllowsWorldJoin("player-1", "zone-1", "stale", DefaultContentVersion) {
		t.Fatal("stale map versions must be denied")
	}
	if cfg.MapVersion != DefaultMapVersion || cfg.ContentVersion != DefaultContentVersion {
		t.Fatalf("default release versions = %q/%q", cfg.MapVersion, cfg.ContentVersion)
	}
	if cfg.MapDir != defaultMapDir || cfg.GuProfiles != defaultGuProfiles {
		t.Fatalf("unexpected asset paths: map=%q profiles=%q", cfg.MapDir, cfg.GuProfiles)
	}
	if cfg.InstancesPerZone != 1 {
		t.Fatalf("default instances per zone = %d, want 1", cfg.InstancesPerZone)
	}
}

func TestDeploymentCannotOverrideMapVersions(t *testing.T) {
	values := map[string]string{
		"MMO_MAP_VERSION":                 "stale-map",
		"MMO_CONTENT_VERSION":             "stale-content",
		"NEXT_PUBLIC_MMO_MAP_VERSION":     "stale-map",
		"NEXT_PUBLIC_MMO_CONTENT_VERSION": "stale-content",
	}
	cfg, err := LoadFromLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadFromLookup returned error: %v", err)
	}
	if cfg.MapVersion != DefaultMapVersion || cfg.ContentVersion != DefaultContentVersion {
		t.Fatalf("deployment overrode shipped versions: %q/%q", cfg.MapVersion, cfg.ContentVersion)
	}
}

func TestInstancesPerZoneBounds(t *testing.T) {
	for _, raw := range []string{"0", "9", "many"} {
		t.Run(raw, func(t *testing.T) {
			values := map[string]string{"MMO_INSTANCES_PER_ZONE": raw}
			cfg, err := LoadFromLookup(func(key string) (string, bool) {
				value, ok := values[key]
				return value, ok
			})
			if err == nil || cfg.Enabled || cfg.AllowsJoin("player-1", "zone-1") {
				t.Fatal("invalid instance count must return a disabled configuration")
			}
		})
	}
}

func TestMaxPlayersPerZoneBounds(t *testing.T) {
	for _, raw := range []string{"0", "101", "many"} {
		t.Run(raw, func(t *testing.T) {
			values := map[string]string{"MMO_MAX_PLAYERS_PER_ZONE": raw}
			cfg, err := LoadFromLookup(func(key string) (string, bool) {
				value, ok := values[key]
				return value, ok
			})
			if err == nil || cfg.Enabled || cfg.AllowsJoin("player-1", "zone-1") {
				t.Fatal("invalid capacity must return a disabled configuration")
			}
		})
	}
}
