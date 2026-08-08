package handlers

import (
	"database/sql"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type PlayerProfileResponse struct {
	Username        string            `json:"username"`
	Rank            int               `json:"rank"`
	SubRank         int               `json:"sub_rank"`
	AptitudeType    string            `json:"aptitude_type"`
	AptitudePercent int               `json:"aptitude_percent"`
	CurrentMana     float64           `json:"current_mana"`
	MaxMana         float64           `json:"max_mana"`
	PrimevalStones  float64           `json:"primeval_stones"`
	ImmortalStones  float64           `json:"immortal_stones"`
	HP              float64           `json:"hp"`
	BaseHP          float64           `json:"base_hp"`
	SectName        string            `json:"sect_name"`
	EquippedGus     []string          `json:"equipped_gus"`
	ScoutingStatus  string            `json:"scouting_status"`
	StatusType      string            `json:"status_type"` // "success", "partial", "failure"
	DominantPath    string            `json:"dominant_path"`
	DaoMarks        map[string]int    `json:"dao_marks"`
	PathMasteries   map[string]string `json:"path_masteries"`
	Bio             string            `json:"bio"`
	AvatarURL       string            `json:"avatar_url"`
}

// getDominantPath determines the player's primary path based on Dao Marks or Path Masteries
func getDominantPath(daoMarks map[string]int, pathMasteries map[string]string) string {
	maxMarks := 0
	dominant := ""
	for path, marks := range daoMarks {
		if marks > maxMarks {
			maxMarks = marks
			dominant = path
		}
	}
	if dominant != "" {
		return strings.Title(dominant)
	}

	masteryWeight := map[string]int{
		"Supreme Grandmaster": 5,
		"Great Grandmaster":   4,
		"Grandmaster":         3,
		"Master":              2,
		"Ordinary":            1,
	}
	maxWeight := 0
	for path, lvl := range pathMasteries {
		w := masteryWeight[lvl]
		if w > maxWeight {
			maxWeight = w
			dominant = path
		}
	}
	if dominant != "" {
		return strings.Title(dominant)
	}
	return "Chưa rõ"
}

// GetPlayerProfile returns another player's profile filtered by Scouting vs Disguise levels
func GetPlayerProfile(c *gin.Context) {
	viewerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	targetUsername := c.Param("username")
	if targetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	cacheKey := "profile_" + viewerID.String() + "_viewing_" + targetUsername
	var cachedResp PlayerProfileResponse
	if err := cache.GetJSON(cacheKey, &cachedResp); err == nil {
		c.JSON(http.StatusOK, cachedResp)
		return
	}

	// 1. Fetch viewer details for scouting calculation
	var viewerRank int
	var viewerGuInventory models.GuInventory
	var viewerPathMasteries models.PathMasteriesMap

	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT rank, `+db.GuInventoryAggregate("players.id")+`, path_masteries
		FROM players
		WHERE id = $1
	`, viewerID).Scan(&viewerRank, &viewerGuInventory, &viewerPathMasteries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load viewer data: " + err.Error()})
		return
	}

	// 2. Fetch target details and target's sect name
	var target models.Player
	var targetSectName sql.NullString
	var targetDaoMarks models.DaoMarksMap

	query := `
		SELECT p.id, p.username, p.rank, p.sub_rank, p.aptitude_type, p.aptitude_percent, p.current_mana, p.max_mana, p.primeval_stones, p.immortal_stones, p.hp, p.base_hp, `+db.GuInventoryAggregate("p.id")+`, p.path_masteries, COALESCE(p.bio, ''), COALESCE(p.avatar_url, ''), p.fake_rank, p.fake_sub_rank, COALESCE(p.fake_vital_gu, ''), p.is_disguised
		FROM players p
		WHERE p.username = $1
	`
	err = db.DB.QueryRowContext(c.Request.Context(), query, targetUsername).Scan(
		&target.ID, &target.Username, &target.Rank, &target.SubRank,
		&target.AptitudeType, &target.AptitudePercent, &target.CurrentMana, &target.MaxMana,
		&target.PrimevalStones, &target.ImmortalStones, &target.HP, &target.BaseHP, &target.GuInventory, &target.PathMasteries,
		&target.Bio, &target.AvatarURL, &target.FakeRank, &target.FakeSubRank, &target.FakeVitalGu, &target.IsDisguised,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ sư này!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tìm kiếm: " + err.Error()})
		return
	}

	if target.Rank >= 6 {
		_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, target.ID).Scan(&targetDaoMarks)
	}
	if targetDaoMarks == nil {
		targetDaoMarks = models.DaoMarksMap{"strength": 0}
	}

	// Fetch sect name separately to keep query simpler and safe
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT s.name 
		FROM sect_members sm 
		JOIN sects s ON sm.sect_id = s.id 
		WHERE sm.player_id = $1
	`, target.ID).Scan(&targetSectName)

	// 3. Compute Scouting Level of Viewer
	scoutingLevel := float64(viewerRank)
	for _, gu := range viewerGuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			switch gu.GuID {
			case "GU_FLESH_EAR":
				scoutingLevel += 1.0
			case "GU_EARTH_COMMUNICATION":
				scoutingLevel += 2.0
			case "GU_CONNECTING_HEAVEN":
				scoutingLevel += 3.0
			case "GU_DIVINE_SENSE":
				scoutingLevel += 4.0
			}
		}
	}
	for path, level := range viewerPathMasteries {
		cleanPath := strings.ToLower(path)
		if strings.Contains(cleanPath, "wisdom") || strings.Contains(cleanPath, "information") {
			switch level {
			case "Master":
				scoutingLevel += 1.0
			case "Grandmaster":
				scoutingLevel += 2.0
			case "Great Grandmaster":
				scoutingLevel += 3.0
			case "Supreme Grandmaster":
				scoutingLevel += 5.0
			}
		}
	}

	// 4. Compute Disguise Level of Target
	disguiseLevel := float64(target.Rank)
	for _, gu := range target.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			switch gu.GuID {
			case "GU_SHADOW", "GU_SHADOW_STEPS":
				disguiseLevel += 1.0
			case "GU_STEALTH_SCALES":
				disguiseLevel += 2.0
			case "GU_CHANGE_FORM":
				disguiseLevel += 3.0
			case "GU_ATTITUDE":
				disguiseLevel += 8.0
			}
		}
	}
	for path, level := range target.PathMasteries {
		cleanPath := strings.ToLower(path)
		if strings.Contains(cleanPath, "wisdom") || strings.Contains(cleanPath, "dark") {
			switch level {
			case "Master":
				disguiseLevel += 1.0
			case "Grandmaster":
				disguiseLevel += 2.0
			case "Great Grandmaster":
				disguiseLevel += 3.0
			case "Supreme Grandmaster":
				disguiseLevel += 5.0
			}
		}
	}

	// 5. Compare and Filter Information
	diff := scoutingLevel - disguiseLevel
	sectName := "Không tông môn"
	if targetSectName.Valid {
		sectName = targetSectName.String
	}

	var resp PlayerProfileResponse
	resp.SectName = sectName

	if diff >= 2 {
		// Full Success
		resp.Username = target.Username
		resp.Bio = target.Bio
		resp.AvatarURL = target.AvatarURL
		resp.Rank = target.Rank
		resp.SubRank = target.SubRank
		resp.AptitudeType = target.AptitudeType
		resp.AptitudePercent = target.AptitudePercent
		resp.CurrentMana = target.CurrentMana
		resp.MaxMana = target.MaxMana
		resp.PrimevalStones = target.PrimevalStones
		resp.ImmortalStones = target.ImmortalStones
		resp.HP = target.HP
		resp.BaseHP = target.BaseHP
		resp.ScoutingStatus = "Đột phá trinh sát thành công! Thấu thị toàn bộ tu vi đối phương."
		resp.StatusType = "success"
		resp.DominantPath = getDominantPath(targetDaoMarks, target.PathMasteries)
		resp.DaoMarks = targetDaoMarks
		resp.PathMasteries = target.PathMasteries

		equipped := []string{}
		for _, gu := range target.GuInventory {
			if gu.Equipped && gu.Status == "ACTIVE" {
				equipped = append(equipped, gu.GuID)
			}
		}
		resp.EquippedGus = equipped

	} else if diff >= 0 {
		// Partial Success
		resp.Username = target.Username
		resp.Bio = target.Bio
		resp.AvatarURL = target.AvatarURL
		resp.Rank = target.Rank
		resp.SubRank = target.SubRank
		resp.AptitudeType = "???"
		resp.AptitudePercent = 0
		resp.CurrentMana = 0
		resp.MaxMana = 0
		resp.PrimevalStones = 0
		resp.ImmortalStones = 0
		resp.HP = 0
		resp.BaseHP = 0
		resp.ScoutingStatus = "Nhìn thấu sơ bộ. Đối phương che giấu một phần thông tin."
		resp.StatusType = "partial"
		resp.DominantPath = getDominantPath(targetDaoMarks, target.PathMasteries)
		resp.DaoMarks = map[string]int{}
		resp.PathMasteries = map[string]string{}

		equipped := []string{}
		for _, gu := range target.GuInventory {
			if gu.Equipped && gu.Status == "ACTIVE" {
				equipped = append(equipped, gu.GuID)
			}
		}
		if len(equipped) > 0 {
			rand.Shuffle(len(equipped), func(i, j int) {
				equipped[i], equipped[j] = equipped[j], equipped[i]
			})
			showCount := 2
			if len(equipped) < 2 {
				showCount = len(equipped)
			}
			shown := equipped[:showCount]
			if len(equipped) > showCount {
				shown = append(shown, "???")
			}
			resp.EquippedGus = shown
		} else {
			resp.EquippedGus = []string{}
		}

	} else {
		// Scouting Failure
		resp.StatusType = "failure"
		resp.DaoMarks = map[string]int{}
		resp.PathMasteries = map[string]string{}

		if target.IsDisguised {
			// Return B's configured fake profile settings
			resp.Username = target.Username
			resp.Rank = target.FakeRank
			resp.SubRank = target.FakeSubRank
			resp.Bio = "Ta chỉ là một cổ sư bình thường tu luyện trong núi..."
			resp.AptitudeType = "???"
			resp.AptitudePercent = 0
			resp.CurrentMana = 0
			resp.MaxMana = 0
			resp.PrimevalStones = 0
			resp.ImmortalStones = 0
			resp.HP = 0
			resp.BaseHP = 0
			resp.SectName = "Không tông môn"
			if target.FakeVitalGu != "" {
				resp.EquippedGus = []string{target.FakeVitalGu}
			} else {
				resp.EquippedGus = []string{}
			}
			// resp.ScoutingStatus = "Trinh sát thất bại! Đối phương ngụy trang tinh vi hoặc tu vi vượt trội."
			resp.DominantPath = "Chưa rõ"
		} else {
			hasAttitudeGu := false
			for _, gu := range target.GuInventory {
				if gu.GuID == "GU_ATTITUDE" && gu.Equipped && gu.Status == "ACTIVE" {
					hasAttitudeGu = true
					break
				}
			}

			if hasAttitudeGu {
				// Perfect disguise via Attitude Gu
				resp.Username = "Ẩn Danh Cổ Sư"
				resp.Bio = "Ta chỉ là một tán tu vô danh..."
				resp.Rank = 1
				resp.SubRank = 1
				resp.AptitudeType = "Đinh"
				resp.AptitudePercent = 25
				resp.CurrentMana = 25
				resp.MaxMana = 25
				resp.PrimevalStones = 0
				resp.ImmortalStones = 0
				resp.HP = 50
				resp.BaseHP = 100
				resp.SectName = "Không tông môn"
				resp.EquippedGus = []string{"GU_SHADOW"}
				// resp.ScoutingStatus = "Thất bại trinh sát! Đối phương ngụy trang hoàn hảo hoặc tu vi/trí đạo vượt trội."
				resp.DominantPath = "Trí Đạo"
				resp.DaoMarks = map[string]int{"wisdom": 50}
				resp.PathMasteries = map[string]string{"wisdom": "Master"}
			} else {
				// Scramble username, randomize rank
				scrambled := target.Username
				if len(scrambled) > 3 {
					scrambled = scrambled[:2] + "****"
				} else {
					scrambled = "***"
				}
				resp.Username = scrambled
				resp.Bio = "???"

				offset := rand.Intn(3) - 1 // -1, 0, +1
				fakeRank := target.Rank + offset
				if fakeRank < 1 {
					fakeRank = 1
				}
				if fakeRank > 9 {
					fakeRank = 9
				}

				resp.Rank = fakeRank
				resp.SubRank = rand.Intn(4) + 1
				resp.AptitudeType = "???"
				resp.AptitudePercent = 0
				resp.CurrentMana = 0
				resp.MaxMana = 0
				resp.PrimevalStones = 0
				resp.ImmortalStones = 0
				resp.HP = 0
				resp.BaseHP = 0
				resp.SectName = "???"
				resp.EquippedGus = []string{}
				// resp.ScoutingStatus = "Trinh sát thất bại! Đối phương che giấu tu vi."
				resp.DominantPath = "Chưa rõ"
			}
		}
	}

	_ = cache.SetJSON(cacheKey, resp, 60*time.Second)
	c.JSON(http.StatusOK, resp)
}
