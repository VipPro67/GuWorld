package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestApplySinhKhiTanHitStacksToCap(t *testing.T) {
	var list models.StatusList
	wantTiers := []int{1, 2, 3, 4, 5, 5, 5}
	for i, want := range wantTiers {
		tier, value := ApplySinhKhiTanHitToList(&list)
		if tier != want {
			t.Fatalf("hit %d: tier = %d, want %d", i+1, tier, want)
		}
		if value != gamefx.SinhKhiTanValueForTier(want) {
			t.Fatalf("hit %d: value = %v, want %v", i+1, value, gamefx.SinhKhiTanValueForTier(want))
		}
		if len(list) != 1 || list[0].ID != gamefx.StatusSinhKhiTan {
			t.Fatalf("hit %d: list should hold exactly one sinh_khi_tan entry: %+v", i+1, list)
		}
		if list[0].ExpiresAt == nil || time.Until(*list[0].ExpiresAt) > 5*time.Hour+time.Minute {
			t.Fatalf("hit %d: expiry should refresh ~5h: %+v", i+1, list[0].ExpiresAt)
		}
	}
}

func TestCleanseSinhKhiTanDowngradesGradually(t *testing.T) {
	var list models.StatusList
	for i := 0; i < 5; i++ {
		ApplySinhKhiTanHitToList(&list)
	}
	for want := 4; want >= 0; want-- {
		tier, found := CleanseSinhKhiTanOneTierFromList(&list)
		if !found {
			t.Fatalf("cleanse to %d: expected found", want)
		}
		if tier != want {
			t.Fatalf("cleanse: tier = %d, want %d", tier, want)
		}
	}
	if len(list) != 0 {
		t.Fatalf("after full cleanse list should be empty: %+v", list)
	}
	if _, found := CleanseSinhKhiTanOneTierFromList(&list); found {
		t.Fatal("cleanse on empty list should report not found")
	}
}

func TestSinhKhiTanIgnoresExpiredEntry(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	list := models.StatusList{{ID: gamefx.StatusSinhKhiTan, Value: -0.99, ExpiresAt: &past}}
	// Hit mới phải tính từ tầng 0 (entry cũ hết hạn), không kế thừa tầng V.
	if tier, _ := ApplySinhKhiTanHitToList(&list); tier != 1 {
		t.Fatalf("hit after expiry: tier = %d, want 1", tier)
	}
	list = models.StatusList{{ID: gamefx.StatusSinhKhiTan, Value: -0.50, ExpiresAt: &past}}
	// Cleanse entry hết hạn thì dọn luôn, báo đã sạch.
	if tier, found := CleanseSinhKhiTanOneTierFromList(&list); !found || tier != 0 || len(list) != 0 {
		t.Fatalf("cleanse expired: tier = %d, found = %v, list = %+v", tier, found, list)
	}
}

func TestMaybeGrantPersistentHitRejectsNonPVP(t *testing.T) { // Nil player must not touch the DB.
	if grants := maybeGrantPersistentHitStatuses(nil, nil, nil, nil); len(grants) != 0 {
		t.Fatal("nil player must not grant")
	}
	// Non-PVP combat must not grant either.
	p := &models.Player{}
	p.ActiveCombat = &models.CombatState{EnemyType: "Monster"}
	st := &simulator.CombatState{
		Allies: []*simulator.CombatantAuto{{ID: "attacker", Actions: []simulator.SimAction{
			{ActionType: "GU_SKILL", Name: "Sát Sinh Khí Cổ", SourceID: "GU_KILL_ENERGY", PersistentStatusKey: "sinh_khi_tan", PersistentPvPOnly: true},
		}}},
	}
	logs := []simulator.CombatLog{{ActorId: "attacker", Damage: 100, SourceGuID: "GU_KILL_ENERGY", HitLanded: true}}
	if grants := maybeGrantPersistentHitStatuses(nil, p, st, logs); len(grants) != 0 {
		t.Fatal("non-PVP combat must not grant")
	}
}

func TestPersistentHitStatusKeysFromSpec(t *testing.T) {
	newState := func() *simulator.CombatState {
		return &simulator.CombatState{
			Allies: []*simulator.CombatantAuto{
				{ID: "attacker", Actions: []simulator.SimAction{
					{ActionType: "GU_SKILL", Name: "Sát Sinh Khí Cổ", SourceID: "GU_KILL_ENERGY", PersistentStatusKey: "sinh_khi_tan", PersistentPvPOnly: true},
					{ActionType: "GU_SKILL", Name: "Cổ Khác", SourceID: "GU_OTHER"},
				}},
			},
		}
	}

	// Spec trên Cổ resolve ra status key, Cổ khác không có.
	keys := persistentHitStatusKeys(newState())
	if keys["GU_KILL_ENERGY"] != "sinh_khi_tan" {
		t.Fatalf("expected sinh_khi_tan for GU_KILL_ENERGY, got %q", keys["GU_KILL_ENERGY"])
	}
	if _, dup := keys["GU_OTHER"]; dup {
		t.Fatal("Gu without hook must not appear in persistent keys")
	}

	// Hit thật (kể cả khiên hấp thụ, Damage>0) match theo SourceGuID;
	// Cổ khác đánh trúng không match.
	hitGu := []simulator.CombatLog{{ActorId: "attacker", Damage: 100, SourceGuID: "GU_KILL_ENERGY", HitLanded: true,
		Detail: "admin dùng [Sát Sinh Khí Cổ] lên Clone Gmail, gây 100 sát thương (còn 0 HP)"}}
	hitOther := []simulator.CombatLog{{ActorId: "attacker", Damage: 100, SourceGuID: "GU_OTHER", HitLanded: true,
		Detail: "admin dùng [Cổ Khác] lên Clone Gmail, gây 100 sát thương (còn 0 HP)"}}
	countHits := func(logs []simulator.CombatLog) int {
		n := 0
		ally := map[string]bool{"attacker": true}
		for _, l := range logs {
			if l.HitLanded && ally[l.ActorId] {
				if _, ok := keys[l.SourceGuID]; ok {
					n++
				}
			}
		}
		return n
	}
	if countHits(hitGu) != 1 {
		t.Fatal("Gu hit must match by SourceGuID")
	}
	if countHits(hitOther) != 0 {
		t.Fatal("damage từ Cổ khác không được tính hit")
	}
	// Miss (né/mù) có SourceGuID nhưng HitLanded=false → không grant.
	miss := []simulator.CombatLog{{ActorId: "attacker", SourceGuID: "GU_KILL_ENERGY", HitLanded: false}}
	if countHits(miss) != 0 {
		t.Fatal("missed strike must not count as hit")
	}
}
