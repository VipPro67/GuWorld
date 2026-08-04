package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
)

func TestOpenMineChestEffect_AuditLog(t *testing.T) {
	// 1. Load env & connect
	var envLoaded bool
	for _, envPath := range []string{".env", "../../.env", "../../../.env", "backend/.env"} {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		t.Log("Warning: could not load env file")
	}

	err := db.Connect()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return
	}
	defer db.DB.Close()

	InventorySvc = services.NewInventoryService(db.DB)
	RecipeSvc = services.NewRecipeService(db.DB)

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Set up a test player
	playerID := uuid.New()
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, 'TestChestOpener', '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'opener@test.com', $2)`, playerID, "REF-OPN-"+playerID.String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}
	defer func() {
		db.DB.Exec(`DELETE FROM players WHERE id = $1`, playerID)
		db.DB.Exec(`DELETE FROM audit_logs WHERE player_id = $1`, playerID)
	}()

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	player := &models.Player{
		Account: models.Account{
			ID: playerID,
		},
		ResourceState: models.ResourceState{
			PrimevalStones: 1000,
		},
	}

	chestSpec := models.ItemSpec{
		ID:   "mine_chest",
		Name: "Rương Mỏ Linh Thạch",
	}

	// Seed item dictionary for fallback or material roll (we need some materials in registry to roll from, e.g. models.ItemRegistry)
	if len(models.ItemRegistry) == 0 {
		models.ItemRegistry = map[string]models.ItemSpec{
			"test_material": {
				ID:   "test_material",
				Name: "Linh Chi Thảo",
				Type: "Material",
				Rank: 1,
			},
		}
	}

	// 3. Call openMineChestEffect
	msg, err := openMineChestEffect(tx, player, chestSpec)
	if err != nil {
		t.Fatalf("openMineChestEffect failed: %v", err)
	}
	if msg == "" {
		t.Errorf("expected non-empty message from openMineChestEffect")
	}

	// Commit tx to write to DB so we can query audit log
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// 4. Verify audit log was written
	var count int
	err = db.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE player_id = $1 AND action = 'item.open_mine_chest'`, playerID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query audit logs: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 audit log, got %d", count)
	}

	// Verify details
	var action, requestSummary string
	err = db.DB.QueryRow(`SELECT action, request_summary FROM audit_logs WHERE player_id = $1`, playerID).Scan(&action, &requestSummary)
	if err != nil {
		t.Fatalf("Failed to query audit log row: %v", err)
	}

	if action != "item.open_mine_chest" {
		t.Errorf("Expected action 'item.open_mine_chest', got %q", action)
	}

	t.Logf("Audit log summary: %s", requestSummary)
}

func TestHealApertureEffect(t *testing.T) {
	var envLoaded bool
	for _, envPath := range []string{".env", "../../.env", "../../../.env", "backend/.env"} {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		t.Log("Warning: could not load env file")
	}

	err := db.Connect()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return
	}
	defer db.DB.Close()

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	playerID := uuid.New()
	// Insert immortal player (Rank 6)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, 'TestImmortalHeal', '', 6, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'immortal_heal@test.com', $2)`, playerID, "REF-IMM-"+playerID.String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}
	defer func() {
		db.DB.Exec(`DELETE FROM players WHERE id = $1`, playerID)
		db.DB.Exec(`DELETE FROM immortal_apertures WHERE player_id = $1`, playerID)
	}()

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Create an aperture row directly with 5000 HP
	apertureID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO immortal_apertures (id, player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, state)
		VALUES ($1, $2, 'HIGH', 30.0, 5000, 10000, 9000000, 10, 30, 0, 10, 'DAMAGED')
	`, apertureID, playerID)
	if err != nil {
		t.Fatalf("Failed to insert immortal aperture: %v", err)
	}

	player := &models.Player{
		Account: models.Account{
			ID: playerID,
		},
		CultivationState: models.CultivationState{
			Rank: 6,
		},
	}

	itemSpec := models.ItemSpec{
		ID:   "test_immortal_water",
		Name: "Tiên Tuyền Thủy",
		Effects: map[string]float64{
			"heal_aperture": 500,
		},
	}

	msg, err := healApertureEffect(tx, player, itemSpec)
	if err != nil {
		t.Fatalf("healApertureEffect failed: %v", err)
	}
	if msg == "" {
		t.Errorf("expected non-empty message")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify updated HP
	var updatedHP int
	err = db.DB.QueryRow(`SELECT aperture_hp FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&updatedHP)
	if err != nil {
		t.Fatalf("Failed to query aperture HP: %v", err)
	}
	if updatedHP != 5500 {
		t.Errorf("Expected aperture HP 5500, got %d", updatedHP)
	}
}

func TestHealApertureEffect_NoApertureError(t *testing.T) {
	var envLoaded bool
	for _, envPath := range []string{".env", "../../.env", "../../../.env", "backend/.env"} {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		t.Log("Warning: could not load env file")
	}

	err := db.Connect()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return
	}
	defer db.DB.Close()

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	playerID := uuid.New()
	// Insert immortal player (Rank 6)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, 'TestImmortalHeal2', '', 6, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'immortal_heal2@test.com', $2)`, playerID, "REF-IMM2-"+playerID.String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}
	defer func() {
		db.DB.Exec(`DELETE FROM players WHERE id = $1`, playerID)
		db.DB.Exec(`DELETE FROM immortal_apertures WHERE player_id = $1`, playerID)
	}()

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	player := &models.Player{
		Account: models.Account{
			ID: playerID,
		},
		CultivationState: models.CultivationState{
			Rank: 6,
		},
	}

	itemSpec := models.ItemSpec{
		ID:   "test_immortal_water",
		Name: "Tiên Tuyền Thủy",
		Effects: map[string]float64{
			"heal_aperture": 500,
		},
	}

	_, err = healApertureEffect(tx, player, itemSpec)
	if err == nil {
		t.Fatalf("expected error because immortal aperture does not exist, but got nil")
	}
	expectedErr := "không tìm thấy Tiên Khiếu của đạo hữu."
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestHealApertureEffect_MortalDenied(t *testing.T) {
	var envLoaded bool
	for _, envPath := range []string{".env", "../../.env", "../../../.env", "backend/.env"} {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		t.Log("Warning: could not load env file")
	}

	err := db.Connect()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return
	}
	defer db.DB.Close()

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	playerID := uuid.New()
	// Insert mortal player (Rank 5)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, 'TestMortalHeal', '', 5, 4, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'mortal_heal@test.com', $2)`, playerID, "REF-MOR-"+playerID.String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}
	defer func() {
		db.DB.Exec(`DELETE FROM players WHERE id = $1`, playerID)
	}()

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	player := &models.Player{
		Account: models.Account{
			ID: playerID,
		},
		CultivationState: models.CultivationState{
			Rank: 5,
		},
	}

	itemSpec := models.ItemSpec{
		ID:   "test_immortal_water",
		Name: "Tiên Tuyền Thủy",
		Effects: map[string]float64{
			"heal_aperture": 500,
		},
	}

	_, err = healApertureEffect(tx, player, itemSpec)
	if err == nil {
		t.Fatalf("expected error because player is mortal (rank 5 < 6), but got nil")
	}
	expectedErr := "Tiên Tuyền Thủy chỉ dành cho Cổ Tiên để trị liệu tiên khiếu."
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}
