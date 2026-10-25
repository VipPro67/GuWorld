package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
)

// cmdCompileRealtimeGu validates authored realtime Gu profiles against Gu.csv
// and the combat contract, then optionally writes the immutable compiled
// artifact for review. It is read-only against CSV data.
func cmdCompileRealtimeGu(args []string) error {
	fs := flag.NewFlagSet("compile-realtime-gu", flag.ExitOnError)
	data := fs.String("data", "data", "data directory")
	profilesPath := fs.String("profiles", "data/mmo/gu/transpiled_profiles.json", "comma-separated realtime Gu profile catalogs")
	report := fs.String("report", "", "write compiled catalog JSON to this path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	entries, err := loadRealtimeGuEntries(*data + "/Gu.csv")
	if err != nil {
		return err
	}
	profiles, err := realtimeeffects.LoadProfileCatalogs(*profilesPath)
	if err != nil {
		return err
	}
	compiled, err := realtimeeffects.CompileCatalog(profiles, entries, realtimeeffects.DefaultMVP())
	if err != nil {
		return err
	}
	if *report != "" {
		data, err := json.MarshalIndent(compiled, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*report, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("realtime Gu compiled: profiles=%d actives=%d hash=%s\n",
		len(compiled.Profiles), len(compiled.DefaultSkills), compiled.Hash)
	for _, skill := range compiled.DefaultSkills {
		fmt.Printf("  slot %d %s mode=%s windup=%d cooldown=%d range=%.1f damage=%.1f shield=%.1f heal=%.1f status=%s\n",
			skill.Slot, skill.ID, skill.Mode, skill.WindupTicks, skill.CooldownTicks,
			skill.Range, skill.Damage, skill.Shield, skill.Heal, skill.StatusID)
	}
	return nil
}
