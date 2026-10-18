package cache

import (
	"math"
	"testing"
	"time"
)

func TestApplyWorldMonsterHitSupportsBlockV2ScaleDamage(t *testing.T) {
	now := time.Date(2026, 10, 4, 6, 0, 0, 0, time.UTC)
	monster := WorldMonster{HP: 100, MaxHP: 100, State: "CHASE", TargetID: "player", Revision: 7}

	outcome := applyWorldMonsterHit(&monster, 200_000_000_000, now, 5*time.Minute)

	if outcome != WorldMonsterFirstKill {
		t.Fatalf("outcome = %q, want %q", outcome, WorldMonsterFirstKill)
	}
	if monster.HP != 0 || monster.State != "DEFEATED" || monster.TargetID != "" {
		t.Fatalf("monster was not defeated cleanly: %+v", monster)
	}
	if monster.Revision != 8 || !monster.LastHitAt.Equal(now) || !monster.RespawnAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("monster transition metadata is wrong: %+v", monster)
	}
}

func TestHitWorldMonsterRejectsNonFiniteDamageBeforeRedis(t *testing.T) {
	previous := Client
	Client = nil
	t.Cleanup(func() { Client = previous })

	for _, damage := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, _, err := HitWorldMonster(t.Context(), "region", "monster", "command", 1, damage, time.Now(), time.Minute, time.Minute)
		if err == nil || err.Error() == "redis unavailable" {
			t.Fatalf("damage %v should be rejected", damage)
		}
	}
}

func TestApplyWorldMonsterHitKeepsLivingMonsterActive(t *testing.T) {
	now := time.Date(2026, 10, 4, 6, 0, 0, 0, time.UTC)
	monster := WorldMonster{HP: 100, MaxHP: 100, State: "CHASE", TargetID: "player", Revision: 3}

	outcome := applyWorldMonsterHit(&monster, 25, now, time.Minute)

	if outcome != WorldMonsterHit || monster.HP != 75 || monster.State != "CHASE" || monster.Revision != 4 {
		t.Fatalf("unexpected hit transition: outcome=%q monster=%+v", outcome, monster)
	}
}

func TestFreshestWorldBossesKeepsOneActorPerSpawn(t *testing.T) {
	older := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	members := []worldBossSetMember{
		{id: "old", monster: WorldMonster{InstanceID: "old", SpawnID: "mapcell:r1:5:5", LastUpdateAt: older, Revision: 40}},
		{id: "new", monster: WorldMonster{InstanceID: "new", SpawnID: "mapcell:r1:5:5", LastUpdateAt: newer, Revision: 2}},
		{id: "other", monster: WorldMonster{InstanceID: "other", SpawnID: "mapcell:r1:9:9", LastUpdateAt: older, Revision: 1}},
		{id: "empty", monster: WorldMonster{InstanceID: "empty", SpawnID: "", LastUpdateAt: older}},
	}

	kept, duplicates := freshestWorldBosses(members)

	if len(kept) != 3 || len(duplicates) != 1 {
		t.Fatalf("kept=%d duplicates=%d, want 3/1", len(kept), len(duplicates))
	}
	if duplicates[0].id != "old" {
		t.Fatalf("duplicate = %q, want the stale instance", duplicates[0].id)
	}
	for _, member := range kept {
		if member.id == "old" {
			t.Fatal("stale duplicate must not be kept")
		}
	}
}
