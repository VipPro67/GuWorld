package sim

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func newTestGuHouse(id, name string, rank int, coreCount, auxCount int) *models.GuHouseCombatProfile {
	cores := make([]string, coreCount)
	for i := 0; i < coreCount; i++ {
		cores[i] = "core_gu_" + string(rune('a'+i))
	}
	return &models.GuHouseCombatProfile{
		ID:              id,
		Name:            name,
		Rank:            rank,
		CoreGuIDs:       cores,
		AuxGuCount:      auxCount,
		MaxHouseHP:      4500,
		CurrentHouseHP:  4500,
		HouseDefense:    80,
		DamageReduction: 0.25,
	}
}

func setupGuHouseTestPlayer(house *models.GuHouseCombatProfile, durability float64, mana float64) (*World, *Entity) {
	w, _ := NewWorld("test_zone", DefaultQueueSize)
	pSkills := [13]combat.Skill{
		{},
		{ID: "HUMAN_SKILL_1", Slot: 1, Name: "Nhân Kỹ 1", Mode: combat.ModeDirection, Range: 5, Damage: 20, CooldownTicks: 20},
		{ID: "HUMAN_SKILL_2", Slot: 2, Name: "Nhân Kỹ 2", Mode: combat.ModeSelf, Shield: 50, CooldownTicks: 40},
	}
	player := Entity{
		ID:          "p1",
		CharacterID: "c1",
		Kind:        EntityPlayer,
		Transform:   Transform{Position: Vec2{X: 10, Y: 10}},
		Vitals:      Vitals{HP: 1000, MaxHP: 1000},
		Radius:      0.5,
		Combat: CombatState{
			Mana:    mana,
			MaxMana: 1000,
			Skills:  pSkills,
		},
		EquippedGuHouse:   house,
		GuHouseDurability: durability,
	}
	_ = w.AddEntity(player)
	return w, w.entities["p1"]
}

func TestCalculateGuHouseUpkeep(t *testing.T) {
	// Scenario 1: Same rank (Rank 7 house, Rank 7 player, 3 cores, 100 aux)
	// U_base = (3 * 0.04 + 100 * 0.0005) * 1.0 = 0.17
	// M = 1.0, F_sub = 1.0 -> 0.17
	u1 := CalculateGuHouseUpkeep(7, 7, 1, 3, 100)
	if math.Abs(u1-0.17) > 1e-4 {
		t.Fatalf("expected upkeep 0.17, got %v", u1)
	}

	// Scenario 2: Over-rank (Rank 7 house, Rank 6 peak player (subrank 4), 3 cores, 100 aux)
	// U_base = 0.17, M = 100 * 1.5 = 150, F_sub = 0.85
	// Upkeep = 0.17 * 150 * 0.85 = 21.675
	u2 := CalculateGuHouseUpkeep(7, 6, 4, 3, 100)
	if math.Abs(u2-21.675) > 1e-3 {
		t.Fatalf("expected upkeep 21.675, got %v", u2)
	}

	// Scenario 3: High rank pilot low rank house (Rank 7 house, Rank 8 player, 3 cores, 100 aux)
	// U_base = 0.17, M = 100^(-1) * 0.8 = 0.008, F_sub = 1.0
	// Upkeep = 0.17 * 0.008 = 0.00136
	u3 := CalculateGuHouseUpkeep(7, 8, 1, 3, 100)
	if math.Abs(u3-0.00136) > 1e-5 {
		t.Fatalf("expected upkeep 0.00136, got %v", u3)
	}
}

func TestGuHouseIntegrationValidationRejections(t *testing.T) {
	house := newTestGuHouse("house_1", "Bát Quái Cổ Ốc", 7, 3, 100)

	// 1. Not equipped
	w1, p1 := setupGuHouseTestPlayer(nil, 100, 200)
	w1.setGuHouseIntegrateLocked(p1, GuHouseIntegrateCommand{Action: "ENTER"})
	if p1.Transformation.IsWindup || p1.Transformation.Active {
		t.Fatalf("should not integrate without equipped house")
	}

	// 2. Broken durability
	w2, p2 := setupGuHouseTestPlayer(house, 0, 200)
	w2.setGuHouseIntegrateLocked(p2, GuHouseIntegrateCommand{Action: "ENTER"})
	if p2.Transformation.IsWindup || p2.Transformation.Active {
		t.Fatalf("should not integrate with broken house")
	}

	// 3. Insufficient mana (need 50)
	w3, p3 := setupGuHouseTestPlayer(house, 100, 30)
	w3.setGuHouseIntegrateLocked(p3, GuHouseIntegrateCommand{Action: "ENTER"})
	if p3.Transformation.IsWindup || p3.Transformation.Active {
		t.Fatalf("should not integrate with insufficient mana")
	}

	// 4. Hard CC active (Stunned)
	w4, p4 := setupGuHouseTestPlayer(house, 100, 200)
	p4.Combat.Statuses = append(p4.Combat.Statuses, StatusEffect{ID: "STUN", DurationTicks: 20, Debuff: true})
	w4.setGuHouseIntegrateLocked(p4, GuHouseIntegrateCommand{Action: "ENTER"})
	if p4.Transformation.IsWindup || p4.Transformation.Active {
		t.Fatalf("should not integrate while stunned")
	}
}

func TestGuHouseWindupAndInterruption(t *testing.T) {
	house := newTestGuHouse("house_1", "Bát Quái Cổ Ốc", 7, 3, 100)
	w, p := setupGuHouseTestPlayer(house, 100, 200)

	// Enter windup
	w.setGuHouseIntegrateLocked(p, GuHouseIntegrateCommand{Action: "ENTER"})
	if !p.Transformation.IsWindup {
		t.Fatalf("expected entity to be in windup")
	}
	if p.Combat.Mana != 150 {
		t.Fatalf("expected 50 mana deducted, got %v", p.Combat.Mana)
	}

	// Hard CC interruption
	w.InterruptTransformation(p, "STUN")
	if p.Transformation.IsWindup || p.Transformation.Active {
		t.Fatalf("expected windup to be cancelled by CC")
	}
}

func TestGuHouseSuccessfulTransformationAndSkillPalette(t *testing.T) {
	house := newTestGuHouse("house_1", "Bát Quái Cổ Ốc", 7, 3, 100)
	w, p := setupGuHouseTestPlayer(house, 100, 200)

	w.setGuHouseIntegrateLocked(p, GuHouseIntegrateCommand{Action: "ENTER"})
	if !p.Transformation.IsWindup {
		t.Fatalf("expected windup to begin")
	}

	// Advance 40 ticks
	for i := 0; i < 40; i++ {
		w.tick++
		w.tickTransformationLocked()
	}

	if p.Transformation.IsWindup {
		t.Fatalf("windup should be finished")
	}
	if !p.Transformation.Active {
		t.Fatalf("transformation should be active")
	}
	if p.Transformation.FormKind != TransformationGuHouse {
		t.Fatalf("expected form kind GU_HOUSE, got %v", p.Transformation.FormKind)
	}
	if p.Transformation.MaxFormHP != 4500 || p.Transformation.CurrentFormHP != 4500 {
		t.Fatalf("expected 4500 Form HP, got %v", p.Transformation.CurrentFormHP)
	}
	if math.Abs(p.Radius-0.9) > 1e-4 { // 0.5 * 1.8 = 0.9
		t.Fatalf("expected radius 0.9, got %v", p.Radius)
	}

	// Verify 6-slot House Strike palette
	if p.Combat.Skills[1].ID != "HOUSE_STRIKE_ARTILLERY" {
		t.Fatalf("slot 1 expected HOUSE_STRIKE_ARTILLERY, got %v", p.Combat.Skills[1].ID)
	}
	if p.Combat.Skills[2].ID != "HOUSE_STRIKE_TRAMPLE" {
		t.Fatalf("slot 2 expected HOUSE_STRIKE_TRAMPLE, got %v", p.Combat.Skills[2].ID)
	}
	if p.Combat.Skills[3].ID != "HOUSE_STRIKE_BARRIER" {
		t.Fatalf("slot 3 expected HOUSE_STRIKE_BARRIER, got %v", p.Combat.Skills[3].ID)
	}
	if p.Combat.Skills[4].ID != "HOUSE_STRIKE_MENDING" {
		t.Fatalf("slot 4 expected HOUSE_STRIKE_MENDING, got %v", p.Combat.Skills[4].ID)
	}
	if p.Combat.Skills[5].ID != "HOUSE_STRIKE_ULTIMATE" {
		t.Fatalf("slot 5 expected HOUSE_STRIKE_ULTIMATE, got %v", p.Combat.Skills[5].ID)
	}
	if p.Combat.Skills[6].ID != "REVERT_GU_HOUSE" {
		t.Fatalf("slot 6 expected REVERT_GU_HOUSE, got %v", p.Combat.Skills[6].ID)
	}

	// Slots > 6 must be rejected
	w.startCastLocked(p, CastCommand{Slot: 7, Mode: combat.ModeDirection})
	events := w.events
	foundReject := false
	for _, ev := range events {
		if ev.Type == combat.EventCastRejected && ev.Reason == "form_skill_locked" {
			foundReject = true
			break
		}
	}
	if !foundReject {
		t.Fatalf("expected slot 7 to be rejected with form_skill_locked")
	}
}

func TestGuHouse100PercentDamageAbsorptionAndMending(t *testing.T) {
	house := newTestGuHouse("house_1", "Bát Quái Cổ Ốc", 7, 3, 100)
	w, p := setupGuHouseTestPlayer(house, 100, 200)

	// Complete transformation directly
	p.Transformation.OriginalRadius = p.Radius
	w.completeGuHouseTransformationLocked(p)

	initialPlayerHP := p.Vitals.HP

	// Apply 1000 damage
	// Damage reduction is 25% -> 750 reduced damage absorbed by Form HP
	spillover := w.applyFormDamageLocked(p, 1000)
	if spillover != 0 {
		t.Fatalf("expected 0 spillover, got %v", spillover)
	}
	if p.Vitals.HP != initialPlayerHP {
		t.Fatalf("player human HP must not be damaged! HP was %v, now %v", initialPlayerHP, p.Vitals.HP)
	}
	expectedFormHP := 4500.0 - 750.0
	if math.Abs(p.Transformation.CurrentFormHP-expectedFormHP) > 1e-4 {
		t.Fatalf("expected form HP %v, got %v", expectedFormHP, p.Transformation.CurrentFormHP)
	}

	// Test Slot 4 Structural Mending: heals 15% Form HP (675)
	w.applySelfSupportLocked(p, p.Combat.Skills[4])
	if p.Vitals.HP != initialPlayerHP {
		t.Fatalf("mending must not heal human HP!")
	}
	if math.Abs(p.Transformation.CurrentFormHP-(expectedFormHP+675)) > 1e-4 {
		t.Fatalf("expected form HP %v, got %v", expectedFormHP+675, p.Transformation.CurrentFormHP)
	}
}

func TestGuHouseStructuralBreachBacklash(t *testing.T) {
	house := newTestGuHouse("house_1", "Bát Quái Cổ Ốc", 7, 3, 100)
	w, p := setupGuHouseTestPlayer(house, 100, 200)

	w.completeGuHouseTransformationLocked(p)

	// Inflict massive lethal damage to Gu House (10,000 damage)
	spillover := w.applyFormDamageLocked(p, 10000)
	if spillover != 0 {
		t.Fatalf("expected 0 spillover on breach, got %v", spillover)
	}

	// Gu House should be broken and deactivated
	if p.Transformation.Active {
		t.Fatalf("transformation should be deactivated upon structural breach")
	}

	// Backlash damage: 30% of 1000 Max HP = 300 damage -> HP should be 700
	if math.Abs(p.Vitals.HP-700) > 1e-4 {
		t.Fatalf("expected player HP 700 after 30%% backlash, got %v", p.Vitals.HP)
	}

	// Player should have STUN status (duration 60 ticks = 3.0s)
	stunned := false
	for _, st := range p.Combat.Statuses {
		if st.ID == "STUN" && st.DurationTicks == 60 {
			stunned = true
			break
		}
	}
	if !stunned {
		t.Fatalf("player must be stunned for 60 ticks upon structural breach")
	}

	// Gu house durability should be 0
	if p.GuHouseDurability != 0 {
		t.Fatalf("expected Gu House durability 0, got %v", p.GuHouseDurability)
	}

	// Restored human skills
	if p.Combat.Skills[1].ID != "HUMAN_SKILL_1" {
		t.Fatalf("human skills must be restored upon breach")
	}

	// Original radius restored
	if math.Abs(p.Radius-0.5) > 1e-4 {
		t.Fatalf("expected radius restored to 0.5, got %v", p.Radius)
	}
}

func TestGuHouseSafeExitTieredDurabilityWear(t *testing.T) {
	house := newTestGuHouse("house_1", "Bát Quái Cổ Ốc", 7, 3, 100)
	w, p := setupGuHouseTestPlayer(house, 100, 200)

	w.completeGuHouseTransformationLocked(p)

	// Lose 50% Form HP (from 4500 to 2250)
	p.Transformation.CurrentFormHP = 2250

	// Voluntary safe exit via Slot 6
	w.startCastLocked(p, CastCommand{Slot: 6, Mode: combat.ModeSelf})

	if p.Transformation.Active {
		t.Fatalf("transformation should be deactivated upon exit")
	}

	// Wear formula: 0.5% + 10% * (1 - 2250 / 4500) = 0.5 + 5.0 = 5.5% wear
	// Expected durability: 100 - 5.5 = 94.5%
	if math.Abs(p.GuHouseDurability-94.5) > 1e-4 {
		t.Fatalf("expected durability 94.5, got %v", p.GuHouseDurability)
	}
	if math.Abs(p.GuHouseWearDamage-5.5) > 1e-4 {
		t.Fatalf("expected wear damage 5.5, got %v", p.GuHouseWearDamage)
	}

	// Human skills restored
	if p.Combat.Skills[1].ID != "HUMAN_SKILL_1" {
		t.Fatalf("human skills must be restored upon voluntary exit")
	}
}

func TestGuHouseUpkeepDrainAndManaDepletion(t *testing.T) {
	house := newTestGuHouse("house_1", "Bát Quái Cổ Ốc", 7, 3, 100)
	// Give only 1.0 mana to force quick depletion
	w, p := setupGuHouseTestPlayer(house, 100, 1.0)

	w.completeGuHouseTransformationLocked(p)

	// Upkeep per tick: 0.17 / 20 = 0.0085 mana per tick
	initialMana := p.Combat.Mana
	w.tick++
	w.tickTransformationLocked()

	if p.Combat.Mana >= initialMana {
		t.Fatalf("expected mana to decrease from upkeep")
	}

	// Now set mana to almost zero
	p.Combat.Mana = 0.0001
	w.tick++
	w.tickTransformationLocked()

	// Should be forced to revert due to mana depletion
	if p.Transformation.Active {
		t.Fatalf("expected transformation to terminate on mana depletion")
	}

	// Should have EXHAUSTION debuff
	exhausted := false
	for _, st := range p.Combat.Statuses {
		if st.ID == "EXHAUSTION" {
			exhausted = true
			break
		}
	}
	if !exhausted {
		t.Fatalf("player must suffer EXHAUSTION debuff when mana depleted")
	}
}
