package simulator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestSimulateRepresentativeLogsGolden(t *testing.T) {
	const firstSeed, seedCount = int64(1), 8
	hash := sha256.New()

	for _, scenario := range []string{"rider", "status", "summon", "death", "displacement", "chance"} {
		for seed := firstSeed; seed < firstSeed+seedCount; seed++ {
			state := representativeCombatState(scenario, seed)
			logs := Simulate(state)
			encoded, err := json.Marshal(logs)
			if err != nil {
				t.Fatalf("marshal %s seed %d logs: %v", scenario, seed, err)
			}
			_, _ = hash.Write([]byte(scenario))
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write(encoded)
			_, _ = hash.Write([]byte{0})
		}
	}

	const want = "f30756554ec0a1cf51a6a5923783e4bc3378af232751371933ba37855af3d907"
	if got := hex.EncodeToString(hash.Sum(nil)); got != want {
		t.Fatalf("representative simulation log hash = %s, want %s", got, want)
	}
}

func TestSimulateRunsTypedTurnStartCounterBeforeLegacyRider(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1200, 1000)
	enemy := newTestCombatant("enemy", "Enemy", 100, 1)
	ally.Speed, enemy.Speed = 30, 10
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: 1000, DamageMult: 1}}
	ally.TurnStartCounterAdds = []CounterAdd{{Key: "thi_y", Delta: 1, Min: 0, Max: 10}}

	logs := Simulate(&CombatState{Seed: 3, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
	if ally.CustomState["thi_y"] != 1 {
		t.Fatalf("typed start-of-turn counter = %d, want 1", ally.CustomState["thi_y"])
	}
	foundPoemRider := false
	for _, log := range logs {
		if strings.Contains(log.Detail, "Tích tụ 1 tầng Thi Ý") {
			foundPoemRider = true
			break
		}
	}
	if !foundPoemRider {
		t.Fatalf("typed counter did not activate the poem rider: %+v", logs)
	}
}

func representativeCombatState(scenario string, seed int64) *CombatState {
	ally := newTestCombatant("ally", "Player", 1200, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1200, 60)
	ally.Speed, enemy.Speed = 30, 20
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Cổ thử", SourceID: "GU_GOLDEN", Rank: 3, Damage: 45}}
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 30}}
	state := &CombatState{Seed: seed, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}

	switch scenario {
	case "rider":
		ally.Actions[0].Mechanics = []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "SOUL_DEVOUR"}}
		ally.HP = 700
	case "status":
		ally.Actions[0].StatusOps = []statusfx.StatusOp{{ID: statusfx.StatusBleed, Value: 18, Turns: 3, Chance: 0.65, SourcePath: "blood"}}
	case "summon":
		mechanic := mech("ON_BATTLE_START", "SUMMON_MONSTER", `{"hp":160,"damage":20,"defense":8,"position":"frontline"}`)
		mechanic.Rank = 3
		ally.Mechanics = []CombatMechanic{mechanic}
	case "death":
		ally.HP = 60
		ally.Speed, enemy.Speed = 1, 100
		ally.Mechanics = []CombatMechanic{mech("ON_DEATH", "REBORN", `{"chance":1,"mode":"in_place","max_uses":1,"revive_hp_percent":0.5}`)}
		enemy.Actions[0].Damage = 90
	case "displacement":
		ally.GridX, ally.GridY = 1, 2
		enemy.GridX, enemy.GridY = 2, 2
		ally.Actions[0].Mechanics = []CombatMechanic{NewCombatMechanic("ON_ACTION", "KNOCKBACK", []byte(`{"steps":2}`))}
	case "chance":
		ally.Actions[0].Mechanics = []CombatMechanic{mech("ON_CAST", "CHAOS_GAMBLE", `{}`)}
	default:
		panic("unknown representative scenario: " + scenario)
	}
	return state
}
