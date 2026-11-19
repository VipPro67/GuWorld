package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type equippedGuHouseInfo struct {
	id         string
	name       string
	rank       int
	durability float64
	auxCount   int
}

// getActiveEquippedGuHouse loads the currently equipped and functioning Immortal Gu House for a player.
func getActiveEquippedGuHouse(tx *sql.Tx, playerID uuid.UUID) (*equippedGuHouseInfo, error) {
	if playerID == uuid.Nil {
		return nil, nil
	}
	type rowQuerier interface {
		QueryRow(string, ...interface{}) *sql.Row
	}
	var q rowQuerier
	if tx != nil {
		q = tx
	} else if db.DB != nil {
		q = db.DB
	} else {
		return nil, nil
	}

	var h equippedGuHouseInfo
	err := q.QueryRow(`
		SELECT id::text, name, rank, current_durability, aux_gu_count
		FROM player_gu_houses
		WHERE player_id = $1 AND is_equipped = true AND current_durability > 0
		LIMIT 1
	`, playerID).Scan(&h.id, &h.name, &h.rank, &h.durability, &h.auxCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &h, nil
}

func playerEffectiveMaxHP(p *models.Player) float64 {
	if p.BaseHP > 0 {
		return gamefx.EffectiveMaxHP(p)
	}
	return math.Max(100, p.HP)
}

// checkLayerTransitionGuard validates whether the player has sufficient cultivation realm,
// an immortal body, or an equipped active Immortal Gu House to cross vertical layers.
func checkLayerTransitionGuard(ctx context.Context, tx *sql.Tx, player *models.Player, fromRegion, toRegion string) error {
	fromMap := worldMapIDForRegion(fromRegion)
	toMap := worldMapIDForRegion(toRegion)
	if fromMap == toMap {
		return nil
	}

	house, _ := getActiveEquippedGuHouse(tx, player.ID)
	maxHP := playerEffectiveMaxHP(player)

	// Case 1: Ascending through Heavenly Wind Qi Wall (Khí Cương Phong Bích) -> White Heaven
	if toMap == "bai_thien" && fromMap != "hac_thien" {
		if player.Rank < 6 {
			// Mortals cannot enter White Heaven - Only Gu Immortals (Rank 6+) with Immortal Body can ascend.
			dmg := maxHP * 0.90
			if dmg < 10 {
				dmg = 10
			}
			player.HP -= dmg
			if player.HP < 1 {
				player.HP = 1
			}
			player.Statuses.Add("severely_wounded", "khi_cuong_phong_bich", 30*time.Minute, 0)
			if db.DB != nil && player.ID != uuid.Nil {
				_, _ = db.DB.ExecContext(ctx, "UPDATE players SET hp=$1, statuses=$2 WHERE id=$3", player.HP, player.Statuses, player.ID)
			}
			return movementDenied(http.StatusForbidden, "🌪️ NGUY CẤP! Khí Cương Phong Bích cuồng bạo xé rách phàm thân! Chỉ có Cổ Tiên (Lục Chuyển trở lên) đúc thành Tiên Thể mới có thể phi thăng Thái Cổ Bạch Thiên!")
		}
	}

	// Case 2: Crossing Astral Vortex / Spatial Shear -> Black Heaven
	if toMap == "hac_thien" {
		if player.Rank < 8 {
			dmg := maxHP * 0.70
			if dmg < 10 {
				dmg = 10
			}
			player.HP -= dmg
			if player.HP < 1 {
				player.HP = 1
			}
			player.Statuses.Add("severely_wounded", "cuc_diem_khong_gian", 20*time.Minute, 0)
			if db.DB != nil && player.ID != uuid.Nil {
				_, _ = db.DB.ExecContext(ctx, "UPDATE players SET hp=$1, statuses=$2 WHERE id=$3", player.HP, player.Statuses, player.ID)
			}
			return movementDenied(http.StatusForbidden, "🌌 CỰC KỲ NGUY HIỂM! Cực Điểm Không Gian ngăn cách Hắc Thiên ẩn chứa Hư Không Liệt Phùng và Thái Cổ Hồn Phong. Chỉ có Bát Chuyển Cổ Tiên trở lên mới có đủ uy áp và thể phách để bước vào Thái Cổ Hắc Thiên!")
		}
	}

	// Case 3: Descending into the Abyss (Vạn Trượng Địa Uyên)
	if toMap == "dia_uyen" {
		if player.Rank < 5 && house == nil {
			dmg := maxHP * 0.50
			if dmg < 10 {
				dmg = 10
			}
			player.HP -= dmg
			if player.HP < 1 {
				player.HP = 1
			}
			player.Statuses.Add("dia_uyen_miasma", "chướng khí địa câu", 15*time.Minute, -0.20)
			if db.DB != nil && player.ID != uuid.Nil {
				_, _ = db.DB.ExecContext(ctx, "UPDATE players SET hp=$1, statuses=$2 WHERE id=$3", player.HP, player.Statuses, player.ID)
			}
			return movementDenied(http.StatusForbidden, "☠️ CHƯỚNG KHÍ ĐỊA CÂU! Áp lực vạn trượng lòng đất và chướng khí độc hại quá nồng nặc. Cần tu vi tối thiểu Bậc 5 (Ngũ Chuyển) hoặc Tiên Cổ Ốc để tiến sâu vào Địa Uyên an toàn!")
		}
	}

	return nil
}

// applyLayerTransitionEffects applies the gameplay settlement (durability wear, essence consumption,
// spatial suppression debuffs) and returns a narrative message describing the transition.
func applyLayerTransitionEffects(ctx context.Context, tx *sql.Tx, player *models.Player, fromMap, toMap string) (string, error) {
	if fromMap == toMap {
		return "", nil
	}

	house, _ := getActiveEquippedGuHouse(tx, player.ID)
	maxHP := playerEffectiveMaxHP(player)

	// Ascending: Surface -> White Heaven
	if toMap == "bai_thien" && fromMap != "hac_thien" {
		if house != nil {
			durLoss := 20.0
			newDur := math.Max(0, house.durability-durLoss)
			guLoss := int(float64(house.auxCount) * 0.10)
			newAux := int(math.Max(0, float64(house.auxCount-guLoss)))
			if tx != nil {
				_, _ = tx.ExecContext(ctx, "UPDATE player_gu_houses SET current_durability=$1, aux_gu_count=$2, updated_at=NOW() WHERE id=$3", newDur, newAux, house.id)
			}
			return fmt.Sprintf("🏰 Tiên Cổ Ốc [%s] cản phá phong nhận của Khí Cương Phong Bích (tiêu hao 20 độ bền), hộ tống đạo hữu phi thăng Thái Cổ Bạch Thiên an toàn!", house.name), nil
		}
		// Gu Immortal (Rank 6+)
		if player.CurrentMana >= 50 {
			player.CurrentMana -= 50
		} else {
			abrasion := maxHP * 0.05
			player.HP = math.Max(1, player.HP-abrasion)
		}
		return "🌪️ Đạo hữu vận dụng Tiên Thể phá vỡ Khí Cương Phong Bích, thuận lợi đăng lâm Thái Cổ Bạch Thiên!", nil
	}

	// Ascending: White Heaven -> Black Heaven
	if toMap == "hac_thien" {
		if player.Rank >= 8 {
			if house != nil {
				durLoss := 10.0
				newDur := math.Max(0, house.durability-durLoss)
				if tx != nil {
					_, _ = tx.ExecContext(ctx, "UPDATE player_gu_houses SET current_durability=$1, updated_at=NOW() WHERE id=$2", newDur, house.id)
				}
				return fmt.Sprintf("🌌 Tiên Cổ Ốc [%s] càn quét Cực Điểm Không Gian, hộ tống Bát Chuyển Cổ Tiên tiến vào Thái Cổ Hắc Thiên vô tận!", house.name), nil
			}
			return "🌌 Đạo hữu phóng thích uy áp Bát Chuyển, phá toái Cực Điểm Không Gian tiến vào Thái Cổ Hắc Thiên vô tận!", nil
		}
		return "🌌 Đạo hữu tiến vào Thái Cổ Hắc Thiên vô tận!", nil
	}

	// Descending: Surface -> Abyss
	if toMap == "dia_uyen" {
		return "🕳️ Đạo hữu lặn sâu qua kẽ nứt Địa Câu, tiến vào Vạn Trượng Địa Uyên âm u lạnh lẽo!", nil
	}

	// Returning: Abyss -> Surface
	if fromMap == "dia_uyen" && toMap == "world_map" {
		return "☀️ Đạo hữu men theo mạch ngầm Địa Câu trở về mặt đất Ngũ Vực!", nil
	}

	// Descent: Black Heaven -> White Heaven
	if fromMap == "hac_thien" && toMap == "bai_thien" {
		return "✨ Đạo hữu từ bóng đêm Hắc Thiên rơi xuống biển mây Thái Cổ Bạch Thiên!", nil
	}

	// Skyfall: White Heaven -> Five Regions (Surface)
	if fromMap == "bai_thien" && toMap == "world_map" {
		if player.Rank < 6 && house == nil {
			fallDmg := maxHP * 0.40
			player.HP = math.Max(1, player.HP-fallDmg)
			return "🪂 Đạo hữu rơi tự do từ Bạch Thiên hạ giới, không có Tiên Thể giảm chấn nên chịu chấn thương va chạm khi chạm đất Ngũ Vực!", nil
		}
		return "🪂 Đạo hữu từ Thái Cổ Bạch Thiên phi thân hạ giới, xuyên qua tầng mây đáp xuống Ngũ Vực an toàn!", nil
	}

	return "", nil
}

// SkyfallDescent allows a character in White Heaven, Black Heaven, or Abyss
// to perform a free-fall descent piercing through cloud/earth layers back to
// the corresponding geographical coordinate on the Five Regions (or lower layer).
func SkyfallDescent(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if rejectRealtimeOwned(c, authID) {
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kết nối cơ sở dữ liệu", err))
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải thông tin nhân vật", err))
		return
	}

	currentMap := worldMapIDForRegion(player.CurrentRegion)
	if currentMap == "world_map" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang ở mặt đất Ngũ Vực, không thể thực hiện Rơi Tự Do (Skyfall)."})
		return
	}

	targetMap := "world_map"
	if currentMap == "hac_thien" {
		targetMap = "bai_thien"
	}

	targetRegion := DeriveRegionAtOnMap(player.X, player.Y, targetMap)
	if targetRegion == "" || targetRegion == HubRegionID {
		if targetMap == "bai_thien" {
			targetRegion = "dai_vuc_bach_thien"
		} else {
			targetRegion = "dai_vuc_trung_chau"
		}
	}

	transition, errMove := ExecuteSpatialTransition(c.Request.Context(), tx, &player, movementSpec{
		kind:         movementPortal,
		x:            player.X,
		y:            player.Y,
		hasTarget:    true,
		targetRegion: targetRegion,
	}, publishPartyRealtimeEvent)
	if errMove != nil {
		if respondMovementError(c, errMove) {
			return
		}
		httperr.Respond(c, httperr.Internal(errMove.Error(), errMove))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xác nhận rơi tự do", err))
		return
	}

	targetName := targetRegion
	if z := ZoneByID(targetRegion); z != nil && z.Name != "" {
		targetName = z.Name
	}

	logMsg := fmt.Sprintf("🪂 Đạo hữu thực hiện Rơi Tự Do (Skyfall), xuyên qua tầng giới đáp xuống %s (%d, %d).", targetName, player.X, player.Y)
	if transition.DebuffMessage != "" {
		logMsg += " " + transition.DebuffMessage
	}

	if transition.PublishPostCommit != nil {
		transition.PublishPostCommit()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": logMsg,
		"player": gin.H{
			"current_region": player.CurrentRegion,
			"world_map_id":   worldMapIDForRegion(player.CurrentRegion),
			"x":              player.X,
			"y":              player.Y,
			"hp":             player.HP,
			"current_mana":   player.CurrentMana,
		},
		"synced_party_members": transition.SyncedMembers,
	})
}
