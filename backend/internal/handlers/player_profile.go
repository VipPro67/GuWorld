package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type UpdateProfileSettingsRequest struct {
	Username    string `json:"username"`
	Bio         string `json:"bio"`
	NewPassword string `json:"new_password"`
	FakeRank    int    `json:"fake_rank"`
	FakeSubRank int    `json:"fake_sub_rank"`
	FakeVitalGu string `json:"fake_vital_gu"`
	IsDisguised bool   `json:"is_disguised"`
}

// GetRenameCost returns the cost type ("free", "primeval", "immortal") and the cost amount for a given rank.
func GetRenameCost(rank int) (string, float64) {
	return gamefx.RenameCost(rank)
}

// GetImmortalPenaltyRates returns the loss rates for dao marks, path masteries, aperture HP, and stress gain for a given immortal rank (6-9).
func GetImmortalPenaltyRates(rank int) (daoLossRate float64, masteryLossRate float64, hpLossRate float64, stressGain float64) {
	return gamefx.ImmortalPenaltyRates(rank)
}

// ApplyImmortalRenamePenalty calculates the reduced dao marks, path masteries, and aperture damage without reducing sub-rank.
func ApplyImmortalRenamePenalty(rank int, daoMarks models.DaoMarksMap, pathMasteries models.PathMasteriesMap, apertureHP float64) (models.DaoMarksMap, models.PathMasteriesMap, float64, float64) {
	daoLossRate, masteryLossRate, hpLossRate, stressGain := GetImmortalPenaltyRates(rank)

	newDao := make(models.DaoMarksMap)
	daoRetain := 1.0 - daoLossRate
	for k, v := range daoMarks {
		newDao[k] = int64(float64(v) * daoRetain)
	}

	newMasteries := make(models.PathMasteriesMap)
	masteryRetain := 1.0 - masteryLossRate
	for k, v := range pathMasteries {
		newMasteries[k] = int(float64(v) * masteryRetain)
	}

	newApertureHP := apertureHP * (1.0 - hpLossRate)

	return newDao, newMasteries, newApertureHP, stressGain
}

// UpdateProfileSettings updates the current user's profile settings (email, bio, password, fake profile)
func UpdateProfileSettings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateProfileSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// Validate Disguise requirements if player wants to activate or configure disguise
	hasConcealment := models.InventoryHasGuCapability(player.GuInventory, "DISGUISE", true)
	hasAttitude := models.InventoryHasGuCapability(player.GuInventory, "PERFECT_DISGUISE", true)
	hasFormChange := models.InventoryHasGuCapability(player.GuInventory, "CHANGE_FORM_DISGUISE", true)

	isDisguised := req.IsDisguised && hasConcealment
	if isDisguised {
		if req.FakeRank != player.Rank {
			if !hasAttitude {
				// Standard stealth only allows hiding/lowering rank ("giả yếu")
				if req.FakeRank > player.Rank || req.FakeRank < 1 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng hiện tại chỉ hỗ trợ giấu khí tức làm yếu đi, không thể giả làm cường giả mạnh hơn tu vi bản thân! Cần cổ trùng có hiệu ứng ngụy trang hoàn mỹ (PERFECT_DISGUISE) để đạt tới mức ngụy trang vô ngại."})
					return
				}
			} else {
				if req.FakeRank < 1 || req.FakeRank > 9 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Tu vi giả chỉ có thể từ Nhất Chuyển (1) đến Cửu Chuyển (9)!"})
					return
				}
			}
		}

		if req.FakeVitalGu != "" {
			if !hasAttitude && !hasFormChange {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cần trang bị cổ trùng có hiệu ứng biến hóa hình thái hoặc ngụy trang hoàn mỹ để tạo dựng Bản Mệnh Cổ giả!"})
				return
			}
			// Verify it is a valid Gu ID in registry or player inventory
			found := false
			for _, gu := range player.GuInventory {
				if gu.GuID == req.FakeVitalGu {
					found = true
					break
				}
			}
			if !found {
				if _, exists := models.GetGuAny(req.FakeVitalGu); !exists {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng ngụy tạo " + req.FakeVitalGu + " không tồn tại trong điển tịch Cổ Trùng!"})
					return
				}
			}
		}
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu không được để trống!"})
		return
	}

	if len(username) < 3 || len(username) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu phải từ 3 đến 30 ký tự!"})
		return
	}

	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) && !unicode.IsDigit(r) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu chỉ được chứa chữ cái, số và khoảng trắng, không chứa ký tự đặc biệt!"})
			return
		}
	}

	oldUsername := player.Username
	renameSuccessMsg := "Cập nhật hồ sơ thành công!"

	if username != player.Username {
		var exists bool
		err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM players WHERE username = $1 AND id <> $2)`, username, player.ID).Scan(&exists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu này đã được sử dụng bởi cổ sư khác!"})
			return
		}

		costType, cost := GetRenameCost(player.Rank)
		if costType == "free" {
			// Rank 1-2: Miễn phí (Tự do đổi tên)
			player.Username = username
		} else if costType == "primeval" {
			// Rank 3-5: Mất phí Nguyên Thạch
			if player.PrimevalStones < cost {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Không đủ Nguyên Thạch! Đổi đạo hiệu ở cấp %d Chuyển cần %.0f Nguyên Thạch.", player.Rank, cost),
				})
				return
			}
			player.PrimevalStones -= cost
			player.Username = username
			renameSuccessMsg = fmt.Sprintf("Đổi đạo hiệu thành công! Đã tiêu hao %.0f Nguyên Thạch.", cost)
		} else if costType == "immortal" {
			// Rank 6-9: Cổ Tiên nghịch thiên cải mệnh, trả giá cực lớn
			if player.ImmortalStones < cost {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch! Cổ Tiên %d Chuyển đổi đạo hiệu cần tối thiểu %.0f Tiên Nguyên Thạch để áp chế thiên cơ.", player.Rank, cost),
				})
				return
			}
			player.ImmortalStones -= cost

			var curApertureHP float64
			var daoJSON []byte
			_ = tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb), COALESCE(aperture_hp, 10000) FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, player.ID).Scan(&daoJSON, &curApertureHP)

			currentDaoMap := make(models.DaoMarksMap)
			if len(daoJSON) > 0 {
				_ = json.Unmarshal(daoJSON, &currentDaoMap)
			}

			daoLossRate, _, _, _ := GetImmortalPenaltyRates(player.Rank)
			newDao, newMasteries, newApertureHP, stressGain := ApplyImmortalRenamePenalty(player.Rank, currentDaoMap, player.PathMasteries, curApertureHP)

			// 1. Cập nhật Đạo Ngân
			player.DaoMarks = newDao
			newDaoJSON, err := json.Marshal(newDao)
			if err == nil {
				_, _ = tx.Exec(`UPDATE immortal_apertures SET dao_marks = $1, aperture_hp = $2 WHERE player_id = $3`, newDaoJSON, newApertureHP, player.ID)
			}
			// 2. Cập nhật Cảnh Giới Lưu Phái
			player.PathMasteries = newMasteries

			// 3. Cập nhật Áp Lực Tiên Khiếu & Tiến Độ (không tụt tiểu cảnh giới)
			player.ApertureStress += stressGain
			player.WallProgress = 0

			player.Username = username
			renameSuccessMsg = fmt.Sprintf("Đổi đạo hiệu thành công! Chịu thiên cơ phản phệ: tiêu hao %.0f Tiên Nguyên Thạch, tụt %.0f%% Đạo Ngân & Cảnh Giới Lưu Phái, tổn hại Tiên Khiếu.", cost, daoLossRate*100)

			// 4. Phát thông báo Thế Giới
			_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("⚡ THIÊN ĐẠO BIẾN ĐỘNG: Cổ Tiên %s nghịch thiên cải mệnh, đổi đạo hiệu thành %s, chịu đại đạo phản phệ tổn thất Đạo Ngân và Tu Vi!", oldUsername, username))
		}
	}

	player.Bio = req.Bio
	player.FakeRank = req.FakeRank
	player.FakeSubRank = req.FakeSubRank
	player.FakeVitalGu = req.FakeVitalGu
	player.IsDisguised = isDisguised

	if req.NewPassword != "" {
		if len(req.NewPassword) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mật khẩu mới phải có ít nhất 6 ký tự"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi mã hóa mật khẩu"})
			return
		}
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6, password_hash = $7, primeval_stones = $8, immortal_stones = $9, sub_rank = $10, wall_progress = $11, aperture_stress = $12, path_masteries = $13 WHERE id = $14`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, string(hash), player.PrimevalStones, player.ImmortalStones, player.SubRank, player.WallProgress, player.ApertureStress, player.PathMasteries, player.ID)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
				return
			}
			logger.S.Errorf("Failed to update profile settings for player %s: %v", player.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu cài đặt hồ sơ: " + err.Error()})
			return
		}
	} else {
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6, primeval_stones = $7, immortal_stones = $8, sub_rank = $9, wall_progress = $10, aperture_stress = $11, path_masteries = $12 WHERE id = $13`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, player.PrimevalStones, player.ImmortalStones, player.SubRank, player.WallProgress, player.ApertureStress, player.PathMasteries, player.ID)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
				return
			}
			logger.S.Errorf("Failed to update profile settings for player %s: %v", player.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu cài đặt hồ sơ: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile settings"})
		return
	}

	// Update token cookie to keep session claim synchronized with the new username
	var sessionID string
	err = tx.QueryRow(`SELECT COALESCE(active_session_id, '') FROM players WHERE id = $1`, player.ID).Scan(&sessionID)
	if err != nil {
		sessionID = ""
	}
	newToken, err := auth.GenerateToken(player.ID, player.Username, sessionID)
	if err == nil {
		setAuthCookie(c, newToken)
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": renameSuccessMsg})
}
