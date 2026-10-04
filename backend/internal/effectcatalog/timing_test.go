package effectcatalog

import "testing"

// Every manifest definition must map to exactly one firing timing, so a newly
// added key/trigger without a timing classification fails loudly here.
func TestEveryDefinitionHasTiming(t *testing.T) {
	for _, def := range Definitions() {
		if _, ok := FieldTiming(def.Section, def.Key); !ok {
			t.Errorf("no timing for field %s.%s", def.Section, def.Key)
		}
	}
	for _, def := range MechanicDefinitions() {
		if _, ok := MechanicTiming(def.Trigger); !ok {
			t.Errorf("no timing for trigger %s/%s", def.Trigger, def.ActionCode)
		}
	}
	for _, def := range StatusDefinitions() {
		if timing, ok := StatusTiming(def.Key); !ok || timing != TimingOnEvent {
			t.Errorf("manifest status %s must be ON_EVENT, got %q ok=%v", def.Key, timing, ok)
		}
	}
}

func TestMechanicTimingGroups(t *testing.T) {
	cases := map[string]Timing{
		"ON_CONSUME":          TimingOnAction,
		"ON_USE":              TimingOnAction,
		"ON_ACTION":           TimingOnAction,
		"ON_TICK":             TimingScheduled,
		"CAPABILITY":          TimingCapability,
		"ON_CAST":             TimingOnEvent,
		"ON_HIT":              TimingOnEvent,
		"BEFORE_HIT":          TimingOnEvent,
		"BEFORE_ENEMY_ATTACK": TimingOnEvent,
		"ON_TURN_START":       TimingOnEvent,
		"ON_BATTLE_START":     TimingOnEvent,
		"ON_DEATH":            TimingOnEvent,
		"PASSIVE":             TimingOnEvent,
		"COMBAT_PASSIVE":      TimingOnEvent,
	}
	for trigger, want := range cases {
		got, ok := MechanicTiming(trigger)
		if !ok || got != want {
			t.Errorf("trigger %s: got %q ok=%v, want %q", trigger, got, ok, want)
		}
	}
}

func TestFieldTimingGroups(t *testing.T) {
	if got, _ := FieldTiming("passive", "stat_base_attack"); got != TimingAlwaysOn {
		t.Errorf("passive.stat_base_attack = %q, want ALWAYS_ON", got)
	}
	if got, _ := FieldTiming("active", "damage_flat"); got != TimingOnAction {
		t.Errorf("active.damage_flat = %q, want ON_ACTION", got)
	}
	if got, _ := FieldTiming("growth", "max_attack"); got != TimingScheduled {
		t.Errorf("growth.max_attack = %q, want SCHEDULED", got)
	}
	if got, _ := FieldTiming("active", "description"); got != TimingMeta {
		t.Errorf("active.description = %q, want META", got)
	}
	if _, ok := FieldTiming("active", "no_such_key"); ok {
		t.Error("unknown field must return ok=false")
	}
}
