package handlers

import (
	"database/sql"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type PlayerProfileResponse struct {
	Username        string              `json:"username"`
	Rank            int                 `json:"rank"`
	SubRank         int                 `json:"sub_rank"`
	AptitudeType    string              `json:"aptitude_type"`
	AptitudePercent int                 `json:"aptitude_percent"`
	CurrentMana     float64             `json:"current_mana"`
	MaxMana         float64             `json:"max_mana"`
	PrimevalStones  float64             `json:"primeval_stones"`
	ImmortalStones  float64             `json:"immortal_stones"`
	HP              float64             `json:"hp"`
	BaseHP          float64             `json:"base_hp"`
	SectName        string              `json:"sect_name"`
	EquippedGus     []string            `json:"equipped_gus"`
	ScoutingStatus  string              `json:"scouting_status"`
	StatusType      string              `json:"status_type"` // "success", "partial", "failure"
	DeltaScore      float64             `json:"delta_score,omitempty"`
	MyScore         *models.ScoreDetail `json:"my_score,omitempty"`
	DominantPath    string              `json:"dominant_path"`
	DaoMarks        models.DaoMarksMap  `json:"dao_marks"`
	PathMasteries   map[string]int      `json:"path_masteries"`
	Bio             string              `json:"bio"`
	AvatarURL       string              `json:"avatar_url"`
	Race            string              `json:"race"`
	Gender          string              `json:"gender"`
	Constitution    string              `json:"constitution"`
	Statuses        models.StatusList   `json:"statuses"`
}

// getDominantPath determines the player's primary path based on Dao Marks or Path Masteries
func getDominantPath(daoMarks models.DaoMarksMap, pathMasteries map[string]int) string {
	maxMarks := int64(0)
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

	maxWeight := 0
	for path, points := range pathMasteries {
		w := gamefx.PathMasteryLevel(points)
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
	var viewerRegion string
	var viewerX, viewerY int
	var viewerPartyID *uuid.UUID
	var viewerSectID sql.NullString
	var viewerGuInventory models.GuInventory
	var viewerPathMasteries models.PathMasteriesMap

	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT rank, current_region, x, y, party_id, `+db.GuInventoryAggregate("players.id")+`, path_masteries
		FROM players
		WHERE id = $1
	`, viewerID).Scan(&viewerRank, &viewerRegion, &viewerX, &viewerY, &viewerPartyID, &viewerGuInventory, &viewerPathMasteries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load viewer data: " + err.Error()})
		return
	}

	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id::text FROM sect_members WHERE player_id = $1`, viewerID).Scan(&viewerSectID)

	// 2. Fetch target details and target's sect name
	var target models.Player
	var targetSectID sql.NullString
	var targetSectName sql.NullString
	var targetDaoMarks models.DaoMarksMap

	query := `
		SELECT p.id, p.username, p.rank, p.sub_rank, p.aptitude_type, p.aptitude_percent, p.current_mana, p.max_mana, p.primeval_stones, p.immortal_stones, p.hp, p.base_hp, p.current_region, p.x, p.y, p.party_id, ` + db.GuInventoryAggregate("p.id") + `, p.path_masteries, COALESCE(p.bio, ''), COALESCE(p.avatar_url, ''), p.fake_rank, p.fake_sub_rank, COALESCE(p.fake_vital_gu, ''), p.is_disguised, COALESCE(p.race, ''), COALESCE(p.gender, ''), COALESCE(p.constitution, ''), p.statuses
		FROM players p
		WHERE p.username = $1
	`
	err = db.DB.QueryRowContext(c.Request.Context(), query, targetUsername).Scan(
		&target.ID, &target.Username, &target.Rank, &target.SubRank,
		&target.AptitudeType, &target.AptitudePercent, &target.CurrentMana, &target.MaxMana,
		&target.PrimevalStones, &target.ImmortalStones, &target.HP, &target.BaseHP,
		&target.CurrentRegion, &target.X, &target.Y, &target.PartyID,
		&target.GuInventory, &target.PathMasteries,
		&target.Bio, &target.AvatarURL, &target.FakeRank, &target.FakeSubRank, &target.FakeVitalGu, &target.IsDisguised,
		&target.Race, &target.Gender, &target.Constitution, &target.Statuses,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ sư này!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tìm kiếm: " + err.Error()})
		return
	}

	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT sm.sect_id::text, s.name 
		FROM sect_members sm 
		JOIN sects s ON sm.sect_id = s.id 
		WHERE sm.player_id = $1
	`, target.ID).Scan(&targetSectID, &targetSectName)

	// Check self-inspection or proximity
	isSelf := viewerID == target.ID
	isSameParty := viewerPartyID != nil && target.PartyID != nil && *viewerPartyID == *target.PartyID
	isSameSect := viewerSectID.Valid && targetSectID.Valid && viewerSectID.String == targetSectID.String

	if !isSelf && !isSameParty && !isSameSect {
		if viewerRegion != target.CurrentRegion {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không thể xem thông tin từ xa! Hãy dùng Suy Diễn Thiên Cơ."})
			return
		}

		investigationRange := 15.0 + float64(viewerRank)*2.0
		for _, gu := range models.FilterEquippedActive(viewerGuInventory) {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				investigationRange += models.GuStatValue(spec, "INVESTIGATION") * 2.0
			}
		}

		dist := math.Hypot(float64(viewerX-target.X), float64(viewerY-target.Y))
		if dist > investigationRange {
			c.JSON(http.StatusForbidden, gin.H{"error": "Đối phương nằm ngoài tầm mắt! Hãy dùng Suy Diễn Thiên Cơ để suy đoán từ xa."})
			return
		}
	}

	// 3. Compute Scouting Level of Viewer
	scoutingLevel := float64(viewerRank)
	for _, gu := range models.FilterEquippedActive(viewerGuInventory) {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			scoutingLevel += models.GuStatValue(spec, "INVESTIGATION")
		}
	}
	for path, points := range viewerPathMasteries {
		cleanPath := strings.ToLower(path)
		if strings.Contains(cleanPath, "wisdom") || strings.Contains(cleanPath, "information") {
			switch gamefx.PathMasteryLevel(points) {
			case 1:
				scoutingLevel += 1.0
			case 2:
				scoutingLevel += 2.0
			case 3:
				scoutingLevel += 3.0
			case 4:
				scoutingLevel += 5.0
			}
		}
	}

	// 4. Compute Disguise Level of Target
	disguiseLevel := float64(target.Rank)
	for _, gu := range models.FilterEquippedActive(target.GuInventory) {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			disguiseLevel += models.GuStatValue(spec, "DISGUISE")
		}
	}
	for path, points := range target.PathMasteries {
		cleanPath := strings.ToLower(path)
		if strings.Contains(cleanPath, "wisdom") || strings.Contains(cleanPath, "dark") {
			switch gamefx.PathMasteryLevel(points) {
			case 1:
				disguiseLevel += 1.0
			case 2:
				disguiseLevel += 2.0
			case 3:
				disguiseLevel += 3.0
			case 4:
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
		resp.BaseHP = gamefx.EffectiveMaxHP(&target)
		resp.ScoutingStatus = "Đột phá trinh sát thành công! Thấu thị toàn bộ tu vi đối phương."
		resp.StatusType = "success"
		resp.DominantPath = getDominantPath(targetDaoMarks, target.PathMasteries)
		resp.DaoMarks = targetDaoMarks
		resp.PathMasteries = target.PathMasteries
		resp.Race = target.Race
		resp.Gender = target.Gender
		resp.Constitution = target.Constitution
		resp.Statuses = target.Statuses.Active()

		equipped := []string{}
		for _, gu := range models.FilterEquippedActive(target.GuInventory) {
			equipped = append(equipped, gu.GuID)
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
		resp.DaoMarks = models.DaoMarksMap{}
		resp.PathMasteries = map[string]int{}

		equipped := []string{}
		for _, gu := range models.FilterEquippedActive(target.GuInventory) {
			equipped = append(equipped, gu.GuID)
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
		resp.DaoMarks = models.DaoMarksMap{}
		resp.PathMasteries = map[string]int{}

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
			hasAttitudeGu := models.InventoryHasGuCapability(target.GuInventory, "PERFECT_DISGUISE", true)

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
				resp.SectName = "Không tông môn"
				if target.FakeVitalGu != "" {
					resp.EquippedGus = []string{target.FakeVitalGu}
				} else {
					resp.EquippedGus = []string{}
				}
				// resp.ScoutingStatus = "Thất bại trinh sát! Đối phương ngụy trang hoàn hảo hoặc tu vi/trí đạo vượt trội."
				resp.DominantPath = "Trí Đạo"
				resp.DaoMarks = models.DaoMarksMap{"wisdom": 50}
				resp.PathMasteries = map[string]int{"wisdom": gamefx.PathMasteryMasterPoints}
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
