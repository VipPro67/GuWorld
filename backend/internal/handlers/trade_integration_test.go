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
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
)

func TestP2PTradeFlow_ConnectingHeaven(t *testing.T) {
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

	// Initialize InventorySvc
	InventorySvc = services.NewInventoryService(db.DB)

	// 2. Set up test players
	player1ID := uuid.New()
	player2ID := uuid.New()

	defer func() {
		db.DB.Exec(`DELETE FROM trades WHERE sender_id IN ($1, $2) OR receiver_id IN ($1, $2)`, player1ID, player2ID)
		db.DB.Exec(`DELETE FROM players WHERE id IN ($1, $2)`, player1ID, player2ID)
	}()

	// Insert Player 1 (TestTrade1, 1000 stones)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'trade1@test.com', 'REF-T1')`, player1ID, "TestTrade1")
	if err != nil {
		t.Fatalf("Failed to insert player 1: %v", err)
	}
	InventorySvc.AddItem(nil, player1ID.String(), "iron_ore", "materials", 5)

	// Insert Player 2 (TestTrade2, 1000 stones)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'trade2@test.com', 'REF-T2')`, player2ID, "TestTrade2")
	if err != nil {
		t.Fatalf("Failed to insert player 2: %v", err)
	}
	InventorySvc.AddItem(nil, player2ID.String(), "copper_ore", "materials", 5)

	// Helper to give Connecting Heaven Gu
	giveConnectingHeaven := func(playerID uuid.UUID) {
		guInv := models.GuInventory{
			{
				InstanceID: uuid.New().String(),
				GuID:       "GU_CONNECTING_HEAVEN",
				Hunger:     100,
				Status:     "SLEEPING",
			},
		}
		guBytes, _ := json.Marshal(guInv)
		_, err := db.DB.Exec(`UPDATE players SET gu_inventory = $1 WHERE id = $2`, guBytes, playerID)
		if err != nil {
			t.Fatalf("failed to give Connecting Heaven Gu: %v", err)
		}
	}

	// Initiate trade request (should fail because players lack Connecting Heaven Gu)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("player_id", player1ID)
	reqBody, _ := json.Marshal(InitiateTradeRequest{ReceiverUsername: "TestTrade2"})
	c.Request = httptest.NewRequest("POST", "/api/trade/initiate", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	InitiateTrade(c)
	if w.Code == http.StatusOK {
		t.Fatalf("InitiateTrade should fail without Connecting Heaven Gu")
	}

	// Give Connecting Heaven to both players
	giveConnectingHeaven(player1ID)
	giveConnectingHeaven(player2ID)

	// Initiate trade request (should succeed now)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player1ID)
	c.Request = httptest.NewRequest("POST", "/api/trade/initiate", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	InitiateTrade(c)
	if w.Code != http.StatusOK {
		t.Fatalf("InitiateTrade failed with status %d: %s", w.Code, w.Body.String())
	}

	var initResp gin.H
	json.Unmarshal(w.Body.Bytes(), &initResp)
	tradeID := initResp["trade_id"].(string)

	// Accept trade from player 2
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player2ID)
	c.Params = []gin.Param{{Key: "id", Value: tradeID}}
	c.Request = httptest.NewRequest("POST", "/api/trade/accept/"+tradeID, nil)
	AcceptTrade(c)
	if w.Code != http.StatusOK {
		t.Fatalf("AcceptTrade failed: %s", w.Body.String())
	}

	// Update offer from player 1 (offer 100 Primeval, 3 iron_ore)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player1ID)
	c.Params = []gin.Param{{Key: "id", Value: tradeID}}
	offer1Body, _ := json.Marshal(TradeOffer{
		PrimevalStones: 100,
		ImmortalStones: 0,
		Gus:            []string{},
		Materials:      map[string]int{"iron_ore": 3},
	})
	c.Request = httptest.NewRequest("POST", "/api/trade/offer/"+tradeID, bytes.NewBuffer(offer1Body))
	c.Request.Header.Set("Content-Type", "application/json")
	UpdateTradeOffer(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTradeOffer player 1 failed: %s", w.Body.String())
	}

	// Update offer from player 2 (offer 50 Primeval, 2 copper_ore)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player2ID)
	c.Params = []gin.Param{{Key: "id", Value: tradeID}}
	offer2Body, _ := json.Marshal(TradeOffer{
		PrimevalStones: 50,
		ImmortalStones: 0,
		Gus:            []string{},
		Materials:      map[string]int{"copper_ore": 2},
	})
	c.Request = httptest.NewRequest("POST", "/api/trade/offer/"+tradeID, bytes.NewBuffer(offer2Body))
	c.Request.Header.Set("Content-Type", "application/json")
	UpdateTradeOffer(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTradeOffer player 2 failed: %s", w.Body.String())
	}

	// Lock trade - Player 1
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player1ID)
	c.Params = []gin.Param{{Key: "id", Value: tradeID}}
	c.Request = httptest.NewRequest("POST", "/api/trade/lock/"+tradeID, nil)
	LockTrade(c)
	if w.Code != http.StatusOK {
		t.Fatalf("LockTrade player 1 failed: %s", w.Body.String())
	}

	// Lock trade - Player 2
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player2ID)
	c.Params = []gin.Param{{Key: "id", Value: tradeID}}
	c.Request = httptest.NewRequest("POST", "/api/trade/lock/"+tradeID, nil)
	LockTrade(c)
	if w.Code != http.StatusOK {
		t.Fatalf("LockTrade player 2 failed: %s", w.Body.String())
	}

	// Confirm trade - Player 1
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player1ID)
	c.Params = []gin.Param{{Key: "id", Value: tradeID}}
	c.Request = httptest.NewRequest("POST", "/api/trade/confirm/"+tradeID, nil)
	ConfirmTrade(c)
	if w.Code != http.StatusOK {
		t.Fatalf("ConfirmTrade player 1 failed: %s", w.Body.String())
	}

	// Confirm trade - Player 2 (should trigger actual swap execution!)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("player_id", player2ID)
	c.Params = []gin.Param{{Key: "id", Value: tradeID}}
	c.Request = httptest.NewRequest("POST", "/api/trade/confirm/"+tradeID, nil)
	ConfirmTrade(c)
	if w.Code != http.StatusOK {
		t.Fatalf("ConfirmTrade player 2 failed: %s", w.Body.String())
	}

	// Verify player balances in database
	var stones1, stones2 int

	db.DB.QueryRow(`SELECT primeval_stones FROM players WHERE id = $1`, player1ID).Scan(&stones1)
	db.DB.QueryRow(`SELECT primeval_stones FROM players WHERE id = $1`, player2ID).Scan(&stones2)

	// Player 1 had 1000, offered 100, received 50 -> 950
	if stones1 != 950 {
		t.Errorf("Expected Player 1 stones to be 950, got %d", stones1)
	}

	// Player 2 had 1000, offered 50, received 100 -> 1050
	if stones2 != 1050 {
		t.Errorf("Expected Player 2 stones to be 1050, got %d", stones2)
	}

	// Check materials from player_inventory_items
	var qty int
	db.DB.QueryRow(`SELECT quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = 'iron_ore'`, player1ID.String()).Scan(&qty)
	if qty != 2 {
		t.Errorf("Expected Player 1 iron_ore=2, got %d", qty)
	}
	db.DB.QueryRow(`SELECT quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = 'copper_ore'`, player1ID.String()).Scan(&qty)
	if qty != 2 {
		t.Errorf("Expected Player 1 copper_ore=2, got %d", qty)
	}
	db.DB.QueryRow(`SELECT quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = 'iron_ore'`, player2ID.String()).Scan(&qty)
	if qty != 3 {
		t.Errorf("Expected Player 2 iron_ore=3, got %d", qty)
	}
	db.DB.QueryRow(`SELECT quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = 'copper_ore'`, player2ID.String()).Scan(&qty)
	if qty != 3 {
		t.Errorf("Expected Player 2 copper_ore=3, got %d", qty)
	}

	// Verify trade status is COMPLETED
	var status string
	db.DB.QueryRow(`SELECT status FROM trades WHERE id = $1`, tradeID).Scan(&status)
	if status != "COMPLETED" {
		t.Errorf("Expected trade status to be COMPLETED, got %s", status)
	}
}
