package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Divination economy: mortals burn primeval stones, immortals may burn immortal
// stones instead. Backfire drains a fixed mana amount, never a percentage.
const (
	divinationPrimevalMin    = 100.0
	divinationBackfireMana   = 20.0
	divineGuPrimevalFactor   = 0.1
	divinationPrimevalFactor = 0.1
	severCluesManaCost       = 20.0
	resonanceScanManaCost    = 50.0
)

// InspectLocalPlayer handles local map inspection with FOV and Party/Sect verification
func InspectLocalPlayer(c *gin.Context) {
	viewerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.InspectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if req.TargetID == "" && req.TargetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TargetID or TargetUsername is required"})
		return
	}

	// 1. Fetch viewer data
	var viewer models.Player
	var viewerSectID sql.NullString
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_region, p.x, p.y, p.party_id,
		       `+db.GuInventoryAggregate("p.id")+`, p.path_masteries
		FROM players p
		WHERE p.id = $1
	`, viewerID).Scan(
		&viewer.ID, &viewer.Username, &viewer.Rank, &viewer.SubRank, &viewer.CurrentRegion, &viewer.X, &viewer.Y, &viewer.PartyID,
		&viewer.GuInventory, &viewer.PathMasteries,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load viewer data: " + err.Error()})
		return
	}

	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id::text FROM sect_members WHERE player_id = $1`, viewerID).Scan(&viewerSectID)

	var viewerDaoMarks models.DaoMarksMap
	if viewer.Rank >= 6 {
		_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, viewer.ID).Scan(&viewerDaoMarks)
	}

	// 2. Fetch target data
	var target models.Player
	var targetSectID sql.NullString
	var targetSectName sql.NullString
	var targetDaoMarks models.DaoMarksMap

	var targetQuery string
	var targetArg interface{}
	if req.TargetID != "" {
		targetQuery = "p.id = $1"
		targetArg = req.TargetID
	} else {
		targetQuery = "LOWER(p.username) = LOWER($1)"
		targetArg = strings.TrimSpace(req.TargetUsername)
	}

	query := `
		SELECT p.id, p.username, p.rank, p.sub_rank, p.aptitude_type, p.aptitude_percent, p.current_mana, p.max_mana,
		       p.primeval_stones, p.immortal_stones, p.hp, p.base_hp, p.current_region, p.x, p.y, p.party_id,
		       ` + db.GuInventoryAggregate("p.id") + `, p.path_masteries, COALESCE(p.bio, ''), COALESCE(p.avatar_url, ''),
		       p.fake_rank, p.fake_sub_rank, COALESCE(p.fake_vital_gu, ''), p.is_disguised, COALESCE(p.race, ''),
		       COALESCE(p.gender, ''), COALESCE(p.constitution, ''), p.statuses
		FROM players p
		WHERE ` + targetQuery

	err = db.DB.QueryRowContext(c.Request.Context(), query, targetArg).Scan(
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn mục tiêu: " + err.Error()})
		return
	}

	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT sm.sect_id::text, s.name 
		FROM sect_members sm 
		JOIN sects s ON sm.sect_id = s.id 
		WHERE sm.player_id = $1
	`, target.ID).Scan(&targetSectID, &targetSectName)

	if target.Rank >= 6 {
		_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, target.ID).Scan(&targetDaoMarks)
	}
	if targetDaoMarks == nil {
		targetDaoMarks = models.DaoMarksMap{"strength": 0}
	}

	// 3. Check Visibility / Exemption: Same Party / Same Sect / Local FOV
	isSameParty := viewer.PartyID != nil && target.PartyID != nil && *viewer.PartyID == *target.PartyID
	isSameSect := viewerSectID.Valid && targetSectID.Valid && viewerSectID.String == targetSectID.String

	if !isSameParty && !isSameSect {
		if viewer.CurrentRegion != target.CurrentRegion {
			c.JSON(http.StatusForbidden, gin.H{"error": "Đối phương ở vùng khác, không thể trinh sát trực tiếp!"})
			return
		}

		// Calculate investigation FOV range
		investigationRange := 15.0 + float64(viewer.Rank)*2.0
		for _, gu := range models.FilterEquippedActive(viewer.GuInventory) {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				investigationRange += models.GuStatValue(spec, "INVESTIGATION") * 2.0
			}
		}

		dist := math.Hypot(float64(viewer.X-target.X), float64(viewer.Y-target.Y))
		if dist > investigationRange {
			c.JSON(http.StatusForbidden, gin.H{"error": "Đối phương nằm ngoài tầm mắt! Hãy dùng Suy Diễn Thiên Cơ để dò tìm từ xa."})
			return
		}
	}

	// 4. Final score only: one engine delta decides everything.
	delta, myScore := gamefx.EvaluateInspect(
		&viewer,
		&target,
		models.FilterEquippedActive(viewer.GuInventory),
		models.FilterEquippedActive(target.GuInventory),
		viewerDaoMarks,
		targetDaoMarks,
	)
	sectName := "Không tông môn"
	if targetSectName.Valid {
		sectName = targetSectName.String
	}

	var resp PlayerProfileResponse
	resp.SectName = sectName
	resp.DeltaScore = delta
	resp.MyScore = &myScore

	if isSameParty || isSameSect || delta >= 30 {
		// Full inspection
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
		resp.ScoutingStatus = "Trinh sát trực tiếp thành công! Thấu thị toàn bộ tu vi đối phương."
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

	} else if delta >= -10 {
		// Partial inspection
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
		// Failure / Disguise
		resp.StatusType = "failure"
		resp.DaoMarks = models.DaoMarksMap{}
		resp.PathMasteries = map[string]int{}

		if target.IsDisguised {
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
			resp.DominantPath = "Chưa rõ"
		} else {
			resp.Username = target.Username[:int(math.Min(2, float64(len(target.Username))))] + "****"
			resp.Bio = "???"
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
			resp.SectName = "???"
			resp.EquippedGus = []string{}
			resp.DominantPath = "Chưa rõ"
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ExecuteDivination handles remote deduction with Dao marks and backfire logic
func ExecuteDivination(c *gin.Context) {
	viewerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.DivinationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if req.TargetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TargetUsername is required for divination"})
		return
	}

	// Fetch target sect name for clue validation (server-side only)
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id::text FROM sect_members WHERE player_id = (SELECT id FROM players WHERE LOWER(username) = LOWER($1))`, strings.TrimSpace(req.TargetUsername)).Scan(&req.TargetSectName)

	// 1. Fetch Viewer (statuses needed for max HP on backfire)
	var viewer models.Player
	var viewerDaoMarks models.DaoMarksMap
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_mana, p.max_mana, p.immortal_stones,
		       p.primeval_stones, p.hp, p.base_hp, p.soul_hp, p.max_soul, p.current_region, p.x, p.y,
		       `+db.GuInventoryAggregate("p.id")+`, p.path_masteries, p.statuses
		FROM players p
		WHERE p.id = $1
	`, viewerID).Scan(
		&viewer.ID, &viewer.Username, &viewer.Rank, &viewer.SubRank, &viewer.CurrentMana, &viewer.MaxMana,
		&viewer.ImmortalStones, &viewer.PrimevalStones, &viewer.HP, &viewer.BaseHP, &viewer.SoulHP, &viewer.MaxSoul,
		&viewer.CurrentRegion, &viewer.X, &viewer.Y, &viewer.GuInventory, &viewer.PathMasteries, &viewer.Statuses,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load viewer data: " + err.Error()})
		return
	}

	if viewer.Rank >= 6 {
		_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, viewer.ID).Scan(&viewerDaoMarks)
	}

	// 2. Resource Validation & Deduction (stones only, never mana)
	primevalCost := 0.0
	immortalStoneCost := 0.0
	resourceBonus := 0.0

	if req.ResourceType == "immortal_stone" {
		if viewer.Rank < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ Tiên nhân mới có thể vận dụng Tiên Nguyên Thạch để suy diễn!"})
			return
		}
		immortalStoneCost = req.ResourceAmount
		if immortalStoneCost < 1.0 {
			immortalStoneCost = 1.0
		}
		if viewer.ImmortalStones < immortalStoneCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Nguyên Thạch!"})
			return
		}
		resourceBonus = immortalStoneCost * 5.0
	} else {
		// Primeval stones (legacy "mana" accepted as alias)
		primevalCost = req.ResourceAmount
		if primevalCost < divinationPrimevalMin {
			primevalCost = divinationPrimevalMin
		}
		if viewer.PrimevalStones < primevalCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nguyên Thạch không đủ để thi triển suy diễn!"})
			return
		}
		resourceBonus = primevalCost * divinationPrimevalFactor
	}

	// 3. Fetch Target
	var target models.Player
	var targetDaoMarks models.DaoMarksMap
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_mana, p.max_mana, p.hp, p.base_hp,
		       p.current_region, p.x, p.y, `+db.GuInventoryAggregate("p.id")+`, p.path_masteries,
		       p.is_disguised, p.fake_rank, p.fake_sub_rank, COALESCE(p.fake_vital_gu, '')
		FROM players p
		WHERE LOWER(p.username) = LOWER($1)
	`, strings.TrimSpace(req.TargetUsername)).Scan(
		&target.ID, &target.Username, &target.Rank, &target.SubRank, &target.CurrentMana, &target.MaxMana,
		&target.HP, &target.BaseHP, &target.CurrentRegion, &target.X, &target.Y,
		&target.GuInventory, &target.PathMasteries, &target.IsDisguised,
		&target.FakeRank, &target.FakeSubRank, &target.FakeVitalGu,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thiên cơ mờ mịt, không tìm thấy nhân quả của cổ sư này!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn thiên cơ: " + err.Error()})
		return
	}

	if target.Rank >= 6 {
		_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, target.ID).Scan(&targetDaoMarks)
	}

	// Validate ItemClueID: check if target actually owns the Gu in player_gu
	itemClueValid := false
	if req.ItemClueID != "" {
		var clueCount int
		err := db.DB.QueryRowContext(c.Request.Context(),
			`SELECT COUNT(*) FROM player_gu WHERE player_id = $1 AND gu_id = $2`,
			target.ID, req.ItemClueID).Scan(&clueCount)
		if err == nil && clueCount > 0 {
			itemClueValid = true
		}
	}

	// Validate SuspectedGu: check if target actually owns the Gu in player_gu
	suspectedGuValid := false
	if req.SuspectedGu != "" {
		var guCount int
		err := db.DB.QueryRowContext(c.Request.Context(),
			`SELECT COUNT(*) FROM player_gu WHERE player_id = $1 AND gu_id = $2`,
			target.ID, req.SuspectedGu).Scan(&guCount)
		if err == nil && guCount > 0 {
			suspectedGuValid = true
		}
	}

	// 4. Evaluate Divination (final delta only)
	statusType, delta, backfireRatio, myScore := gamefx.EvaluateDivination(
		&viewer,
		&target,
		models.FilterEquippedActive(viewer.GuInventory),
		models.FilterEquippedActive(target.GuInventory),
		viewerDaoMarks,
		targetDaoMarks,
		req,
		resourceBonus,
		itemClueValid,
		suspectedGuValid,
	)

	// Deduct resources & apply damage in transaction
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi transaction"})
		return
	}
	defer tx.Rollback()

	newStones := viewer.PrimevalStones - primevalCost
	if newStones < 0 {
		newStones = 0
	}
	newImmortalStones := viewer.ImmortalStones - immortalStoneCost
	if newImmortalStones < 0 {
		newImmortalStones = 0
	}

	newMana := viewer.CurrentMana
	newHP := viewer.HP
	newSoulHP := viewer.SoulHP

	var backfireDamage, backfireSoulDamage float64

	if statusType == "backfire" {
		// Fixed mana drain, never a percentage.
		newMana -= divinationBackfireMana
		if newMana < 0 {
			newMana = 0
		}
		// HP / Soul damage is a bounded fraction of the attacker's own maxima, so a
		// bad roll against a high-rank target can never be instantly lethal.
		maxHP := gamefx.EffectiveMaxHP(&viewer)
		if maxHP <= 0 {
			maxHP = viewer.BaseHP
		}
		maxSoul := float64(viewer.MaxSoul)

		backfireDamage = maxHP * backfireRatio
		backfireSoulDamage = maxSoul * backfireRatio

		newHP -= backfireDamage
		if newHP < 1.0 {
			newHP = 1.0
		}
		newSoulHP -= int(backfireSoulDamage)
		if newSoulHP < 1 {
			newSoulHP = 1
		}

		// Insert backfire alert into world traces for target awareness
		_, _ = tx.ExecContext(c.Request.Context(), `
			INSERT INTO world_residual_traces(region_id, x, y, player_id, action_type, dominant_path, dao_pulse_strength, expires_at)
			VALUES($1, $2, $3, $4, 'backfire_alert', 'wisdom', $5, NOW() + INTERVAL '10 minutes')
		`, viewer.CurrentRegion, viewer.X, viewer.Y, target.ID, backfireDamage)
	}

	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, current_mana = $3, hp = $4, soul_hp = $5
		WHERE id = $6
	`, newStones, newImmortalStones, newMana, newHP, newSoulHP, viewer.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật tài nguyên: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit transaction"})
		return
	}

	// 5. Construct Response
	resp := models.DivinationResponse{
		StatusType:   statusType,
		DeltaScore:   delta,
		MyScore:      &myScore,
		CanApplyMark: statusType == "full",
	}

	switch statusType {
	case "full":
		gus := []string{}
		for _, g := range models.FilterEquippedActive(target.GuInventory) {
			gus = append(gus, g.GuID)
		}
		resp.Report = &models.DivinationReport{
			TargetID:      target.ID,
			Username:      target.Username,
			Region:        target.CurrentRegion,
			Coordinates:   &models.Coordinates{X: target.X, Y: target.Y},
			ApproxRank:    target.Rank,
			SubRank:       target.SubRank,
			DominantPath:  getDominantPath(targetDaoMarks, target.PathMasteries),
			DetectedGus:   gus,
			StatusSummary: "Định vị thiên cơ hoàn hảo! Thấu thị vị trí và tu vi chân thực.",
			IsDisguised:   target.IsDisguised,
		}
		resp.Message = "Suy diễn đại thành công! Đã nắm bắt chính xác tọa độ và khí tức mục tiêu."

	case "partial":
		gus := []string{}
		for _, g := range models.FilterEquippedActive(target.GuInventory) {
			gus = append(gus, g.GuID)
		}
		if len(gus) > 2 {
			gus = gus[:2]
		}
		approxRank := target.Rank
		if target.IsDisguised && target.FakeRank > 0 {
			approxRank = target.FakeRank
		}
		resp.Report = &models.DivinationReport{
			TargetID:      target.ID,
			Username:      target.Username,
			Region:        target.CurrentRegion,
			Coordinates:   nil, // Hidden coordinates
			ApproxRank:    approxRank,
			DominantPath:  getDominantPath(targetDaoMarks, target.PathMasteries),
			DetectedGus:   gus,
			StatusSummary: "Nhìn thấu sơ bộ. Đối phương có màn sương phòng hộ.",
			IsDisguised:   target.IsDisguised,
		}
		resp.Message = "Suy diễn tiểu thành công! Nắm được phương vị khu vực và tu vi ước lượng."

	case "fogged":
		resp.Message = "Màn sương thiên cơ quá dày đặc! Kẻ thù che giấu nhân quả, suy diễn thất bại."

	case "backfire":
		resp.BackfireDamage = backfireDamage
		resp.BackfireSoulDamage = backfireSoulDamage
		resp.Message = "Thiên cơ phản phệ! Đối phương tu vi/phòng ngự trí đạo vượt trội, bạn bị trọng thương linh hồn và tổn thất 20 Chân Nguyên!"
	}

	c.JSON(http.StatusOK, resp)
}

// ApplyTrackingMark places a persistent tracking impression on a target
func ApplyTrackingMark(c *gin.Context) {
	viewerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.ApplyMarkRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TargetID is required"})
		return
	}

	targetUUID, err := uuid.Parse(req.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TargetID"})
		return
	}

	var viewerRank int
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT rank FROM players WHERE id = $1`, viewerID).Scan(&viewerRank)

	// Check active marks count
	var activeCount int
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*) FROM player_tracking_marks WHERE tracker_id = $1 AND expires_at > NOW()
	`, viewerID).Scan(&activeCount)

	maxAllowed := 3
	if viewerRank >= 6 {
		maxAllowed = 10
	}

	if activeCount >= maxAllowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đã đạt giới hạn số lượng mục tiêu theo dõi đồng thời!"})
		return
	}

	markType := req.MarkType
	if markType != "blood" && markType != "wisdom" {
		markType = "wisdom"
	}

	duration := time.Duration(viewerRank*30) * time.Minute
	if markType == "blood" {
		duration = time.Duration(viewerRank*2) * time.Hour
	}
	expiresAt := time.Now().Add(duration)

	_, err = db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO player_tracking_marks(tracker_id, target_id, mark_type, power, expires_at)
		VALUES($1, $2, $3, $4, $5)
	`, viewerID, targetUUID, markType, float64(viewerRank)*10.0, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi gieo ấn ký: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"mark_type":    markType,
		"expires_at":   expiresAt,
		"active_marks": activeCount + 1,
		"message":      "Đã gieo ấn ký truy tung thành công!",
	})
}

// GetTrackedTargets returns all active tracked targets with their real-time coordinates
func GetTrackedTargets(c *gin.Context) {
	viewerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT m.id, m.tracker_id, m.target_id, m.mark_type, m.power, m.expires_at, m.created_at,
		       p.username, COALESCE(p.avatar_url, ''), p.current_region, p.x, p.y
		FROM player_tracking_marks m
		JOIN players p ON m.target_id = p.id
		WHERE m.tracker_id = $1 AND m.expires_at > NOW()
		ORDER BY m.expires_at DESC
	`, viewerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn danh sách theo dõi: " + err.Error()})
		return
	}
	defer rows.Close()

	targets := []models.TrackingMark{}
	for rows.Next() {
		var tm models.TrackingMark
		if err := rows.Scan(
			&tm.ID, &tm.TrackerID, &tm.TargetID, &tm.MarkType, &tm.Power, &tm.ExpiresAt, &tm.CreatedAt,
			&tm.TargetUsername, &tm.TargetAvatar, &tm.CurrentRegion, &tm.X, &tm.Y,
		); err == nil {
			targets = append(targets, tm)
		}
	}

	c.JSON(http.StatusOK, gin.H{"tracked_targets": targets})
}

// SeverClues purges tracking marks and grid traces left by this player
func SeverClues(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var player models.Player
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, rank, current_mana, immortal_stones, current_region, x, y,
		       `+db.GuInventoryAggregate("players.id")+`, path_masteries
		FROM players WHERE id = $1
	`, playerID).Scan(
		&player.ID, &player.Rank, &player.CurrentMana, &player.ImmortalStones,
		&player.CurrentRegion, &player.X, &player.Y, &player.GuInventory, &player.PathMasteries,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải thông tin cổ sư: " + err.Error()})
		return
	}

	// Fixed mana cost, never a percentage.
	manaCost := severCluesManaCost
	if player.CurrentMana < manaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chân nguyên không đủ để thi triển Đoạn Tuyệt Thiên Cơ!"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi transaction"})
		return
	}
	defer tx.Rollback()

	// 1. Deduct mana
	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_mana = current_mana - $1, last_sever_clues_at = NOW()
		WHERE id = $2
	`, manaCost, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật chân nguyên: " + err.Error()})
		return
	}

	// 2. Delete tracking marks pointing to this player
	resMarks, err := tx.ExecContext(c.Request.Context(), `
		DELETE FROM player_tracking_marks WHERE target_id = $1
	`, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thanh tẩy ấn ký: " + err.Error()})
		return
	}
	purgedMarks, _ := resMarks.RowsAffected()

	// 3. Delete cell residual traces at current position
	resTraces, err := tx.ExecContext(c.Request.Context(), `
		DELETE FROM world_residual_traces
		WHERE player_id = $1 AND region_id = $2 AND x = $3 AND y = $4
	`, player.ID, player.CurrentRegion, player.X, player.Y)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa tàn dư ô bản đồ: " + err.Error()})
		return
	}
	purgedTraces, _ := resTraces.RowsAffected()

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		return
	}

	c.JSON(http.StatusOK, models.SeverCluesResponse{
		Success:           true,
		PurgedMarksCount:  int(purgedMarks),
		PurgedTracesCount: int(purgedTraces),
		Message:           "Đã chém đứt toàn bộ ấn ký thiên cơ đang âm thầm theo dõi bạn và xóa sạch tàn dư khí tức tại vị trí này!",
	})
}

// ResonanceScan handles Gu resonance and Dao pulse detection across regions
func ResonanceScan(c *gin.Context) {
	viewerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.ResonanceScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	var viewer models.Player
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, rank, current_mana, immortal_stones, primeval_stones, current_region, x, y
		FROM players WHERE id = $1
	`, viewerID).Scan(
		&viewer.ID, &viewer.Rank, &viewer.CurrentMana, &viewer.ImmortalStones,
		&viewer.PrimevalStones, &viewer.CurrentRegion, &viewer.X, &viewer.Y,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải thông tin người chơi"})
		return
	}

	// Cost validation
	scanRegion := req.ScanRegion
	if scanRegion == "" {
		scanRegion = viewer.CurrentRegion
	}

	isCrossRegion := !strings.EqualFold(scanRegion, viewer.CurrentRegion)
	if isCrossRegion && viewer.Rank < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phàm nhân không thể cảm ứng Đạo ngân xuyên qua Ngũ Vực Giới Bích!"})
		return
	}

	if viewer.Rank >= 6 {
		tntCost := 2.0
		if isCrossRegion {
			tntCost = 5.0
		}
		if viewer.ImmortalStones < tntCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Nguyên Thạch!"})
			return
		}
		_, _ = db.DB.ExecContext(c.Request.Context(), `UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, tntCost, viewer.ID)
	} else {
		// Fixed mana cost, never a percentage.
		manaCost := resonanceScanManaCost
		if viewer.CurrentMana < manaCost || viewer.PrimevalStones < 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cần 50 Chân Nguyên và 200 Nguyên Thạch để cảm ứng Cổ trùng!"})
			return
		}
		_, _ = db.DB.ExecContext(c.Request.Context(), `
			UPDATE players
			SET current_mana = current_mana - $1, primeval_stones = primeval_stones - 200
			WHERE id = $2
		`, manaCost, viewer.ID)
	}

	// Query nearby active players in scan region
	scanRadius := float64(viewer.Rank * 20)
	if viewer.Rank >= 6 {
		scanRadius = 500.0 // Whole region
	}

	pings := []models.ResonancePing{}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT p.id, p.x, p.y, `+db.GuInventoryAggregate("p.id")+`
		FROM players p
		WHERE p.current_region = $1 AND p.id != $2
	`, scanRegion, viewer.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pID uuid.UUID
			var pX, pY int
			var pGus models.GuInventory
			if err := rows.Scan(&pID, &pX, &pY, &pGus); err == nil {
				dist := math.Hypot(float64(viewer.X-pX), float64(viewer.Y-pY))
				if !isCrossRegion && dist > scanRadius {
					continue
				}

				// Check if target has concealment
				hasConceal := false
				for _, g := range models.FilterEquippedActive(pGus) {
					if spec, ok := models.GetGu(g.GuID, g.Rank); ok {
						_, conceal := models.GuNativeMechanic(spec, "CONCEAL_GU")
						_, perfect := models.GuNativeMechanic(spec, "PERFECT_DISGUISE")
						hasConceal = conceal || perfect
					}
				}

				if !hasConceal {
					// Check matching gus
					for _, g := range models.FilterEquippedActive(pGus) {
						if spec, ok := models.GetGu(g.GuID, g.Rank); ok {
							matchedPath := req.TargetPath == ""
							if !matchedPath {
								for _, p := range spec.Path {
									if strings.EqualFold(p, req.TargetPath) {
										matchedPath = true
										break
									}
								}
							}
							if matchedPath {
								if req.MinRank == 0 || g.Rank >= req.MinRank {
									pings = append(pings, models.ResonancePing{
										X:          pX,
										Y:          pY,
										Intensity:  "high",
										GuID:       g.GuID,
										EntityType: "player",
									})
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// Query world residual traces (Dao Pulses)
	daoPulses := []models.DaoPulsePing{}
	traceRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT region_id, x, y, dominant_path, action_type, EXTRACT(EPOCH FROM (expires_at - NOW()))::int
		FROM world_residual_traces
		WHERE region_id = $1 AND expires_at > NOW()
	`, scanRegion)
	if err == nil {
		defer traceRows.Close()
		for traceRows.Next() {
			var d models.DaoPulsePing
			if err := traceRows.Scan(&d.Region, &d.X, &d.Y, &d.Path, &d.Event, &d.ExpiresInSeconds); err == nil {
				daoPulses = append(daoPulses, d)
			}
		}
	}

	c.JSON(http.StatusOK, models.ResonanceScanResponse{
		Pings:           pings,
		DaoPulses:       daoPulses,
		CooldownSeconds: 120,
	})
}

// ExecuteDivineGu handles deducing a Gu worm without needing a target player
func ExecuteDivineGu(c *gin.Context) {
	viewerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.DivineGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	req.GuQuery = strings.TrimSpace(req.GuQuery)
	if req.GuQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập tên hoặc ID Cổ trùng cần suy tính"})
		return
	}

	// 1. Fetch Viewer
	var viewer models.Player
	var viewerDaoMarks models.DaoMarksMap
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_mana, p.max_mana, p.immortal_stones,
		       p.primeval_stones, p.hp, p.base_hp, p.soul_hp, p.max_soul, p.current_region, p.x, p.y,
		       `+db.GuInventoryAggregate("p.id")+`, p.path_masteries
		FROM players p
		WHERE p.id = $1
	`, viewerID).Scan(
		&viewer.ID, &viewer.Username, &viewer.Rank, &viewer.SubRank, &viewer.CurrentMana, &viewer.MaxMana,
		&viewer.ImmortalStones, &viewer.PrimevalStones, &viewer.HP, &viewer.BaseHP, &viewer.SoulHP, &viewer.MaxSoul,
		&viewer.CurrentRegion, &viewer.X, &viewer.Y, &viewer.GuInventory, &viewer.PathMasteries,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin người suy diễn: " + err.Error()})
		return
	}

	if viewer.Rank >= 6 {
		_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, viewer.ID).Scan(&viewerDaoMarks)
	}

	// 2. Resource Validation & Deduction (stones only, never mana)
	primevalCost := 0.0
	immortalStoneCost := 0.0
	resourceBonus := 0.0

	if req.ResourceType == "immortal_stone" {
		if viewer.Rank < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ Tiên nhân mới có thể vận dụng Tiên Nguyên Thạch để suy diễn!"})
			return
		}
		immortalStoneCost = req.ResourceAmount
		if immortalStoneCost < 1.0 {
			immortalStoneCost = 1.0
		}
		if viewer.ImmortalStones < immortalStoneCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Nguyên Thạch!"})
			return
		}
		resourceBonus = immortalStoneCost * 6.0
	} else {
		// Primeval stones (legacy "mana" accepted as alias)
		primevalCost = req.ResourceAmount
		if primevalCost < divinationPrimevalMin {
			primevalCost = divinationPrimevalMin
		}
		if viewer.PrimevalStones < primevalCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nguyên Thạch không đủ để thi triển suy tính!"})
			return
		}
		resourceBonus = primevalCost * divineGuPrimevalFactor
	}

	// 3. Resolve Target Gu: Name / Query -> Gu ID & Spec from Dictionary
	targetGu, found := models.FindGuByNameOrID(req.GuQuery, req.TargetRank)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thiên cơ mờ mịt, không tìm thấy cổ trùng này trong thiên địa thư quyển!"})
		return
	}

	matchIDs := pq.Array([]string{targetGu.ID, strings.ToLower(targetGu.ID), strings.ToUpper(targetGu.ID)})

	// 4. Query Holders across all 3 domains: Players, Sect Vaults, and Market/Auctions
	type holderData struct {
		holderType     string
		holderName     string
		holderUsername string
		region         string
		x, y           int
		hasCoords      bool
		isVital        bool
		equipped       bool
		state          string
		quantity       int
		priceInfo      string
		activeConceal  bool
		defense        float64
	}
	var allHolders []holderData
	maxHolderDefense := 0.0
	playerCount := 0
	sectVaultCount := 0
	marketCount := 0
	groundDropCount := 0

	// 4a. Player Inventories & Apertures (player_gu)
	queryPlayers := `
		SELECT p.id, p.username, p.current_region, p.x, p.y, p.rank, p.sub_rank,
		       COALESCE(pg.gu_rank, pg.rank, 1), pg.state, pg.equipped, pg.is_vital, pg.location,
		       p.is_disguised, (p.active_concealment IS NOT NULL AND p.active_concealment != '{}'::jsonb),
		       ` + db.GuInventoryAggregate("p.id") + `, p.path_masteries
		FROM player_gu pg
		JOIN players p ON pg.player_id = p.id
		WHERE (LOWER(pg.gu_id) = LOWER($1) OR pg.gu_id = ANY($3))
		  AND ($2 = 0 OR pg.gu_rank = $2 OR pg.rank = $2)
		  AND p.id <> '00000000-0000-0000-0000-000000000000'::uuid
	`
	pRows, err := db.DB.QueryContext(c.Request.Context(), queryPlayers, targetGu.ID, targetGu.Rank, matchIDs)
	if err != nil {
		if logger.S != nil {
			logger.S.Errorf("scouting: ExecuteDivineGu queryPlayers error: %v", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm chứng đầy đủ vị trí Cổ trùng"})
		return
	} else {
		defer pRows.Close()
		for pRows.Next() {
			var pID uuid.UUID
			var pUsername, pRegion, pState, pLocation string
			var pX, pY, pRank, pSubRank, pgRank int
			var pEquipped, pIsVital, pDisguised, pActiveConceal bool
			var pGus models.GuInventory
			var pMasteries models.PathMasteriesMap

			if err := pRows.Scan(
				&pID, &pUsername, &pRegion, &pX, &pY, &pRank, &pSubRank,
				&pgRank, &pState, &pEquipped, &pIsVital, &pLocation,
				&pDisguised, &pActiveConceal, &pGus, &pMasteries,
			); err != nil {
				if logger.S != nil {
					logger.S.Errorf("scouting: ExecuteDivineGu player scan error: %v", err)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu Cổ trùng trong túi người chơi không hợp lệ"})
				return
			}
			targetPlayer := models.Player{
				Account:          models.Account{ID: pID, Username: pUsername, IsDisguised: pDisguised},
				CultivationState: models.CultivationState{Rank: pRank, SubRank: pSubRank},
				WorldState:       models.WorldState{CurrentRegion: pRegion, X: pX, Y: pY},
				DaoState:         models.DaoState{PathMasteries: pMasteries},
			}
			def := gamefx.CalculateConcealmentDefense(&targetPlayer, models.FilterEquippedActive(pGus), nil, viewer.CurrentRegion)
			if pActiveConceal {
				def += 50.0
			}
			// Cổ nằm kho/ngủ chỉ hưởng một nửa che giấu của chủ; Cổ trang bị hưởng full.
			effDef := def
			if !pEquipped {
				effDef = def * 0.5
			}
			if effDef > maxHolderDefense {
				maxHolderDefense = effDef
			}

			locState := "inventory"
			if pEquipped {
				locState = "equipped"
			} else if pState == "SLEEPING" {
				locState = "sleeping"
			}

			allHolders = append(allHolders, holderData{
				holderType:     "player",
				holderName:     pUsername,
				holderUsername: pUsername,
				region:         pRegion,
				x:              pX,
				y:              pY,
				hasCoords:      true,
				isVital:        pIsVital,
				equipped:       pEquipped,
				state:          locState,
				quantity:       1,
				activeConceal:  pActiveConceal || pDisguised,
				defense:        def,
			})
			playerCount++
		}
		if err := pRows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc hết vị trí Cổ trùng trong túi người chơi"})
			return
		}
	}

	// 4b. Sect Vaults (sect_vault_gu)
	querySects := `
		SELECT s.id, s.name, s.region, COALESCE(s.map_x, 0), COALESCE(s.map_y, 0), s.level,
		       vg.gu_id, vg.rank, vg.quantity, vg.cost
		FROM sect_vault_gu vg
		JOIN sects s ON s.id = vg.sect_id
		WHERE (LOWER(vg.gu_id) = LOWER($1) OR vg.gu_id = ANY($3))
		  AND ($2 = 0 OR vg.rank = $2)
		  AND vg.quantity > 0
	`
	sRows, err := db.DB.QueryContext(c.Request.Context(), querySects, targetGu.ID, targetGu.Rank, matchIDs)
	if err != nil {
		if logger.S != nil {
			logger.S.Errorf("scouting: ExecuteDivineGu querySects error: %v", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm chứng đầy đủ kho Tông Môn"})
		return
	} else {
		defer sRows.Close()
		for sRows.Next() {
			var sID, sName, sRegion, guID string
			var sMapX, sMapY, sLevel, guRank, qty, cost int
			if err := sRows.Scan(&sID, &sName, &sRegion, &sMapX, &sMapY, &sLevel, &guID, &guRank, &qty, &cost); err != nil {
				if logger.S != nil {
					logger.S.Errorf("scouting: ExecuteDivineGu sect scan error: %v", err)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu Cổ trùng trong kho Tông Môn không hợp lệ"})
				return
			}
			sectDef := float64(sLevel*20 + 10)
			if sectDef > maxHolderDefense {
				maxHolderDefense = sectDef
			}

			allHolders = append(allHolders, holderData{
				holderType:    "sect_vault",
				holderName:    fmt.Sprintf("Bảo Khố Tông Môn [%s]", sName),
				region:        sRegion,
				x:             sMapX,
				y:             sMapY,
				hasCoords:     sMapX != 0 || sMapY != 0,
				state:         "sect_vault",
				quantity:      qty,
				priceInfo:     fmt.Sprintf("Cống hiến đổi: %d điểm", cost),
				activeConceal: false,
				defense:       sectDef,
			})
			sectVaultCount += qty
		}
		if err := sRows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc hết kho Tông Môn"})
			return
		}
	}

	// 4c. Marketplace & Auction Listings (auction)
	queryAuctions := `
		SELECT a.id, a.seller_id, p.username, p.current_region, p.x, p.y,
		       a.currency, a.start_price, a.current_price, a.listing_type,
		       COUNT(*)::int AS quantity,
		       MIN(CASE WHEN elem->>'rank' ~ '^[0-9]+$' THEN (elem->>'rank')::int ELSE 1 END) AS gu_rank
		FROM auction a
		JOIN players p ON a.seller_id = p.id,
		LATERAL jsonb_array_elements(
			CASE 
				WHEN jsonb_typeof(a.item_data) = 'array' THEN a.item_data 
				ELSE '[]'::jsonb 
			END
		) AS elem
		WHERE LOWER(a.item_type) = 'gu'
		  AND a.status IN ('active', 'ACTIVE')
		  AND a.end_time > NOW()
		  AND (LOWER(elem->>'gu_id') = LOWER($1) OR LOWER(elem->>'id') = LOWER($1) OR elem->>'gu_id' = ANY($3) OR elem->>'id' = ANY($3))
		  AND ($2 = 0 OR CASE WHEN elem->>'rank' ~ '^[0-9]+$' THEN (elem->>'rank')::int ELSE 1 END = $2)
		GROUP BY a.id, a.seller_id, p.username, p.current_region, p.x, p.y,
		         a.currency, a.start_price, a.current_price, a.listing_type
	`
	aRows, err := db.DB.QueryContext(c.Request.Context(), queryAuctions, targetGu.ID, targetGu.Rank, matchIDs)
	if err != nil {
		if logger.S != nil {
			logger.S.Errorf("scouting: ExecuteDivineGu queryAuctions error: %v", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm chứng đầy đủ Cổ trùng trên chợ"})
		return
	} else {
		defer aRows.Close()
		for aRows.Next() {
			var aID uuid.UUID
			var aSellerID uuid.UUID
			var sellerName, sellerRegion, currency, listingType string
			var sellerX, sellerY, guRank, qty int
			var startPrice, currentPrice float64
			if err := aRows.Scan(
				&aID, &aSellerID, &sellerName, &sellerRegion, &sellerX, &sellerY,
				&currency, &startPrice, &currentPrice, &listingType, &qty, &guRank,
			); err != nil {
				if logger.S != nil {
					logger.S.Errorf("scouting: ExecuteDivineGu auction scan error: %v", err)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu niêm yết Cổ trùng không hợp lệ"})
				return
			}
			currLabel := "Nguyên Thạch"
			if strings.EqualFold(currency, "immortal") || strings.EqualFold(currency, "immortal_stone") || strings.EqualFold(currency, "immortal_stones") {
				currLabel = "Tiên Nguyên Thạch"
			}

			allHolders = append(allHolders, holderData{
				holderType:     "auction_listing",
				holderName:     fmt.Sprintf("Sàn Đấu Giá (Người bán: %s)", sellerName),
				holderUsername: sellerName,
				region:         sellerRegion,
				x:              sellerX,
				y:              sellerY,
				hasCoords:      false,
				state:          "auction_listing",
				quantity:       qty,
				priceInfo:      fmt.Sprintf("%.0f %s (%s)", currentPrice, currLabel, listingType),
				activeConceal:  false,
				defense:        15.0,
			})
			marketCount += qty
		}
		if err := aRows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc hết niêm yết Cổ trùng"})
			return
		}
	}

	// Gu escrowed by asynchronous trade offers and SELL buyback orders are
	// physically owned by the Bảo Hoàng Thiên system player. Classify only
	// active, unexpired escrow as market inventory instead of attributing it to
	// that synthetic player (or counting stale escrow forever).
	escrowRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT source, creator_name, region, x, y, gu_id, gu_rank
		FROM (
			SELECT 'trade_offer'::text AS source, p.username AS creator_name,
			       p.current_region AS region, p.x, p.y, pg.gu_id,
			       COALESCE(pg.gu_rank, pg.rank, 1) AS gu_rank
			FROM trade_offers t
			JOIN players p ON p.id = t.creator_id
			CROSS JOIN LATERAL jsonb_array_elements_text(
				CASE WHEN jsonb_typeof(t.offered_data->'gus') = 'array'
				     THEN t.offered_data->'gus' ELSE '[]'::jsonb END
			) gu_instance_id
			JOIN player_gu pg ON pg.id = gu_instance_id
			WHERE UPPER(t.status) = 'ACTIVE' AND t.end_time > NOW()
			UNION ALL
			SELECT 'buyback_listing'::text, p.username, p.current_region, p.x, p.y,
			       pg.gu_id, COALESCE(pg.gu_rank, pg.rank, 1)
			FROM buyback_orders b
			JOIN players p ON p.id = b.buyer_id
			CROSS JOIN LATERAL jsonb_array_elements_text(
				CASE WHEN jsonb_typeof(b.item_data) = 'array'
				     THEN b.item_data ELSE '[]'::jsonb END
			) gu_instance_id
			JOIN player_gu pg ON pg.id = gu_instance_id
			WHERE UPPER(b.status) = 'ACTIVE' AND UPPER(b.order_type) = 'SELL'
			  AND LOWER(b.item_type) = 'gu' AND b.end_time > NOW()
		) escrow
		WHERE (LOWER(gu_id) = LOWER($1) OR gu_id = ANY($3))
		  AND ($2 = 0 OR gu_rank = $2)
	`, targetGu.ID, targetGu.Rank, matchIDs)
	if err != nil {
		if logger.S != nil {
			logger.S.Errorf("scouting: ExecuteDivineGu queryMarketEscrow error: %v", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm chứng đầy đủ Cổ trùng ký gửi trên chợ"})
		return
	}
	defer escrowRows.Close()
	for escrowRows.Next() {
		var source, creatorName, region, guID string
		var x, y, guRank int
		if err := escrowRows.Scan(&source, &creatorName, &region, &x, &y, &guID, &guRank); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu Cổ trùng ký gửi trên chợ không hợp lệ"})
			return
		}
		label := "Đơn trao đổi"
		if source == "buyback_listing" {
			label = "Đơn rao bán"
		}
		allHolders = append(allHolders, holderData{
			holderType: source, holderName: fmt.Sprintf("%s (Người đăng: %s)", label, creatorName),
			holderUsername: creatorName, region: region, x: x, y: y,
			state: source, quantity: 1, priceInfo: "Ký gửi tại Bảo Hoàng Thiên", defense: 15,
		})
		marketCount++
	}
	if err := escrowRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc hết Cổ trùng ký gửi trên chợ"})
		return
	}

	// 4d. Unexpired Gu dropped on the world map. Death drops have already been
	// removed from player_gu, so omitting this domain incorrectly declares them
	// destroyed and can also violate Immortal Gu uniqueness.
	dropRows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT d.region_id, d.x, d.y, d.loser_name,
		       EXTRACT(EPOCH FROM (d.expires_at - NOW()))::int,
		       elem->>'gu_id',
		       CASE WHEN elem->>'rank' ~ '^[0-9]+$' THEN (elem->>'rank')::int ELSE 1 END
		FROM map_ground_drops d
		CROSS JOIN LATERAL jsonb_array_elements(
			CASE WHEN jsonb_typeof(d.loot_data->'gus') = 'array'
			     THEN d.loot_data->'gus' ELSE '[]'::jsonb END
		) elem
		WHERE d.expires_at > NOW()
		  AND (LOWER(elem->>'gu_id') = LOWER($1) OR elem->>'gu_id' = ANY($3))
		  AND ($2 = 0 OR CASE WHEN elem->>'rank' ~ '^[0-9]+$' THEN (elem->>'rank')::int ELSE 1 END = $2)
	`, targetGu.ID, targetGu.Rank, matchIDs)
	if err != nil {
		if logger.S != nil {
			logger.S.Errorf("scouting: ExecuteDivineGu queryGroundDrops error: %v", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm chứng đầy đủ Cổ trùng rơi trên bản đồ"})
		return
	} else {
		defer dropRows.Close()
		for dropRows.Next() {
			var region, loserName, guID string
			var x, y, expiresIn, guRank int
			if err := dropRows.Scan(&region, &x, &y, &loserName, &expiresIn, &guID, &guRank); err != nil {
				if logger.S != nil {
					logger.S.Errorf("scouting: ExecuteDivineGu ground drop scan error: %v", err)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu Cổ trùng rơi trên bản đồ không hợp lệ"})
				return
			}
			allHolders = append(allHolders, holderData{
				holderType: "ground_drop",
				holderName: fmt.Sprintf("Chiến lợi phẩm rơi của %s", loserName),
				region:     region, x: x, y: y, hasCoords: true,
				state: "ground_drop", quantity: 1,
				priceInfo: fmt.Sprintf("Tan biến sau %d giây", expiresIn),
				defense:   10,
			})
			groundDropCount++
		}
		if err := dropRows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc hết Cổ trùng rơi trên bản đồ"})
			return
		}
	}

	totalWorldCount := playerCount + sectVaultCount + marketCount + groundDropCount

	// 5. Query Recipe Clues
	var reqGusJSON, reqMatsJSON sql.NullString
	var successRate sql.NullFloat64
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT required_gus::text, required_materials::text, success_rate
		FROM refining_recipes
		WHERE target_gu_id = $1 AND ($2 = 0 OR target_rank = $2)
		LIMIT 1
	`, targetGu.ID, targetGu.Rank).Scan(&reqGusJSON, &reqMatsJSON, &successRate)

	// 6. Evaluate Divination (final delta only)
	statusType, delta, backfireHPRatio, backfireSoulRatio, myScore := gamefx.EvaluateGuDivination(
		&viewer,
		models.FilterEquippedActive(viewer.GuInventory),
		viewerDaoMarks,
		targetGu,
		totalWorldCount,
		maxHolderDefense,
		resourceBonus,
	)

	// 7. Transaction: Apply cost & backfire damage
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi transaction"})
		return
	}
	defer tx.Rollback()

	newStones := viewer.PrimevalStones - primevalCost
	if newStones < 0 {
		newStones = 0
	}
	newImmortalStones := viewer.ImmortalStones - immortalStoneCost
	if newImmortalStones < 0 {
		newImmortalStones = 0
	}

	newMana := viewer.CurrentMana
	newHP := viewer.HP
	newSoulHP := viewer.SoulHP
	var backfireDamage, backfireSoulDamage float64

	if statusType == "backfire" {
		// Fixed mana drain, never a percentage.
		newMana -= divinationBackfireMana
		if newMana < 0 {
			newMana = 0
		}
		// Bounded fraction of the diviner's own maxima, same rule as /divine.
		maxHP := gamefx.EffectiveMaxHP(&viewer)
		if maxHP <= 0 {
			maxHP = viewer.BaseHP
		}
		backfireDamage = maxHP * backfireHPRatio
		backfireSoulDamage = float64(viewer.MaxSoul) * backfireSoulRatio

		newHP -= backfireDamage
		if newHP < 1.0 {
			newHP = 1.0
		}
		newSoulHP -= int(backfireSoulDamage)
		if newSoulHP < 1 {
			newSoulHP = 1
		}
	}

	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, current_mana = $3, hp = $4, soul_hp = $5
		WHERE id = $6
	`, newStones, newImmortalStones, newMana, newHP, newSoulHP, viewer.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật tài nguyên: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu dữ liệu: " + err.Error()})
		return
	}

	// 8. Build Report
	isImmortal := targetGu.Rank >= 6
	isRefined := totalWorldCount > 0

	var loreStatus string
	if isImmortal {
		if isRefined {
			loreStatus = fmt.Sprintf("Tiên Cổ Đã Xuất Thế — Thiên địa độc nhất vô nhị! Hiện đang tồn tại trong thế giới (%d nơi lưu giữ).", totalWorldCount)
		} else {
			loreStatus = "Tiên Cổ Vô Chủ — Độc nhất vô nhị trong thiên địa! Hiện tại chưa ai luyện thành (hoặc đã bị hủy diệt / thất lạc). Cổ trùng đang lưu lạc hoang dã hoặc chờ đợi cổ sư luyện chế!"
		}
	} else {
		if isRefined {
			loreStatus = fmt.Sprintf("Phàm Cổ — Hiện có tổng cộng %d con trong thiên hạ (Cổ Sư: %d, Kho Tông Môn: %d, Đang Treo Bán: %d, Rơi trên bản đồ: %d).", totalWorldCount, playerCount, sectVaultCount, marketCount, groundDropCount)
		} else {
			loreStatus = "Phàm Cổ — Hiện không có bản thể nào được lưu giữ trong tay Cổ Sư, Tông Môn hay Chợ Đấu Giá."
		}
	}

	foodHint := ""
	if statusType == "full" || statusType == "partial" {
		foodHint = strings.Join(targetGu.FoodTypes, ", ")
		if foodHint == "" {
			foodHint = "Hấp thu thiên địa nguyên khí / tinh hoa đặc thù"
		}
	} else {
		foodHint = "Thiên cơ mờ ảo, chưa thấy rõ thức ăn"
	}

	recipeHint := ""
	if reqGusJSON.Valid && reqGusJSON.String != "" {
		if statusType == "full" {
			recipeHint = fmt.Sprintf("Bí Phương: Cần phối hợp %s và linh tài %s (Tỉ lệ cơ bản: %.1f%%)", reqGusJSON.String, reqMatsJSON.String, successRate.Float64*100)
		} else if statusType == "partial" {
			recipeHint = fmt.Sprintf("Manh mối luyện chế: Cần phối hợp cổ trùng thuộc lưu phái %s và linh tài đặc thù.", strings.Join(targetGu.Path, ", "))
		} else {
			recipeHint = "Bí phương bị đạo ngân thiên địa che phủ."
		}
	} else {
		recipeHint = "Chưa tìm thấy bí phương lưu truyền trong nhân gian."
	}

	naturalHabitat := fmt.Sprintf("Lưu phái: %s. Thường sinh trưởng ở nơi hội tụ đạo ngân thiên nhiên.", strings.Join(targetGu.Path, ", "))

	var holderInfos []models.DivineGuHolderInfo
	if statusType == "full" {
		for _, h := range allHolders {
			info := models.DivineGuHolderInfo{
				HolderType:     h.holderType,
				HolderUsername: h.holderUsername,
				HolderName:     h.holderName,
				Region:         h.region,
				IsVital:        h.isVital,
				Equipped:       h.equipped,
				LocationState:  h.state,
				Quantity:       h.quantity,
				PriceInfo:      h.priceInfo,
				Concealed:      h.activeConceal,
			}
			if h.hasCoords && strings.EqualFold(viewer.CurrentRegion, h.region) {
				info.Coordinates = &models.Coordinates{X: h.x, Y: h.y}
			}
			holderInfos = append(holderInfos, info)
		}
	} else if statusType == "partial" {
		for _, h := range allHolders {
			name := h.holderName
			if h.holderType == "player" {
				if h.activeConceal {
					name = "Cổ sư bí ẩn"
				} else if len(h.holderUsername) > 2 {
					name = h.holderUsername[:2] + "****"
				}
			}
			holderInfos = append(holderInfos, models.DivineGuHolderInfo{
				HolderType:     h.holderType,
				HolderUsername: name,
				HolderName:     name,
				Region:         h.region,
				Equipped:       h.equipped,
				LocationState:  h.state,
				Quantity:       h.quantity,
				PriceInfo:      h.priceInfo,
				Concealed:      h.activeConceal,
			})
		}
	}

	report := &models.DivineGuReport{
		GuID:            targetGu.ID,
		GuName:          targetGu.Name,
		Rank:            targetGu.Rank,
		Path:            strings.Join(targetGu.Path, ", "),
		IsRefined:       isRefined,
		WorldCount:      totalWorldCount,
		PlayerCount:     playerCount,
		SectVaultCount:  sectVaultCount,
		MarketCount:     marketCount,
		GroundDropCount: groundDropCount,
		Holders:         holderInfos,
		FoodHint:        foodHint,
		RecipeHint:      recipeHint,
		NaturalHabitat:  naturalHabitat,
		LoreStatus:      loreStatus,
	}

	var msg string
	switch statusType {
	case "full":
		msg = fmt.Sprintf("Suy tính hoàn toàn thấu triệt thiên cơ của [%s]!", targetGu.Name)
	case "partial":
		msg = fmt.Sprintf("Bắt được một luồng thiên cơ mờ ảo về [%s].", targetGu.Name)
	case "fogged":
		msg = fmt.Sprintf("Thiên cơ hỗn loạn, đạo ngân xung đột, chỉ nhìn thấy hình bóng mờ mịt của [%s].", targetGu.Name)
	case "backfire":
		msg = fmt.Sprintf("Dòm ngó thiên cơ của [%s] vượt quá năng lực! Bị Thiên Cơ Phản Phệ chịu %.0f sát thương linh hồn!", targetGu.Name, backfireDamage+backfireSoulDamage)
		report.Holders = nil
		report.FoodHint = "Bị phản phệ làm mù quáng thần thức"
		report.RecipeHint = "Bí phương bị đạo ngân tiên thiên phản phệ che khuất"
	}

	c.JSON(http.StatusOK, models.DivineGuResponse{
		StatusType:         statusType,
		DeltaScore:         delta,
		MyScore:            &myScore,
		Report:             report,
		BackfireDamage:     backfireDamage,
		BackfireSoulDamage: backfireSoulDamage,
		Message:            msg,
	})
}
