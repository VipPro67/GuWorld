package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
)

func testAdminBoss() cache.WorldMonster {
	return cache.WorldMonster{
		InstanceID: "inst-1", SpawnID: "spawn-1", RegionID: "r1",
		X: 10, Y: 10, SpawnX: 10, SpawnY: 10,
		HP: 100, MaxHP: 100, State: MonsterStateIdle, Revision: 5,
		RespawnSeconds: 600,
	}
}

func floatPtr(v float64) *float64 { return &v }
func stringPtr(v string) *string  { return &v }

func TestApplyAdminWorldBossUpdate(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

	t.Run("teleport inside region", func(t *testing.T) {
		boss := testAdminBoss()
		event, err := applyAdminWorldBossUpdate(&boss, adminWorldBossUpdateRequest{X: floatPtr(20), Y: floatPtr(30)}, 0, 0, 100, 100, now)
		if err != nil || event != "monster_moved" {
			t.Fatalf("event=%q err=%v", event, err)
		}
		if boss.X != 20 || boss.Y != 30 || boss.Revision != 6 || !boss.LastUpdateAt.Equal(now) {
			t.Fatalf("unexpected boss after teleport: %+v", boss)
		}
	})

	t.Run("teleport outside region is rejected", func(t *testing.T) {
		boss := testAdminBoss()
		if _, err := applyAdminWorldBossUpdate(&boss, adminWorldBossUpdateRequest{X: floatPtr(500)}, 0, 0, 100, 100, now); err == nil {
			t.Fatal("expected out-of-bounds error")
		}
		if boss.X != 10 || boss.Revision != 5 {
			t.Fatalf("boss must stay untouched on validation failure: %+v", boss)
		}
	})

	t.Run("hp is clamped to max", func(t *testing.T) {
		boss := testAdminBoss()
		event, err := applyAdminWorldBossUpdate(&boss, adminWorldBossUpdateRequest{HP: floatPtr(999)}, 0, 0, 100, 100, now)
		if err != nil || boss.HP != 100 || event != "" {
			t.Fatalf("event=%q hp=%v err=%v (unchanged hp must not write)", event, boss.HP, err)
		}
	})

	t.Run("hp zero marks defeated with respawn timer", func(t *testing.T) {
		boss := testAdminBoss()
		event, err := applyAdminWorldBossUpdate(&boss, adminWorldBossUpdateRequest{HP: floatPtr(0)}, 0, 0, 100, 100, now)
		if err != nil || event != "monster_defeated" {
			t.Fatalf("event=%q err=%v", event, err)
		}
		if boss.State != MonsterStateDefeated || boss.HP != 0 {
			t.Fatalf("boss not defeated: %+v", boss)
		}
		if want := now.Add(monsterRespawnDelay(boss.RespawnSeconds)); !boss.RespawnAt.Equal(want) {
			t.Fatalf("respawn_at=%v want %v", boss.RespawnAt, want)
		}
	})

	t.Run("idle revives at spawn with full hp", func(t *testing.T) {
		boss := testAdminBoss()
		boss.State, boss.HP, boss.X, boss.Y = MonsterStateDefeated, 0, 55, 55
		boss.RespawnAt = now.Add(-time.Minute)
		event, err := applyAdminWorldBossUpdate(&boss, adminWorldBossUpdateRequest{State: stringPtr("idle")}, 0, 0, 100, 100, now)
		if err != nil || event != "monster_respawned" {
			t.Fatalf("event=%q err=%v", event, err)
		}
		if boss.State != MonsterStateIdle || boss.HP != boss.MaxHP || boss.X != boss.SpawnX || boss.Y != boss.SpawnY {
			t.Fatalf("boss not revived at spawn: %+v", boss)
		}
		if !boss.RespawnAt.IsZero() {
			t.Fatalf("respawn_at must be cleared, got %v", boss.RespawnAt)
		}
	})

	t.Run("empty request and bad state are rejected", func(t *testing.T) {
		boss := testAdminBoss()
		if _, err := applyAdminWorldBossUpdate(&boss, adminWorldBossUpdateRequest{}, 0, 0, 100, 100, now); err == nil {
			t.Fatal("expected empty request error")
		}
		if _, err := applyAdminWorldBossUpdate(&boss, adminWorldBossUpdateRequest{State: stringPtr("FLYING")}, 0, 0, 100, 100, now); err == nil {
			t.Fatal("expected invalid state error")
		}
	})
}
