package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/authz"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminUpdatePlayerRequest struct {
	Rank                *int               `json:"rank"`
	SubRank             *int               `json:"sub_rank"`
	AptitudeType        *string            `json:"aptitude_type"`
	AptitudePercent     *float64           `json:"aptitude_percent"`
	AptitudeGrade       *string            `json:"aptitude_grade"`
	CurrentMana         *float64           `json:"current_mana"`
	MaxMana             *float64           `json:"max_mana"`
	HP                  *float64           `json:"hp"`
	BaseHP              *float64           `json:"base_hp"`
	WallProgress        *float64           `json:"wall_progress"`
	HumanQi             *float64           `json:"human_qi"`
	Lifespan            *float64           `json:"lifespan"`
	CurrentRegion       *string            `json:"current_region"`
	X                   *int               `json:"x"`
	Y                   *int               `json:"y"`
	IsAdmin             *bool              `json:"is_admin"`
	SoulHP              *float64           `json:"soul_hp"`
	MaxSoul             *float64           `json:"max_soul"`
	SoulTier            *string            `json:"soul_tier"`
	ExploreCount        *int               `json:"explore_count"`
	ApertureStress      *float64           `json:"aperture_stress"`
	Bio                 *string            `json:"bio"`
	AvatarURL           *string            `json:"avatar_url"`
	Race                *string            `json:"race"`
	Gender              *string            `json:"gender"`
	Constitution        *string            `json:"constitution"`
	Statuses            *models.StatusList `json:"statuses"`
	CurrentThought      *float64           `json:"current_thought"`
	MaxThought          *float64           `json:"max_thought"`
	ThoughtRegenRate    *float64           `json:"thought_regen_rate"`
	TimeSpeedMultiplier *float64           `json:"time_speed_multiplier"`
	NeedsSoulRefine     *bool              `json:"needs_soul_refine"`
	SelectedTitle       *string            `json:"selected_title"`
	SelectedFrame       *string            `json:"selected_frame"`
	FakeRank            *int               `json:"fake_rank"`
	FakeSubRank         *int               `json:"fake_sub_rank"`
	FakeVitalGu         *string            `json:"fake_vital_gu"`
	IsDisguised         *bool              `json:"is_disguised"`
	NeedsOnboarding     *bool              `json:"needs_onboarding"`
	ImmortalJade        *float64           `json:"immortal_jade"`
}

// AdminUpdatePlayer allows admins to edit a wide range of player attributes in one call.
func AdminUpdatePlayer(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminUpdatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	if _, err := loadPlayerForStatus(tx, playerID, true); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Rank != nil {
		rank := *req.Rank
		if rank < 1 {
			rank = 1
		} else if rank > 9 {
			rank = 9
		}
		sets = append(sets, "rank=$"+strconv.Itoa(argIdx))
		args = append(args, rank)
		argIdx++
	}
	if req.SubRank != nil {
		subRank := *req.SubRank
		if subRank < 1 {
			subRank = 1
		} else if subRank > 4 {
			subRank = 4
		}
		sets = append(sets, "sub_rank=$"+strconv.Itoa(argIdx))
		args = append(args, subRank)
		argIdx++
	}
	if req.AptitudeType != nil {
		sets = append(sets, "aptitude_type=$"+strconv.Itoa(argIdx))
		args = append(args, *req.AptitudeType)
		argIdx++
	}
	if req.AptitudePercent != nil {
		pct := *req.AptitudePercent
		if pct < 0 {
			pct = 0
		} else if pct > 100 {
			pct = 100
		}
		sets = append(sets, "aptitude_percent=$"+strconv.Itoa(argIdx))
		args = append(args, pct)
		argIdx++
	}
	if req.AptitudeGrade != nil {
		sets = append(sets, "aptitude_grade=$"+strconv.Itoa(argIdx))
		args = append(args, *req.AptitudeGrade)
		argIdx++
	}
	if req.CurrentMana != nil {
		val := math.Max(0, *req.CurrentMana)
		sets = append(sets, "current_mana=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.MaxMana != nil {
		val := math.Max(0, *req.MaxMana)
		sets = append(sets, "max_mana=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.HP != nil {
		val := math.Max(0, *req.HP)
		sets = append(sets, "hp=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.BaseHP != nil {
		val := math.Max(0, *req.BaseHP)
		sets = append(sets, "base_hp=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.WallProgress != nil {
		val := math.Min(1.0, math.Max(0, *req.WallProgress))
		sets = append(sets, "wall_progress=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.HumanQi != nil {
		val := math.Max(0, *req.HumanQi)
		sets = append(sets, "human_qi=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.Lifespan != nil {
		val := math.Max(0, *req.Lifespan)
		sets = append(sets, "lifespan=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.CurrentRegion != nil {
		sets = append(sets, "current_region=$"+strconv.Itoa(argIdx))
		args = append(args, *req.CurrentRegion)
		argIdx++
	}
	if req.X != nil {
		sets = append(sets, "x=$"+strconv.Itoa(argIdx))
		args = append(args, *req.X)
		argIdx++
	}
	if req.Y != nil {
		sets = append(sets, "y=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Y)
		argIdx++
	}
	if req.IsAdmin != nil {
		sets = append(sets, "is_admin=$"+strconv.Itoa(argIdx))
		args = append(args, *req.IsAdmin)
		argIdx++
	}
	if req.SoulHP != nil {
		val := math.Max(0, *req.SoulHP)
		sets = append(sets, "soul_hp=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.MaxSoul != nil {
		val := math.Max(0, *req.MaxSoul)
		sets = append(sets, "max_soul=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.SoulTier != nil {
		sets = append(sets, "soul_tier=$"+strconv.Itoa(argIdx))
		args = append(args, *req.SoulTier)
		argIdx++
	}
	if req.ExploreCount != nil {
		val := *req.ExploreCount
		if val < 0 {
			val = 0
		}
		sets = append(sets, "explore_count=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.ApertureStress != nil {
		val := math.Max(0, *req.ApertureStress)
		sets = append(sets, "aperture_stress=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.Bio != nil {
		sets = append(sets, "bio=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Bio)
		argIdx++
	}
	if req.AvatarURL != nil {
		sets = append(sets, "avatar_url=$"+strconv.Itoa(argIdx))
		args = append(args, *req.AvatarURL)
		argIdx++
	}
	if req.Race != nil {
		race := *req.Race
		if race != "" && !gamefx.ValidRaces()[race] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chủng tộc không hợp lệ"})
			return
		}
		sets = append(sets, "race=$"+strconv.Itoa(argIdx))
		args = append(args, race)
		argIdx++
	}
	if req.Gender != nil {
		gender := *req.Gender
		if gender != "" && !gamefx.ValidGenders()[gender] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Giới tính không hợp lệ"})
			return
		}
		sets = append(sets, "gender=$"+strconv.Itoa(argIdx))
		args = append(args, gender)
		argIdx++
	}
	if req.Constitution != nil {
		constitution := *req.Constitution
		if constitution != "" && !gamefx.ValidConstitutions()[constitution] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thể chất không hợp lệ"})
			return
		}
		sets = append(sets, "constitution=$"+strconv.Itoa(argIdx))
		args = append(args, constitution)
		argIdx++
	}
	if req.Statuses != nil {
		statuses := *req.Statuses
		for _, s := range statuses {
			if s.ID != "" && !gamefx.ValidStatuses()[s.ID] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái không hợp lệ: " + s.ID})
				return
			}
		}
		sets = append(sets, "statuses=$"+strconv.Itoa(argIdx))
		args = append(args, statuses)
		argIdx++
	}
	if req.CurrentThought != nil {
		sets = append(sets, "current_thought=$"+strconv.Itoa(argIdx))
		args = append(args, math.Max(0, *req.CurrentThought))
		argIdx++
	}
	if req.MaxThought != nil {
		sets = append(sets, "max_thought=$"+strconv.Itoa(argIdx))
		args = append(args, math.Max(0, *req.MaxThought))
		argIdx++
	}
	if req.ThoughtRegenRate != nil {
		sets = append(sets, "thought_regen_rate=$"+strconv.Itoa(argIdx))
		args = append(args, math.Max(0, *req.ThoughtRegenRate))
		argIdx++
	}
	if req.TimeSpeedMultiplier != nil {
		sets = append(sets, "time_speed_multiplier=$"+strconv.Itoa(argIdx))
		args = append(args, math.Max(0.1, *req.TimeSpeedMultiplier))
		argIdx++
	}
	if req.NeedsSoulRefine != nil {
		sets = append(sets, "needs_soul_refine=$"+strconv.Itoa(argIdx))
		args = append(args, *req.NeedsSoulRefine)
		argIdx++
	}
	if req.SelectedTitle != nil {
		sets = append(sets, "selected_title=$"+strconv.Itoa(argIdx))
		args = append(args, *req.SelectedTitle)
		argIdx++
	}
	if req.SelectedFrame != nil {
		sets = append(sets, "selected_frame=$"+strconv.Itoa(argIdx))
		args = append(args, *req.SelectedFrame)
		argIdx++
	}
	if req.FakeRank != nil {
		sets = append(sets, "fake_rank=$"+strconv.Itoa(argIdx))
		args = append(args, *req.FakeRank)
		argIdx++
	}
	if req.FakeSubRank != nil {
		sets = append(sets, "fake_sub_rank=$"+strconv.Itoa(argIdx))
		args = append(args, *req.FakeSubRank)
		argIdx++
	}
	if req.FakeVitalGu != nil {
		sets = append(sets, "fake_vital_gu=$"+strconv.Itoa(argIdx))
		args = append(args, *req.FakeVitalGu)
		argIdx++
	}
	if req.IsDisguised != nil {
		sets = append(sets, "is_disguised=$"+strconv.Itoa(argIdx))
		args = append(args, *req.IsDisguised)
		argIdx++
	}
	if req.NeedsOnboarding != nil {
		sets = append(sets, "needs_onboarding=$"+strconv.Itoa(argIdx))
		args = append(args, *req.NeedsOnboarding)
		argIdx++
	}

	if req.ImmortalJade != nil {
		_, err := tx.Exec(`INSERT INTO wallets (user_id, immortal_jade) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET immortal_jade = EXCLUDED.immortal_jade, updated_at = NOW()`, playerID, *req.ImmortalJade)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật Tiên Ngọc"})
			return
		}
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có trường nào để cập nhật"})
		return
	}
	args = append(args, playerID)
	if _, err := tx.Exec(`UPDATE players SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(argIdx), args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật thông tin người chơi"})
		return
	}
	if req.IsAdmin != nil {
		if *req.IsAdmin {
			if _, err := tx.Exec(`INSERT INTO player_admin_roles (player_id, role_key) VALUES ($1, $2) ON CONFLICT DO NOTHING`, playerID, authz.SuperAdminRole); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cấp quyền quản trị"})
				return
			}
		} else if _, err := tx.Exec(`DELETE FROM player_admin_roles WHERE player_id = $1`, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thu hồi quyền quản trị"})
			return
		}
	}

	if req.Rank != nil && *req.Rank >= 6 {
		var apID string
		var internalYears float64
		if err := tx.QueryRow(`SELECT id, internal_years_passed FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apID, &internalYears); err == nil {
			rankStart := math.Floor(internalYears)
			nextYear, _ := models.GetNextTribulationInfo(*req.Rank, rankStart, internalYears)
			guardedNext := guardNextTribulationYear(nextYear, internalYears)
			_, _ = tx.Exec(`UPDATE immortal_apertures SET rank_start_year = $1, next_tribulation_year = $2 WHERE id = $3`, rankStart, guardedNext, apID)
		}
	}

	tx.Commit()

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã cập nhật thông tin của đạo hữu %s: %s", username, formatChangedFields(sets, args)))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.update_player",
		EntityType:     "player",
		EntityID:       playerID.String(),
		RequestSummary: strings.Join(sets, ", "),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật thông tin người chơi"})
}
