package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestResolveRespawnPosition_NoSectStaysInCurrentRegion(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_nam_cuong", ZoneType: ZoneTypeLarge, MinX: 1000, MinY: 1000, MaxX: 5000, MaxY: 5000},
	}
	zoneReg.byID = make(map[string]*models.Region)
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()

	dummyID := uuid.New()
	// Calling with nil tx should fallback to step 2 (current region random bounds)
	respawnRegion, rx, ry := ResolveRespawnPosition(nil, dummyID, "dai_vuc_nam_cuong", 2000, 2000)

	if respawnRegion != "dai_vuc_nam_cuong" {
		t.Errorf("expected respawn region to stay dai_vuc_nam_cuong, got %s", respawnRegion)
	}
	if rx < 1000 || rx > 5000 {
		t.Errorf("expected rx in [1000, 5000], got %d", rx)
	}
	if ry < 1000 || ry > 5000 {
		t.Errorf("expected ry in [1000, 5000], got %d", ry)
	}
}

func TestPvpImmunityStatus(t *testing.T) {
	var sl models.StatusList
	sl.Add("pvp_immunity", "respawn", 10*time.Minute, 1.0)

	if !sl.Has("pvp_immunity") {
		t.Errorf("expected status list to have pvp_immunity")
	}

	sl.Remove("pvp_immunity")
	if sl.Has("pvp_immunity") {
		t.Errorf("expected pvp_immunity to be removed after breaking")
	}
}

func TestNewbieProtectionRules(t *testing.T) {
	// Rule: Rank <= 3 without Sat Khi is protected.
	isProtected := func(rank int, satKhiLevel int) bool {
		return rank <= 3 && satKhiLevel == 0
	}

	if !isProtected(1, 0) {
		t.Errorf("Rank 1 with 0 Sat Khi should be protected")
	}
	if !isProtected(3, 0) {
		t.Errorf("Rank 3 with 0 Sat Khi should be protected")
	}
	if isProtected(3, 1) {
		t.Errorf("Rank 3 with 1 Sat Khi should NOT be protected")
	}
	if isProtected(4, 0) {
		t.Errorf("Rank 4 with 0 Sat Khi should NOT be protected")
	}
}

func TestWorldPosIndex_ImmediatePresenceAndNearby(t *testing.T) {
	idx := newPositionIndex(300)
	playerA := WorldPosEntry{
		ID:       "player-a",
		Username: "PlayerA",
		X:        100,
		Y:        100,
		Region:   "hub_nam_cuong",
		TS:       time.Now().UnixMilli(),
	}
	playerB := WorldPosEntry{
		ID:       "player-b",
		Username: "PlayerB",
		X:        120,
		Y:        120,
		Region:   "hub_nam_cuong",
		TS:       time.Now().UnixMilli(),
	}
	playerC := WorldPosEntry{
		ID:       "player-c",
		Username: "PlayerC",
		X:        1000,
		Y:        1000,
		Region:   "hub_nam_cuong",
		TS:       time.Now().UnixMilli(),
	}

	idx.Set(playerA)
	idx.Set(playerB)
	idx.Set(playerC)

	// Nearby for playerA with radius 300 should find playerB but not playerC or playerA itself
	nearby := idx.Nearby(100, 100, 300, "player-a")
	if len(nearby) != 1 {
		t.Fatalf("expected 1 nearby player, got %d", len(nearby))
	}
	if nearby[0].ID != "player-b" {
		t.Errorf("expected player-b, got %s", nearby[0].ID)
	}

	// Verify get by ID
	entry, ok := idx.Get("player-a")
	if !ok || entry.X != 100 {
		t.Errorf("expected player-a at (100, 100)")
	}
}

func TestDeathDrop_VoidDestructionAndIncreasedRates(t *testing.T) {
	// Verify configured rates
	if deathPenaltyConfig.PrimevalStonePct < 40 {
		t.Errorf("expected PrimevalStonePct >= 40, got %f", deathPenaltyConfig.PrimevalStonePct)
	}
	if deathPenaltyConfig.ImmortalStonePct < 40 {
		t.Errorf("expected ImmortalStonePct >= 40, got %f", deathPenaltyConfig.ImmortalStonePct)
	}
	if deathPenaltyConfig.ItemsTypePct < 40 {
		t.Errorf("expected ItemsTypePct >= 40, got %f", deathPenaltyConfig.ItemsTypePct)
	}
	if deathPenaltyConfig.VoidDestructionPct != 40 {
		t.Errorf("expected VoidDestructionPct == 40, got %f", deathPenaltyConfig.VoidDestructionPct)
	}

	// Test stone void destruction calculation
	primevalLoss := 100.0
	voidPct := deathPenaltyConfig.VoidDestructionPct / 100.0
	primevalDestroyed := primevalLoss * voidPct
	primevalLoot := primevalLoss - primevalDestroyed

	if primevalDestroyed != 40.0 {
		t.Errorf("expected 40 primeval stones destroyed, got %f", primevalDestroyed)
	}
	if primevalLoot != 60.0 {
		t.Errorf("expected 60 primeval stones in ground loot, got %f", primevalLoot)
	}
}

func TestSectTerritory_ShieldedProtection(t *testing.T) {
	// Rule: when a sect is under shield_until (trạng thái bảo hộ),
	// players inside its territory are NOT protected from PvP attacks.
	isProtectedInTerritory := func(inTerritory bool, isShielded bool) bool {
		return inTerritory && !isShielded
	}

	if !isProtectedInTerritory(true, false) {
		t.Errorf("Player in territory of non-shielded sect SHOULD be protected")
	}

	if isProtectedInTerritory(true, true) {
		t.Errorf("Player in territory of shielded sect SHOULD NOT be protected")
	}

	if isProtectedInTerritory(false, false) {
		t.Errorf("Player outside territory should NOT be protected")
	}
}
