package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
)

type rawTranspileBlock struct {
	ID         string             `json:"id"`
	Trigger    string             `json:"trigger"`
	Target     rawTranspileTarget `json:"target"`
	Operations []rawTranspileOp   `json:"operations"`
}

type rawTranspileTarget struct {
	Selector string `json:"selector"`
}

type rawTranspileOp struct {
	Op       string  `json:"op"`
	SubType  string  `json:"sub_type"`
	Stat     string  `json:"stat"`
	Resource string  `json:"resource"`
	StatusID string  `json:"status_id"`
	Code     string  `json:"code"`
	Turns    int     `json:"turns"`
	Chance   float64 `json:"chance"`
	Value    float64 `json:"value"`
}

type TranspileStats struct {
	TotalCSVEntries    int
	Overridden         int
	NotActiveSkipped   int
	PassivesGenerated  int
	ActivesGenerated   int
	ModeTargetCount    int
	ModeDirectionCount int
	ModeGroundCount    int
	ModeSelfCount      int
	ShapesCount        map[string]int
}

func cmdTranspileRealtimeGu(args []string) error {
	fs := flag.NewFlagSet("transpile-realtime-gu", flag.ExitOnError)
	data := fs.String("data", "data", "data directory containing Gu.csv")
	overridesPath := fs.String("overrides", "", "comma-separated existing profile catalogs to preserve as overrides")
	output := fs.String("output", "data/mmo/gu/transpiled_profiles.json", "output path for the complete generated profile catalog")
	compileReport := fs.String("compile-report", "", "optional path to write compiled catalog JSON directly")
	statsFlag := fs.Bool("stats", true, "print detailed transpilation statistics")
	if err := fs.Parse(args); err != nil {
		return err
	}

	csvPath := filepath.Join(*data, "Gu.csv")
	t, err := readCSV(csvPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", csvPath, err)
	}

	// Load handcrafted overrides
	overrideMap := map[string]realtimeeffects.GuProfile{}
	if *overridesPath != "" {
		catalogs, err := realtimeeffects.LoadProfileCatalogs(*overridesPath)
		if err != nil {
			return fmt.Errorf("load overrides: %w", err)
		}
		for _, p := range catalogs.Profiles {
			overrideMap[fmt.Sprintf("%s:%d", p.GUID, p.Rank)] = p
		}
	}

	stats := TranspileStats{
		TotalCSVEntries: len(t.rows),
		ShapesCount:     make(map[string]int),
	}

	catalog := realtimeeffects.ProfileCatalog{
		SchemaVersion: realtimeeffects.ProfileSchemaVersion,
		Profiles:      make([]realtimeeffects.GuProfile, 0, len(t.rows)),
	}

	for _, row := range t.rows {
		guID := t.get(row, "id")
		rank, _ := strconv.Atoi(t.get(row, "rank"))
		if guID == "" || rank < 1 {
			continue
		}
		key := fmt.Sprintf("%s:%d", guID, rank)

		// Check if handcrafted override exists
		if override, ok := overrideMap[key]; ok {
			catalog.Profiles = append(catalog.Profiles, override)
			stats.Overridden++
			if override.Active != nil {
				stats.ActivesGenerated++
				switch override.Active.Mode {
				case combat.ModeTarget:
					stats.ModeTargetCount++
				case combat.ModeDirection:
					stats.ModeDirectionCount++
				case combat.ModeGround:
					stats.ModeGroundCount++
				case combat.ModeSelf:
					stats.ModeSelfCount++
				}
				if override.Active.Shape != "" {
					stats.ShapesCount[string(override.Active.Shape)]++
				}
			} else {
				stats.PassivesGenerated++
			}
			continue
		}

		pathStr := t.get(row, "path")
		primaryRole := t.get(row, "primary_role")
		baseMana, _ := strconv.ParseFloat(t.get(row, "base_mana_cost"), 64)
		effectsJSON := t.get(row, "effects")

		var blocks []rawTranspileBlock
		if effectsJSON != "" {
			_ = json.Unmarshal([]byte(effectsJSON), &blocks)
		}

		// Check for world-only/not-active triggers
		isNotActive := false
		for _, b := range blocks {
			if b.Trigger == "CAPABILITY" || b.Trigger == "ON_CONSUME" || b.Trigger == "ON_USE" {
				isNotActive = true
				break
			}
		}
		if isNotActive {
			stats.NotActiveSkipped++
			continue
		}

		profile := transpileGuProfile(guID, rank, pathStr, primaryRole, baseMana, blocks)
		if profile.Active != nil {
			stats.ActivesGenerated++
			switch profile.Active.Mode {
			case combat.ModeTarget:
				stats.ModeTargetCount++
			case combat.ModeDirection:
				stats.ModeDirectionCount++
			case combat.ModeGround:
				stats.ModeGroundCount++
			case combat.ModeSelf:
				stats.ModeSelfCount++
			}
			if profile.Active.Shape != "" {
				stats.ShapesCount[string(profile.Active.Shape)]++
			}
		} else {
			stats.PassivesGenerated++
		}

		catalog.Profiles = append(catalog.Profiles, profile)
	}

	// Sort profiles deterministically by ID and rank
	sort.SliceStable(catalog.Profiles, func(i, j int) bool {
		if catalog.Profiles[i].GUID != catalog.Profiles[j].GUID {
			return catalog.Profiles[i].GUID < catalog.Profiles[j].GUID
		}
		return catalog.Profiles[i].Rank < catalog.Profiles[j].Rank
	})

	// Ensure output dir exists
	if *output != "" {
		if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		dataBytes, err := json.MarshalIndent(catalog, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal catalog: %w", err)
		}
		if err := os.WriteFile(*output, append(dataBytes, '\n'), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", *output, err)
		}
		fmt.Printf("Transpiled %d profiles written to %s\n", len(catalog.Profiles), *output)
	}

	if *statsFlag {
		fmt.Printf("\n=== Transpilation Summary ===\n")
		fmt.Printf("Total Gu in CSV:      %d\n", stats.TotalCSVEntries)
		fmt.Printf("Handcrafted overrides: %d\n", stats.Overridden)
		fmt.Printf("World/Not-Active:     %d (skipped from combat hotbar)\n", stats.NotActiveSkipped)
		fmt.Printf("Profiles Generated:   %d (Actives: %d, Passives: %d)\n", len(catalog.Profiles), stats.ActivesGenerated, stats.PassivesGenerated)
		fmt.Printf("Active Modes Breakdown:\n")
		fmt.Printf("  - TARGET:    %d\n", stats.ModeTargetCount)
		fmt.Printf("  - DIRECTION: %d\n", stats.ModeDirectionCount)
		fmt.Printf("  - GROUND:    %d\n", stats.ModeGroundCount)
		fmt.Printf("  - SELF:      %d\n", stats.ModeSelfCount)
		fmt.Printf("Geometric Shapes Breakdown:\n")
		for s, count := range stats.ShapesCount {
			fmt.Printf("  - %-10s: %d\n", s, count)
		}
		fmt.Println("=============================")
	}

	// If requested, also compile and validate the generated catalog
	if *compileReport != "" {
		entries, err := loadRealtimeGuEntries(csvPath)
		if err != nil {
			return fmt.Errorf("load entries for compile: %w", err)
		}
		compiled, err := realtimeeffects.CompileCatalog(catalog, entries, realtimeeffects.DefaultMVP())
		if err != nil {
			return fmt.Errorf("validation/compilation failed: %w", err)
		}
		compBytes, err := json.MarshalIndent(compiled, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*compileReport, append(compBytes, '\n'), 0o644); err != nil {
			return err
		}
		fmt.Printf("Compiled catalog validated and written to %s (Hash: %s)\n", *compileReport, compiled.Hash)
	}

	return nil
}

func transpileGuProfile(guID string, rank int, pathStr, primaryRole string, baseMana float64, blocks []rawTranspileBlock) realtimeeffects.GuProfile {
	profile := realtimeeffects.GuProfile{
		GUID:    guID,
		Rank:    rank,
		Slot:    0, // catalog-only by default; equipped Gu are reslotted dynamically
		Visuals: inferVisuals(pathStr, guID),
	}

	// Check if this Gu has passive blocks only
	hasActiveTrigger := false
	for _, b := range blocks {
		switch b.Trigger {
		case "ON_ACTION", "ON_CAST", "ON_HIT", "BEFORE_HIT", "ON_TURN_START", "ON_BATTLE_START", "BEFORE_ENEMY_ATTACK", "ON_TICK", "ON_DEATH":
			hasActiveTrigger = true
		}
	}

	if !hasActiveTrigger {
		// Pure passive Gu
		var passiveEffects []realtimeeffects.EffectProfile
		for _, b := range blocks {
			for _, op := range b.Operations {
				if op.Op == "STAT_MOD" && op.Stat != "" {
					val := op.Value
					if val == 0 {
						val = float64(rank * 2)
					}
					passiveEffects = append(passiveEffects, realtimeeffects.EffectProfile{
						Type:  "STAT_MOD",
						Stat:  op.Stat,
						Value: val,
					})
				}
			}
		}
		if len(passiveEffects) == 0 {
			// Fallback stat boost if no explicit op
			passiveEffects = append(passiveEffects, realtimeeffects.EffectProfile{
				Type:  "STAT_MOD",
				Stat:  "ATTACK",
				Value: float64(rank * 2),
			})
		}
		profile.Passives = []realtimeeffects.PassiveProfile{
			{
				Trigger: "ALWAYS_ON",
				Effects: passiveEffects,
			},
		}
		return profile
	}

	// Active Gu
	windupMS, recoveryMS, cooldownMS, manaCost := calculatePacing(rank, baseMana)

	// Determine dominant target selector
	selector := "TARGET"
	for _, b := range blocks {
		if b.Target.Selector != "" {
			selector = b.Target.Selector
			break
		}
	}

	mode, shape, rangeUnits, radiusUnits, angleDeg, lengthUnits, widthUnits, maxTargets, isProj := calculateRangeAndMode(selector, primaryRole, pathStr, rank)

	active := &realtimeeffects.ActiveProfile{
		Mode:           mode,
		Shape:          shape,
		AngleDeg:       angleDeg,
		LengthUnits:    lengthUnits,
		WidthUnits:     widthUnits,
		WindupMS:       windupMS,
		RecoveryMS:     recoveryMS,
		CooldownMS:     cooldownMS,
		RangeUnits:     rangeUnits,
		ManaCost:       manaCost,
		MovementPolicy: "SLOW",
		MaxTargets:     maxTargets,
		RadiusUnits:    radiusUnits,
	}

	if isProj {
		active.Projectile = &realtimeeffects.ProjectileProfile{
			SpeedUnitsPerSec: 18.0,
			RadiusUnits:      0.35,
			LifetimeTicks:    40,
		}
		if shape == combat.ShapeRectangle {
			active.Projectile.PierceCount = 3
		}
	}

	// Translate operations into realtime effects
	var effects []realtimeeffects.EffectProfile
	hasDamage := false
	hasHeal := false
	hasShield := false
	hasStatus := false

	for _, b := range blocks {
		for _, op := range b.Operations {
			switch op.Op {
			case "DAMAGE":
				val := scaleMMODamage(op.Value, rank)
				effects = append(effects, realtimeeffects.EffectProfile{
					Type:  "DAMAGE",
					Value: val,
				})
				hasDamage = true
			case "SHIELD":
				val := scaleMMODamage(op.Value, rank)
				effects = append(effects, realtimeeffects.EffectProfile{
					Type:  "SHIELD",
					Value: val,
				})
				hasShield = true
			case "HEAL", "DIRECT_HP":
				val := scaleMMODamage(op.Value, rank)
				effects = append(effects, realtimeeffects.EffectProfile{
					Type:  "HEAL",
					Value: val,
				})
				hasHeal = true
			case "CLEANSE":
				effects = append(effects, realtimeeffects.EffectProfile{
					Type: "CLEANSE",
				})
			case "STATUS":
				if op.StatusID != "" && !hasStatus {
					eff := mapStatusEffect(op.StatusID, op.Turns, op.Value, rank)
					effects = append(effects, eff)
					hasStatus = true
				}
			}
		}
	}

	// Ensure active skill has at least one valid payload
	if len(effects) == 0 {
		if mode == combat.ModeSelf {
			effects = append(effects, realtimeeffects.EffectProfile{
				Type:  "SHIELD",
				Value: float64(rank * 30),
			})
			hasShield = true
		} else {
			effects = append(effects, realtimeeffects.EffectProfile{
				Type:  "DAMAGE",
				Value: float64(rank * 25),
			})
			hasDamage = true
		}
	} else if mode == combat.ModeSelf && !hasHeal && !hasShield {
		// Self skill must have heal, shield, status, or cleanse
		hasStatusOrCleanse := false
		for _, e := range effects {
			if e.Type == "STATUS" || e.Type == "CLEANSE" {
				hasStatusOrCleanse = true
				break
			}
		}
		if !hasStatusOrCleanse {
			effects = append(effects, realtimeeffects.EffectProfile{
				Type:  "SHIELD",
				Value: float64(rank * 30),
			})
		}
	} else if mode != combat.ModeSelf && !hasDamage {
		// Non-self skill must have damage or status
		hasStatus := false
		for _, e := range effects {
			if e.Type == "STATUS" {
				hasStatus = true
				break
			}
		}
		if !hasStatus {
			effects = append(effects, realtimeeffects.EffectProfile{
				Type:  "DAMAGE",
				Value: float64(rank * 25),
			})
		}
	}

	active.Effects = effects
	profile.Active = active
	return profile
}

func calculatePacing(rank int, baseMana float64) (windupMS, recoveryMS, cooldownMS uint64, manaCost float64) {
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	windups := []uint64{0, 150, 200, 300, 400, 500, 600, 700, 800, 1000}
	recoveries := []uint64{0, 150, 150, 200, 200, 250, 250, 300, 300, 350}
	cooldowns := []uint64{0, 4000, 6000, 8000, 12000, 16000, 20000, 25000, 30000, 35000}
	minMana := []float64{0, 4, 6, 8, 12, 16, 20, 25, 30, 40}

	windupMS = windups[rank]
	recoveryMS = recoveries[rank]
	cooldownMS = cooldowns[rank]
	manaCost = minMana[rank]
	if baseMana > manaCost {
		manaCost = baseMana
	}
	return
}

func calculateRangeAndMode(selector, primaryRole, pathStr string, rank int) (mode combat.Mode, shape combat.Shape, rangeUnits, radiusUnits, angleDeg, lengthUnits, widthUnits float64, maxTargets int, isProj bool) {
	switch selector {
	case "SELF", "ALL_ALLIES":
		mode = combat.ModeSelf
		maxTargets = 1
		rangeUnits = 0
	case "COL":
		mode = combat.ModeGround
		shape = combat.ShapeCircle
		rangeUnits = 10.0
		radiusUnits = 2.5
		maxTargets = 5
	case "ALL_ENEMIES":
		mode = combat.ModeGround
		shape = combat.ShapeCone
		angleDeg = 75.0
		rangeUnits = 8.0
		radiusUnits = 8.0
		maxTargets = 8
	case "ROW":
		mode = combat.ModeDirection
		shape = combat.ShapeRectangle
		rangeUnits = 14.0
		lengthUnits = 14.0
		widthUnits = 1.8
		radiusUnits = 0.5
		maxTargets = 1
		isProj = true
	default: // TARGET or empty
		p := strings.ToLower(pathStr)
		if primaryRole == "RANGED" || strings.Contains(p, "light") || strings.Contains(p, "lightning") || strings.Contains(p, "water") || strings.Contains(p, "fire") || strings.Contains(p, "poison") {
			mode = combat.ModeDirection
			rangeUnits = 16.0
			radiusUnits = 0.35
			maxTargets = 1
			isProj = true
		} else {
			mode = combat.ModeTarget
			rangeUnits = 9.0
			maxTargets = 1
		}
	}
	return
}

func scaleMMODamage(val float64, rank int) float64 {
	caps := []float64{0, 40, 75, 150, 250, 400, 650, 900, 1200, 1600}
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	cap := caps[rank]
	if val <= 0 {
		return float64(rank * 20)
	}
	if val > cap {
		return cap
	}
	return val
}

func mapStatusEffect(statusID string, turns int, val float64, rank int) realtimeeffects.EffectProfile {
	id := strings.ToUpper(strings.TrimSpace(statusID))
	t := turns
	if t <= 0 {
		t = 2
	}
	if t > 5 {
		t = 5
	}

	eff := realtimeeffects.EffectProfile{
		Type:        "STATUS",
		StatusID:    id,
		MaxStacks:   1,
		StackPolicy: "REFRESH",
	}

	debuffTrue := true
	debuffFalse := false

	switch id {
	case "FREEZE", "STUN", "SLEEP", "CHARM", "CONFUSE", "FEAR":
		dur := uint64(t * 1000)
		if dur > 2500 {
			dur = 2500
		}
		if dur < 1000 {
			dur = 1000
		}
		eff.DurationMS = dur
		eff.Debuff = &debuffTrue
		eff.PvPDrCategory = "HARD_CC"
		zero := 0.0
		eff.MoveMultiplier = &zero

	case "BIND":
		dur := uint64(t * 1200)
		if dur > 3000 {
			dur = 3000
		}
		eff.DurationMS = dur
		eff.Debuff = &debuffTrue
		eff.PvPDrCategory = "ROOT"
		zero := 0.0
		eff.MoveMultiplier = &zero

	case "SLOW", "SLOW1":
		dur := uint64(t * 1500)
		eff.DurationMS = dur
		eff.Debuff = &debuffTrue
		eff.PvPDrCategory = "SLOW"
		mult := 0.6
		eff.MoveMultiplier = &mult

	case "POISON", "BLEED", "BURN", "SOUL_WOUND":
		dur := uint64(t * 1500)
		eff.DurationMS = dur
		eff.PeriodMS = 500
		tick := val / float64(2*t)
		if tick <= 0 {
			tick = float64(rank * 5)
		}
		eff.TickValue = tick
		eff.Debuff = &debuffTrue
		eff.PvPDrCategory = "DOT"

	case "HASTE":
		eff.DurationMS = uint64(t * 2000)
		hasteMult := 1.35
		eff.MoveMultiplier = &hasteMult
		eff.Debuff = &debuffFalse
		eff.PvPDrCategory = "BUFF"

	case "ATK_UP", "DAMAGE_REDUCTION_UP", "EVASION", "INVINCIBLE", "LAST_STAND", "STEALTH", "FLYING":
		eff.DurationMS = uint64(t * 2000)
		eff.Debuff = &debuffFalse
		eff.PvPDrCategory = "BUFF"

	default: // SILENCE, GU_SEAL, ARMOR_BREAK, WEAK, BLIND, etc.
		dur := uint64(t * 1500)
		if dur < 1000 {
			dur = 1000
		}
		eff.DurationMS = dur
		eff.Debuff = &debuffTrue
		eff.PvPDrCategory = "DEBUFF"
	}
	return eff
}

func inferVisuals(pathStr, guID string) realtimeeffects.VisualProfile {
	p := strings.ToLower(pathStr)
	switch {
	case strings.Contains(p, "fire"), strings.Contains(p, "viêm"):
		return realtimeeffects.VisualProfile{Icon: "gu/fire", CastVFX: "cast/fire", ProjectileVFX: "projectiles/fireball", ImpactVFX: "impact/fire_burst"}
	case strings.Contains(p, "water"), strings.Contains(p, "thủy"):
		return realtimeeffects.VisualProfile{Icon: "gu/water", CastVFX: "cast/water", ProjectileVFX: "projectiles/water_arrow", ImpactVFX: "impact/water"}
	case strings.Contains(p, "ice"), strings.Contains(p, "frost"), strings.Contains(p, "băng"):
		return realtimeeffects.VisualProfile{Icon: "gu/ice", CastVFX: "cast/frost", ProjectileVFX: "projectiles/water_arrow", ImpactVFX: "impact/frost"}
	case strings.Contains(p, "lightning"), strings.Contains(p, "lôi"):
		return realtimeeffects.VisualProfile{Icon: "gu/lightning", CastVFX: "cast/lightning", ProjectileVFX: "projectiles/spark", ImpactVFX: "impact/lightning_strike"}
	case strings.Contains(p, "poison"), strings.Contains(p, "độc"):
		return realtimeeffects.VisualProfile{Icon: "gu/poison", CastVFX: "cast/poison", ProjectileVFX: "projectiles/poison_spit", ImpactVFX: "impact/poison_cloud"}
	case strings.Contains(p, "strength"), strings.Contains(p, "lực"), strings.Contains(p, "earth"), strings.Contains(p, "thổ"):
		return realtimeeffects.VisualProfile{Icon: "gu/earth", CastVFX: "cast/earth", ImpactVFX: "impact/heavy_strike"}
	case strings.Contains(p, "light"), strings.Contains(p, "quang"), strings.Contains(p, "moon"):
		return realtimeeffects.VisualProfile{Icon: "gu/light", CastVFX: "cast/light", ProjectileVFX: "projectiles/moon_blade", ImpactVFX: "impact/light_slash"}
	case strings.Contains(p, "dark"), strings.Contains(p, "ám"), strings.Contains(p, "shadow"):
		return realtimeeffects.VisualProfile{Icon: "gu/dark", CastVFX: "cast/dark", ProjectileVFX: "projectiles/dark_orb", ImpactVFX: "impact/dark_burst"}
	case strings.Contains(p, "sword"), strings.Contains(p, "blade"), strings.Contains(p, "kiếm"), strings.Contains(p, "đao"):
		return realtimeeffects.VisualProfile{Icon: "gu/sword", CastVFX: "cast/sword", ImpactVFX: "impact/sword_slash"}
	case strings.Contains(p, "wind"), strings.Contains(p, "phong"):
		return realtimeeffects.VisualProfile{Icon: "gu/wind", CastVFX: "cast/wind", ProjectileVFX: "projectiles/wind_blade", ImpactVFX: "impact/wind_slash"}
	default:
		return realtimeeffects.VisualProfile{Icon: "gu/placeholder", CastVFX: "cast/placeholder", ProjectileVFX: "projectiles/water_arrow", ImpactVFX: "impact/placeholder"}
	}
}
