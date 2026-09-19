package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetTeleportTemplates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/teleport/templates", GetTeleportTemplates)

	req, _ := http.NewRequest("GET", "/api/teleport/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var templates []models.TeleportTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &templates); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(templates))
	}

	tier1 := templates[0]
	if tier1.Tier != models.TeleportTier1 || tier1.MaxRange != 1000 {
		t.Fatalf("unexpected tier1 data: %+v", tier1)
	}

	tier3 := templates[2]
	if tier3.Tier != models.TeleportTier3 || !tier3.IsCrossRegion {
		t.Fatalf("unexpected tier3 data: %+v", tier3)
	}
}

func TestEvaluateTeleportLink(t *testing.T) {
	ownerID := uuid.New()
	otherPlayerID := uuid.New()
	sect1ID := "sect-alpha"
	sect2ID := "sect-beta"

	sourceArray := models.TeleportArray{
		ID:                uuid.New().String(),
		Name:              "Trận Nam Cương",
		OwnerID:           ownerID.String(),
		RegionID:          "Southern_Border",
		X:                 100,
		Y:                 100,
		Tier:              models.TeleportTier1,
		MaxRange:          1000,
		IsCrossRegion:     false,
		DirectionType:     models.TeleportDirectionTwoWay,
		AccessPolicy:      models.TeleportAccessPublic,
		CurrentDurability: 50,
		MaxDurability:     50,
		IsActive:          true,
		FeeEnabled:        true,
		FeeCurrency:       "primeval_stones",
		FeeAmount:         100,
	}

	destArrayWithinRange := models.TeleportArray{
		ID:                uuid.New().String(),
		Name:              "Trận Nam Cương 2",
		OwnerID:           ownerID.String(),
		RegionID:          "Southern_Border",
		X:                 300,
		Y:                 200,
		Tier:              models.TeleportTier1,
		MaxRange:          1000,
		IsCrossRegion:     false,
		DirectionType:     models.TeleportDirectionTwoWay,
		AccessPolicy:      models.TeleportAccessPublic,
		CurrentDurability: 50,
		MaxDurability:     50,
		IsActive:          true,
		FeeEnabled:        true,
		FeeCurrency:       "primeval_stones",
		FeeAmount:         50,
	}

	destArrayOutOfRange := models.TeleportArray{
		ID:                uuid.New().String(),
		Name:              "Trận Nam Cương Xa",
		OwnerID:           ownerID.String(),
		RegionID:          "Southern_Border",
		X:                 2000,
		Y:                 2000,
		Tier:              models.TeleportTier1,
		MaxRange:          1000,
		IsCrossRegion:     false,
		DirectionType:     models.TeleportDirectionTwoWay,
		AccessPolicy:      models.TeleportAccessPublic,
		CurrentDurability: 50,
		MaxDurability:     50,
		IsActive:          true,
	}

	destArrayOtherRegion := models.TeleportArray{
		ID:                uuid.New().String(),
		Name:              "Trận Bắc Nguyên",
		OwnerID:           ownerID.String(),
		RegionID:          "Northern_Plains",
		X:                 500,
		Y:                 500,
		Tier:              models.TeleportTier3,
		MaxRange:          999999,
		IsCrossRegion:     true,
		DirectionType:     models.TeleportDirectionTwoWay,
		AccessPolicy:      models.TeleportAccessPublic,
		CurrentDurability: 100,
		MaxDurability:     100,
		IsActive:          true,
	}

	// 1. Test Owner Jump within range -> Fee is 0 for owner
	ownerPlayer := &models.Player{
		Account: models.Account{ID: ownerID},
	}
	item := evaluateTeleportLink(ownerPlayer, nil, &sourceArray, &destArrayWithinRange)
	if !item.CanUse {
		t.Fatalf("expected owner to be able to use link, got reason: %s", item.Reason)
	}
	if item.FeeToPay != 0 {
		t.Fatalf("expected owner fee to be 0, got %d", item.FeeToPay)
	}
	if item.Distance != 200 { // max(|100-300|, |100-200|) = 200
		t.Fatalf("expected distance 200, got %d", item.Distance)
	}

	// 2. Test Other Player Jump within range -> Fee is 50
	guestPlayer := &models.Player{
		Account: models.Account{ID: otherPlayerID},
	}
	itemGuest := evaluateTeleportLink(guestPlayer, nil, &sourceArray, &destArrayWithinRange)
	if !itemGuest.CanUse {
		t.Fatalf("expected guest to use public link, got: %s", itemGuest.Reason)
	}
	if itemGuest.FeeToPay != 50 {
		t.Fatalf("expected guest fee 50, got %d", itemGuest.FeeToPay)
	}

	// 3. Test Out of Range
	itemOutOfRange := evaluateTeleportLink(ownerPlayer, nil, &sourceArray, &destArrayOutOfRange)
	if itemOutOfRange.CanUse {
		t.Fatalf("expected out-of-range link to be unusable")
	}

	// 4. Test Cross-Region with Tier 1 source -> should fail
	itemCrossRegionFail := evaluateTeleportLink(ownerPlayer, nil, &sourceArray, &destArrayOtherRegion)
	if itemCrossRegionFail.CanUse {
		t.Fatalf("expected Tier 1 source unable to cross region")
	}

	// 5. Test Cross-Region with Tier 3 source -> should succeed
	tier3Source := sourceArray
	tier3Source.Tier = models.TeleportTier3
	tier3Source.IsCrossRegion = true
	itemCrossRegionOk := evaluateTeleportLink(ownerPlayer, nil, &tier3Source, &destArrayOtherRegion)
	if !itemCrossRegionOk.CanUse {
		t.Fatalf("expected Tier 3 source to succeed cross region, got: %s", itemCrossRegionOk.Reason)
	}

	// 6. Test Sect Access Policy
	sectArray := destArrayWithinRange
	sectArray.AccessPolicy = models.TeleportAccessSect
	sectArray.SectID = &sect1ID

	// Same sect member
	itemSectSame := evaluateTeleportLink(guestPlayer, &sect1ID, &sourceArray, &sectArray)
	if !itemSectSame.CanUse {
		t.Fatalf("expected same sect member to use sect array, got: %s", itemSectSame.Reason)
	}
	if itemSectSame.FeeToPay != 0 {
		t.Fatalf("expected sect member fee to be waived, got %d", itemSectSame.FeeToPay)
	}

	// Different sect member
	itemSectDiff := evaluateTeleportLink(guestPlayer, &sect2ID, &sourceArray, &sectArray)
	if itemSectDiff.CanUse {
		t.Fatalf("expected different sect member to be blocked from sect array")
	}
}

func TestFormationMasteryEffects(t *testing.T) {
	tplTier1 := teleportTemplates[models.TeleportTier1]
	tplTier3 := teleportTemplates[models.TeleportTier3]

	// 1. Novice Player: Rank 2, 0 Chân Ý Trận Đạo
	novicePlayer := &models.Player{}
	novicePlayer.ID = uuid.New()
	novicePlayer.Rank = 2
	novicePlayer.PathMasteries = models.PathMasteriesMap{"formation": 0}
	novicePlayer.DaoMarks = map[string]int64{"formation": 0}
	novicePlayer.PrimevalStones = 5000

	rateNovice, costNTNovice, costTTNovice, discNovice, gainNovice, lvlNovice, _ := calculateBuildStats(novicePlayer, tplTier1)
	if lvlNovice != 0 {
		t.Fatalf("expected level 0 for novice, got %d", lvlNovice)
	}
	if rateNovice != 70.0 {
		t.Fatalf("expected 70.0%% success rate for novice tier 1, got %.1f", rateNovice)
	}
	if discNovice != 0.0 || costNTNovice != 3000.0 || costTTNovice != 0.0 {
		t.Fatalf("expected 0%% discount and 3000 NT cost, got disc=%.0f, cost=%.0f", discNovice, costNTNovice)
	}
	if gainNovice != 15 {
		t.Fatalf("expected 15 mastery gain for tier 1, got %d", gainNovice)
	}

	// 2. Master Player: Rank 3, 150 Chân Ý (Đại Sư), 500 Đạo Ngân
	masterPlayer := &models.Player{}
	masterPlayer.ID = uuid.New()
	masterPlayer.Rank = 3
	masterPlayer.PathMasteries = models.PathMasteriesMap{"formation": 150}
	masterPlayer.DaoMarks = map[string]int64{"formation": 500}
	masterPlayer.PrimevalStones = 5000

	rateMaster, costNTMaster, _, discMaster, _, lvlMaster, lvlNameVN := calculateBuildStats(masterPlayer, tplTier1)
	if lvlMaster != 1 || lvlNameVN != "Đại Sư" {
		t.Fatalf("expected Master (1), got %d (%s)", lvlMaster, lvlNameVN)
	}
	// Expected rate = 70 + 15 (Master) + 3 (Rank 3 vs 2) + 1 (500 Dao Marks / 500) = 89.0%
	if rateMaster != 89.0 {
		t.Fatalf("expected 89.0%% rate for master, got %.1f", rateMaster)
	}
	// Expected cost = 3000 * (1 - 0.15) = 2550 NT
	if discMaster != 15.0 || costNTMaster != 2550.0 {
		t.Fatalf("expected 15%% discount and 2550 NT, got disc=%.0f, cost=%.0f", discMaster, costNTMaster)
	}

	// 3. Grandmaster Player: Rank 6, 1200 Chân Ý (Tông Sư)
	grandmasterPlayer := &models.Player{}
	grandmasterPlayer.ID = uuid.New()
	grandmasterPlayer.Rank = 6
	grandmasterPlayer.PathMasteries = models.PathMasteriesMap{"formation": 1200}
	grandmasterPlayer.DaoMarks = map[string]int64{"formation": 0}
	grandmasterPlayer.PrimevalStones = 200000
	grandmasterPlayer.ImmortalStones = 100

	rateGM, costNTGM, costTTGM, discGM, gainGM, lvlGM, _ := calculateBuildStats(grandmasterPlayer, tplTier3)
	if lvlGM != 2 {
		t.Fatalf("expected Grandmaster level 2, got %d", lvlGM)
	}
	// Expected Tier 3 rate = 45 (base) + 30 (Grandmaster) + 0 (Rank 6 vs 6) = 75.0%
	if rateGM != 75.0 {
		t.Fatalf("expected 75.0%% rate for GM tier 3, got %.1f", rateGM)
	}
	// Expected cost = 100,000 * 0.70 = 70,000 NT, 50 * 0.70 = 35 TT
	if discGM != 30.0 || costNTGM != 70000.0 || costTTGM != 35.0 {
		t.Fatalf("expected 30%% discount (70,000 NT, 35 TT), got disc=%.0f, costNT=%.0f, costTT=%.0f", discGM, costNTGM, costTTGM)
	}
	if gainGM != 200 {
		t.Fatalf("expected 200 mastery gain for tier 3, got %d", gainGM)
	}
}
