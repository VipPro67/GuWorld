package sim

import "testing"

func TestSuspendedPlayerTakesNoDamageAndIsNotTargeted(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "suspend", RulesetVersion: "v1"}, 8, 7)
	if err != nil {
		t.Fatal(err)
	}
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 0, Y: 0}
	monster := newAIMonster("mob", Vec2{X: 2, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}

	w.entities["player"].Suspended = true
	w.applyDamageLocked(w.entities["mob"], w.entities["player"], 100, "BASIC_ATTACK")
	if hp := w.entities["player"].Vitals.HP; hp != player.Vitals.HP {
		t.Fatalf("suspended player took damage: hp=%v want %v", hp, player.Vitals.HP)
	}

	for i := 0; i < 60; i++ {
		w.Step()
	}
	if hp := w.entities["player"].Vitals.HP; hp != player.Vitals.HP {
		t.Fatalf("monster damaged a suspended player: hp=%v want %v", hp, player.Vitals.HP)
	}

	w.entities["player"].Suspended = false
	for i := 0; i < 60; i++ {
		w.Step()
	}
	if hp := w.entities["player"].Vitals.HP; hp >= player.Vitals.HP {
		t.Fatalf("resumed player was not attacked again: hp=%v", hp)
	}
}
