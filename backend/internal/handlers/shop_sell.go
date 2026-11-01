package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type SellShopRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
	Rank     int    `json:"rank"` // 0 = auto; >0 = the specific rank being sold
}

type SellShopItemInfo struct {
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
	Rank     int    `json:"rank"`
}

type SellShopBulkRequest struct {
	PlayerID string             `json:"player_id" binding:"required"`
	Items    []SellShopItemInfo `json:"items" binding:"required,dive"`
}

// resolveVillageMerchant loads (creating if needed) the region's default village
// merchant that backs the legacy caravan sell endpoints.
func resolveVillageMerchant(tx *sql.Tx, regionID string) (string, merchantProfile, error) {
	if err := ensureRegionMerchants(regionID); err != nil {
		return "", merchantProfile{}, err
	}
	merchantID := "village_" + regionID
	var profileID string
	if err := tx.QueryRow(`SELECT profile_id FROM merchants WHERE id=$1 AND is_active`, merchantID).Scan(&profileID); err != nil {
		return "", merchantProfile{}, err
	}
	profile, err := loadMerchantProfile(tx, profileID)
	return merchantID, profile, err
}

// runCaravanSell is the shared body for /api/shop/sell and /sell-bulk: it sells
// `lines` to the player's region village merchant (dynamic price + quota) and
// writes the response.
func runCaravanSell(c *gin.Context, authID uuid.UUID, lines []merchantSellLine, action string) {
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể mua sắm!"})
		return
	}
	merchantID, profile, err := resolveVillageMerchant(tx, player.CurrentRegion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thương nhân khu vực"})
		return
	}
	primeval, immortal, summaries, removedGuIDs, guSold, err := executeMerchantSale(tx, &player, merchantID, &profile, lines)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	player.PrimevalStones += primeval
	player.ImmortalStones += immortal
	RebuildEquippedGus(&player)
	if guSold {
		if err := db.RemovePlayerGus(tx, removedGuIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory changes"})
			return
		}
	}
	if _, err := tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2, equipped_gus=$3 WHERE id=$4`,
		player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save sell transaction"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	gainSummary := formatGainSummary(primeval, immortal)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         action,
		EntityType:     "shop",
		RequestSummary: fmt.Sprintf("Bán cho %s: %s | Tổng nhận: %s", profile.Name, strings.Join(summaries, ", "), gainSummary),
		Metadata:       map[string]interface{}{"merchant_id": merchantID, "lines": len(lines)},
		IPAddress:      c.ClientIP(),
		Success:        true,
	})
	notify.Send(db.DB, authID, "shop_sell", "Bán Hàng",
		fmt.Sprintf("Đã bán cho %s, thu về %s.", profile.Name, gainSummary))
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công, nhận %s!", gainSummary)})
}

// SellShopItem sells one entity to the region village merchant.
func SellShopItem(c *gin.Context) {
	var req SellShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}
	line := merchantSellLine{ItemID: req.ItemID, ItemType: req.ItemType, Quantity: req.Quantity, Rank: req.Rank}
	runCaravanSell(c, authID, []merchantSellLine{line}, "shop.sell_item")
}

// SellShopItemsBulk sells many entities to the region village merchant.
func SellShopItemsBulk(c *gin.Context) {
	var req SellShopBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm nào được chọn"})
		return
	}
	if len(req.Items) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi lần thanh lý tối đa 50 loại vật phẩm!"})
		return
	}
	lines := make([]merchantSellLine, 0, len(req.Items))
	for _, it := range req.Items {
		lines = append(lines, merchantSellLine{ItemID: it.ItemID, ItemType: it.ItemType, Quantity: it.Quantity, Rank: it.Rank})
	}
	runCaravanSell(c, authID, lines, "shop.sell_bulk")
}
