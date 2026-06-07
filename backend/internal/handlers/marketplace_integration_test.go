package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestPlaceBid_RefundLogic(t *testing.T) {
	// Set gin to test mode to suppress debug output
	gin.SetMode(gin.TestMode)

	// 1. Load environment variables & connect to the database
	var envLoaded bool
	for _, envPath := range []string{".env", "../../.env", "../../../.env", "backend/.env"} {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		t.Log("Warning: could not load env file, assuming system environment variables are set")
	}

	err := db.Connect()
	if err != nil {
		t.Skipf("Skipping database integration test because connection failed: %v", err)
		return
	}
	defer db.DB.Close()

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Set up test players
	player1ID := uuid.New()
	player2ID := uuid.New()

	// Clean up just in case
	defer func() {
		db.DB.Exec(`DELETE FROM auction WHERE seller_id IN ($1, $2) OR highest_bidder_id IN ($1, $2)`, player1ID, player2ID)
		db.DB.Exec(`DELETE FROM players WHERE id IN ($1, $2)`, player1ID, player2ID)
	}()

	// Insert player 1 (initial stones = 1000)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'player1@test.com', 'REF-P1')`, player1ID, "TestPlayer1")
	if err != nil {
		t.Fatalf("Failed to insert player 1: %v", err)
	}

	// Insert player 2 (initial stones = 1000)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'player2@test.com', 'REF-P2')`, player2ID, "TestPlayer2")
	if err != nil {
		t.Fatalf("Failed to insert player 2: %v", err)
	}

	// 3. Set up active auction (seller is player 2)
	auctionID := uuid.New().String()
	endTime := time.Now().Add(1 * time.Hour)
	_, err = db.DB.Exec(`INSERT INTO auction(id, seller_id, item_type, item_data, quantity, currency, start_price, current_price, highest_bidder_id, end_time, status, listing_type, created_at)
		VALUES ($1, $2, 'Material', '"test_item"', 1, 'primeval', 100.0, 100.0, NULL, $3, 'ACTIVE', 'AUCTION', NOW())`,
		auctionID, player2ID, endTime)
	if err != nil {
		t.Fatalf("Failed to insert auction listing: %v", err)
	}

	// Helper function to call PlaceBid
	callPlaceBid := func(playerID uuid.UUID, amount float64) (*httptest.ResponseRecorder, gin.H) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("player_id", playerID)

		reqBody, _ := json.Marshal(PlaceBidRequest{
			ListingID: auctionID,
			Amount:    amount,
		})
		c.Request = httptest.NewRequest("POST", "/api/shop/bao-hoang/bid", bytes.NewBuffer(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		PlaceBid(c)

		var resp gin.H
		json.Unmarshal(w.Body.Bytes(), &resp)
		return w, resp
	}

	// Helper to check stones in database
	getStones := func(playerID uuid.UUID) int {
		var stones int
		err := db.DB.QueryRow(`SELECT primeval_stones FROM players WHERE id = $1`, playerID).Scan(&stones)
		if err != nil {
			t.Fatalf("failed to query player stones: %v", err)
		}
		return stones
	}

	// 4. Test Bid 1: Player 1 bids 150 (initial state: no bidder, starts at 100)
	w, resp := callPlaceBid(player1ID, 150)
	if w.Code != http.StatusOK {
		t.Fatalf("Bid 1 failed with status %d: %v", w.Code, resp)
	}

	// Verify player 1 stones (1000 - 150 = 850)
	if s := getStones(player1ID); s != 850 {
		t.Errorf("Expected Player 1 stones to be 850, got %d", s)
	}

	// 5. Test Bid 2: Player 1 (already highest bidder) bids 200 (raise bid)
	w, resp = callPlaceBid(player1ID, 200)
	if w.Code != http.StatusOK {
		t.Fatalf("Bid 2 failed with status %d: %v", w.Code, resp)
	}

	// Verify Player 1 stones:
	// They previously bid 150 (charged 150).
	// They now bid 200. Only 50 additional stones should be deducted.
	// 850 - 50 = 800.
	if s := getStones(player1ID); s != 800 {
		t.Errorf("Expected Player 1 stones to be 800, got %d", s)
	}

	// 6. Test Bid 3: Player 2 (different player) bids 250
	w, resp = callPlaceBid(player2ID, 250)
	if w.Code != http.StatusOK {
		t.Fatalf("Bid 3 failed with status %d: %v", w.Code, resp)
	}

	// Verify Player 2 stones:
	// Initial: 1000. Bid: 250.
	// 1000 - 250 = 750.
	if s := getStones(player2ID); s != 750 {
		t.Errorf("Expected Player 2 stones to be 750, got %d", s)
	}

	// Verify Player 1 stones (fully refunded 200):
	// Previous bid was 200, so they should be refunded 200.
	// 800 + 200 = 1000.
	if s := getStones(player1ID); s != 1000 {
		t.Errorf("Expected Player 1 stones to be refunded to 1000, got %d", s)
	}
}

func TestCreateAndBuyTrade_ConnectingHeavenLogic(t *testing.T) {
	// Set gin to test mode to suppress debug output
	gin.SetMode(gin.TestMode)

	// 1. Load environment variables & connect to the database
	var envLoaded bool
	for _, envPath := range []string{".env", "../../.env", "../../../.env", "backend/.env"} {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		t.Log("Warning: could not load env file, assuming system environment variables are set")
	}
	err := db.Connect()
	if err != nil {
		t.Skipf("Skipping database integration test because connection failed: %v", err)
		return
	}
	defer db.DB.Close()

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Set up test players
	player1ID := uuid.New()
	player2ID := uuid.New()
	player3ID := uuid.New() // Observer/Stealer

	// Clean up just in case
	defer func() {
		db.DB.Exec(`DELETE FROM auction WHERE seller_id IN ($1, $2, $3) OR highest_bidder_id IN ($1, $2, $3)`, player1ID, player2ID, player3ID)
		db.DB.Exec(`DELETE FROM players WHERE id IN ($1, $2, $3)`, player1ID, player2ID, player3ID)
	}()

	// Insert player 1 (Seller, initial stones = 1000)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code, materials_inventory)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'player1_trade@test.com', 'REF-P1T', '{"iron_ore": 10}')`, player1ID, "TestPlayer1_Trade")
	if err != nil {
		t.Fatalf("Failed to insert player 1: %v", err)
	}

	// Insert player 2 (Buyer, initial stones = 1000)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'player2_trade@test.com', 'REF-P2T')`, player2ID, "TestPlayer2_Trade")
	if err != nil {
		t.Fatalf("Failed to insert player 2: %v", err)
	}

	// Insert player 3 (Stealer, initial stones = 1000)
	_, err = db.DB.Exec(`INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, hp, base_hp, current_region, email, ref_code)
		VALUES ($1, $2, '', 1, 1, 'Giáp', 90, 100.0, 100.0, 1000, 100.0, 100.0, 'Nam Hoang', 'player3_trade@test.com', 'REF-P3T')`, player3ID, "TestPlayer3_Trade")
	if err != nil {
		t.Fatalf("Failed to insert player 3: %v", err)
	}

	// Helper function to call CreateListing
	callCreateListing := func(playerID uuid.UUID, listingType, targetBuyer string) (*httptest.ResponseRecorder, gin.H) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("player_id", playerID)

		reqBody, _ := json.Marshal(CreateListingRequest{
			ItemType:            "Material",
			ItemID:              "iron_ore",
			Quantity:            5,
			Price:               100.0,
			Currency:            "primeval",
			DurationHours:       12,
			ListingType:         listingType,
			TargetBuyerUsername: targetBuyer,
		})
		c.Request = httptest.NewRequest("POST", "/api/shop/bao-hoang/listings", bytes.NewBuffer(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		CreateListing(c)

		var resp gin.H
		json.Unmarshal(w.Body.Bytes(), &resp)
		return w, resp
	}

	// Helper function to call BuyListing
	callBuyListing := func(playerID uuid.UUID, listingID string) (*httptest.ResponseRecorder, gin.H) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("player_id", playerID)

		reqBody, _ := json.Marshal(BuyListingRequest{
			ListingID: listingID,
			Quantity:  5,
		})
		c.Request = httptest.NewRequest("POST", "/api/shop/bao-hoang/buy", bytes.NewBuffer(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		BuyListing(c)

		var resp gin.H
		json.Unmarshal(w.Body.Bytes(), &resp)
		return w, resp
	}

	// Helper to give a player Connecting Heaven Gu
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
			t.Fatalf("failed to update player gu_inventory: %v", err)
		}
	}

	// Helper to check stones in database
	getStones := func(playerID uuid.UUID) int {
		var stones int
		err := db.DB.QueryRow(`SELECT primeval_stones FROM players WHERE id = $1`, playerID).Scan(&stones)
		if err != nil {
			t.Fatalf("failed to query player stones: %v", err)
		}
		return stones
	}

	// Test 1: Create direct trade listing when seller DOES NOT have Connecting Heaven Gu -> Should fail
	w, resp := callCreateListing(player1ID, "TRADE", "TestPlayer2_Trade")
	if w.Code == http.StatusOK {
		t.Fatalf("Listing creation should have failed because seller lacks Connecting Heaven Gu, got response: %v", resp)
	}

	// Give Seller the Connecting Heaven Gu
	giveConnectingHeaven(player1ID)

	// Test 2: Create direct trade listing when seller has Connecting Heaven Gu -> Should succeed, basic listing fee is 0
	w, resp = callCreateListing(player1ID, "TRADE", "TestPlayer2_Trade")
	if w.Code != http.StatusOK {
		t.Fatalf("Listing creation failed with status %d: %v", w.Code, resp)
	}

	listingID := resp["listing_id"].(string)

	// Verify seller paid 0 basic fee (stones remain 1000)
	if s := getStones(player1ID); s != 1000 {
		t.Errorf("Expected Seller stones to be 1000, got %d", s)
	}

	// Test 3: Buyer (Player 2) tries to buy but DOES NOT have Connecting Heaven Gu -> Should fail
	w, resp = callBuyListing(player2ID, listingID)
	if w.Code == http.StatusOK {
		t.Fatalf("Purchase should have failed because buyer lacks Connecting Heaven Gu, got response: %v", resp)
	}

	// Give Buyer the Connecting Heaven Gu
	giveConnectingHeaven(player2ID)

	// Test 4: Stealer (Player 3) tries to buy the private trade -> Should fail with 403 Forbidden
	w, resp = callBuyListing(player3ID, listingID)
	if w.Code == http.StatusOK {
		t.Fatalf("Stealer should not have been able to purchase private trade listing, got response: %v", resp)
	}

	// Test 5: Buyer buys the listing -> Should succeed, 0 commission (Seller receives full 100 stones)
	w, resp = callBuyListing(player2ID, listingID)
	if w.Code != http.StatusOK {
		t.Fatalf("Buyer failed to complete direct trade buyout: %v", resp)
	}

	// Verify buyer stones: 1000 - (5 * 100) = 500
	if s := getStones(player2ID); s != 500 {
		t.Errorf("Expected Buyer stones to be 500, got %d", s)
	}

	// Verify seller stones: 1000 + (5 * 100) = 1500 (No commission deducted)
	if s := getStones(player1ID); s != 1500 {
		t.Errorf("Expected Seller stones to be 1500 (full payment), got %d", s)
	}
}
