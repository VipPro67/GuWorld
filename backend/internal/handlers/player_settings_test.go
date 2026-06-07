package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

func TestUpdateProfileSettings_UsernameRules(t *testing.T) {
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

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Set up test players
	player1ID := uuid.New()
	player2ID := uuid.New()

	defer func() {
		db.DB.Exec(`DELETE FROM players WHERE id IN ($1, $2)`, player1ID, player2ID)
	}()

	// Insert Player 1: "OrigPlayerOne" with email "p1@test.com"
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'p1@test.com', 'REF-P1')`, player1ID, "OrigPlayerOne")
	if err != nil {
		t.Fatalf("Failed to insert player 1: %v", err)
	}

	// Insert Player 2: "OrigPlayerTwo" with email "p2@test.com"
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'p2@test.com', 'REF-P2')`, player2ID, "OrigPlayerTwo")
	if err != nil {
		t.Fatalf("Failed to insert player 2: %v", err)
	}

	// Subtest 1: Update to a valid username with spaces and accented characters, check trimming
	t.Run("Valid username update with spaces and Vietnamese accents", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("player_id", player1ID)

		reqBody, _ := json.Marshal(UpdateProfileSettingsRequest{
			Username: "  Cổ Sư Đẹp Trai 123  ",
			Bio:      "Hello World",
		})
		c.Request = httptest.NewRequest("PUT", "/api/player/profile/settings", bytes.NewBuffer(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateProfileSettings(c)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify database contains trimmed username, and email remains "p1@test.com"
		var dbUsername, dbEmail string
		err := db.DB.QueryRow(`SELECT username, email FROM players WHERE id = $1`, player1ID).Scan(&dbUsername, &dbEmail)
		if err != nil {
			t.Fatalf("Failed to query player 1 from DB: %v", err)
		}

		if dbUsername != "Cổ Sư Đẹp Trai 123" {
			t.Errorf("Expected username to be trimmed to 'Cổ Sư Đẹp Trai 123', got '%s'", dbUsername)
		}
		if dbEmail != "p1@test.com" {
			t.Errorf("Expected email to remain 'p1@test.com', got '%s'", dbEmail)
		}
	})

	// Subtest 2: Update with invalid characters (symbols)
	t.Run("Reject invalid username with special characters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("player_id", player1ID)

		reqBody, _ := json.Marshal(UpdateProfileSettingsRequest{
			Username: "CổSư@ĐộcLạ",
			Bio:      "Invalid name",
		})
		c.Request = httptest.NewRequest("PUT", "/api/player/profile/settings", bytes.NewBuffer(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateProfileSettings(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Subtest 3: Update to an already existing username
	t.Run("Reject duplicate username", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("player_id", player1ID)

		reqBody, _ := json.Marshal(UpdateProfileSettingsRequest{
			Username: "OrigPlayerTwo", // Player 2 has this username
			Bio:      "Should fail",
		})
		c.Request = httptest.NewRequest("PUT", "/api/player/profile/settings", bytes.NewBuffer(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateProfileSettings(c)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected status 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}
	})
}
