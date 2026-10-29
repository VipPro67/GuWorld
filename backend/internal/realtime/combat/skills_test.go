package combat

import "testing"

func TestDefaultLoadoutIsValidAndCoversAllModes(t *testing.T) {
	loadout := DefaultLoadout()
	if len(loadout) != 6 {
		t.Fatalf("slots=%d, want 6", len(loadout))
	}
	modes := map[Mode]bool{}
	for _, skill := range loadout {
		if !skill.Valid() {
			t.Fatalf("skill %+v is invalid", skill)
		}
		if _, ok := SkillBySlot(skill.Slot); !ok {
			t.Fatalf("slot %d is not addressable", skill.Slot)
		}
		modes[skill.Mode] = true
	}
	for _, mode := range []Mode{ModeTarget, ModeDirection, ModeGround, ModeSelf} {
		if !modes[mode] {
			t.Fatalf("loadout is missing mode %q", mode)
		}
	}
	if _, ok := SkillBySlot(7); ok {
		t.Fatal("slot 7 must not exist")
	}
}

func TestInvalidSkillsAreRejected(t *testing.T) {
	skill := DefaultLoadout()[1]
	skill.Slot = 13
	if skill.Valid() {
		t.Fatal("slot outside 1..12 accepted")
	}
	skill = DefaultLoadout()[1]
	skill.ProjectileSpeed = 0
	if skill.Valid() {
		t.Fatal("direction skill without projectile speed accepted")
	}
	skill = DefaultLoadout()[3]
	skill.Shield = 0
	if skill.Valid() {
		t.Fatal("self skill without heal/shield accepted")
	}
}
