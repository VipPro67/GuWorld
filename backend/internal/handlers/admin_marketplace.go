package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// AdminListAuctions returns paginated marketplace auction listings with search & filters.
func AdminListAuctions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	itemType := strings.TrimSpace(c.Query("item_type"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseQuery := `
		FROM auction a
		JOIN players s ON s.id = a.seller_id
		LEFT JOIN players b ON b.id = a.highest_bidder_id
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (s.username ILIKE $%d OR b.username ILIKE $%d OR a.item_type ILIKE $%d OR a.item_data::text ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if status != "" && status != "ALL" {
		baseQuery += fmt.Sprintf(" AND a.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if itemType != "" && itemType != "ALL" {
		baseQuery += fmt.Sprintf(" AND a.item_type = $%d", argIdx)
		args = append(args, itemType)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đếm phiên đấu giá", err))
		return
	}

	selectQuery := fmt.Sprintf(`
		SELECT a.id, a.seller_id, s.username, a.item_type, a.item_data, a.quantity, a.currency,
		       a.start_price, a.current_price, a.highest_bidder_id, b.username, a.end_time,
		       a.status, a.listing_type, a.created_at, a.duration_hours
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), selectQuery, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc danh sách đấu giá", err))
		return
	}
	defer rows.Close()

	list := make([]gin.H, 0)
	for rows.Next() {
		var id, sellerID uuid.UUID
		var highestBidderID uuid.NullUUID
		var sellerName, itemTypeVal, currencyVal, statusVal, listingTypeVal string
		var highestBidderName sql.NullString
		var quantity int64
		var startPrice, currentPrice float64
		var itemDataRaw []byte
		var endTime, createdAt interface{}
		var durationHours int

		if err := rows.Scan(&id, &sellerID, &sellerName, &itemTypeVal, &itemDataRaw, &quantity, &currencyVal,
			&startPrice, &currentPrice, &highestBidderID, &highestBidderName, &endTime,
			&statusVal, &listingTypeVal, &createdAt, &durationHours); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu đấu giá", err))
			return
		}

		var itemData interface{} = map[string]interface{}{}
		if len(itemDataRaw) > 0 {
			_ = json.Unmarshal(itemDataRaw, &itemData)
		}

		var bidderIDStr *string
		if highestBidderID.Valid {
			str := highestBidderID.UUID.String()
			bidderIDStr = &str
		}

		list = append(list, gin.H{
			"id":                  id.String(),
			"seller_id":           sellerID.String(),
			"seller_name":         sellerName,
			"item_type":           itemTypeVal,
			"item_data":           itemData,
			"quantity":            quantity,
			"currency":            currencyVal,
			"start_price":         startPrice,
			"current_price":       currentPrice,
			"highest_bidder_id":   bidderIDStr,
			"highest_bidder_name": highestBidderName.String,
			"end_time":            endTime,
			"status":              statusVal,
			"listing_type":        listingTypeVal,
			"created_at":          createdAt,
			"duration_hours":      durationHours,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminCancelAuction cancels an auction listing and flags it as cancelled.
func AdminCancelAuction(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	auctionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên đấu giá không hợp lệ"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể mở transaction", err))
		return
	}
	defer tx.Rollback()

	var status string
	var sellerID uuid.UUID
	var itemType string
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT status, seller_id, item_type FROM auction WHERE id = $1 FOR UPDATE`, auctionID).Scan(&status, &sellerID, &itemType); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên đấu giá"})
		return
	}

	if status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể hủy phiên đấu giá đang hoạt động (active)"})
		return
	}

	_, err = tx.ExecContext(c.Request.Context(), `UPDATE auction SET status = 'cancelled' WHERE id = $1`, auctionID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể hủy phiên đấu giá", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể commit transaction", err))
		return
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "auction",
		EntityID:       auctionID.String(),
		Action:         "admin.cancel_auction",
		RequestSummary: fmt.Sprintf("seller_id=%s item_type=%s", sellerID.String(), itemType),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy phiên đấu giá thành công"})
}

// AdminListTrades returns direct player-to-player trade records.
func AdminListTrades(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseQuery := `
		FROM trades t
		JOIN players s ON s.id = t.sender_id
		JOIN players r ON r.id = t.receiver_id
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (s.username ILIKE $%d OR r.username ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if status != "" && status != "ALL" {
		baseQuery += fmt.Sprintf(" AND t.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đếm lịch sử giao dịch", err))
		return
	}

	selectQuery := fmt.Sprintf(`
		SELECT t.id, t.sender_id, s.username, t.receiver_id, r.username,
		       t.sender_offer, t.receiver_offer, t.status, t.created_at, t.updated_at
		%s
		ORDER BY t.created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), selectQuery, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc danh sách giao dịch", err))
		return
	}
	defer rows.Close()

	list := make([]gin.H, 0)
	for rows.Next() {
		var id, senderID, receiverID uuid.UUID
		var senderName, receiverName, statusVal string
		var senderOfferRaw, receiverOfferRaw []byte
		var createdAt, updatedAt interface{}

		if err := rows.Scan(&id, &senderID, &senderName, &receiverID, &receiverName, &senderOfferRaw, &receiverOfferRaw, &statusVal, &createdAt, &updatedAt); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu giao dịch", err))
			return
		}

		var senderOffer interface{} = map[string]interface{}{}
		if len(senderOfferRaw) > 0 {
			_ = json.Unmarshal(senderOfferRaw, &senderOffer)
		}
		var receiverOffer interface{} = map[string]interface{}{}
		if len(receiverOfferRaw) > 0 {
			_ = json.Unmarshal(receiverOfferRaw, &receiverOffer)
		}

		list = append(list, gin.H{
			"id":             id.String(),
			"sender_id":      senderID.String(),
			"sender_name":    senderName,
			"receiver_id":    receiverID.String(),
			"receiver_name":  receiverName,
			"sender_offer":   senderOffer,
			"receiver_offer": receiverOffer,
			"status":         statusVal,
			"created_at":     createdAt,
			"updated_at":     updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
