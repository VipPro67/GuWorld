package config

import "testing"

func TestLoadFromLookupDefaultsToDisabledAndClosedAdmission(t *testing.T) {
	config, err := LoadFromLookup(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("LoadFromLookup returned error: %v", err)
	}
	if config.Enabled || config.PhaserClientEnabled {
		t.Fatal("realtime flags must default to disabled")
	}
	if config.AllowsJoin("player-1", "zone-1") {
		t.Fatal("empty allowlists must deny admission")
	}
	if config.AllowsWorldJoin("player-1", "zone-1", DefaultMapVersion, DefaultContentVersion) {
		t.Fatal("disabled defaults and empty allowlists must deny the v1 join handshake")
	}
	if config.MapVersion != DefaultMapVersion || config.ContentVersion != DefaultContentVersion {
		t.Fatalf("default release versions = %q/%q", config.MapVersion, config.ContentVersion)
	}
	if config.MapDir != defaultMapDir {
		t.Fatalf("default map dir = %q, want %q", config.MapDir, defaultMapDir)
	}
	if config.GuProfiles != defaultGuProfiles {
		t.Fatalf("default Gu profile path = %q, want %q", config.GuProfiles, defaultGuProfiles)
	}
	if config.InstancesPerZone != 1 {
		t.Fatalf("default instances per zone = %d, want 1", config.InstancesPerZone)
	}
}

func TestAllowsJoinRequiresBothPlayerAndZoneAllowlist(t *testing.T) {
	values := map[string]string{
		"REALTIME_WORLD_ENABLED": "true",
		"MMO_ZONE_IDS":           "zone-1, zone-2",
		"MMO_ALLOWLIST":          "player-1",
	}
	config, err := LoadFromLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadFromLookup returned error: %v", err)
	}
	if !config.AllowsJoin("player-1", "zone-2") {
		t.Fatal("configured player and zone should be admitted")
	}
	if config.AllowsJoin("player-2", "zone-2") {
		t.Fatal("non-allowlisted player must be denied")
	}
	if config.AllowsJoin("player-1", "zone-3") {
		t.Fatal("non-allowlisted zone must be denied")
	}
	if !config.AllowsWorldJoin("player-1", "zone-2", DefaultMapVersion, DefaultContentVersion) {
		t.Fatal("matching map/content versions should be admitted")
	}
	if config.AllowsWorldJoin("player-1", "zone-2", "stale", DefaultContentVersion) {
		t.Fatal("stale map version must be denied")
	}
}

func TestDeploymentCannotOverrideMapVersions(t *testing.T) {
	values := map[string]string{
		"MMO_MAP_VERSION":                 "stale-map",
		"MMO_CONTENT_VERSION":             "stale-content",
		"NEXT_PUBLIC_MMO_MAP_VERSION":     "stale-map",
		"NEXT_PUBLIC_MMO_CONTENT_VERSION": "stale-content",
	}
	config, err := LoadFromLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadFromLookup returned error: %v", err)
	}
	if config.MapVersion != DefaultMapVersion || config.ContentVersion != DefaultContentVersion {
		t.Fatalf("deployment overrode shipped versions: %q/%q", config.MapVersion, config.ContentVersion)
	}
}

func TestInvalidRolloutConfigFailsClosed(t *testing.T) {
	values := map[string]string{
		"REALTIME_WORLD_ENABLED": "sometimes",
		"MMO_ZONE_IDS":           "zone-1",
		"MMO_ALLOWLIST":          "player-1",
	}
	config, err := LoadFromLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err == nil {
		t.Fatal("expected invalid boolean to return an error")
	}
	if config.Enabled || config.AllowsJoin("player-1", "zone-1") {
		t.Fatal("invalid configuration must fail closed")
	}
}

func TestInstancesPerZoneBounds(t *testing.T) {
	for _, raw := range []string{"0", "9", "many"} {
		t.Run(raw, func(t *testing.T) {
			values := map[string]string{"MMO_INSTANCES_PER_ZONE": raw}
			config, err := LoadFromLookup(func(key string) (string, bool) {
				value, ok := values[key]
				return value, ok
			})
			if err == nil || config.Enabled || config.AllowsJoin("player-1", "zone-1") {
				t.Fatal("invalid instance count must return a disabled configuration")
			}
		})
	}
}

func TestMaxPlayersPerZoneBounds(t *testing.T) {
	for _, raw := range []string{"0", "101", "many"} {
		t.Run(raw, func(t *testing.T) {
			values := map[string]string{"MMO_MAX_PLAYERS_PER_ZONE": raw}
			config, err := LoadFromLookup(func(key string) (string, bool) {
				value, ok := values[key]
				return value, ok
			})
			if err == nil || config.Enabled || config.AllowsJoin("player-1", "zone-1") {
				t.Fatal("invalid capacity must return a disabled configuration")
			}
		})
	}
}

func TestAllowsJoinWithWildcard(t *testing.T) {
	for _, wildcard := range []string{"*", "all"} {
		t.Run("wildcard_"+wildcard, func(t *testing.T) {
			values := map[string]string{
				"REALTIME_WORLD_ENABLED": "true",
				"MMO_ZONE_IDS":           "zone-1",
				"MMO_ALLOWLIST":          wildcard,
			}
			cfg, err := LoadFromLookup(func(key string) (string, bool) {
				return values[key], true
			})
			if err != nil {
				t.Fatalf("LoadFromLookup returned error: %v", err)
			}
			if !cfg.AllowsJoin("random-player-uuid-1", "zone-1") {
				t.Fatalf("expected player to be admitted under %s allowlist", wildcard)
			}
			if !cfg.AllowsJoin("random-player-uuid-2", "zone-1") {
				t.Fatalf("expected second player to be admitted under %s allowlist", wildcard)
			}
			if cfg.AllowsJoin("random-player-uuid-1", "other-zone") {
				t.Fatal("non-allowlisted zone must still be rejected")
			}
		})
	}
}
