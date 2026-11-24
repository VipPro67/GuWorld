package simulator

import "testing"

func TestChainDamagePrimitiveMatchesLegacyHook(t *testing.T) {
	newTargets := func() ([]*CombatantAuto, *CombatantAuto, *CombatantAuto) {
		primary := &CombatantAuto{ID: "1", Name: "Primary", HP: 500, MaxHP: 500}
		secondary := &CombatantAuto{ID: "2", Name: "Secondary", HP: 500, MaxHP: 500}
		return []*CombatantAuto{primary, secondary}, primary, secondary
	}
	legacyTargets, legacyPrimary, legacySecondary := newTargets()
	legacyLogs := []CombatLog{}
	legacy := &combatActionRunner{turn: 2, turnLogs: &legacyLogs}
	legacy.applyChainDamage(combatPhaseContext{
		actor:       &CombatantAuto{ID: "9", Name: "Caster"},
		action:      &SimAction{Damage: 100, Mechanics: []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "CHAIN_LIGHTNING"}}},
		firstTarget: legacyPrimary, targets: legacyTargets,
	})

	typedTargets, typedPrimary, typedSecondary := newTargets()
	typedLogs := []CombatLog{}
	typed := &combatActionRunner{turn: 2, turnLogs: &typedLogs}
	typed.applyChainDamage(combatPhaseContext{
		actor: &CombatantAuto{ID: "9", Name: "Caster"},
		action: &SimAction{Damage: 100, OperationEffects: []SimActionOperation{{Action: &SimAction{ChainDamages: []ChainDamage{{
			Fraction: 0.5, MinDamage: 30, TargetCount: 1, LogAction: "CHAIN_LIGHTNING",
			Message: "[Lôi Đạo - Lôi Điện Truyền Dẫn] Tia lôi điện phóng lan sang {target}, gây thêm {damage} sát thương điện giật!",
		}}}}}},
		firstTarget: typedPrimary, targets: typedTargets,
	})

	if legacyPrimary.HP != typedPrimary.HP || legacySecondary.HP != typedSecondary.HP || legacySecondary.HP != 450 {
		t.Fatalf("legacy and typed HP differ: legacy=(%v,%v), typed=(%v,%v)", legacyPrimary.HP, legacySecondary.HP, typedPrimary.HP, typedSecondary.HP)
	}
	if len(legacyLogs) != 1 || len(typedLogs) != 1 || legacyLogs[0] != typedLogs[0] {
		t.Fatalf("legacy and typed logs differ: legacy=%+v typed=%+v", legacyLogs, typedLogs)
	}
}
