package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const sampleEffects = `[
  {"id":"A","trigger":"ON_CAST","target":{"selector":"TARGET"},"operations":[
    {"op":"DAMAGE","sub_type":"FLAT","value":120},
    {"op":"SHIELD","sub_type":"FLAT","value":50},
    {"op":"STAT_MOD","sub_type":"ADD_FLAT","value":10,"stat":"ATTACK"},
    {"op":"STATUS","status_id":"BLEED","value":5,"turns":2},
    {"op":"UNIQUE","code":"REBORN","params":{"chance":0.3}},
    {"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.35}
  ]}
]`

func TestPowerOf(t *testing.T) {
	p, err := powerOf(sampleEffects)
	if err != nil {
		t.Fatal(err)
	}
	if p.Damage != 120 || p.Shield != 50 || p.Stat != 10 || p.Status != 10 {
		t.Fatalf("unexpected components: %+v", p)
	}
	if p.Heal != 0 || p.DirectHP != 0 {
		t.Fatalf("unexpected zero components: %+v", p)
	}
	if got := p.total(map[string]float64{}); got != 190 {
		t.Fatalf("total = %v, want 190", got)
	}
}

func TestScaleCombatFlats(t *testing.T) {
	out, err := scaleCombatFlats(sampleEffects, 2)
	if err != nil {
		t.Fatal(err)
	}
	p, err := powerOf(out)
	if err != nil {
		t.Fatal(err)
	}
	if p.Damage != 240 || p.Shield != 100 {
		t.Fatalf("combat flats not scaled: %+v", p)
	}
	if p.Stat != 10 || p.Status != 10 {
		t.Fatalf("non-combat values were scaled: %+v", p)
	}
	// The multiplier op must be untouched.
	var blocks []struct {
		Operations []struct {
			Op      string          `json:"op"`
			SubType string          `json:"sub_type"`
			Value   json.RawMessage `json:"value"`
		} `json:"operations"`
	}
	if err := json.Unmarshal([]byte(out), &blocks); err != nil {
		t.Fatal(err)
	}
	for _, op := range blocks[0].Operations {
		if op.Op == "DAMAGE" && op.SubType == "MULTIPLIER" {
			v, _ := toFloat(op.Value)
			if v != 1.35 {
				t.Fatalf("multiplier changed to %v", v)
			}
		}
	}
}

func TestScaleCombatFlatsObjectShape(t *testing.T) {
	in := `{"schema_version":2,"blocks":[{"id":"A","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]}]}`
	out, err := scaleCombatFlats(in, 2)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &root); err != nil {
		t.Fatalf("object shape not preserved: %v", err)
	}
	if _, ok := root["schema_version"]; !ok {
		t.Fatalf("schema_version dropped: %s", out)
	}
}

func TestClassifyFood(t *testing.T) {
	guIDs := map[string]bool{"GU_FOOD": true}
	items := map[string]itemRef{
		"apple": {rank: 3, path: []string{"Wood Path"}, itemType: "Food"},
		"iron":  {rank: 2, path: []string{"Metal Path"}, itemType: "Material"},
	}
	cases := []struct {
		food     string
		wantKind string
	}{
		{"apple", "ok"},
		{"iron", "ok"},
		{"GU_FOOD", "ok"},
		{"primeval_stone", "ok"},
		{"ghost_item", "dangling"},
		{"GU_GHOST", "dangling"},
	}
	for _, c := range cases {
		kind, _ := classifyFood(c.food, guIDs, items)
		if kind != c.wantKind {
			t.Errorf("classifyFood(%q) = %q, want %q", c.food, kind, c.wantKind)
		}
	}
}

func TestSelectMaterialsPrefersPathAndRank(t *testing.T) {
	materials := map[string]itemRef{
		"blood_r6": {rank: 6, path: []string{"Blood Path"}, itemType: "Material"},
		"blood_r5": {rank: 5, path: []string{"Blood Path"}, itemType: "Material"},
		"fire_r6":  {rank: 6, path: []string{"Fire Path"}, itemType: "Material"},
		"fire_r7":  {rank: 7, path: []string{"Fire Path"}, itemType: "Material"},
	}
	got := selectMaterials([]string{"Blood Path"}, 6, materials, 2, 10)
	if _, ok := got["blood_r6"]; !ok {
		t.Fatalf("expected rank-matched path material, got %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 materials, got %d: %v", len(got), got)
	}
}

func TestTranspileRealtimeGuCanProcessCatalog(t *testing.T) {
	outPath := t.TempDir() + "/transpiled.json"
	reportPath := t.TempDir() + "/compiled.json"
	err := cmdTranspileRealtimeGu([]string{
		"-data", "../../data",
		"-overrides", "",
		"-output", outPath,
		"-compile-report", reportPath,
		"-stats=false",
	})
	if err != nil {
		t.Fatalf("transpile failed: %v", err)
	}
}

func TestTranspileGuProfileKeepsCastableEffectBesideWorldHook(t *testing.T) {
	profile := transpileGuProfile("GU_MIXED", 2, "Water Path", "Attack", 5, []rawTranspileBlock{
		{ID: "ACTIVE_SKILL", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{{Op: "DAMAGE", SubType: "FLAT", Value: 60}}},
		{ID: "PORTAL", Trigger: "CAPABILITY", Operations: []rawTranspileOp{{Op: "UNIQUE"}}},
	})
	if profile.Active == nil || len(profile.Active.Effects) != 1 || profile.Active.Effects[0].Type != "DAMAGE" {
		t.Fatalf("castable action was lost beside domain hook: %+v", profile)
	}
}

func TestTranspileGuProfileDoesNotInventCombatPayload(t *testing.T) {
	profile := transpileGuProfile("GU_UNSUPPORTED_ACTION", 3, "Fire Path", "Attack", 5, []rawTranspileBlock{
		{ID: "ACTIVE_SKILL", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{{Op: "STAT_MOD", Stat: "ATTACK", Value: 100}}},
	})
	if profile.Active != nil {
		t.Fatalf("unsupported stat action became a castable skill: %+v", profile.Active)
	}

	profile = transpileGuProfile("GU_WORLD_ONLY", 3, "Fire Path", "Attack", 5, []rawTranspileBlock{
		{ID: "PORTAL", Trigger: "CAPABILITY", Operations: []rawTranspileOp{{Op: "UNIQUE"}}},
	})
	if profile.Active != nil || len(profile.Passives) != 0 {
		t.Fatalf("world-only Gu became a synthetic combat skill: %+v", profile)
	}

	profile = transpileGuProfile("GU_UNAUTHORED_STATUS", 3, "Ice Path", "Attack", 5, []rawTranspileBlock{
		{ID: "ACTIVE_SKILL", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{{Op: "STATUS"}}},
	})
	if profile.Active != nil {
		t.Fatalf("turn-based status without realtime duration became a castable skill: %+v", profile.Active)
	}

	profile = transpileGuProfile("GU_DAMAGE_WITH_STATUS", 3, "Ice Path", "Attack", 5, []rawTranspileBlock{
		{ID: "ACTIVE_SKILL", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{
			{Op: "DAMAGE", SubType: "FLAT", Value: 50},
			{Op: "STATUS", Value: 5},
		}},
	})
	if profile.Active == nil || len(profile.Active.Effects) != 1 || profile.Active.Effects[0].Type != "DAMAGE" {
		t.Fatalf("safe action effect was dropped with an unauthored status: %+v", profile.Active)
	}

	profile = transpileGuProfile("GU_PARTIAL_ACTION", 3, "Fire Path", "Attack", 5, []rawTranspileBlock{
		{ID: "ACTIVE_DAMAGE", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{{Op: "DAMAGE", Value: 50, SubType: "FLAT"}}},
		{ID: "ACTIVE_BUFF", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "ALL_ALLIES"}, Operations: []rawTranspileOp{{Op: "STAT_MOD", Stat: "ATTACK", Value: 5, SubType: "ADD_FLAT"}}},
	})
	if profile.Active != nil {
		t.Fatalf("partially modeled action was exposed as a castable skill: %+v", profile.Active)
	}
}

func TestTranspileGuProfileAppliesDamageMultiplier(t *testing.T) {
	profile := transpileGuProfile("GU_MULTIPLIER", 2, "Strength Path", "Attack", 5, []rawTranspileBlock{
		{ID: "ACTIVE_SKILL", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{
			{Op: "DAMAGE", SubType: "FLAT", Value: 60},
			{Op: "DAMAGE", SubType: "MULTIPLIER", Value: 3},
		}},
	})
	if profile.Active == nil || len(profile.Active.Effects) != 1 || profile.Active.Effects[0].Value != 75 {
		t.Fatalf("damage multiplier was not folded into base damage: %+v", profile.Active)
	}
}

func TestRealtimeCastBlockRejectsEffectsWithUnsupportedTargets(t *testing.T) {
	cases := []rawTranspileBlock{
		{ID: "SELF_DAMAGE", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "SELF"}, Operations: []rawTranspileOp{{Op: "DAMAGE", SubType: "FLAT", Value: 10}}},
		{ID: "TARGET_SHIELD", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{{Op: "SHIELD", SubType: "FLAT", Value: 10}}},
		{ID: "ALL_ALLIES_HEAL", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "ALL_ALLIES"}, Operations: []rawTranspileOp{{Op: "HEAL", SubType: "FLAT", Resource: "HP", Value: 10}}},
		{ID: "MULTIPLIER_ONLY", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{{Op: "DAMAGE", SubType: "MULTIPLIER", Value: 2}}},
	}
	for _, block := range cases {
		if realtimeCastBlockSupported(block) {
			t.Errorf("unsupported target was accepted: %+v", block)
		}
	}
	if realtimeCastBlocksSupported([]rawTranspileBlock{
		{ID: "SINGLE", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "TARGET"}, Operations: []rawTranspileOp{{Op: "DAMAGE", SubType: "FLAT", Value: 10}}},
		{ID: "AREA", Trigger: "ON_ACTION", Target: rawTranspileTarget{Selector: "ALL_ENEMIES"}, Operations: []rawTranspileOp{{Op: "DAMAGE", SubType: "FLAT", Value: 10}}},
	}) {
		t.Fatal("one cast with incompatible damage targeting modes was accepted")
	}
}

func TestTranspileRealtimeProfilesCoverEveryFullyModeledAction(t *testing.T) {
	data, err := readCSV("../../data/Gu.csv")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range data.rows {
		blocks, err := decodeTranspileBlocks(data.get(row, "effects"))
		if err != nil {
			t.Fatalf("decode %s: %v", data.get(row, "id"), err)
		}
		var actionBlocks []rawTranspileBlock
		for _, block := range blocks {
			if strings.EqualFold(strings.TrimSpace(block.Trigger), "ON_ACTION") {
				actionBlocks = append(actionBlocks, block)
			}
		}
		if len(actionBlocks) == 0 {
			continue
		}
		allSupported := realtimeCastBlocksSupported(actionBlocks)
		profile := transpileGuProfile(
			data.get(row, "id"), 1, data.get(row, "path"), data.get(row, "primary_role"), 1, blocks,
		)
		if got := profile.Active != nil; got != allSupported {
			t.Errorf("%s: active profile=%t, fully modeled ON_ACTION=%t", profile.GUID, got, allSupported)
		}
	}
}
