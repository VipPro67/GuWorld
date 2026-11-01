package sim

import (
	"math"
	"testing"
	"time"
)

func TestPlayerShieldDecaysEachSecondAndFloorsToZero(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Vitals = Vitals{HP: 100, MaxHP: 100, Shield: 100}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	for tick := 1; tick < ShieldDecayIntervalTicks; tick++ {
		w.Step()
	}
	if got := w.entities["player"].Vitals.Shield; got != 100 {
		t.Fatalf("shield decayed before the interval elapsed: %v", got)
	}
	w.Step()
	want := 100 * (1 - ShieldDecayPerSecond)
	if got := w.entities["player"].Vitals.Shield; math.Abs(got-want) > 1e-6 {
		t.Fatalf("shield after one interval = %v, want %v", got, want)
	}
	for tick := 0; tick < ShieldDecayIntervalTicks*60; tick++ {
		w.Step()
	}
	if got := w.entities["player"].Vitals.Shield; got != 0 {
		t.Fatalf("shield did not decay to zero: %v", got)
	}
}

func TestShieldDecayDropsRemainderBelowFloor(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Vitals = Vitals{HP: 100, MaxHP: 100, Shield: ShieldDecayMin + 0.05}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < ShieldDecayIntervalTicks; tick++ {
		w.Step()
	}
	if got := w.entities["player"].Vitals.Shield; got != 0 {
		t.Fatalf("shield remainder %v was not dropped", got)
	}
}

func TestMonsterShieldDoesNotDecay(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	monster := newTargetMonster("mob", Vec2{X: 3})
	monster.Vitals.Shield = 100
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < ShieldDecayIntervalTicks*5; tick++ {
		w.Step()
	}
	if got := w.entities["mob"].Vitals.Shield; got != 100 {
		t.Fatalf("monster shield decayed to %v", got)
	}
}

func TestDecayedShieldAgesCheckpointByElapsedTime(t *testing.T) {
	cases := []struct {
		name    string
		shield  float64
		elapsed time.Duration
		want    float64
	}{
		{name: "no shield", shield: 0, elapsed: time.Minute, want: 0},
		{name: "no elapsed", shield: 50, elapsed: 0, want: 50},
		{name: "negative elapsed", shield: 50, elapsed: -time.Second, want: 50},
		{name: "one second", shield: 50, elapsed: time.Second, want: 45},
		{name: "below floor", shield: 1.05, elapsed: time.Second, want: 0},
		{name: "long offline", shield: 500, elapsed: 24 * time.Hour, want: 0},
		{name: "nan shield", shield: math.NaN(), elapsed: time.Second, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DecayedShield(tc.shield, tc.elapsed); math.Abs(got-tc.want) > 1e-6 {
				t.Fatalf("DecayedShield(%v, %v) = %v, want %v", tc.shield, tc.elapsed, got, tc.want)
			}
		})
	}
}
