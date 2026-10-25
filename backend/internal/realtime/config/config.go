// Package config contains the fail-closed rollout configuration for the new
// realtime world runtime. It deliberately does not change legacy world ownership.
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

// Config is the process-level admission policy for realtime world sessions.
// Empty zone or player allowlists deny all joins, even when Enabled is true.
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
		return disabled(), fmt.Errorf("realtime config: environment lookup is nil")
	}

	enabled, err := readBool(lookup, "REALTIME_WORLD_ENABLED", false)
	if err != nil {
		return disabled(), err
	}
	phaserEnabled, err := readBool(lookup, "PHASER_WORLD_CLIENT_ENABLED", false)
	if err != nil {
		return disabled(), err
	}
	admissionPaused, err := readBool(lookup, "MMO_ADMISSION_PAUSED", false)
	if err != nil {
		return disabled(), err
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
		Enabled:             enabled,
		PhaserClientEnabled: phaserEnabled,
		AdmissionPaused:     admissionPaused,
		MaxPlayersPerZone:   maxPlayers,
		InstancesPerZone:    instances,
		MapVersion:          boundedVersion(value(lookup, "MMO_MAP_VERSION")),
		ContentVersion:      boundedVersion(value(lookup, "MMO_CONTENT_VERSION")),
		MapDir:              mapDir(lookup),
		GuProfiles:          guProfiles(lookup),
		ZoneIDs:             parseSet(value(lookup, "MMO_ZONE_IDS")),
		PlayerAllowlist:     parseSet(value(lookup, "MMO_ALLOWLIST")),
	}, nil
}

func (c Config) AllowsWorldJoin(playerID, zoneID, mapVersion, contentVersion string) bool {
	return c.AllowsJoin(playerID, zoneID) && c.MapVersion != "" && c.ContentVersion != "" &&
		mapVersion == c.MapVersion && contentVersion == c.ContentVersion
}

// AllowsJoin checks all server-side rollout gates. The allowlist is required;
// an empty allowlist never means "allow everyone", but '*' or 'all' grants access to all players / zones.
func (c Config) AllowsJoin(playerID, zoneID string) bool {
	if !c.Enabled || strings.TrimSpace(playerID) == "" || strings.TrimSpace(zoneID) == "" {
		return false
	}
	_, playerWildcard := c.PlayerAllowlist["*"]
	if !playerWildcard {
		_, playerWildcard = c.PlayerAllowlist["all"]
	}
	_, playerAllowed := c.PlayerAllowlist[strings.TrimSpace(playerID)]

	_, zoneWildcard := c.ZoneIDs["*"]
	if !zoneWildcard {
		_, zoneWildcard = c.ZoneIDs["all"]
	}
	_, zoneAllowed := c.ZoneIDs[strings.TrimSpace(zoneID)]

	return (playerWildcard || playerAllowed) && (zoneWildcard || zoneAllowed)
}

func readBool(lookup func(string) (string, bool), name string, fallback bool) (bool, error) {
	raw, ok := lookup(name)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fallback, fmt.Errorf("realtime config: %s must be true or false", name)
	}
	return parsed, nil
}

func value(lookup func(string) (string, bool), name string) string {
	raw, _ := lookup(name)
	return raw
}

func boundedVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 80 {
		return ""
	}
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

func parseSet(raw string) map[string]struct{} {
	values := make(map[string]struct{})
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			values[value] = struct{}{}
		}
	}
	return values
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
