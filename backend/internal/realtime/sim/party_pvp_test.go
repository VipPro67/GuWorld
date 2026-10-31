package sim

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func TestPlayerCanHarmAnotherPlayerOnlyInPvPDifferentParty(t *testing.T) {
	for _, test := range []struct {
		name       string
		pvp        bool
		partyA     string
		partyB     string
		sourcePvP  bool
		targetPvP  bool
		wantReject bool
	}{
		{name: "pvp different party both opted in", pvp: true, partyA: "a", partyB: "b", sourcePvP: true, targetPvP: true, wantReject: false},
		{name: "pvp different party attacker opted in", pvp: true, partyA: "a", partyB: "b", sourcePvP: true, wantReject: false},
		{name: "pvp different party target opted in", pvp: true, partyA: "a", partyB: "b", targetPvP: true, wantReject: false},
		{name: "pvp different party neither opted in", pvp: true, partyA: "a", partyB: "b", wantReject: true},
		{name: "pvp same party", pvp: true, partyA: "a", partyB: "a", sourcePvP: true, targetPvP: true, wantReject: true},
		{name: "pve", pvp: false, partyA: "a", partyB: "b", sourcePvP: true, targetPvP: true, wantReject: true},
		{name: "pvp no parties", pvp: true, sourcePvP: true, wantReject: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, _ := NewWorld("arena", 8)
			w.SetPvPEnabled(test.pvp)
			first := newCombatPlayer("player-a", Vec2{X: 0, Y: 0})
			second := newCombatPlayer("player-b", Vec2{X: 2, Y: 0})
			first.PartyID, second.PartyID = test.partyA, test.partyB
			first.PvPEnabled, second.PvPEnabled = test.sourcePvP, test.targetPvP
			if err := w.AddEntity(first); err != nil {
				t.Fatal(err)
			}
			if err := w.AddEntity(second); err != nil {
				t.Fatal(err)
			}
			if err := w.SubmitCast(CastCommand{EntityID: "player-a", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "player-b"}); err != nil {
				t.Fatal(err)
			}
			events := collectEvents(t, w, 6)
			rejected := hasEvent(events, combat.EventCastRejected)
			if rejected != test.wantReject {
				t.Fatalf("rejected=%v want=%v events=%+v", rejected, test.wantReject, events)
			}
			if !test.wantReject {
				if hp := entityHP(t, w, "player-b"); hp >= 100 {
					t.Fatalf("pvp cast did not damage target: %v", hp)
				}
			} else if hp := entityHP(t, w, "player-b"); hp != 100 {
				t.Fatalf("blocked pvp cast damaged target: %v", hp)
			}
		})
	}
}

func TestBossThreatSelectsTopDamageAndSplashHitsParty(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	boss := newAIMonster("boss", Vec2{X: 0, Y: 0})
	boss.Vitals = Vitals{HP: 10_000, MaxHP: 10_000}
	boss.AI.SplashRadius = 2
	boss.AI.SplashDamage = 5
	boss.AI.ThinkTicks = 1
	boss.AI.AttackCooldownTicks = 1
	boss.AI.AttackRange = 2
	boss.AI.AggroRadius = 10
	boss.AI.LeashRadius = 20
	first := newCombatPlayer("player-a", Vec2{X: 1.2, Y: 0})
	second := newCombatPlayer("player-b", Vec2{X: 1.4, Y: 0.2})
	first.PartyID, second.PartyID = "party", "party"
	for _, entity := range []Entity{boss, first, second} {
		if err := w.AddEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	// First player deals more threat than the second.
	w.applyDamageLocked(w.entities["player-a"], w.entities["boss"], 100, "A")
	w.applyDamageLocked(w.entities["player-b"], w.entities["boss"], 20, "B")
	w.Step()
	w.Step()
	if w.entities["boss"].AI.Threat != "player-a" {
		t.Fatalf("boss threat=%q, want player-a", w.entities["boss"].AI.Threat)
	}
	if entityHP(t, w, "player-a") >= 100 {
		t.Fatal("boss did not attack the top-threat player")
	}
	if entityHP(t, w, "player-b") >= 100 {
		t.Fatal("boss splash did not hit the nearby party member")
	}
}

func TestBossThreatClearsOnLeash(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	boss := newAIMonster("boss", Vec2{X: 0, Y: 0})
	boss.AI.LeashRadius = 5
	boss.AI.AggroRadius = 4
	boss.AI.ThinkTicks = 1
	boss.Transform.Position = Vec2{X: 6, Y: 0}
	player := newCombatPlayer("player-a", Vec2{X: 6.5, Y: 0})
	if err := w.AddEntity(boss); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	w.applyDamageLocked(w.entities["player-a"], w.entities["boss"], 10, "A")
	w.Step()
	if len(w.entities["boss"].AI.ThreatTable) != 0 || w.entities["boss"].AI.Threat != "" {
		t.Fatalf("leash did not clear threat: %+v", w.entities["boss"].AI)
	}
}
