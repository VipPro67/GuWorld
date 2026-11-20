package handlers

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestFindNeighborZoneOnAtlas(t *testing.T) {
	previousZones, previousByID := zoneReg.zones, zoneReg.byID
	t.Cleanup(func() {
		zoneReg.zones, zoneReg.byID = previousZones, previousByID
	})

	zoneReg.zones = []models.Region{
		{ID: "thai_gia_cang", Name: "Thái Gia Cảng", ZoneType: ZoneTypeNamed, Width: 1000, Height: 1000, AtlasX: 8000, AtlasY: 6000, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000},
		{ID: "thien_luu_hai_vuc", Name: "Thiên Lưu Hải Vực", ZoneType: ZoneTypeNamed, Width: 1000, Height: 1000, AtlasX: 9000, AtlasY: 6000, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000},
		{ID: "dh_gioi_bich_dong_bac", Name: "Đông Bắc Giới Bích", ZoneType: ZoneTypeNamed, Width: 1000, Height: 1000, AtlasX: 9000, AtlasY: 5000, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000},
		{ID: "cu_lang_vuc", Name: "Cự Lãng Vực", ZoneType: ZoneTypeNamed, Width: 1000, Height: 1000, AtlasX: 9000, AtlasY: 7000, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000},
		{ID: "bt_thai_co_van_hai", Name: "Bạch Thiên Vân Hải", ZoneType: ZoneTypeNamed, Width: 5000, Height: 5000, AtlasX: 0, AtlasY: 0, MinX: 0, MinY: 0, MaxX: 5000, MaxY: 5000},
		{ID: "bt_quang_minh_tinh_doi", Name: "Quang Minh Tinh Đới", ZoneType: ZoneTypeNamed, Width: 5000, Height: 5000, AtlasX: 5000, AtlasY: 0, MinX: 0, MinY: 0, MaxX: 5000, MaxY: 5000},
	}
	zoneReg.byID = make(map[string]*models.Region, len(zoneReg.zones))
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}

	thienLuu := zoneReg.byID["thien_luu_hai_vuc"]
	if thienLuu == nil {
		t.Fatal("thien_luu_hai_vuc not found")
	}

	// 1. Moving left from thien_luu_hai_vuc -> thai_gia_cang
	left := FindNeighborZoneOnAtlas(thienLuu, -1, 0)
	if left == nil || left.ID != "thai_gia_cang" {
		t.Fatalf("expected left neighbor thai_gia_cang, got %+v", left)
	}

	// 2. Moving right from thien_luu_hai_vuc -> nil (end of world)
	right := FindNeighborZoneOnAtlas(thienLuu, 1, 0)
	if right != nil {
		t.Fatalf("expected nil for right neighbor of thien_luu_hai_vuc, got %s", right.ID)
	}

	// 3. Moving top from thien_luu_hai_vuc -> dh_gioi_bich_dong_bac
	top := FindNeighborZoneOnAtlas(thienLuu, 0, -1)
	if top == nil || top.ID != "dh_gioi_bich_dong_bac" {
		t.Fatalf("expected top neighbor dh_gioi_bich_dong_bac, got %+v", top)
	}

	// 4. Moving bottom from thien_luu_hai_vuc -> cu_lang_vuc
	bottom := FindNeighborZoneOnAtlas(thienLuu, 0, 1)
	if bottom == nil || bottom.ID != "cu_lang_vuc" {
		t.Fatalf("expected bottom neighbor cu_lang_vuc, got %+v", bottom)
	}

	// 5. Layer check: bt_thai_co_van_hai moving right -> bt_quang_minh_tinh_doi (not a world_map zone)
	btVanHai := zoneReg.byID["bt_thai_co_van_hai"]
	btRight := FindNeighborZoneOnAtlas(btVanHai, 1, 0)
	if btRight == nil || btRight.ID != "bt_quang_minh_tinh_doi" {
		t.Fatalf("expected bt_quang_minh_tinh_doi for bt_thai_co_van_hai right neighbor, got %+v", btRight)
	}
}

func TestResolveCellArrivalEdgeTransition(t *testing.T) {
	previousZones, previousByID := zoneReg.zones, zoneReg.byID
	t.Cleanup(func() {
		zoneReg.zones, zoneReg.byID = previousZones, previousByID
	})

	zoneReg.zones = []models.Region{
		{ID: "thai_gia_cang", Name: "Thái Gia Cảng", ZoneType: ZoneTypeNamed, Width: 1000, Height: 1000, AtlasX: 8000, AtlasY: 6000, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000},
		{ID: "thien_luu_hai_vuc", Name: "Thiên Lưu Hải Vực", ZoneType: ZoneTypeNamed, Width: 1000, Height: 1000, AtlasX: 9000, AtlasY: 6000, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000},
	}
	zoneReg.byID = make(map[string]*models.Region, len(zoneReg.zones))
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}

	// Case 1: Player in thien_luu_hai_vuc tries to step right past 1000 (no right neighbor)
	var playerAtRightEdge models.Player
	playerAtRightEdge.ID = uuid.New()
	playerAtRightEdge.CurrentRegion = "thien_luu_hai_vuc"
	playerAtRightEdge.X = 1000
	playerAtRightEdge.Y = 997
	playerAtRightEdge.Rank = 5

	msg, _, err := resolveCellArrivalWithInstance(context.Background(), nil, &playerAtRightEdge, 1001, 997, "", false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "ranh giới cùng cực của thế giới") {
		t.Fatalf("expected end of world blockage message, got: %q", msg)
	}
	if playerAtRightEdge.CurrentRegion != "thien_luu_hai_vuc" || playerAtRightEdge.X != 1000 {
		t.Fatalf("player should remain at (1000, 997) in thien_luu_hai_vuc, got (%d, %d) in %s", playerAtRightEdge.X, playerAtRightEdge.Y, playerAtRightEdge.CurrentRegion)
	}

	// Case 2: Player in thien_luu_hai_vuc steps left past 0 (has left neighbor thai_gia_cang)
	var playerAtLeftEdge models.Player
	playerAtLeftEdge.ID = uuid.New()
	playerAtLeftEdge.CurrentRegion = "thien_luu_hai_vuc"
	playerAtLeftEdge.X = 0
	playerAtLeftEdge.Y = 500
	playerAtLeftEdge.Rank = 5

	msg2, _, err2 := resolveCellArrivalWithInstance(context.Background(), nil, &playerAtLeftEdge, -1, 500, "", false, nil, nil)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if !strings.Contains(msg2, "Thái Gia Cảng") {
		t.Fatalf("expected transition to Thái Gia Cảng, got message: %q", msg2)
	}
	if playerAtLeftEdge.CurrentRegion != "thai_gia_cang" {
		t.Fatalf("expected player region to be thai_gia_cang, got %s", playerAtLeftEdge.CurrentRegion)
	}
	if playerAtLeftEdge.X != 996 || playerAtLeftEdge.Y != 500 {
		t.Fatalf("expected player coords to be (996, 500), got (%d, %d)", playerAtLeftEdge.X, playerAtLeftEdge.Y)
	}

	// Case 3: Player in aperture_interior tries to step outside (x=20)
	var aperturePlayer models.Player
	aperturePlayer.ID = uuid.New()
	aperturePlayer.CurrentRegion = "aperture_interior"
	aperturePlayer.X = 19
	aperturePlayer.Y = 10
	aperturePlayer.Rank = 6
	loadedInst := uuid.New().String()
	loadedInstNull := sql.NullString{String: loadedInst, Valid: true}
	msg3, _, err3 := resolveCellArrivalWithInstance(context.Background(), nil, &aperturePlayer, 20, 10, "", false, &loadedInstNull, nil)
	if err3 != nil {
		t.Fatalf("unexpected error: %v", err3)
	}
	if !strings.Contains(msg3, "Khiếu Bích") {
		t.Fatalf("expected Khiếu Bích barrier message, got: %q", msg3)
	}
	if aperturePlayer.X != 19 {
		t.Fatalf("player in aperture should not move past 19, got %d", aperturePlayer.X)
	}
}

func TestValidateMovementTargetWalkEdge(t *testing.T) {
	// Walking to edge cells (e.g. x: -1, y: 0) should be allowed to trigger edge transition
	if err := validateMovementTarget(movementSpec{kind: movementWalk, hasTarget: true, x: -1, y: 0, allowBorderWalk: true}); err != nil {
		t.Fatalf("walking to border cell x=-1 should be allowed: %v", err)
	}
	if err := validateMovementTarget(movementSpec{kind: movementWalk, hasTarget: true, x: 0, y: -1, allowBorderWalk: true}); err != nil {
		t.Fatalf("walking to border cell y=-1 should be allowed: %v", err)
	}
	// Excessively negative coordinates should still be rejected
	if err := validateMovementTarget(movementSpec{kind: movementWalk, hasTarget: true, x: -15, y: 0, allowBorderWalk: true}); err == nil {
		t.Fatal("walking to x=-15 should be rejected as out of bounds")
	}
	// Non-walk targets (e.g. portal, teleport) must still reject negative coordinates
	if err := validateMovementTarget(movementSpec{kind: movementTeleport, hasTarget: true, x: -1, y: 0}); err == nil {
		t.Fatal("teleport to x=-1 should be rejected")
	}
}
