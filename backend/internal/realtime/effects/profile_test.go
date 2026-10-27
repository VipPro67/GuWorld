package effects

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func TestCompileCommittedProfiles(t *testing.T) {
	profiles, err := LoadProfiles("../../../data/mmo/gu/transpiled_profiles.json")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := loadGuCSV("../../../data/Gu.csv")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := CompileCatalog(profiles, entries, DefaultMVP())
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled.Profiles) != 4427 || len(compiled.DefaultSkills) != 5 {
		t.Fatalf("profiles=%d actives=%d, want 4427/5", len(compiled.Profiles), len(compiled.DefaultSkills))
	}
	if compiled.Hash == "" {
		t.Fatal("compiled catalog hash is empty")
	}
	if compiled.Hash != "68826df06d723a2d756b009a26edb11e0b9ed9da7f0a97a1f9a622c867813b94" {
		t.Fatalf("compiled catalog drifted; update this baseline: %s", compiled.Hash)
	}
	arrow, ok := compiled.Profile("GU_WATER_ARROW", 1)
	if !ok || arrow.Active == nil {
		t.Fatal("water arrow profile missing")
	}
	if _, ok := compiled.Profile("GU_WATER_ARROW", 9); ok {
		t.Fatal("missing rank must not inherit another rank's realtime profile")
	}
	if arrow.Active.Skill.Mode != combat.ModeDirection || arrow.Active.Skill.ProjectileSpeed != 18 || arrow.Active.Skill.WindupTicks != 4 {
		t.Fatalf("unexpected water arrow compile: %+v", arrow.Active.Skill)
	}
	hammer, _ := compiled.Profile("GU_SPIRAL_HAMMER_2", 2)
	if hammer.Active == nil || hammer.Active.Skill.Mode != combat.ModeGround || hammer.Active.Skill.Radius != 2.5 || hammer.Active.Skill.MaxTargets != 5 {
		t.Fatalf("unexpected hammer compile: %+v", hammer.Active)
	}
	frost, _ := compiled.Profile("GU_WHITE_FROST", 3)
	if frost.Active == nil || frost.Active.Skill.StatusID != "FREEZE" || frost.Active.Skill.StatusDurationTicks != 30 || frost.Active.Skill.StatusMoveMultiplier != 0 || !frost.Active.Skill.StatusDebuff {
		t.Fatalf("unexpected frost compile: %+v", frost.Active)
	}
	strength, _ := compiled.Profile("GU_STRENGTH", 1)
	if strength.Active != nil || len(strength.Passives) != 1 || len(strength.Passives[0].Effects) != 5 {
		t.Fatalf("unexpected strength compile: %+v", strength)
	}
	for _, skill := range compiled.DefaultSkills {
		if !skill.Valid() || skill.Slot < 1 || skill.Slot > 6 {
			t.Fatalf("compiled skill invalid: %+v", skill)
		}
	}

	alpha, ok := compiled.Profile("GU_ANCIENT_BODY_1", 1)
	if !ok || alpha.Active == nil {
		t.Fatal("alpha catalog profile missing")
	}
	if alpha.Slot != 0 || alpha.Active.Skill.Slot != 1 || alpha.Active.Skill.Mode != combat.ModeSelf {
		t.Fatalf("catalog-only profile should compile with slot 0 and a reslotted skill: %+v", alpha)
	}
}

func TestAdminEquippedGuCompileAndReslot(t *testing.T) {
	profiles, err := LoadProfiles("../../../data/mmo/gu/transpiled_profiles.json")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := loadGuCSV("../../../data/Gu.csv")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := CompileCatalog(profiles, entries, DefaultMVP())
	if err != nil {
		t.Fatal(err)
	}
	loadout := BuildLoadout(compiled, []EquippedGu{
		{InstanceID: "bell", GU: "GU_HEALING_BELL_2", Rank: 2},
		{InstanceID: "dragon", GU: "GU_IMMORTAL_DRAGON_SLAYING_SWORD", Rank: 7},
		{InstanceID: "strength", GU: "GU_STRENGTH_INFUSION", Rank: 4},
	})
	if len(loadout.Skills) != 3 || len(loadout.Skipped) != 0 {
		t.Fatalf("admin Gu loadout skills=%+v skipped=%+v", loadout.Skills, loadout.Skipped)
	}
	if loadout.Skills[0].ID != "GU_HEALING_BELL_2" || loadout.Skills[0].Slot != 1 || loadout.Skills[0].Damage != 60 ||
		loadout.Skills[1].ID != "GU_IMMORTAL_DRAGON_SLAYING_SWORD" || loadout.Skills[1].Slot != 2 || loadout.Skills[1].StatusID != "BLEED" ||
		loadout.Skills[2].ID != "GU_STRENGTH_INFUSION" || loadout.Skills[2].Slot != 3 || loadout.Skills[2].Damage != 45 || loadout.Skills[2].Shield != 45 {
		t.Fatalf("admin Gu were not compiled into ordered hotbar slots: %+v", loadout.Skills)
	}
}

func TestLoadProfileCatalogsRejectsEmptyAndBadPaths(t *testing.T) {
	if _, err := LoadProfileCatalogs(" , "); err == nil {
		t.Fatal("empty catalog list was accepted")
	}
	if _, err := LoadProfileCatalogs("../../../data/mmo/gu/transpiled_profiles.json,../../../data/mmo/gu/missing.json"); err == nil {
		t.Fatal("missing catalog was accepted")
	}
}

func TestCompilerRejectsBrokenProfiles(t *testing.T) {
	base := func() ProfileCatalog {
		return ProfileCatalog{SchemaVersion: 1, Profiles: []GuProfile{{
			GUID: "GU_TEST", Rank: 1, Slot: 2,
			Active: &ActiveProfile{
				Mode: combat.ModeDirection, WindupMS: 100, CooldownMS: 1000, RangeUnits: 10, ManaCost: 1,
				Projectile: &ProjectileProfile{SpeedUnitsPerSec: 10, RadiusUnits: 0.2, LifetimeTicks: 20},
				Effects:    []EffectProfile{{Type: "DAMAGE", Value: 10}},
			},
		}}}
	}
	entries := []GuEntry{{ID: "GU_TEST", Rank: 1, Name: "Test"}}
	if _, err := CompileCatalog(base(), entries, nil); err != nil {
		t.Fatalf("valid profile failed: %v", err)
	}

	cases := map[string]func(*ProfileCatalog){
		"schema":        func(c *ProfileCatalog) { c.SchemaVersion = 2 },
		"missing gu":    func(c *ProfileCatalog) { c.Profiles[0].GUID = "GU_MISSING" },
		"slot range":    func(c *ProfileCatalog) { c.Profiles[0].Slot = 7 },
		"windup":        func(c *ProfileCatalog) { c.Profiles[0].Active.WindupMS = MaxWindupMS + 1 },
		"no projectile": func(c *ProfileCatalog) { c.Profiles[0].Active.Projectile = nil },
		"bad effect":    func(c *ProfileCatalog) { c.Profiles[0].Active.Effects = []EffectProfile{{Type: "UNIQUE"}} },
		"active stat_mod": func(c *ProfileCatalog) {
			c.Profiles[0].Active.Effects = []EffectProfile{{Type: "STAT_MOD", Stat: "attack", Value: 5}}
		},
		"movement policy": func(c *ProfileCatalog) { c.Profiles[0].Active.MovementPolicy = "SPRINT" },
		"status duration": func(c *ProfileCatalog) {
			c.Profiles[0].Active.Effects = []EffectProfile{{Type: "STATUS", StatusID: "FREEZE", PvPDrCategory: "HARD_CC"}}
		},
		"mvp missing": func(c *ProfileCatalog) {
			_ = c
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			catalog := base()
			mutate(&catalog)
			required := []EntryRef(nil)
			if name == "mvp missing" {
				required = DefaultMVP()
			}
			if _, err := CompileCatalog(catalog, entries, required); err == nil {
				t.Fatal("invalid profile was accepted")
			}
		})
	}
}

func TestCompilerRejectsDuplicateSlots(t *testing.T) {
	profile := func(gu string) GuProfile {
		return GuProfile{
			GUID: gu, Rank: 1, Slot: 2,
			Active: &ActiveProfile{
				Mode: combat.ModeTarget, WindupMS: 100, CooldownMS: 1000, RangeUnits: 5, ManaCost: 1,
				Effects: []EffectProfile{{Type: "DAMAGE", Value: 5}},
			},
		}
	}
	catalog := ProfileCatalog{SchemaVersion: 1, Profiles: []GuProfile{profile("GU_A"), profile("GU_B")}}
	entries := []GuEntry{{ID: "GU_A", Rank: 1}, {ID: "GU_B", Rank: 1}}
	if _, err := CompileCatalog(catalog, entries, nil); err == nil {
		t.Fatal("duplicate slot was accepted")
	}
}
