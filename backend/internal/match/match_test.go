package match

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func testParticipant(x, y float64) *Participant {
	return &Participant{
		ID: uuid.New(), Name: "test",
		X: x, Y: y,
		HP: 200, MaxHP: 200,
		Mana: 100, MaxMana: 100,
		Attack: 10, Defense: 10, Speed: 10,
	}
}

func targetSkill(guID string) Skill {
	return Skill{
		GuID: guID, Rank: 1,
		Profile: models.MobaProfile{
			CastMode: models.MobaCastTarget, Shape: models.MobaShapeSingle,
			CastRange: 12, CooldownMs: 3000, MaxTargets: 1,
		},
		Payload:  SkillPayload{DamageFlat: 30},
		ManaCost: 10,
	}
}

func TestCastAppliesDamageCooldownAndMana(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 10)
	caster.Kit = []Skill{targetSkill("GU_TEST_NUKE")}
	target := testParticipant(10, 10)
	m.AddParticipant(caster)
	m.AddParticipant(target)

	if !m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_NUKE", Rank: 1, Mode: models.MobaCastTarget, TargetID: target.ID.String(),
	}}}) {
		t.Fatal("input should be accepted")
	}
	m.Step()

	// ResolveBaseDamage(30, 10) == 20.
	if target.HP != 180 {
		t.Fatalf("target hp = %v, want 180", target.HP)
	}
	if spent := 100 - caster.Mana; spent < 9.5 || spent > 10.1 {
		t.Fatalf("mana spent = %v, want ~10", spent)
	}
	if ready := caster.Cooldowns["GU_TEST_NUKE"]; ready != 1+ticksForMs(3000) {
		t.Fatalf("cooldown ready at tick %d, want %d", ready, 1+ticksForMs(3000))
	}

	// The cooldown must refuse an immediate second cast.
	m.SubmitInput(caster.ID, Input{Seq: 2, Casts: []CastInput{{
		GuID: "GU_TEST_NUKE", Rank: 1, Mode: models.MobaCastTarget, TargetID: target.ID.String(),
	}}})
	m.Step()
	if target.HP != 180 {
		t.Fatalf("cooldown did not block the second cast, hp = %v", target.HP)
	}
}

func TestCastRejectsModeMismatch(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 10)
	caster.Kit = []Skill{targetSkill("GU_TEST_NUKE")}
	target := testParticipant(10, 10)
	m.AddParticipant(caster)
	m.AddParticipant(target)

	// A TARGET skill fired as a ground cast must be refused.
	m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_NUKE", Rank: 1, Mode: models.MobaCastGround, AX: 10, AY: 10,
	}}})
	m.Step()
	if target.HP != 200 {
		t.Fatalf("mode mismatch should not deal damage, hp = %v", target.HP)
	}
}

func TestLineSkillshotPierces(t *testing.T) {
	run := func(pierce bool) (float64, float64) {
		m := NewMatch(uuid.New())
		caster := testParticipant(2, 10)
		caster.Kit = []Skill{{
			GuID: "GU_TEST_LINE", Rank: 1,
			Profile: models.MobaProfile{
				CastMode: models.MobaCastDirection, Shape: models.MobaShapeLine,
				Length: 16, Width: 1.2, CastRange: 16, Pierce: pierce, MaxTargets: 3,
			},
			Payload: SkillPayload{DamageFlat: 30},
		}}
		near := testParticipant(8, 10)
		far := testParticipant(12, 10)
		m.AddParticipant(caster)
		m.AddParticipant(near)
		m.AddParticipant(far)

		m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
			GuID: "GU_TEST_LINE", Rank: 1, Mode: models.MobaCastDirection, AX: 1, AY: 0,
		}}})
		m.Step()
		return near.HP, far.HP
	}

	if near, far := run(true); near != 180 || far != 180 {
		t.Fatalf("piercing line should hit both, got near=%v far=%v", near, far)
	}
	if near, far := run(false); near != 180 || far != 200 {
		t.Fatalf("non-piercing line should stop at the nearest, got near=%v far=%v", near, far)
	}
}

func TestDelayedGroundCastCanBeDodged(t *testing.T) {
	run := func(moveAway bool) float64 {
		m := NewMatch(uuid.New())
		caster := testParticipant(8, 12)
		caster.Kit = []Skill{{
			GuID: "GU_TEST_AOE", Rank: 1,
			Profile: models.MobaProfile{
				CastMode: models.MobaCastGround, Shape: models.MobaShapeCircle,
				Radius: 4, CastRange: 14, CastTimeMs: 1200, MaxTargets: 5,
			},
			Payload: SkillPayload{DamageFlat: 30},
		}}
		victim := testParticipant(20, 12)
		m.AddParticipant(caster)
		m.AddParticipant(victim)

		m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
			GuID: "GU_TEST_AOE", Rank: 1, Mode: models.MobaCastGround, AX: 20, AY: 12,
		}}})
		for tick := 1; tick <= 30; tick++ {
			if moveAway {
				m.SubmitInput(victim.ID, Input{Seq: tick, Move: &MoveInput{DX: 0, DY: 1}})
			}
			m.Step()
		}
		return victim.HP
	}

	if hp := run(false); hp != 180 {
		t.Fatalf("standing inside the marker should be hit, hp = %v", hp)
	}
	if hp := run(true); hp != 200 {
		t.Fatalf("walking out of the marker should dodge, hp = %v", hp)
	}
}

func TestEmptyFrameMarshalsListsAsArrays(t *testing.T) {
	// Go marshals a nil slice as null, which is not iterable on the client.
	// Every list in the frame must serialize as [] when empty.
	m := NewMatch(uuid.New())
	m.AddParticipant(testParticipant(1, 1))

	raw, err := json.Marshal(m.Snapshot())
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	for _, expected := range []string{`"events":[]`, `"projectiles":[]`, `"participants":[`} {
		if !strings.Contains(string(raw), expected) {
			t.Fatalf("frame JSON missing %s: %s", expected, raw)
		}
	}
	if strings.Contains(string(raw), `"events":null`) {
		t.Fatalf("frame JSON serialized a list as null: %s", raw)
	}
}

func TestMatchSetupMarshalsObstaclesAsArray(t *testing.T) {
	m := NewMatch(uuid.New())
	raw, err := json.Marshal(m.MatchSetup())
	if err != nil {
		t.Fatalf("marshal setup: %v", err)
	}
	if !strings.Contains(string(raw), `"obstacles":[`) {
		t.Fatalf("setup JSON missing an obstacle array: %s", raw)
	}
}

func TestFrameCarriesCastTelegraph(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 12)
	caster.Kit = []Skill{groundAoeSkill("GU_TEST_AOE", 3, 700)}
	m.AddParticipant(caster)

	m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_AOE", Rank: 1, Mode: models.MobaCastGround, AX: 10, AY: 12,
	}}})
	m.Step()

	// The opponent must be able to draw where this will land, so the frame has
	// to carry the shape and the aim point, not just the Gu id.
	cast := m.Snapshot().Participants[0].Cast
	if cast == nil {
		t.Fatal("frame should carry the winding-up cast")
	}
	if cast.Shape != string(models.MobaShapeCircle) || cast.Radius != 3 {
		t.Fatalf("telegraph shape = %s r%v, want CIRCLE r3", cast.Shape, cast.Radius)
	}
	if cast.AimX != 10 || cast.AimY != 12 {
		t.Fatalf("telegraph aim = (%v,%v), want (10,12)", cast.AimX, cast.AimY)
	}
	if cast.EndTick <= cast.StartTick {
		t.Fatalf("telegraph window = %d..%d, want a real wind-up", cast.StartTick, cast.EndTick)
	}
}

func TestDummyDodgesTelegraphedGroundAoE(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 12)
	caster.Kit = []Skill{groundAoeSkill("GU_TEST_AOE", 3, 700)}
	dummy := NewDummy(uuid.New(), 10, 12)
	m.AddParticipant(caster)
	m.AddParticipant(dummy)

	m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_AOE", Rank: 1, Mode: models.MobaCastGround, AX: 10, AY: 12,
	}}})
	for tick := 0; tick < 16; tick++ {
		m.Step()
	}

	if dummy.X == 10 && dummy.Y == 12 {
		t.Fatal("dummy should have stepped out of the telegraphed marker")
	}
	if dummy.HP != DummyHP {
		t.Fatalf("dummy should have dodged the marker, hp = %v", dummy.HP)
	}
}

func TestDummyDoesNotDodgePointAndClick(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 12)
	skill := targetSkill("GU_TEST_NUKE")
	skill.Profile.CastTimeMs = 700
	caster.Kit = []Skill{skill}
	dummy := NewDummy(uuid.New(), 10, 12)
	m.AddParticipant(caster)
	m.AddParticipant(dummy)

	m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_NUKE", Rank: 1, Mode: models.MobaCastTarget, TargetID: dummy.ID.String(),
	}}})
	for tick := 0; tick < 16; tick++ {
		m.Step()
	}

	// Point-and-click cannot be dodged, so the dummy should stay put and eat it.
	if dummy.X != 10 || dummy.Y != 12 {
		t.Fatalf("dummy moved against an undodgeable cast: (%v,%v)", dummy.X, dummy.Y)
	}
	if dummy.HP == DummyHP {
		t.Fatal("point-and-click should have landed")
	}
}

func TestCastingRootsTheActor(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 12)
	caster.Kit = []Skill{groundAoeSkill("GU_TEST_AOE", 3, 700)}
	m.AddParticipant(caster)

	m.SubmitInput(caster.ID, Input{Seq: 1, Move: &MoveInput{DX: 1, DY: 0}, Casts: []CastInput{{
		GuID: "GU_TEST_AOE", Rank: 1, Mode: models.MobaCastGround, AX: 10, AY: 12,
	}}})
	for tick := 0; tick < 5; tick++ {
		m.Step()
	}
	if caster.X != 5 {
		t.Fatalf("a winding-up cast must root the caster, x = %v", caster.X)
	}
}

func groundAoeSkill(guID string, radius float64, castTimeMs int) Skill {
	return Skill{
		GuID: guID, Rank: 1,
		Profile: models.MobaProfile{
			CastMode: models.MobaCastGround, Shape: models.MobaShapeCircle,
			Radius: radius, CastRange: 14, CastTimeMs: castTimeMs, MaxTargets: 5,
		},
		Payload: SkillPayload{DamageFlat: 30},
	}
}

func TestMatchEndsWhenOneSideDies(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 10)
	caster.Kit = []Skill{targetSkill("GU_TEST_NUKE")}
	victim := testParticipant(10, 10)
	victim.HP, victim.MaxHP = 15, 15
	m.AddParticipant(caster)
	m.AddParticipant(victim)

	m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_NUKE", Rank: 1, Mode: models.MobaCastTarget, TargetID: victim.ID.String(),
	}}})
	m.Step()

	if !m.Ended {
		t.Fatal("match should end when only one participant is alive")
	}
	if m.Winner != caster.ID {
		t.Fatalf("winner = %v, want the caster", m.Winner)
	}
}

func TestCastPreservedWhenMoveInputFollows(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 10)
	caster.Kit = []Skill{targetSkill("GU_TEST_NUKE")}
	victim := testParticipant(10, 10)
	m.AddParticipant(caster)
	m.AddParticipant(victim)

	// Cast input arrives first
	m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_NUKE", Rank: 1, Mode: models.MobaCastTarget, TargetID: victim.ID.String(),
	}}})
	// Follow-up move input arrives in the same tick without casts
	m.SubmitInput(caster.ID, Input{Seq: 2, Move: &MoveInput{DX: 1, DY: 0}})

	m.Step()

	if victim.HP != 180 {
		t.Fatalf("victim hp = %v, want 180 (cast was dropped when move followed)", victim.HP)
	}
}

func TestGroundCastClampedToMaxRange(t *testing.T) {
	m := NewMatch(uuid.New())
	caster := testParticipant(5, 10)
	caster.Kit = []Skill{groundAoeSkill("GU_TEST_AOE", 3, 0)}
	m.AddParticipant(caster)

	// Aim far beyond CastRange (14) at (50, 10)
	m.SubmitInput(caster.ID, Input{Seq: 1, Casts: []CastInput{{
		GuID: "GU_TEST_AOE", Rank: 1, Mode: models.MobaCastGround, AX: 50, AY: 10,
	}}})
	m.Step()

	// Cast should have been clamped and accepted (AimPoint at 5 + 14 = 19, not dropped)
	if caster.Cast != nil || caster.Cooldowns["GU_TEST_AOE"] == 0 {
		// Because castTimeMs is 0, it resolved on the same tick, putting it on cooldown
		if caster.Cooldowns["GU_TEST_AOE"] == 0 {
			t.Fatal("ground cast beyond max range should have been clamped and fired, but was rejected")
		}
	}
}

