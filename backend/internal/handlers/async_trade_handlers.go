package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type CreateTradeOfferRequest struct {
	Offered        models.TradeOfferData `json:"offered" binding:"required"`
	Requested      models.TradeOfferData `json:"requested" binding:"required"`
	DurationHours  int                   `json:"duration_hours" binding:"required"`
	TargetUsername string                `json:"target_username"`
}

type AcceptTradeOfferRequest struct {
	OfferID string `json:"offer_id" binding:"required"`
}

type CancelTradeOfferRequest struct {
	OfferID string `json:"offer_id" binding:"required"`
}

type OfferedGuInfo struct {
	InstanceID string `json:"instance_id"`
	GuID       string `json:"gu_id"`
	GuName     string `json:"gu_name"`
	Rank       int    `json:"rank"`
}

type TradeOfferResponse struct {
	ID               string                `json:"id"`
	CreatorID        string                `json:"creator_id"`
	CreatorUsername  string                `json:"creator_username"`
	TargetID         *string               `json:"target_id,omitempty"`
	TargetUsername   *string               `json:"target_username,omitempty"`
	Offered          models.TradeOfferData `json:"offered"`
	OfferedGuDetails []OfferedGuInfo       `json:"offered_gu_details,omitempty"`
	Requested        models.TradeOfferData `json:"requested"`
	Status           string                `json:"status"`
	DurationHours    int                   `json:"duration_hours"`
	EndTime          time.Time             `json:"end_time"`
	CreatedAt        time.Time             `json:"created_at"`
}

func isTradeOfferEmpty(data models.TradeOfferData) bool {
	if data.PrimevalStones > 0 || data.ImmortalStones > 0 {
		return false
	}
	if len(data.Gus) > 0 {
		return false
	}
	for _, qty := range data.Materials {
		if qty > 0 {
			return false
		}
	}
	for _, rate := range data.Recipes {
		if rate > 0 {
			return false
		}
	}
	return true
}

func validateTradeOfferData(data models.TradeOfferData) error {
	if data.PrimevalStones < 0 || data.ImmortalStones < 0 ||
		!pricing.IsValidAmount(data.PrimevalStones) || !pricing.IsValidAmount(data.ImmortalStones) {
		return fmt.Errorf("số lượng đá trao đổi không hợp lệ")
	}
	if hasDuplicateGuInstanceIDs(data.Gus) {
		return fmt.Errorf("danh sách Cổ trùng không được chứa ID trùng lặp hoặc rỗng")
	}
	for _, quantity := range data.Materials {
		if quantity < 0 {
			return fmt.Errorf("số lượng nguyên liệu trao đổi không hợp lệ")
		}
	}
	for _, completionRate := range data.Recipes {
		if completionRate < 0 || completionRate > 100 {
			return fmt.Errorf("độ hoàn thiện Bí phương phải từ 0 đến 100")
		}
	}
	return nil
}

// System BHTH UUID sentinel for locking offered Gu items
var BHTHSystemUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// CreateTradeOffer posts a new 1-sided async trade offer on BHTH
func CreateTradeOffer(c *gin.Context) {
	var req CreateTradeOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.DurationHours < 1 || req.DurationHours > 168 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời hạn đơn trao đổi từ 1 đến 168 giờ"})
		return
	}
	if err := validateTradeOfferData(req.Offered); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm đưa ra không hợp lệ: " + err.Error()})
		return
	}
	if err := validateTradeOfferData(req.Requested); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm yêu cầu không hợp lệ: " + err.Error()})
		return
	}
	if isTradeOfferEmpty(req.Offered) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn phải đưa ra ít nhất một vật phẩm, Cổ trùng, bí phương hoặc tiền tệ để trao đổi"})
		return
	}
	if isTradeOfferEmpty(req.Requested) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn phải yêu cầu ít nhất một vật phẩm, Cổ trùng, bí phương hoặc tiền tệ muốn nhận"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	creator, err := evaluateManaTx(tx, authID)
	if err != nil || gamefx.IsDeadState(&creator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể tạo đơn trao đổi!"})
		return
	}
	if !HasBaoHoangAccess(creator.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Thông Thiên Cổ và Thần Niệm Cổ để kết nối Bảo Hoàng Thiên!"})
		return
	}

	var activeCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM trade_offers WHERE creator_id = $1 AND status = 'ACTIVE' AND end_time > NOW()`, creator.ID).Scan(&activeCount)
	if activeCount >= 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn chỉ có thể tạo tối đa 10 đơn trao đổi đang hoạt động cùng lúc trên Bảo Hoàng Thiên"})
		return
	}

	targetPlayerID := creator.ID // default: public offer (sentinel: targetPlayerID == creator.ID)
	if req.TargetUsername != "" {
		var tID uuid.UUID
		err := tx.QueryRow(`SELECT id FROM players WHERE username = $1`, req.TargetUsername).Scan(&tID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không tìm thấy đạo hữu %s để chỉ định trao đổi", req.TargetUsername)})
			return
		}
		if tID == creator.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể chỉ định chính bản thân làm đối tác trao đổi"})
			return
		}
		targetPlayerID = tID
	}

	// Validate & Lock Offered Items
	offered := req.Offered
	if offered.PrimevalStones > 0 {
		if creator.PrimevalStones < offered.PrimevalStones {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch đưa trao đổi (Cần %.3f, hiện có %.3f)", offered.PrimevalStones, creator.PrimevalStones)})
			return
		}
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2`, pricing.Quantize(offered.PrimevalStones), creator.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ Nguyên Thạch"})
			return
		}
	}
	if offered.ImmortalStones > 0 {
		if creator.Rank < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ sư dưới Lục Chuyển không thể dùng Tiên Nguyên Thạch"})
			return
		}
		if creator.ImmortalStones < offered.ImmortalStones {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch (Cần %.3f, hiện có %.3f)", offered.ImmortalStones, creator.ImmortalStones)})
			return
		}
		_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, pricing.Quantize(offered.ImmortalStones), creator.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ Tiên Nguyên Thạch"})
			return
		}
	}

	creatorMats := getPlayerMaterialsMap(tx, creator.ID)
	for matID, qty := range offered.Materials {
		if qty <= 0 {
			continue
		}
		if creatorMats[matID] < qty {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s (Cần %d, hiện có %d)", matID, qty, creatorMats[matID])})
			return
		}
		if err := deductMaterialTx(tx, creator.ID, matID, qty); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ nguyên liệu trao đổi"})
			return
		}
	}

	for _, guInstID := range offered.Gus {
		var ownerID uuid.UUID
		var isVital, equipped, isLocked bool
		var state string
		err := tx.QueryRow(`SELECT player_id, is_vital, equipped, is_locked, state FROM player_gu WHERE id = $1`, guInstID).Scan(&ownerID, &isVital, &equipped, &isLocked, &state)
		if err != nil || ownerID != creator.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không sở hữu Cổ trùng đưa trao đổi"})
			return
		}
		if isVital {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể dùng Bổn Mạng Cổ làm vật trao đổi!"})
			return
		}
		if equipped {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể giao dịch Cổ trùng đang trang bị!"})
			return
		}
		if isLocked {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể giao dịch Cổ trùng đang bị khóa!"})
			return
		}
		if state == "DEAD" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể giao dịch Cổ trùng đã tử vong!"})
			return
		}
		// Lock Gu to system BHTH UUID
		_, err = tx.Exec(`UPDATE player_gu SET player_id = $1, equipped = false WHERE id = $2`, BHTHSystemUUID, guInstID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khóa Cổ trùng trao đổi"})
			return
		}
	}

	for recID, rate := range offered.Recipes {
		recProgress, err := RecipeSvc.Get(tx, creator.ID.String(), recID)
		if err != nil || recProgress == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn chưa khai mở Bí phương %s để đem trao đổi", recID)})
			return
		}
		if err := RecipeSvc.Delete(tx, creator.ID.String(), recID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khóa Bí phương trao đổi"})
			return
		}
		if rate <= 0 {
			offered.Recipes[recID] = recProgress.CompletionRate
		}
	}

	offeredJSON, _ := json.Marshal(offered)
	requestedJSON, _ := json.Marshal(req.Requested)
	endTime := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)

	var offerID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO trade_offers (creator_id, player_id, offered_data, requested_data, status, duration_hours, end_time)
		VALUES ($1, $2, $3, $4, 'ACTIVE', $5, $6)
		RETURNING id`,
		creator.ID, targetPlayerID, offeredJSON, requestedJSON, req.DurationHours, endTime,
	).Scan(&offerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo đơn trao đổi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &creator.ID, Action: "marketplace.create_trade_offer",
		EntityType: "trade_offer", EntityID: offerID.String(),
		RequestSummary: "Đăng đơn trao đổi tự động mới",
		IPAddress:      c.ClientIP(), Success: true,
	})

	updatedCreator := creator
	if tx2, err2 := db.DB.Begin(); err2 == nil {
		updatedCreator, _ = evaluateManaTx(tx2, creator.ID)
		_ = tx2.Commit()
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Đăng đơn trao đổi tự động thành công!",
		"offer_id": offerID.String(),
		"player":   updatedCreator,
	})
}

// GetTradeOffers lists active trade offers
func GetTradeOffers(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
		return
	}
	player, err := evaluateManaTx(tx, authID)
	_ = tx.Commit()
	if err != nil || !HasBaoHoangAccess(player.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Thông Thiên Cổ và Thần Niệm Cổ để kết nối Bảo Hoàng Thiên!"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT t.id, t.creator_id, p.username, t.player_id, COALESCE(tp.username, ''),
		       t.offered_data, t.requested_data, t.status, t.duration_hours, t.end_time, t.created_at
		FROM trade_offers t
		JOIN players p ON p.id = t.creator_id
		LEFT JOIN players tp ON tp.id = t.player_id AND t.player_id != t.creator_id
		WHERE t.status = 'ACTIVE' AND t.end_time > NOW()
		  AND (t.player_id = t.creator_id OR t.player_id = $1 OR t.creator_id = $1)
		ORDER BY t.created_at DESC LIMIT 100`, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query error"})
		return
	}
	defer rows.Close()

	var allGuInstIDs []string
	res := []TradeOfferResponse{}
	for rows.Next() {
		var o TradeOfferResponse
		var pID uuid.UUID
		var targetUName string
		var offJSON, reqJSON []byte
		if err := rows.Scan(&o.ID, &o.CreatorID, &o.CreatorUsername, &pID, &targetUName, &offJSON, &reqJSON, &o.Status, &o.DurationHours, &o.EndTime, &o.CreatedAt); err == nil {
			_ = json.Unmarshal(offJSON, &o.Offered)
			_ = json.Unmarshal(reqJSON, &o.Requested)
			if pID != uuid.Nil && o.CreatorID != pID.String() {
				pIDStr := pID.String()
				o.TargetID = &pIDStr
				if targetUName != "" {
					o.TargetUsername = &targetUName
				}
			}
			allGuInstIDs = append(allGuInstIDs, o.Offered.Gus...)
			res = append(res, o)
		}
	}

	if len(allGuInstIDs) > 0 {
		guRows, err := db.DB.Query(`SELECT id, gu_id, gu_rank FROM player_gu WHERE id = ANY($1::text[])`, pq.Array(allGuInstIDs))
		if err == nil {
			defer guRows.Close()
			guInfoMap := make(map[string]OfferedGuInfo)
			for guRows.Next() {
				var info OfferedGuInfo
				if err := guRows.Scan(&info.InstanceID, &info.GuID, &info.Rank); err == nil {
					info.GuName = getGuName(info.GuID, info.Rank)
					guInfoMap[info.InstanceID] = info
				}
			}
			for i := range res {
				for _, instID := range res[i].Offered.Gus {
					if info, found := guInfoMap[instID]; found {
						res[i].OfferedGuDetails = append(res[i].OfferedGuDetails, info)
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, res)
}

// AcceptTradeOffer accepts an active trade offer with 100% exact match
func AcceptTradeOffer(c *gin.Context) {
	var req AcceptTradeOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	offerUUID, err := uuid.Parse(req.OfferID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OfferID không hợp lệ"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	acceptor, err := evaluateManaTx(tx, authID)
	if err != nil || gamefx.IsDeadState(&acceptor) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể chấp nhận trao đổi!"})
		return
	}
	if !HasBaoHoangAccess(acceptor.GuInventory) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn cần sở hữu Cổ trùng Thông Thiên Cổ và Thần Niệm Cổ để giao dịch Bảo Hoàng Thiên!"})
		return
	}

	var creatorID uuid.UUID
	var playerID uuid.UUID
	var offJSON, reqJSON []byte
	var status string
	var endTime time.Time

	err = tx.QueryRow(`
		SELECT creator_id, player_id, offered_data, requested_data, status, end_time
		FROM trade_offers WHERE id = $1 FOR UPDATE`, offerUUID,
	).Scan(&creatorID, &playerID, &offJSON, &reqJSON, &status, &endTime)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Đơn trao đổi không tồn tại"})
		return
	}
	if status != "ACTIVE" || time.Now().After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn trao đổi không còn khả dụng hoặc đã hết hạn"})
		return
	}
	if creatorID == acceptor.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tự chấp nhận đơn trao đổi của mình"})
		return
	}
	// Check if this is a targeted trade offer for someone else
	if playerID != uuid.Nil && playerID != creatorID && playerID != acceptor.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đơn trao đổi này được chỉ định riêng cho đạo hữu khác"})
		return
	}

	// Verify creator dual-gu access
	creator, err := evaluateManaTx(tx, creatorID)
	if err != nil || !HasBaoHoangAccess(creator.GuInventory) {
		// Restore creator items and cancel offer
		var offered models.TradeOfferData
		_ = json.Unmarshal(offJSON, &offered)
		_ = restoreTradeOfferItems(tx, creatorID, offered)
		_, _ = tx.Exec(`UPDATE trade_offers SET status = 'CANCELLED' WHERE id = $1`, offerUUID)
		_ = tx.Commit()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người tạo đơn không còn đủ điều kiện kết nối Bảo Hoàng Thiên. Đơn trao đổi đã bị hủy và trả lại vật phẩm."})
		return
	}

	var offered, requested models.TradeOfferData
	_ = json.Unmarshal(offJSON, &offered)
	_ = json.Unmarshal(reqJSON, &requested)
	if offered.Materials == nil {
		offered.Materials = map[string]int64{}
	}
	if offered.Recipes == nil {
		offered.Recipes = map[string]float64{}
	}
	if requested.Materials == nil {
		requested.Materials = map[string]int64{}
	}
	if requested.Recipes == nil {
		requested.Recipes = map[string]float64{}
	}

	// Validate Exact Match of Acceptor assets against requested
	if requested.PrimevalStones > 0 {
		if acceptor.PrimevalStones < requested.PrimevalStones {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không đủ Nguyên Thạch yêu cầu (Cần %.3f, hiện có %.3f)", requested.PrimevalStones, acceptor.PrimevalStones)})
			return
		}
	}
	if requested.ImmortalStones > 0 {
		if acceptor.Rank < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ sư dưới Lục Chuyển không thể giao dịch Tiên Nguyên Thạch"})
			return
		}
		if acceptor.ImmortalStones < requested.ImmortalStones {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không đủ Tiên Nguyên Thạch yêu cầu (Cần %.3f, hiện có %.3f)", requested.ImmortalStones, acceptor.ImmortalStones)})
			return
		}
	}
	acceptorMats := getPlayerMaterialsMap(tx, acceptor.ID)
	for matID, qty := range requested.Materials {
		if qty <= 0 {
			continue
		}
		if acceptorMats[matID] < qty {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không đủ nguyên liệu %s (Cần %d, hiện có %d)", matID, qty, acceptorMats[matID])})
			return
		}
	}
	// For Gus in requested: match by Gu ID or instance IDs
	var matchedAcceptorGus []string
	for _, reqGuID := range requested.Gus {
		// Find matching Gu instance in acceptor inventory
		found := false
		for _, g := range acceptor.GuInventory {
			if (g.GuID == reqGuID || g.InstanceID == reqGuID) && g.Status != "DEAD" && !g.Equipped && !g.IsVital {
				// Check not already matched
				alreadyUsed := false
				for _, usedID := range matchedAcceptorGus {
					if usedID == g.InstanceID {
						alreadyUsed = true
						break
					}
				}
				if !alreadyUsed {
					matchedAcceptorGus = append(matchedAcceptorGus, g.InstanceID)
					found = true
					break
				}
			}
		}
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không có Cổ trùng yêu cầu: %s", reqGuID)})
			return
		}
	}
	for _, guInstID := range matchedAcceptorGus {
		var ownerID uuid.UUID
		var isVital, equipped, isLocked bool
		var state string
		if err := tx.QueryRow(`SELECT player_id, is_vital, equipped, is_locked, state FROM player_gu WHERE id = $1 FOR UPDATE`, guInstID).
			Scan(&ownerID, &isVital, &equipped, &isLocked, &state); err != nil || ownerID != acceptor.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng yêu cầu không còn thuộc sở hữu của bạn"})
			return
		}
		if isVital || equipped || isLocked || state == "DEAD" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng yêu cầu đang trang bị, bị khóa, là Bổn Mệnh Cổ hoặc đã chết"})
			return
		}
	}
	for reqRecID := range requested.Recipes {
		recProgress, err := RecipeSvc.Get(tx, acceptor.ID.String(), reqRecID)
		if err != nil || recProgress == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bạn không có Bí phương yêu cầu: %s", reqRecID)})
			return
		}
	}

	// 1. Deduct requested from Acceptor & give to Creator
	if requested.PrimevalStones > 0 {
		amt := pricing.Quantize(requested.PrimevalStones)
		_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2`, amt, acceptor.ID)
		_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, amt, creator.ID)
	}
	if requested.ImmortalStones > 0 {
		amt := pricing.Quantize(requested.ImmortalStones)
		_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, amt, acceptor.ID)
		_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, amt, creator.ID)
	}
	for matID, qty := range requested.Materials {
		if qty <= 0 {
			continue
		}
		_ = deductMaterialTx(tx, acceptor.ID, matID, qty)
		_ = addMaterialTx(tx, creator.ID, matID, qty)
	}
	for _, guInstID := range matchedAcceptorGus {
		var guID string
		var rank int
		if err := tx.QueryRow(`SELECT gu_id, gu_rank FROM player_gu WHERE id = $1`, guInstID).Scan(&guID, &rank); err == nil && rank >= 6 {
			_ = db.LogGuHistory(tx, guInstID, guID, rank, "bhth_trade_offer", &acceptor.ID, &creator.ID,
				fmt.Sprintf("Trao đổi Bảo Hoàng Thiên từ %s sang %s", acceptor.Username, creator.Username), nil)
		}
		result, err := tx.Exec(`UPDATE player_gu SET player_id = $1, equipped = false WHERE id = $2 AND player_id = $3`, creator.ID, guInstID, acceptor.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi chuyển giao Cổ trùng yêu cầu"})
			return
		}
		if affected, err := result.RowsAffected(); err != nil || affected != 1 {
			c.JSON(http.StatusConflict, gin.H{"error": "Cổ trùng yêu cầu đã thay đổi, vui lòng thử lại"})
			return
		}
	}
	for reqRecID := range requested.Recipes {
		recProgress, _ := RecipeSvc.Get(tx, acceptor.ID.String(), reqRecID)
		rate := 0.0
		if recProgress != nil {
			rate = recProgress.CompletionRate
		}
		_ = transferRecipe(tx, acceptor.ID.String(), creator.ID.String(), reqRecID, rate)
	}

	// 2. Deliver offered from BHTH lock to Acceptor
	if offered.PrimevalStones > 0 {
		amt := pricing.Quantize(offered.PrimevalStones)
		_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, amt, acceptor.ID)
	}
	if offered.ImmortalStones > 0 {
		amt := pricing.Quantize(offered.ImmortalStones)
		_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, amt, acceptor.ID)
	}
	for matID, qty := range offered.Materials {
		if qty <= 0 {
			continue
		}
		_ = addMaterialTx(tx, acceptor.ID, matID, qty)
	}
	for _, guInstID := range offered.Gus {
		var guID string
		var rank int
		if err := tx.QueryRow(`SELECT gu_id, gu_rank FROM player_gu WHERE id = $1`, guInstID).Scan(&guID, &rank); err == nil && rank >= 6 {
			_ = db.LogGuHistory(tx, guInstID, guID, rank, "bhth_trade_offer", &creator.ID, &acceptor.ID,
				fmt.Sprintf("Trao đổi Bảo Hoàng Thiên từ %s sang %s", creator.Username, acceptor.Username), nil)
		}
		_, _ = tx.Exec(`UPDATE player_gu SET player_id = $1, equipped = false WHERE id = $2`, acceptor.ID, guInstID)
	}
	for offRecID, offRate := range offered.Recipes {
		finalRate := offRate
		if existing, err := RecipeSvc.Get(tx, acceptor.ID.String(), offRecID); err == nil && existing != nil {
			if existing.CompletionRate > finalRate {
				finalRate = existing.CompletionRate
			}
		}
		if finalRate > 100.0 {
			finalRate = 100.0
		}
		_ = RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       acceptor.ID.String(),
			RecipeID:       offRecID,
			CompletionRate: finalRate,
			IsUnlocked:     true,
		})
	}

	// Mark completed and record acceptor as player_id
	_, err = tx.Exec(`UPDATE trade_offers SET status = 'COMPLETED', player_id = $1 WHERE id = $2`, acceptor.ID, offerUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái đơn trao đổi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &acceptor.ID, Action: "marketplace.accept_trade_offer",
		EntityType: "trade_offer", EntityID: offerUUID.String(),
		RequestSummary: fmt.Sprintf("Chấp nhận đơn trao đổi của %s", creatorID.String()),
		IPAddress:      c.ClientIP(), Success: true,
	})

	// Notify the trade offer creator that their offer was accepted
	notify.Send(db.DB, creatorID, "trade_offer_accepted", "Đơn Trao Đổi Đã Khớp",
		fmt.Sprintf("Đạo hữu %s đã chấp nhận đơn trao đổi auto của bạn. Giao dịch hoàn tất!", acceptor.Username))

	updatedAcceptor := acceptor
	if tx2, err2 := db.DB.Begin(); err2 == nil {
		updatedAcceptor, _ = evaluateManaTx(tx2, acceptor.ID)
		_ = tx2.Commit()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Trao đổi tự động thành công!",
		"player":  updatedAcceptor,
	})
}

// CancelTradeOffer cancels an active trade offer and restores locked offered assets to creator
func CancelTradeOffer(c *gin.Context) {
	var req CancelTradeOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	offerUUID, err := uuid.Parse(req.OfferID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OfferID không hợp lệ"})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	creator, err := evaluateManaTx(tx, authID)
	if err != nil || gamefx.IsDeadState(&creator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể hủy đơn trao đổi!"})
		return
	}

	var creatorID uuid.UUID
	var offJSON []byte
	var status string

	err = tx.QueryRow(`
		SELECT creator_id, offered_data, status
		FROM trade_offers WHERE id = $1 FOR UPDATE`, offerUUID,
	).Scan(&creatorID, &offJSON, &status)

	if err != nil || creatorID != creator.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn trao đổi không tồn tại hoặc bạn không có quyền hủy"})
		return
	}
	if status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đơn trao đổi không còn hoạt động"})
		return
	}

	var offered models.TradeOfferData
	_ = json.Unmarshal(offJSON, &offered)

	if err := restoreTradeOfferItems(tx, creator.ID, offered); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn trả vật phẩm"})
		return
	}

	_, err = tx.Exec(`UPDATE trade_offers SET status = 'CANCELLED' WHERE id = $1`, offerUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật đơn trao đổi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit error"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &creator.ID, Action: "marketplace.cancel_trade_offer",
		EntityType: "trade_offer", EntityID: offerUUID.String(),
		RequestSummary: "Hủy đơn trao đổi tự động",
		IPAddress:      c.ClientIP(), Success: true,
	})

	tx2, _ := db.DB.Begin()
	updatedCreator, _ := evaluateManaTx(tx2, creator.ID)
	_ = tx2.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Hủy đơn trao đổi thành công! Đã hoàn lại toàn bộ vật phẩm khóa.",
		"player":  updatedCreator,
	})
}

// restoreTradeOfferItems restores locked offer items back to creator
func restoreTradeOfferItems(tx *sql.Tx, creatorID uuid.UUID, offered models.TradeOfferData) error {
	if offered.PrimevalStones > 0 {
		if _, err := tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, pricing.Quantize(offered.PrimevalStones), creatorID); err != nil {
			return err
		}
	}
	if offered.ImmortalStones > 0 {
		if _, err := tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, pricing.Quantize(offered.ImmortalStones), creatorID); err != nil {
			return err
		}
	}
	for matID, qty := range offered.Materials {
		if qty <= 0 {
			continue
		}
		if err := addMaterialTx(tx, creatorID, matID, qty); err != nil {
			return err
		}
	}
	for _, guInstID := range offered.Gus {
		if _, err := tx.Exec(`UPDATE player_gu SET player_id = $1 WHERE id = $2`, creatorID, guInstID); err != nil {
			return err
		}
	}
	for recID, recRate := range offered.Recipes {
		_ = RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       creatorID.String(),
			RecipeID:       recID,
			CompletionRate: recRate,
			IsUnlocked:     true,
		})
	}
	return nil
}
