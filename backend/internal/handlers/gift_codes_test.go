package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/services"
)

func TestRedeemGiftCode_MultipleTimes(t *testing.T) {
	gin.SetMode(gin.TestMode)

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

	PlayerSvc = services.NewPlayerService(db.DB)
	InventorySvc = services.NewInventoryService(db.DB)

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Set up a test player
	playerID := uuid.New()
	db.DB.Exec(`DELETE FROM players WHERE username = 'TestCodeRedeemer'`)
	defer func() {
		db.DB.Exec(`DELETE FROM players WHERE id = $1`, playerID)
	}()

	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, 'TestCodeRedeemer', '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'redeemer@test.com', $2)`, playerID, "REF-RED-"+playerID.String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}

	// 3. Set up a test gift code with max_per_player = 2
	giftCodeID := uuid.New()
	codeStr := strings.ToUpper("TEST_MULTIPLE_REDEEM_" + playerID.String()[:8])
	rewardsJSON := `[{"type": "primeval_stones", "amount": 100}]`
	expiresAt := time.Now().Add(24 * time.Hour)

	db.DB.Exec(`DELETE FROM gift_codes WHERE code = $1`, codeStr)
	defer func() {
		db.DB.Exec(`DELETE FROM gift_codes WHERE id = $1`, giftCodeID)
	}()

	_, err = db.DB.Exec(`INSERT INTO gift_codes (id, code, rewards, max_uses, used_count, max_per_player, expires_at, is_active)
		VALUES ($1, $2, $3, 10, 0, 2, $4, true)`, giftCodeID, codeStr, rewardsJSON, expiresAt)
	if err != nil {
		t.Fatalf("Failed to insert test gift code: %v", err)
	}

	// Helper to send redemption request
	redeemFn := func() (int, string) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("player_id", playerID)

		reqBody, _ := json.Marshal(RedeemGiftCodeRequest{
			Code: codeStr,
		})
		c.Request = httptest.NewRequest("POST", "/api/player/redeem-gift-code", bytes.NewBuffer(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		RedeemGiftCode(c)
		return w.Code, w.Body.String()
	}

	// 1st redemption should succeed
	code1, body1 := redeemFn()
	if code1 != http.StatusOK {
		t.Fatalf("First redemption failed: status %d, response: %s", code1, body1)
	}

	// 2nd redemption should succeed (since max_per_player = 2)
	code2, body2 := redeemFn()
	if code2 != http.StatusOK {
		t.Fatalf("Second redemption failed: status %d, response: %s", code2, body2)
	}

	// 3rd redemption should fail (exceeds max_per_player)
	code3, body3 := redeemFn()
	if code3 != http.StatusBadRequest {
		t.Fatalf("Third redemption expected status 400, got %d: %s", code3, body3)
	}
}
