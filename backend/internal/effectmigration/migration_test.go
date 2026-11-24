package effectmigration

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFormatSpecificConverters(t *testing.T) {
	tests := []struct {
		name          string
		format        Format
		wantConverted int
		wantAmbiguous int
	}{
		{"gu", FormatGu, 5, 0}, {"item", FormatItem, 2, 0}, {"gamefx", FormatGameFX, 3, 0}, {"aperture", FormatAperture, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Convert(tt.format, "fixture", fixture(t, tt.name))
			if len(r.Report.Converted) != tt.wantConverted || len(r.Report.Ambiguous) != tt.wantAmbiguous {
				t.Fatalf("report = %+v", r.Report)
			}
			if tt.format != FormatAperture {
				var envelope struct {
					SchemaVersion int               `json:"schema_version"`
					Blocks        []json.RawMessage `json:"blocks"`
				}
				if err := json.Unmarshal(r.Envelope, &envelope); err != nil || envelope.SchemaVersion != 2 || len(envelope.Blocks) != tt.wantConverted {
					t.Fatalf("envelope %s: %v", r.Envelope, err)
				}
			}
		})
	}
}

func TestConvertItemUsesTypedOperationsAndRejectsNoOpAliases(t *testing.T) {
	r := ConvertItem("legacy-item", []byte(`{"restore_hp":25,"random_race":1,"hunger":10}`))
	if r.Report.MigrationGate || len(r.Report.Unsupported) != 1 || r.Report.Unsupported[0].Key != "hunger" {
		t.Fatalf("report should reject the no-op hunger alias: %+v", r.Report)
	}
	var envelope struct {
		Blocks []struct {
			ID         string      `json:"id"`
			Operations []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(envelope.Blocks))
	}
	gotByOp := map[string]operation{}
	for _, block := range envelope.Blocks {
		for _, op := range block.Operations {
			gotByOp[op.Op] = op
		}
	}
	if got := gotByOp["HEAL"]; got.Resource != "HP" || got.Value != float64(25) {
		t.Fatalf("restore_hp conversion = %+v", got)
	}
	if got := gotByOp["RANDOMIZE_TRAIT"]; got.SubType != "RACE" {
		t.Fatalf("random_race conversion = %+v", got)
	}
}

func TestConvertGuDreamImmunityToTypedRule(t *testing.T) {
	r := ConvertGu("legacy-dream-immunity", []byte(`{"passive":{"immune_dream_status":true}}`))
	if !r.Report.Validated || len(r.Report.Converted) != 1 || len(r.Report.Unsupported) != 0 {
		t.Fatalf("dream immunity conversion report = %+v", r.Report)
	}
	var envelope struct {
		Blocks []struct {
			Operations []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Blocks) != 1 || len(envelope.Blocks[0].Operations) != 1 {
		t.Fatalf("converted blocks = %+v", envelope.Blocks)
	}
	op := envelope.Blocks[0].Operations[0]
	if op.Op != "SET_RULE" || op.SubType != "IMMUNE_DREAM_STATUS" || op.Enabled == nil || !*op.Enabled {
		t.Fatalf("converted dream immunity = %+v", op)
	}
}

func TestConvertGuBlockV2IsCanonicalAndIdempotent(t *testing.T) {
	array := []byte(`[{"id":"BASIC_HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10,"unit":"FLAT"}]}]`)
	first := ConvertGu("v2", array)
	if !first.Report.MigrationGate || first.Report.Total != 0 || !first.Report.Validated {
		t.Fatalf("array audit = %+v", first.Report)
	}
	second := ConvertGu("v2", first.Envelope)
	if !second.Report.MigrationGate || string(first.Envelope) != string(second.Envelope) {
		t.Fatalf("not idempotent: first=%s second=%s report=%+v", first.Envelope, second.Envelope, second.Report)
	}
	bad := ConvertGu("bad", []byte(`{"schema_version":3,"blocks":[]}`))
	if bad.Report.MigrationGate || len(bad.Report.Unsupported) == 0 {
		t.Fatalf("unsupported version accepted: %+v", bad.Report)
	}
}

func TestAmbiguousNeverGuessesAndGateStaysClosed(t *testing.T) {
	r := ConvertGu("ambiguous", []byte(`{"active":{"sleep":2}}`))
	if len(r.Report.Ambiguous) != 1 || len(r.Report.SuggestedMapping) != 1 || r.Report.MigrationGate {
		t.Fatalf("report = %+v", r.Report)
	}
}

func TestDuplicateAliasesAndExplicitDivergenceAreReported(t *testing.T) {
	r := convertFlatNative(FormatItem, "item", []byte(`{"restore_hp":10,"heal":20}`), "ON_CONSUME", map[string]string{"restore_hp": "HEAL", "heal": "HEAL"})
	if len(r.Report.Duplicate) != 1 || r.Report.MigrationGate {
		t.Fatalf("duplicate report = %+v", r.Report)
	}
	g := ConvertGu("gu", []byte(`{"active":{"damage_flat":1},"blocks":[{"id":"OTHER","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":2}]}]}`))
	if len(g.Report.DivergentExplicitBlockLegacy) != 1 {
		t.Fatalf("divergence report = %+v", g.Report)
	}
}

func TestReportJSONUsesStableMachineReadableArrays(t *testing.T) {
	r := ConvertItem("empty", []byte(`{}`))
	b, err := json.Marshal(r.Report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"converted", "golden_parity", "differential_parity", "exact_parity", "unsupported", "ambiguous", "duplicate", "divergent_explicit_block_legacy", "suggested_mapping"} {
		if _, ok := decoded[key].([]any); !ok {
			t.Fatalf("%s is not an array: %s", key, b)
		}
	}
}

func TestEvaluateMigrationGateLevels(t *testing.T) {
	finding := Finding{Format: FormatGu, SourceID: "fixture", Key: "damage_flat"}
	tests := []struct {
		name   string
		report Report
		code   string
		pass   bool
	}{
		{"unsupported", Report{Validated: true, Unsupported: []Finding{finding}}, "unsupported_effect", false},
		{"ambiguous", Report{Validated: true, Ambiguous: []Finding{finding}}, "ambiguous_effect", false},
		{"divergence", Report{Validated: true, DivergentExplicitBlockLegacy: []Finding{finding}}, "differential_divergence", false},
		{"missing parity", Report{Total: 1, Validated: true, Converted: []Finding{finding}}, "missing_parity", false},
		{"shadow mismatch", Report{Total: 1, Validated: true, Converted: []Finding{finding}, ExactParity: []Finding{finding}, ShadowMismatchCount: 1}, "shadow_mismatch", false},
		{"empty envelope", Report{Validated: true}, "", true},
		{"shadow not compared", Report{Total: 1, Validated: true, Converted: []Finding{finding}, ExactParity: []Finding{finding}}, "shadow_not_compared", false},
		{"unbound shadow", Report{Total: 1, Validated: true, Converted: []Finding{finding}, ExactParity: []Finding{finding}, ShadowCompared: true}, "shadow_evidence_unbound", false},
		{"selected zero-mismatch shadow", Report{Total: 1, Validated: true, Converted: []Finding{finding}, ExactParity: []Finding{finding}, ShadowCompared: true, ShadowSnapshotID: "snapshot-1", ShadowCatalogRevision: 7}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateMigrationGate(tt.report)
			if got.MigrationGate != tt.pass || got.ErrorCode != tt.code {
				t.Fatalf("decision = %+v", got)
			}
		})
	}
}

func TestParityEvidenceMustCoverEachConvertedEffect(t *testing.T) {
	one := Finding{Format: FormatGu, SourceID: "fixture", Key: "one"}
	two := Finding{Format: FormatGu, SourceID: "fixture", Key: "two"}
	report := Report{Total: 2, Validated: true, Converted: []Finding{one, two}}
	RecordGoldenParity(&report, []Finding{one, one})
	if report.MigrationGate || report.ErrorCode != "missing_parity" || len(report.ExactParity) != 1 {
		t.Fatalf("duplicate evidence opened gate: %+v", report)
	}
}

func TestMalformedExplicitGuBlocksFail(t *testing.T) {
	r := ConvertGu("bad-blocks", []byte(`{"blocks":{"not":"an array"}}`))
	if r.Report.MigrationGate || len(r.Report.Unsupported) != 1 {
		t.Fatalf("malformed explicit blocks passed: %+v", r.Report)
	}
}

func TestConversionNeedsRealParityEvidence(t *testing.T) {
	r := ConvertGu("fixture", []byte(`{"active":{"damage_flat":2}}`))
	if !r.Report.Validated || r.Report.MigrationGate || len(r.Report.ExactParity) != 0 {
		t.Fatalf("conversion incorrectly established parity: %+v", r.Report)
	}
	RecordGoldenParity(&r.Report, append([]Finding(nil), r.Report.Converted...))
	if r.Report.MigrationGate || r.Report.ErrorCode != "shadow_not_compared" || len(r.Report.ExactParity) != 1 {
		t.Fatalf("golden evidence bypassed shadow gate: %+v", r.Report)
	}
	RecordShadowParityFor(&r.Report, "snapshot-1", 7, 0)
	if !r.Report.MigrationGate {
		t.Fatalf("bound zero-mismatch shadow did not open gate: %+v", r.Report)
	}
}

func TestMatchingExplicitBlocksProvideDifferentialParity(t *testing.T) {
	legacy := []byte(`{"active":{"damage_flat":2}}`)
	first := ConvertGu("fixture", legacy)
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(first.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	withBlocks := []byte(`{"active":{"damage_flat":2},"blocks":` + string(envelope["blocks"]) + `}`)
	result := ConvertGu("fixture", withBlocks)
	if result.Report.MigrationGate || result.Report.ErrorCode != "shadow_not_compared" || len(result.Report.DifferentialParity) != 1 || len(result.Report.ExactParity) != 1 {
		t.Fatalf("matching differential result bypassed shadow gate: %+v", result.Report)
	}
	RecordShadowParityFor(&result.Report, "snapshot-2", 9, 0)
	if !result.Report.MigrationGate {
		t.Fatalf("bound zero-mismatch shadow did not open gate: %+v", result.Report)
	}
}

func TestShadowEvidenceMustBeBoundToSelectedCatalog(t *testing.T) {
	finding := Finding{Format: FormatGu, SourceID: "fixture", Key: "damage_flat"}
	report := Report{Total: 1, Validated: true, Converted: []Finding{finding}}
	RecordGoldenParity(&report, []Finding{finding})
	RecordShadowParity(&report, 0)
	if report.MigrationGate || report.ErrorCode != "shadow_evidence_unbound" {
		t.Fatalf("unbound evidence opened gate: %+v", report)
	}
	RecordShadowParityFor(&report, "snapshot-3", 11, 0)
	if !report.MigrationGate || !report.ShadowParity {
		t.Fatalf("selected evidence did not open gate: %+v", report)
	}
}

func TestEmptyEnvelopePassesGate(t *testing.T) {
	r := ConvertItem("empty", []byte(`{}`))
	if !r.Report.Validated || !r.Report.MigrationGate || r.Report.Total != 0 {
		t.Fatalf("empty envelope report = %+v; envelope=%s", r.Report, r.Envelope)
	}
}

func TestGuDomainMappingsUseRuntimeTriggersAndUnits(t *testing.T) {
	r := ConvertGu("domain", []byte(`{"active":{"capture_bonus":0.2,"portal_create":true,"random_portal":false},"passive":{"time_speed_multiplier":2,"success_rate_bonus":15,"path_conflict_reduction":100},"growth":{"max_hp":10,"hours_to_max":24}}`))
	if !r.Report.Validated {
		t.Fatalf("candidate did not validate: %s", r.Envelope)
	}
	var envelope struct {
		Blocks []struct {
			Trigger    string      `json:"trigger"`
			Operations []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	wantTrigger := map[string]string{"CAPTURE_BONUS": "CAPABILITY", "PORTAL_CREATE": "CAPABILITY", "TIME_SPEED": "ALWAYS_ON", "REFINE_SUCCESS": "ALWAYS_ON", "PATH_CONFLICT_REDUCTION": "ALWAYS_ON", "HP": "ALWAYS_ON"}
	wantUnit := map[string]string{"TIME_SPEED": "RATIO", "REFINE_SUCCESS": "PERCENT", "PATH_CONFLICT_REDUCTION": "PERCENT"}
	for _, b := range envelope.Blocks {
		op := b.Operations[0]
		key := op.Code
		if key == "" {
			key = op.Stat
		}
		if got := wantTrigger[key]; got != "" && b.Trigger != got {
			t.Errorf("%s trigger = %s, want %s", key, b.Trigger, got)
		}
		if got := wantUnit[key]; got != "" && op.Unit != got {
			t.Errorf("%s unit = %s, want %s", key, op.Unit, got)
		}
	}
}

func TestGuPeriodicResourceAliasesBecomeTypedOperations(t *testing.T) {
	r := ConvertGu("ticks", []byte(`{"other":[{"event_trigger":"ON_TICK","action_code":"MANA_REGEN_TICK","params":{"value":4}},{"event_trigger":"ON_TICK","action_code":"GENERATE_STONES_TICK","params":{"value":7}}]}`))
	if !r.Report.Validated {
		t.Fatalf("candidate did not validate: %s; report=%+v", r.Envelope, r.Report)
	}
	var envelope struct {
		Blocks []struct {
			Operations []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	want := map[string]operation{
		"MANA":           {Op: "PERIODIC_RESOURCE", SubType: "REGEN_PER_HOUR", Resource: "MANA", Value: float64(4)},
		"PRIMEVAL_STONE": {Op: "PERIODIC_RESOURCE", SubType: "GENERATION_WORKER_TICK", Resource: "PRIMEVAL_STONE", Value: float64(7)},
	}
	for _, block := range envelope.Blocks {
		got := block.Operations[0]
		wantOp, ok := want[got.Resource]
		if !ok || got.Op != wantOp.Op || got.SubType != wantOp.SubType || got.Value != wantOp.Value {
			t.Errorf("periodic operation = %+v; want one of %+v", got, want)
		}
		delete(want, got.Resource)
	}
	if len(want) != 0 {
		t.Errorf("missing periodic operations: %+v", want)
	}
}

func TestGuConsumeAliasesBecomeTypedOperations(t *testing.T) {
	r := ConvertGu("consume-resources", []byte(`{"other":[{"event_trigger":"ON_CONSUME","action_code":"ADD_LIFESPAN_YEARS","params":{"value":3}},{"event_trigger":"ON_CONSUME","action_code":"ADD_SOUL_MAX","params":{"value":5}},{"event_trigger":"ON_CONSUME","action_code":"RESTORE_MENTAL","params":{"value":7}},{"event_trigger":"ON_CONSUME","action_code":"RANK_PROGRESS"},{"event_trigger":"ON_CONSUME","action_code":"MAN_BEAST_BURIAL"},{"event_trigger":"ON_CONSUME","action_code":"REDUCE_STRESS","params":{"value":25}},{"event_trigger":"ON_CONSUME","action_code":"INCREASE_TIME_SPEED","params":{"value":0.5}},{"event_trigger":"ON_CONSUME","action_code":"DECREASE_TIME_SPEED","params":{"value":-0.5}},{"event_trigger":"ON_CONSUME","action_code":"THIRD_WATCH"}]}`))
	if !r.Report.Validated {
		t.Fatalf("candidate did not validate: %s; report=%+v", r.Envelope, r.Report)
	}
	var envelope struct {
		Blocks []struct {
			ID         string      `json:"id"`
			Operations []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	want := map[string]operation{
		"LIFESPAN": {Op: "GRANT_RESOURCE", Resource: "LIFESPAN", Value: float64(3)},
		"SOUL":     {Op: "MODIFY_MAX_RESOURCE", Resource: "SOUL", Value: float64(5)},
		"THOUGHT":  {Op: "RESTORE_RESOURCE", Resource: "THOUGHT", Value: float64(7)},
	}
	wantMajorRealm := false
	wantAperture := map[string]operation{
		"REDUCE_STRESS":       {Op: "MODIFY_APERTURE", SubType: "STRESS_REDUCE", Value: float64(25)},
		"INCREASE_TIME_SPEED": {Op: "MODIFY_APERTURE", SubType: "TIME_RATE_ADD", Value: 0.5},
		"DECREASE_TIME_SPEED": {Op: "MODIFY_APERTURE", SubType: "TIME_RATE_SUBTRACT", Value: 0.1},
		"THIRD_WATCH":         {Op: "MODIFY_APERTURE", SubType: "TIME_RATE_MULTIPLY", Value: float64(3)},
	}
	for _, block := range envelope.Blocks {
		got := block.Operations[0]
		if block.ID == "MAN_BEAST_BURIAL" {
			if got.Op != "ADVANCE_MAJOR_REALM" {
				t.Errorf("major realm operation = %+v; want ADVANCE_MAJOR_REALM", got)
			}
			wantMajorRealm = true
			continue
		}
		if block.ID == "RANK_PROGRESS" {
			if got.Op != "ADVANCE_PROGRESSION" {
				t.Errorf("progression operation = %+v; want ADVANCE_PROGRESSION", got)
			}
			continue
		}
		if wantOp, ok := wantAperture[block.ID]; ok {
			gotValue, _ := got.Value.(float64)
			wantValue, _ := wantOp.Value.(float64)
			if got.Op != wantOp.Op || got.SubType != wantOp.SubType || math.Abs(gotValue-wantValue) > 1e-9 {
				t.Errorf("aperture operation = %+v; want %+v", got, wantOp)
			}
			delete(wantAperture, block.ID)
			continue
		}
		wantOp, ok := want[got.Resource]
		if !ok || got.Op != wantOp.Op || got.Value != wantOp.Value {
			t.Errorf("consume resource operation = %+v; want one of %+v", got, want)
		}
		delete(want, got.Resource)
	}
	if !wantMajorRealm {
		t.Error("MAN_BEAST_BURIAL was not converted")
	}
	if len(want) != 0 {
		t.Errorf("missing consume resource operations: %+v", want)
	}
	if len(wantAperture) != 0 {
		t.Errorf("missing aperture operations: %+v", wantAperture)
	}
}

func TestGuPathDamageModifierBecomesTypedStatMods(t *testing.T) {
	r := ConvertGu("path-damage", []byte(`{"other":[{"event_trigger":"ALWAYS_ON","action_code":"PATH_DAMAGE_MODIFIER","params":{"value":{"kill":0.25,"fire":0.1}}}]}`))
	if !r.Report.Validated {
		t.Fatalf("candidate did not validate: %s; report=%+v", r.Envelope, r.Report)
	}
	var envelope struct {
		Blocks []struct {
			Operations []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Blocks) != 1 || len(envelope.Blocks[0].Operations) != 2 {
		t.Fatalf("blocks = %+v; want one block with two path stats", envelope.Blocks)
	}
	want := []operation{
		{Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_DAMAGE", Path: "fire", Value: 0.1},
		{Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_DAMAGE", Path: "kill", Value: 0.25},
	}
	for i, got := range envelope.Blocks[0].Operations {
		if got.Op != want[i].Op || got.SubType != want[i].SubType || got.Stat != want[i].Stat || got.Path != want[i].Path || got.Value != want[i].Value {
			t.Errorf("operation %d = %+v, want %+v", i, got, want[i])
		}
	}
}

func TestGuStarThoughtRestoreBecomesPercentManaHeal(t *testing.T) {
	r := ConvertGu("star-thought-restore", []byte(`{"other":[{"event_trigger":"ON_ACTION","action_code":"STAR_THOUGHT_RESTORE"}]}`))
	if !r.Report.Validated {
		t.Fatalf("candidate did not validate: %s; report=%+v", r.Envelope, r.Report)
	}
	var envelope struct {
		Blocks []struct {
			Trigger string      `json:"trigger"`
			Target  target      `json:"target"`
			Ops     []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Blocks) != 1 || len(envelope.Blocks[0].Ops) != 1 {
		t.Fatalf("blocks = %+v; want one heal block", envelope.Blocks)
	}
	b := envelope.Blocks[0]
	op := b.Ops[0]
	if b.Trigger != "ON_ACTION" || b.Target.Selector != "TARGET" || op.Target == nil || op.Target.Selector != "SELF" || op.Op != "HEAL" || op.SubType != "RESOURCE" || op.Resource != "MANA" || op.Unit != "PERCENT" || op.Value != float64(30) {
		t.Fatalf("typed heal = trigger %s block target %+v op %+v; want operation-level self mana restore 30%%", b.Trigger, b.Target, op)
	}
}

func TestGuSimpleStatusHooksBecomeTypedStatusOperations(t *testing.T) {
	raw := []byte(`{"other":[{"event_trigger":"ON_ACTION","action_code":"WOOD_BIND"},{"event_trigger":"ON_ACTION","action_code":"ICE_DEEP_FREEZE"},{"event_trigger":"ON_ACTION","action_code":"LIGHT_BLINDING_FLASH"},{"event_trigger":"ON_ACTION","action_code":"SOUND_DISRUPT_SILENCE"},{"event_trigger":"ON_ACTION","action_code":"SOUND_RESONANCE"},{"event_trigger":"ON_ACTION","action_code":"ICE_FROSTBITE"},{"event_trigger":"ON_ACTION","action_code":"CHARM_GHOST_COME"},{"event_trigger":"ON_ACTION","action_code":"FIVE_RESTRICTION_DEEP_LIGHT"}]}`)
	r := ConvertGu("simple-status-hooks", raw)
	if !r.Report.Validated {
		t.Fatalf("candidate did not validate: %s; report=%+v", r.Envelope, r.Report)
	}
	var envelope struct {
		Blocks []struct {
			Trigger string      `json:"trigger"`
			Target  target      `json:"target"`
			Ops     []operation `json:"operations"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(r.Envelope, &envelope); err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		id    string
		value float64
		turns int
	}{
		"WOOD_BIND": {"BIND", 0, 2}, "ICE_DEEP_FREEZE": {"FREEZE", 0, 1},
		"LIGHT_BLINDING_FLASH": {"BLIND", 60, 2}, "SOUND_DISRUPT_SILENCE": {"SILENCE", 0, 2},
		"SOUND_RESONANCE": {"SOUL_WOUND", 25, 3}, "ICE_FROSTBITE": {"SLOW_DAMAGE", 35, 3},
		"CHARM_GHOST_COME": {"CHARM", 0, 2}, "FIVE_RESTRICTION_DEEP_LIGHT": {"GU_SEAL", 0, 3},
	}
	if len(envelope.Blocks) != len(want) {
		t.Fatalf("blocks = %d, want %d", len(envelope.Blocks), len(want))
	}
	for i, b := range envelope.Blocks {
		code := []string{"WOOD_BIND", "ICE_DEEP_FREEZE", "LIGHT_BLINDING_FLASH", "SOUND_DISRUPT_SILENCE", "SOUND_RESONANCE", "ICE_FROSTBITE", "CHARM_GHOST_COME", "FIVE_RESTRICTION_DEEP_LIGHT"}[i]
		w := want[code]
		if b.Trigger != "ON_ACTION" || b.Target.Selector != "TARGET" || len(b.Ops) != 1 || b.Ops[0].Op != "STATUS" || b.Ops[0].StatusID != w.id || b.Ops[0].Value != w.value || b.Ops[0].Turns != float64(w.turns) || b.Ops[0].Chance != float64(0) || b.Ops[0].ApplicationMode != "DIRECT" {
			t.Errorf("%s converted block = %+v; want guaranteed STATUS %s value %v turns %d on TARGET", code, b, w.id, w.value, w.turns)
		}
	}
}

func TestGuSummonAliasBecomesTypedOperation(t *testing.T) {
	r := ConvertGu("summoner", []byte(`{"other":[{"event_trigger":"ON_BATTLE_START","action_code":"SUMMON_MONSTER","params":{"monster_id":"monster_wolf_leader","hp_scale":0.5,"damage_scale":0.33,"position":"frontline"}}]}`))
	if !r.Report.Validated {
		t.Fatalf("summon candidate did not validate: %s; report=%+v", r.Envelope, r.Report)
	}
	if !bytes.Contains(r.Envelope, []byte(`"op":"SUMMON"`)) || bytes.Contains(r.Envelope, []byte(`"code":"SUMMON_MONSTER"`)) {
		t.Fatalf("summon alias was not converted to typed operation: %s", r.Envelope)
	}
}

func TestBlockedActionsRemainStatusMetadata(t *testing.T) {
	r := ConvertGameFX("status", []byte(`{"blocked_actions":["explore"]}`))
	if len(r.Report.Converted) != 0 || len(r.Report.Unsupported) != 1 {
		t.Fatalf("blocked_actions was converted as an effect: %+v", r.Report)
	}
}
