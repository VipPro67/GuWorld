package handlers

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
	hasConcealment := models.InventoryHasGuMechanic(player.GuInventory, "DISGUISE", true)
	hasAttitude := models.InventoryHasGuMechanic(player.GuInventory, "PERFECT_DISGUISE", true)
	hasFormChange := models.InventoryHasGuMechanic(player.GuInventory, "CHANGE_FORM_DISGUISE", true)

	if req.IsDisguised {
		if !hasConcealment {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không trang bị cổ trùng ngụy trang nào hoạt động! Cần trang bị các cổ trùng như Ẩn Lân Cổ, Biến Hóa Cổ, Thái Độ Cổ..."})
			return
		}

		if req.FakeRank != player.Rank {
			if !hasAttitude {
				// Standard stealth only allows hiding/lowering rank ("giả yếu")
				if req.FakeRank > player.Rank || req.FakeRank < 1 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng hiện tại chỉ hỗ trợ giấu khí tức làm yếu đi, không thể giả làm cường giả mạnh hơn tu vi bản thân! Chỉ có Thái Độ Cổ mới đạt tới mức ngụy trang vô ngại."})
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
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cần trang bị Biến Hóa Cổ hoặc Thái Độ Cổ để tạo dựng Bản Mệnh Cổ giả!"})
				return
			}
			// Optional: verify they own this Gu or it is a valid Gu ID in registry
			found := false
			for _, gu := range player.GuInventory {
				if gu.GuID == req.FakeVitalGu {
					found = true
					break
				}
			}
			if req.FakeVitalGu != "GU_SHADOW" && !found {
				// Allow standard fakes or anything they actually own
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không sở hữu cổ trùng " + req.FakeVitalGu + " để thiết lập ngụy trang!"})
				return
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
		player.Username = username
	}

	player.Bio = req.Bio
	player.FakeRank = req.FakeRank
	player.FakeSubRank = req.FakeSubRank
	player.FakeVitalGu = req.FakeVitalGu
	player.IsDisguised = req.IsDisguised

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
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6, password_hash = $7 WHERE id = $8`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, string(hash), player.ID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
			return
		}
	} else {
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6 WHERE id = $7`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, player.ID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
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

	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": "Cập nhật hồ sơ thành công!"})
}
