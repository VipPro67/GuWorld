// Package config contains tuning and shipped release defaults for the
// realtime world runtime. Public maps are open to every signed-in player.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultMaxPlayersPerZone = 5
const defaultMapDir = "data/mmo/maps"
const defaultGuProfiles = "data/mmo/gu/transpiled_profiles.json"

// Map versions are part of the shipped MMO release, not deployment options.
// Keep them aligned with backend/data/mmo/maps and frontend/public/game/maps.
const DefaultMapVersion = "regions-v1"
const DefaultContentVersion = "region-content-v1"

// Config is the process-level runtime configuration for realtime world sessions.
// Zone existence and authoritative character location are checked by the manager
// and admission loader; player access is not controlled by deployment allowlists.
type Config struct {
	Enabled             bool
	PhaserClientEnabled bool
	AdmissionPaused     bool
	MaxPlayersPerZone   int
	InstancesPerZone    int
	MapVersion          string
	ContentVersion      string
	MapDir              string
	GuProfiles          string
	ZoneIDs             map[string]struct{}
	PlayerAllowlist     map[string]struct{}
}

// Load reads the realtime rollout flags from the process environment. Invalid
// values return a disabled configuration so a typo cannot expose the new mode.
func Load() (Config, error) {
	return LoadFromLookup(os.LookupEnv)
}

// LoadFromLookup makes environment parsing explicit and keeps callers from
// depending on package-global mutable configuration.
func LoadFromLookup(lookup func(string) (string, bool)) (Config, error) {
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}

	instances := 1
	if raw, ok := lookup("MMO_INSTANCES_PER_ZONE"); ok && strings.TrimSpace(raw) != "" {
		parsed, parseErr := strconv.Atoi(strings.TrimSpace(raw))
		if parseErr != nil || parsed < 1 || parsed > 8 {
			return disabled(), fmt.Errorf("realtime config: MMO_INSTANCES_PER_ZONE must be an integer from 1 to 8")
		}
		instances = parsed
	}

	maxPlayers := defaultMaxPlayersPerZone
	if raw, ok := lookup("MMO_MAX_PLAYERS_PER_ZONE"); ok && strings.TrimSpace(raw) != "" {
		parsed, parseErr := strconv.Atoi(strings.TrimSpace(raw))
		if parseErr != nil || parsed < 1 || parsed > 100 {
			return disabled(), fmt.Errorf("realtime config: MMO_MAX_PLAYERS_PER_ZONE must be an integer from 1 to 100")
		}
		maxPlayers = parsed
	}

	return Config{
		Enabled:             true,
		PhaserClientEnabled: true,
		MaxPlayersPerZone:   maxPlayers,
		InstancesPerZone:    instances,
		MapVersion:          DefaultMapVersion,
		ContentVersion:      DefaultContentVersion,
		MapDir:              mapDir(lookup),
		GuProfiles:          guProfiles(lookup),
		ZoneIDs:             map[string]struct{}{"*": {}},
		PlayerAllowlist:     map[string]struct{}{"*": {}},
	}, nil
}

func (c Config) AllowsWorldJoin(playerID, zoneID, mapVersion, contentVersion string) bool {
	return c.AllowsJoin(playerID, zoneID) && c.MapVersion != "" && c.ContentVersion != "" &&
		mapVersion == c.MapVersion && contentVersion == c.ContentVersion
}

// AllowsJoin admits signed-in players to zones loaded in the runtime catalogue.
// The manager rejects unknown zone IDs and the join loader verifies character location.
func (c Config) AllowsJoin(playerID, zoneID string) bool {
	return c.Enabled && strings.TrimSpace(playerID) != "" && strings.TrimSpace(zoneID) != ""
}

func value(lookup func(string) (string, bool), name string) string {
	raw, _ := lookup(name)
	return raw
}

func mapDir(lookup func(string) (string, bool)) string {
	raw := strings.TrimSpace(value(lookup, "MMO_MAP_DIR"))
	if raw == "" || len(raw) > 512 {
		return defaultMapDir
	}
	return raw
}

func guProfiles(lookup func(string) (string, bool)) string {
	raw := strings.TrimSpace(value(lookup, "MMO_GU_PROFILES"))
	if raw == "" || len(raw) > 512 {
		return defaultGuProfiles
	}
	return raw
}

func disabled() Config {
	return Config{
		MaxPlayersPerZone: defaultMaxPlayersPerZone,
		InstancesPerZone:  1,
		MapDir:            defaultMapDir,
		GuProfiles:        defaultGuProfiles,
		ZoneIDs:           map[string]struct{}{},
		PlayerAllowlist:   map[string]struct{}{},
	}
}
