package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

func SettleExpiredAuctions(c *gin.Context) {
	if err := settleExpiredAuctions(); err != nil {
		logger.S.Errorf("Failed to settle expired auctions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to settle expired auctions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired auctions settled"})
}

func settleExpiredAuctions() error {
	// Cheap guard: bail before any heavy row fetch / JSON unmarshal when no
	// auction is actually expired. The 30s ticker would otherwise repeat this
	// query 2880×/day per server even with zero activity.
	var hasExpired bool
	if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM auction WHERE status='ACTIVE' AND end_time <= NOW())`).Scan(&hasExpired); err != nil {
		return err
	}
	if !hasExpired {
		return nil
	}

	rows, err := db.DB.Query(`SELECT id,seller_id,highest_bidder_id,item_type,item_data,quantity,current_price,currency,listing_type,duration_hours FROM auction WHERE status='ACTIVE' AND end_time <= NOW() ORDER BY end_time LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var sellerID uuid.UUID
		var highestBidderID sql.NullString
		var itemType string
		var rawItemData []byte
		var qty int64
		var durationHours int
		var currentPrice float64
		var currency, listingType string
		if err := rows.Scan(&id, &sellerID, &highestBidderID, &itemType, &rawItemData, &qty, &currentPrice, &currency, &listingType, &durationHours); err != nil {
			logger.S.Errorf("Failed to scan expired auction row: %v", err)
			continue
		}

		var itemData string
		if err := json.Unmarshal(rawItemData, &itemData); err != nil {
			itemData = string(rawItemData)
		}

		if err := settleAuction(id, sellerID, highestBidderID, itemType, itemData, qty, currentPrice, currency, listingType, durationHours); err != nil {
			logger.S.Errorf("Failed to settle auction %s: %v", id, err)
			continue
		}
	}

	return rows.Err()
}

func settleAuction(auctionID string, sellerID uuid.UUID, highestBidderID sql.NullString, itemType, itemData string, qty int64, currentPrice float64, currency string, listingType string, durationHours int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	seller, err := evaluateManaTx(tx, sellerID)
	if err != nil {
		return err
	}
	sellerBefore := append(models.GuInventory(nil), seller.GuInventory...)

	if highestBidderID.Valid {
		buyerID, err := uuid.Parse(highestBidderID.String)
		if err != nil {
			return err
		}

		buyer, err := evaluateManaTx(tx, buyerID)
		if err != nil {
			return err
		}
		buyerBefore := append(models.GuInventory(nil), buyer.GuInventory...)

		// ResourcePoint: rebind the point into the buyer's aperture before
		// any money moves. A business failure means the transfer can never
		// happen — refund the escrowed bid and cancel cleanly.
		if itemType == "ResourcePoint" {
			if err := bindResourcePointToBuyer(tx, &buyer, sellerID, itemData); err != nil {
				var transferErr *resourcePointTransferError
				if !errors.As(err, &transferErr) {
					return err
				}
				return rescindResourcePointAuction(tx, auctionID, buyerID, currentPrice, currency, itemData)
			}
		}

		// Time-based commission fee
		commission := math.Round(currentPrice*getCommissionPct(listingType, durationHours)*1000) / 1000
		payout := currentPrice - commission

		if currency == "primeval" {
			seller.PrimevalStones += payout
		} else {
			seller.ImmortalStones += payout
		}

		if itemType != "ResourcePoint" {
			if err := addAuctionItemToPlayer(tx, &buyer, itemType, itemData, qty); err != nil {
				return err
			}
		}

		if err := updatePlayerInventory(tx, buyer, buyerBefore); err != nil {
			return err
		}
		if err := updatePlayerInventory(tx, seller, sellerBefore); err != nil {
			return err
		}

		if _, err := tx.Exec(`UPDATE auction SET status='COMPLETED' WHERE id=$1`, auctionID); err != nil {
			return err
		}

		itemName := GetItemName(itemType, []byte(itemData))
		auditlog.Log(tx, auditlog.Entry{
			PlayerID: &buyerID, Action: "marketplace.settle_auction_win", EntityType: "auction",
			EntityID:       auctionID,
			RequestSummary: fmt.Sprintf("người thắng đấu giá=%s (%s) | người bán=%s (%s) | vật phẩm=%s | sl=%d | giá=%.3f %s | loại=%s", buyer.Username, buyerID, seller.Username, sellerID, itemName, qty, currentPrice, currency, listingType),
			Success:        true,
		})
	} else {
		if itemType == "ResourcePoint" {
			// Điểm tài nguyên chưa bao giờ rời Tiên Khiếu: chỉ gỡ cờ listed.
			if err := unlistResourcePoint(tx, itemData); err != nil {
				return err
			}
		} else {
			if err := addAuctionItemToPlayer(tx, &seller, itemType, itemData, qty); err != nil {
				return err
			}
			if err := updatePlayerInventory(tx, seller, sellerBefore); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`UPDATE auction SET status='CANCELLED' WHERE id=$1`, auctionID); err != nil {
			return err
		}

		itemName := GetItemName(itemType, []byte(itemData))
		auditlog.Log(tx, auditlog.Entry{
			PlayerID: &sellerID, Action: "marketplace.settle_auction_expired", EntityType: "auction",
			EntityID:       auctionID,
			RequestSummary: fmt.Sprintf("đấu giá hết hạn không có người mua | người bán=%s (%s) | vật phẩm=%s | sl=%d | giá khởi điểm=%.3f %s | loại=%s", seller.Username, sellerID, itemName, qty, currentPrice, currency, listingType),
			Success:        true,
		})
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Notifications after successful commit (best-effort).
	currencyName := "Nguyên Thạch"
	if currency == "immortal" {
		currencyName = "Tiên Nguyên Thạch"
	}
	if highestBidderID.Valid {
		bidderUUID, _ := uuid.Parse(highestBidderID.String)
		notify.Send(db.DB, bidderUUID, "auction_won", "Thắng đấu giá",
			fmt.Sprintf("Bạn đã thắng đấu giá với giá %.3f %s!", currentPrice, currencyName))
		notify.Send(db.DB, sellerID, "auction_sold", "Đấu giá hoàn tất",
			fmt.Sprintf("Mặt hàng đã được bán với giá %.3f %s.", currentPrice, currencyName))
	} else {
		notify.Send(db.DB, sellerID, "auction_expired", "Đấu giá hết hạn",
			"Phiên đấu giá đã hết hạn và không có người đặt giá.")
	}
	return nil
}

func addAuctionItemToPlayer(tx *sql.Tx, player *models.Player, itemType, itemData string, qty int64) error {
	if itemType == "ResourcePoint" {
		// Điểm tài nguyên phải đi qua bindResourcePointToBuyer: nhét lung tung
		// sẽ thành material rác trong túi đồ.
		return fmt.Errorf("ResourcePoint transfers must go through bindResourcePointToBuyer")
	}
	if itemType == "Gu" {
		var gus []models.GuInstance
		if err := json.Unmarshal([]byte(itemData), &gus); err == nil {
			if qty > int64(len(gus)) {
				qty = int64(len(gus))
			}
			for i := int64(0); i < qty; i++ {
				gu := gus[i]
				if gu.InstanceID == "" {
					// Tiên Cổ không được regen id: snapshot thiếu id = dữ liệu hỏng.
					if gu.Rank >= 6 {
						return fmt.Errorf("auction Gu %s R%d thiếu instance_id", gu.GuID, gu.Rank)
					}
					gu.InstanceID = uuid.New().String()
				}
				gu.Status = "SLEEPING"
				gu.Equipped = false
				gu.IsVital = false
				player.GuInventory = append(player.GuInventory, gu)

				guName := gu.GuID
				if guSpec, ok := models.GetGu(gu.GuID, gu.Rank); ok && guSpec.Name != "" {
					guName = guSpec.Name
				}
				summary := fmt.Sprintf("Cổ Sư [%s] nhận Cổ trùng [%s] R%d từ Bảo Hoàng Thiên.", player.Username, guName, gu.Rank)
				_ = db.LogGuHistory(tx, gu.InstanceID, gu.GuID, gu.Rank, "MARKET_SOLD", nil, &player.ID, summary, map[string]interface{}{
					"gu_name": guName,
				})
			}
		} else {
			// Tương thích ngược với định dạng cũ (chỉ lưu string guID)
			for i := int64(0); i < qty; i++ {
				g := mapGuFromRaw(itemData)
				player.GuInventory = append(player.GuInventory, g)
			}
		}
		return nil
	}

	if itemType == "Recipe" {
		var recipeInfo models.UnlockedRecipe
		if err := json.Unmarshal([]byte(itemData), &recipeInfo); err == nil {
			existing, err := RecipeSvc.Get(tx, player.ID.String(), recipeInfo.RecipeID)
			if err == nil {
				completionRate := existing.CompletionRate + recipeInfo.CompletionRate
				if completionRate > 100.0 {
					completionRate = 100.0
				}
				RecipeSvc.Upsert(tx, &services.RecipeProgress{
					PlayerID:       player.ID.String(),
					RecipeID:       recipeInfo.RecipeID,
					CompletionRate: completionRate,
					IsUnlocked:     true,
				})
			} else {
				RecipeSvc.Upsert(tx, &services.RecipeProgress{
					PlayerID:       player.ID.String(),
					RecipeID:       recipeInfo.RecipeID,
					CompletionRate: recipeInfo.CompletionRate,
					IsUnlocked:     true,
				})
			}
		}
		return nil
	}

	InventorySvc.AddItem(tx, player.ID.String(), itemData, "materials", qty)
	return nil
}

// resourcePointTransferError marks a business (not infrastructure) failure to
// move a resource point: the point is no longer listed, the seller no longer
// owns it, or the buyer cannot receive it. The settlement loop treats it as
// final (refund + cancel) instead of retrying forever.
type resourcePointTransferError struct{ msg string }

func (e *resourcePointTransferError) Error() string { return e.msg }

// bindResourcePointToBuyer rebinds a listed APERTURE-scope resource point from
// the seller into the buyer's aperture inside the caller's transaction. Canon:
// a staked Mâm Nguyệt or a sold Phong Tuyết Đỉnh changes hands once payment
// settles (Reverend Insanity ch. 1303 & 2153).
func bindResourcePointToBuyer(tx *sql.Tx, buyer *models.Player, sellerID uuid.UUID, itemData string) error {
	var point ResourcePointItemData
	if err := json.Unmarshal([]byte(itemData), &point); err != nil || point.PointID == "" {
		return &resourcePointTransferError{"Dữ liệu điểm tài nguyên không hợp lệ"}
	}
	pointID, err := uuid.Parse(point.PointID)
	if err != nil {
		return &resourcePointTransferError{"Dữ liệu điểm tài nguyên không hợp lệ"}
	}

	// 1. Lock the point row: it must still exist and still hang off a seller's
	// aperture (guards the "seller lost the aperture meanwhile" settle case).
	var apertureID sql.NullString
	err = tx.QueryRow(`
		SELECT aperture_id
		FROM aperture_resource_points
		WHERE id = $1
		FOR UPDATE
	`, pointID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		return &resourcePointTransferError{"Điểm tài nguyên không còn tồn tại"}
	} else if err != nil {
		return fmt.Errorf("lỗi truy vấn điểm tài nguyên: %w", err)
	}
	if !apertureID.Valid {
		return &resourcePointTransferError{"Người bán không còn sở hữu điểm tài nguyên này"}
	}
	var sellerStillOwns bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM immortal_apertures WHERE id = $1 AND player_id = $2)`, apertureID.String, sellerID).Scan(&sellerStillOwns); err != nil {
		return fmt.Errorf("lỗi kiểm tra Tiên Khiếu người bán: %w", err)
	}
	if !sellerStillOwns {
		return &resourcePointTransferError{"Người bán không còn sở hữu điểm tài nguyên này"}
	}

	// 2. The buyer must have an aperture with room for one more point.
	var buyerApertureID string
	var buyerPoints int
	err = tx.QueryRow(`
		SELECT a.id::text, (SELECT COUNT(1) FROM aperture_resource_points WHERE aperture_id = a.id)
		FROM immortal_apertures a
		WHERE a.player_id = $1
		FOR UPDATE OF a
	`, buyer.ID).Scan(&buyerApertureID, &buyerPoints)
	if err == sql.ErrNoRows {
		return &resourcePointTransferError{"Đạo hữu chưa khai mở Tiên Khiếu — không thể nhận điểm tài nguyên"}
	} else if err != nil {
		return fmt.Errorf("lỗi kiểm tra Tiên Khiếu: %w", err)
	}
	if buyerPoints >= aperturePointPointGenCap {
		return &resourcePointTransferError{fmt.Sprintf("Tiên Khiếu đã có tối đa %d điểm tài nguyên.", aperturePointPointGenCap)}
	}

	// 3. Rebind under the listed guard: 0 rows = a concurrent trade won the row.
	res, err := tx.Exec(`
		UPDATE aperture_resource_points
		SET aperture_id = $1, listed = false, updated_at = NOW()
		WHERE id = $2 AND listed = true
	`, buyerApertureID, pointID)
	if err != nil {
		return fmt.Errorf("lỗi chuyển giao điểm tài nguyên: %w", err)
	}
	if rows, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("lỗi chuyển giao điểm tài nguyên: %w", err)
	} else if rows == 0 {
		return &resourcePointTransferError{"Điểm tài nguyên vừa được giao dịch bởi người khác"}
	}
	return nil
}

// rescindResourcePointAuction settles a ResourcePoint auction whose transfer
// failed for good: the bid was fully escrowed at bid time, so refund the
// winner, cancel the auction and strip the stale listed flag. The caller is
// expected to return right away (commit already happened here).
func rescindResourcePointAuction(tx *sql.Tx, auctionID string, bidderID uuid.UUID, currentPrice float64, currency string, itemData string) error {
	refundCol := "primeval_stones"
	if currency == "immortal" {
		refundCol = "immortal_stones"
	}
	if _, err := tx.Exec(fmt.Sprintf(`UPDATE players SET %s = %s + $1 WHERE id = $2`, refundCol, refundCol), currentPrice, bidderID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE auction SET status='CANCELLED' WHERE id=$1`, auctionID); err != nil {
		return err
	}
	if err := unlistResourcePoint(tx, itemData); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &bidderID, Action: "marketplace.settle_auction_rescinded", EntityType: "auction",
		EntityID:       auctionID,
		RequestSummary: fmt.Sprintf("đấu giá điểm tài nguyên không thể chuyển giao | hoàn trả=%.3f %s", currentPrice, currency),
		Success:        true,
	})
	notify.Send(db.DB, bidderID, "auction_rescinded", "Đấu Giá Hủy Giao Dịch",
		fmt.Sprintf("Điểm tài nguyên không thể chuyển giao cho đạo hữu, %.3f %s đã được hoàn trả.", currentPrice, currency))
	return nil
}

// unlistResourcePoint clears the listed flag once a ResourcePoint listing is
// cancelled or expires: the point itself never left anyone's aperture.
func unlistResourcePoint(tx *sql.Tx, itemData string) error {
	var point ResourcePointItemData
	if err := json.Unmarshal([]byte(itemData), &point); err != nil || point.PointID == "" {
		return nil
	}
	if _, err := uuid.Parse(point.PointID); err != nil {
		return nil
	}
	if _, err := tx.Exec(`UPDATE aperture_resource_points SET listed = false, updated_at = NOW() WHERE id = $1 AND listed = true`, point.PointID); err != nil {
		return err
	}
	return nil
}

func updatePlayerInventory(tx *sql.Tx, player models.Player, before models.GuInventory) error {
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE players SET equipped_gus=$1, primeval_stones=$2, immortal_stones=$3, human_qi=$4 WHERE id=$5`, player.EquippedGus, player.PrimevalStones, player.ImmortalStones, player.HumanQi, player.ID)
	return err
}
