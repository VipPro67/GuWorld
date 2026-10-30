package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// Nghi thức Thăng Tiên 3 khâu (spec "Nát Khiếu → Dẫn Khí → Thả Cổ"):
//
//	Khâu 1 — Nát Khiếu: Ngũ Chuyển Đỉnh Phong có cổ bản mệnh liều mạng trùng kích
//	         không khiếu của mình cho đến khi vỡ nát, hấp dẫn 2 khí thiên địa.
//	Khâu 2 — Dẫn Khí: khảo nghiệm tâm tính + cân bằng 2 khí thiên địa với Nhân Khí.
//	         Nhân Khí dưới Min ⇒ thất bại; ngưỡng Nhân Khí định phẩm phúc địa
//	         Sơ/Trung/Thượng/Chí Cao. Sau đó Địa Tai bản yếu giáng lâm — phải sống sót.
//	Khâu 3 — Thả Cổ: thả cổ 5C của các lưu phái đạt Đại Sư vào khiếu để tiêu hủy,
//	         nhận lại 300 Đạo Ngân mỗi con theo lưu phái — xác định đạo chủ tu.
//
// Phiên Thăng Tiên có thời hạn (AscensionDeadline) kể từ lúc nát khiếu; hết hạn
// hoặc thất bại giữa chừng thì mất part chuẩn bị (chân nguyên đã đốt, 20% Nhân Khí)
// nhưng giữ nguyên 5C Đỉnh Phong.
const (
	// AscensionMinHumanQi là ngưỡng MIN Nhân Khí để Dẫn Khí thành công (Phúc Địa Sơ).
	AscensionMinHumanQi = 100000.0
	// AscensionMidHumanQi / AscensionHighHumanQi là ngưỡng Phúc Địa Trung / Thượng.
	AscensionMidHumanQi  = 230000.0
	AscensionHighHumanQi = 370000.0
	// AscensionExtremeHumanQi là ngưỡng Phúc Địa Chí Cao.
	AscensionExtremeHumanQi = 480000.0

	// AscensionDeadline là thời hạn phiên Thăng Tiên sau khi nát khiếu.
	AscensionDeadline = 2 * time.Hour

	// AscensionShatterBaseChance / step là tỉ lệ nát khiếu mỗi lần trùng kích.
	AscensionShatterBaseChance = 0.15
	AscensionShatterStepChance = 0.15
	AscensionShatterMaxChance  = 0.90
	AscensionShatterCostRatio  = 0.30

	// AscensionDaoMarksPerGu là Đạo Ngân nhận lại cho mỗi cổ 5C bị tiêu hủy khi thả cổ.
	AscensionDaoMarksPerGu = 300

	// AscensionFailQiLoss là tỉ lệ Nhân Khí hụt khi phiên thất bại giữa chừng.
	AscensionFailQiLoss = 0.20
)

// Ascension session steps.
const (
	AscensionStepShattering = "SHATTERING"
	AscensionStepGuidQi     = "GUID_QI"
	AscensionStepCalamity   = "CALAMITY"
	AscensionStepRelease    = "RELEASE"
	AscensionStepFailed     = "FAILED"
	AscensionStepCompleted  = "COMPLETED"
)

// AscensionGradeForHumanQi maps Nhân Khí to the Phúc Địa grade. An empty
// string means the Nhân Khí is below the MIN requirement (thăng tiên thất bại).
func AscensionGradeForHumanQi(qi float64) string {
	switch {
	case qi >= AscensionExtremeHumanQi:
		return "EXTREME"
	case qi >= AscensionHighHumanQi:
		return "HIGH"
	case qi >= AscensionMidHumanQi:
		return "MID"
	case qi >= AscensionMinHumanQi:
		return "LOW"
	}
	return ""
}

// AscensionGradeName returns the display name of a Phúc Địa grade.
func AscensionGradeName(grade string) string {
	switch grade {
	case "EXTREME":
		return "Chí Cao"
	case "HIGH":
		return "Thượng"
	case "MID":
		return "Trung"
	case "LOW":
		return "Sơ"
	}
	return "Không Xác Định"
}

// ascensionTemperOption is one choice inside a tâm tính trial stage.
type ascensionTemperOption struct {
	Text       string  `json:"text"`
	SoulHPCost int     `json:"soul_hp_cost"`
	ManaRatio  float64 `json:"mana_ratio"`  // fraction of max mana
	StonesCost float64 `json:"stones_cost"` // primeval stones
}

// ascensionTemperStages là khảo nghiệm tâm tính (3 tầng: Định Tâm → Ký Địa → Thông Thiên).
// Mỗi tầng phải chọn 1 phương án đủ điều kiện mới qua; không afforded phương án nào
// nghĩa là chuẩn bị thiếu — đợi hết thời hạn sẽ thất bại.
var ascensionTemperStages = [][]ascensionTemperOption{
	{
		{Text: "Tĩnh tọa điều tức, lấy thần thức áp chế tâm ma (tiêu hao 15 Soul HP)", SoulHPCost: 15},
		{Text: "Vận chân nguyên hòa giải 2 khí thiên địa (tiêu hao 25% Chân Nguyên tối đa)", ManaRatio: 0.25},
	},
	{
		{Text: "Mở huyệt Long Không, dẫn địa mạch nhập khiếu (tiêu hao 25 Soul HP)", SoulHPCost: 25},
		{Text: "Đốt nguyên thạch trấn yểm địa mạch (tiêu hao 2.000 Nguyên Thạch)", StonesCost: 2000},
	},
	{
		{Text: "Ngửa mặt đón thiên khí, lấy Nhân Khí làm mồi (tiêu hao 35 Soul HP)", SoulHPCost: 35},
		{Text: "Dốc toàn chân nguyên đón tiếp 2 khí thiên địa (tiêu hao 40% Chân Nguyên tối đa)", ManaRatio: 0.40},
	},
}

// ascensionSession mirrors one row of ascension_sessions.
type ascensionSession struct {
	ID            uuid.UUID  `json:"id"`
	PlayerID      uuid.UUID  `json:"player_id"`
	Step          string     `json:"step"`
	Attempts      int        `json:"attempts"`
	TemperStage   int        `json:"temper_stage"`
	TemperChoices []int      `json:"temper_choices"`
	AscensionQi   float64    `json:"ascension_qi"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func scanAscensionSession(row *sql.Row) (*ascensionSession, error) {
	var s ascensionSession
	var choices []byte
	var expires sql.NullTime
	err := row.Scan(&s.ID, &s.PlayerID, &s.Step, &s.Attempts, &s.TemperStage, &choices, &s.AscensionQi, &expires, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if len(choices) > 0 {
		_ = json.Unmarshal(choices, &s.TemperChoices)
	}
	if expires.Valid {
		t := expires.Time
		s.ExpiresAt = &t
	}
	return &s, nil
}

// loadAscensionSession reads the player's session, lazily expiring stale live
// sessions (deadline passed) with the mid-ritual failure penalty.
func loadAscensionSession(tx *sql.Tx, player *models.Player) (*ascensionSession, error) {
	row := tx.QueryRow(`
		SELECT id, player_id, step, attempts, temper_stage, temper_choices, ascension_qi, expires_at, created_at, updated_at
		FROM ascension_sessions WHERE player_id = $1
	`, player.ID)
	s, err := scanAscensionSession(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if s.Step != AscensionStepShattering && s.Step != AscensionStepGuidQi && s.Step != AscensionStepCalamity && s.Step != AscensionStepRelease {
		return s, nil
	}
	if s.Step != AscensionStepShattering && s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt) {
		if failErr := failAscensionSession(tx, player, s, "Hết thời hạn Thăng Tiên! 2 khí thiên địa tán loạn, khiếu hư tổn."); failErr != nil {
			return nil, failErr
		}
		return s, nil
	}
	return s, nil
}

// failAscensionSession marks the session FAILED and applies the mid-ritual
// penalty: lose part of the Nhân Khí, keep Rank 5 Peak.
func failAscensionSession(tx *sql.Tx, player *models.Player, s *ascensionSession, reason string) error {
	qiLoss := 0.0
	if player.HumanQi > 0 {
		qiLoss = player.HumanQi * AscensionFailQiLoss
		player.HumanQi -= qiLoss
	}
	player.WallProgress = 1.0 // giữ 5C Đỉnh Phong, chỉ mất part chuẩn bị
	if _, err := tx.Exec(`
		UPDATE players SET human_qi = $1, wall_progress = $2 WHERE id = $3
	`, player.HumanQi, player.WallProgress, player.ID); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE ascension_sessions SET step = $1, updated_at = NOW() WHERE id = $2`, AscensionStepFailed, s.ID)
	if err != nil {
		return err
	}
	s.Step = AscensionStepFailed
	SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Đạo hữu %s thất bại trong nghi thức Thăng Tiên (%s)", player.Account.Username, reason))
	return nil
}

// hasVitalGu reports whether the player owns an intact (non-dead) vital Gu.
// Khâu 1 (Nát Khiếu) yêu cầu cổ bản mệnh trấn giữ không khiếu.
func hasVitalGu(player *models.Player) bool {
	for _, inst := range player.GuInventory {
		if inst.IsVital && inst.Status != "DEAD" {
			return true
		}
	}
	return false
}

// eligibleAscensionReleaseGu returns the player's rank-5 Gu whose path has
// reached Đại Sư (level 1) — the pool for the Thả Cổ phase.
func eligibleAscensionReleaseGu(player *models.Player) []models.GuInstance {
	eligible := []models.GuInstance{}
	for _, inst := range player.GuInventory {
		if inst.Rank != 5 || inst.IsVital || inst.Status == "DEAD" {
			continue
		}
		spec, ok := models.GetGu(inst.GuID, inst.Rank)
		if !ok {
			continue
		}
		for _, p := range spec.Path {
			clean := gamefx.CleanPath(p)
			if clean == "" || clean == heavenDaoPath {
				continue
			}
			if gamefx.PathMasteryLevel(player.PathMasteries[clean]) >= 1 {
				eligible = append(eligible, inst)
				break
			}
		}
	}
	return eligible
}

// ── Handlers ──────────────────────────────────────────────────────────────

// GetAscensionSession returns the player's current ascension session state,
// including preparation checklist data for the UI.
func GetAscensionSession(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	session, err := loadAscensionSession(tx, &player)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn phiên Thăng Tiên"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	resp := gin.H{
		"session":      session,
		"has_vital_gu": hasVitalGu(&player),
		"min_human_qi": AscensionMinHumanQi,
		"grade_thresholds": gin.H{
			"LOW":     AscensionMinHumanQi,
			"MID":     AscensionMidHumanQi,
			"HIGH":    AscensionHighHumanQi,
			"EXTREME": AscensionExtremeHumanQi,
		},
	}
	if session != nil && session.Step == AscensionStepRelease {
		resp["eligible_gu"] = eligibleAscensionReleaseGu(&player)
	}
	resp["temper_stages"] = ascensionTemperStages
	c.JSON(http.StatusOK, resp)
}

// AscensionShatter implements Khâu 1: repeated risky wall-shattering attempts.
// Each attempt burns a large chunk of primeval essence; the chance to shatter
// grows with every failed attempt. Requires a vital Gu.
func AscensionShatter(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	if gamefx.HasBlockedAction(&player, gamefx.ActionCultivate) {
		c.JSON(http.StatusForbidden, gin.H{"error": statusBlockedMessage(&player, gamefx.ActionCultivate)})
		return
	}
	if gamefx.HasBlockedAction(&player, gamefx.ActionBreakthrough) {
		c.JSON(http.StatusForbidden, gin.H{"error": statusBlockedMessage(&player, gamefx.ActionBreakthrough)})
		return
	}

	// Must be Rank 5 Peak with full wall progress.
	if player.Rank != 5 || player.SubRank != 4 || player.WallProgress < 1.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phải đạt Ngũ Chuyển Đỉnh Phong với vách khiếu đủ 100% mới có thể nát khiếu thăng tiên"})
		return
	}

	// Khâu 1 yêu cầu cổ bản mệnh trấn giữ không khiếu.
	if !hasVitalGu(&player) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phải có Cổ Bản Mệnh trấn giữ không khiếu mới dám liều mạng nát khiếu!"})
		return
	}

	session, err := loadAscensionSession(tx, &player)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn phiên Thăng Tiên"})
		return
	}
	if session != nil {
		switch session.Step {
		case AscensionStepGuidQi, AscensionStepCalamity, AscensionStepRelease:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phiên Thăng Tiên đang diễn ra, hãy hoàn thành các khâu còn lại!"})
			return
		case AscensionStepShattering:
			// continue accumulating attempts
		default: // FAILED / COMPLETED — start over
			if _, err := tx.Exec(`DELETE FROM ascension_sessions WHERE id = $1`, session.ID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi reset phiên Thăng Tiên"})
				return
			}
			session = nil
		}
	}

	// Cost: 30% of max mana — liều mạng phải trả giá bằng chân nguyên.
	cost := math.Max(1.0, gamefx.EffectiveMaxMana(&player)*AscensionShatterCostRatio)
	if player.CurrentMana < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần %.0f Chân Nguyên cho mỗi lần trùng kích nát khiếu (Hiện có: %.0f)", cost, player.CurrentMana)})
		return
	}
	player.CurrentMana -= cost
	player.LastManaUpdate = time.Now()

	attempts := 0
	if session != nil {
		attempts = session.Attempts
	}
	chance := math.Min(AscensionShatterMaxChance, AscensionShatterBaseChance+AscensionShatterStepChance*float64(attempts))
	shattered := rand.Float64() < chance

	if shattered {
		expires := time.Now().Add(AscensionDeadline)
		if session == nil {
			_, err = tx.Exec(`
				INSERT INTO ascension_sessions (player_id, step, attempts, temper_stage, temper_choices, ascension_qi, expires_at)
				VALUES ($1, $2, $3, 0, '[]'::jsonb, 0, $4)
			`, player.ID, AscensionStepGuidQi, attempts+1, expires)
		} else {
			_, err = tx.Exec(`
				UPDATE ascension_sessions SET step = $1, attempts = $2, expires_at = $3, updated_at = NOW() WHERE id = $4
			`, AscensionStepGuidQi, attempts+1, expires, session.ID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi mở phiên Thăng Tiên"})
			return
		}
	} else {
		if session == nil {
			_, err = tx.Exec(`
				INSERT INTO ascension_sessions (player_id, step, attempts, temper_stage, temper_choices, ascension_qi)
				VALUES ($1, $2, 1, 0, '[]'::jsonb, 0)
			`, player.ID, AscensionStepShattering)
		} else {
			_, err = tx.Exec(`UPDATE ascension_sessions SET attempts = $1, updated_at = NOW() WHERE id = $2`, attempts+1, session.ID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên Thăng Tiên"})
			return
		}
	}

	if _, err := tx.Exec(`UPDATE players SET current_mana = $1, last_mana_update = $2 WHERE id = $3`, player.CurrentMana, player.LastManaUpdate, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật chân nguyên"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)

	msg := fmt.Sprintf("Trùng kích nát khiếu thất bại! (Tỉ lệ vỡ: %.0f%%) Không khiếu rung chuyển dữ dội, hãy dồn chân nguyên tiếp.", chance*100)
	if shattered {
		msg = fmt.Sprintf("Không khiếu vỡ nát! 2 khí thiên địa trào dâng. Đạo hữu có %s để Dẫn Khí, đấu Địa Tai và Thả Cổ — thất bại sẽ mất hết chuẩn bị!", AscensionDeadline.String())
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "ascension.shatter",
		EntityType:     "player",
		EntityID:       player.ID.String(),
		RequestSummary: fmt.Sprintf("Trùng kích nát khiếu lần %d (tiêu hao %.0f chân nguyên): vỡ=%v", attempts+1, cost, shattered),
		Metadata: map[string]interface{}{
			"attempt":   attempts + 1,
			"chance":    chance,
			"shattered": shattered,
			"mana_cost": cost,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "shattered": shattered, "message": msg})
}

// AscensionTemperRequest selects one option in the current tâm tính stage.
type AscensionTemperRequest struct {
	OptionIndex int `json:"option_index" binding:"required"`
}

// AscensionTemper implements Khâu 2a: the tâm tính trial + the Nhân Khí balance.
// Completing the last stage locks in the Phúc Địa grade from Nhân Khí (below the
// MIN requirement = failure) and moves the session to the CALAMITY step.
func AscensionTemper(c *gin.Context) {
	var req AscensionTemperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	session, err := loadAscensionSession(tx, &player)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn phiên Thăng Tiên"})
		return
	}
	if session == nil || session.Step != AscensionStepGuidQi {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong giai đoạn Dẫn Khí của nghi thức Thăng Tiên"})
		return
	}
	if session.TemperStage >= len(ascensionTemperStages) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khảo nghiệm tâm tính đã hoàn thành"})
		return
	}

	stage := ascensionTemperStages[session.TemperStage]
	if req.OptionIndex < 0 || req.OptionIndex >= len(stage) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phương án không hợp lệ"})
		return
	}
	opt := stage[req.OptionIndex]

	// Affordability checks.
	if player.SoulHP < opt.SoulHPCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Soul HP (cần %d, hiện có %d)", opt.SoulHPCost, player.SoulHP)})
		return
	}
	manaCost := gamefx.EffectiveMaxMana(&player) * opt.ManaRatio
	if manaCost > 0 && player.CurrentMana < manaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Chân Nguyên (cần %.0f, hiện có %.0f)", manaCost, player.CurrentMana)})
		return
	}
	if opt.StonesCost > 0 && player.PrimevalStones < opt.StonesCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch (cần %.0f, hiện có %.0f)", opt.StonesCost, player.PrimevalStones)})
		return
	}

	// Apply costs.
	player.SoulHP -= opt.SoulHPCost
	if player.SoulHP < 0 {
		player.SoulHP = 0
	}
	player.CurrentMana -= manaCost
	player.PrimevalStones -= opt.StonesCost

	newStage := session.TemperStage + 1
	choices := append(session.TemperChoices, req.OptionIndex)

	if newStage >= len(ascensionTemperStages) {
		// Final balance check: Nhân Khí below the MIN requirement = thất bại.
		if player.HumanQi < AscensionMinHumanQi {
			if err := failAscensionSession(tx, &player, session, "Nhân Khí bất túc, không thể cân bằng 2 khí thiên địa"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên Thăng Tiên"})
				return
			}
			if _, err := tx.Exec(`UPDATE players SET soul_hp = $1, current_mana = $2, primeval_stones = $3 WHERE id = $4`, player.SoulHP, player.CurrentMana, player.PrimevalStones, player.ID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cổ sư"})
				return
			}
			if err := tx.Commit(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
				return
			}
			ClearPendingPlayerQueues(player.ID.String())
			PublishPlayerSnapshot(&player, PlayerEventUpdated)
			c.JSON(http.StatusOK, gin.H{"success": false, "session": session, "message": "Nhân Khí dưới ngưỡng tối thiểu 100.000 — không thể cân bằng 2 khí thiên địa. Thăng tiên thất bại!"})
			return
		}
		// Grade is locked in from the current Nhân Khí.
		_, err = tx.Exec(`
			UPDATE ascension_sessions SET step = $1, temper_stage = $2, temper_choices = $3, ascension_qi = $4, updated_at = NOW() WHERE id = $5
		`, AscensionStepCalamity, newStage, choicesJSON(choices), player.HumanQi, session.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên Thăng Tiên"})
			return
		}
		session.Step = AscensionStepCalamity
		session.TemperStage = newStage
		session.TemperChoices = choices
		session.AscensionQi = player.HumanQi
	} else {
		_, err = tx.Exec(`
			UPDATE ascension_sessions SET temper_stage = $1, temper_choices = $2, updated_at = NOW() WHERE id = $3
		`, newStage, choicesJSON(choices), session.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên Thăng Tiên"})
			return
		}
		session.TemperStage = newStage
		session.TemperChoices = choices
	}

	if _, err := tx.Exec(`UPDATE players SET soul_hp = $1, current_mana = $2, primeval_stones = $3 WHERE id = $4`, player.SoulHP, player.CurrentMana, player.PrimevalStones, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cổ sư"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)

	msg := "Khảo nghiệm tâm tính vượt qua. Hãy tiếp tục điều hòa 2 khí thiên địa."
	if session.Step == AscensionStepCalamity {
		grade := AscensionGradeForHumanQi(session.AscensionQi)
		msg = fmt.Sprintf("2 khí thiên địa đã thuận theo Nhân Khí! Phúc Địa định phẩm: %s. ĐỊA TAI BẢN YẾU giáng lâm — chuẩn bị nghênh chiến!", AscensionGradeName(grade))
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "ascension.temper",
		EntityType:     "player",
		EntityID:       player.ID.String(),
		RequestSummary: fmt.Sprintf("Khảo nghiệm tâm tính tầng %d/%d (phương án %d)", session.TemperStage, len(ascensionTemperStages), req.OptionIndex),
		Metadata: map[string]interface{}{
			"stage":        session.TemperStage,
			"option":       req.OptionIndex,
			"completed":    session.Step == AscensionStepCalamity,
			"ascension_qi": session.AscensionQi,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "session": session, "message": msg})
}

// AscensionEngageCalamity starts Địa Tai bản yếu as an isolated realtime fight.
func AscensionEngageCalamity(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if reason := RealtimeRuntimeError(); reason != "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "realtime_unavailable", "error": "Địa Tai realtime hiện chưa khả dụng."})
		return
	}
	handedOff, err := middleware.BeginRealtimeHandoff(authID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "realtime_checkpoint_unavailable", "error": "Không thể đồng bộ trạng thái trước khi vào Địa Tai."})
		return
	}
	if handedOff {
		defer middleware.AbortRealtimeHandoff(authID)
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin cổ sư"})
		return
	}
	if err := checkRegionAction(player.CurrentRegion, "combat"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if err := checkRegionEquippedGus(player.CurrentRegion, player.GuInventory); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	session, err := loadAscensionSession(tx, &player)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn phiên Thăng Tiên"})
		return
	}
	if session == nil || session.Step != AscensionStepCalamity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Địa Tai chưa giáng lâm — hãy hoàn thành Dẫn Khí trước"})
		return
	}

	// Build the Địa Tai bản yếu wave: one earth-calamity boss + one minion,
	// scaled to the ascending player's own rank (5).
	const bossTemplateID = "monster_earth_calamity_boss"
	var boss models.MonsterTemplate
	var bossGu []byte
	if err := tx.QueryRow(`
		SELECT id, name, type, max_hp, base_attack, base_defense, gu_inventory, ai_profile, loot_table_id, description
		FROM monster_templates WHERE id = $1
	`, bossTemplateID).Scan(
		&boss.ID, &boss.Name, &boss.Type,
		&boss.MaxHP, &boss.BaseAttack, &boss.BaseDefense,
		&bossGu, &boss.AIProfile, &boss.LootTableID, &boss.Description,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy mẫu Địa Tai"})
		return
	}
	bossHP, bossAtk, bossDef := models.ScaleMonsterStats(boss.MaxHP, boss.BaseAttack, boss.BaseDefense, 5, 4, boss.Type)

	instanceID := uuid.New()
	config, err := json.Marshal(map[string]any{
		"template_id": bossTemplateID, "name": "Địa Tai Bản Yếu", "max_hp": bossHP,
		"attack_damage": bossAtk, "defense": bossDef, "rank": 5, "sub_rank": 4,
		"current_hp": player.HP, "current_mana": player.CurrentMana,
		"x": 20.0, "y": 24.0,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo phiên realtime"})
		return
	}
	var originZone, currentInstance string
	if err := tx.QueryRow(`
		SELECT COALESCE(NULLIF(world_zone_id,''),current_region), COALESCE(world_instance_id,'')
		FROM players WHERE id=$1
	`, player.ID).Scan(&originZone, &currentInstance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc vị trí hiện tại"})
		return
	}
	if currentInstance != "" {
		c.JSON(http.StatusConflict, gin.H{"code": "combat_instance_conflict", "error": "Nhân vật đang ở trong một phiên realtime riêng."})
		return
	}
	var hasPrivateCombat bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM realtime_combat_instances WHERE player_id=$1 AND status='active'
			UNION ALL
			SELECT 1 FROM realtime_combat_session_participants WHERE player_id=$1 AND status='active'
		)
	`, player.ID).Scan(&hasPrivateCombat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra phiên realtime đang mở"})
		return
	}
	if hasPrivateCombat {
		c.JSON(http.StatusConflict, gin.H{"code": "combat_instance_conflict", "error": "Nhân vật đang có một phiên realtime riêng chưa kết thúc."})
		return
	}
	if originZone == "" {
		originZone = player.CurrentRegion
	}
	if _, err := tx.Exec(`
		INSERT INTO realtime_combat_sessions (id,mode,zone_id,origin_zone_id,payload)
		VALUES ($1,'ascension',$2,$3,'{}'::jsonb)
	`, instanceID, practiceRealtimeMapID, originZone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phiên chiến đấu realtime"})
		return
	}
	if _, err := tx.Exec(`
		INSERT INTO realtime_combat_session_participants (session_id,player_id,payload)
		VALUES ($1,$2,$3)
	`, instanceID, player.ID, config); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể đăng ký nhân vật vào phiên realtime"})
		return
	}
	if _, err := tx.Exec(`
		UPDATE players SET world_zone_id=$2,world_instance_id=$3,
			world_location_revision=world_location_revision+1
		WHERE id=$1
	`, player.ID, practiceRealtimeMapID, "combat:"+instanceID.String()); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể chuyển vào bản đồ chiến đấu realtime"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}
	if handedOff {
		middleware.FinishRealtimeHandoff(authID)
	}

	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated)
	c.JSON(http.StatusOK, gin.H{
		"success": true, "message": "ĐỊA TAI BẢN YẾU giáng lâm trên bản đồ realtime riêng!",
		"instance_id": instanceID, "zone_id": practiceRealtimeMapID,
	})
}

func completeAscensionCalamity(tx *sql.Tx, playerID uuid.UUID) error {
	result, err := tx.Exec(`
		UPDATE ascension_sessions SET step = $1, updated_at = NOW()
		WHERE player_id = $2 AND step = $3
	`, AscensionStepRelease, playerID, AscensionStepCalamity)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count != 1 {
		return fmt.Errorf("expected one ascension calamity session for %s, updated %d", playerID, count)
	}
	return nil
}

// AscensionReleaseRequest lists the rank-5 Gu instances to sacrifice (Thả Cổ).
type AscensionReleaseRequest struct {
	GuInstanceIDs []string `json:"gu_instance_ids" binding:"required"`
}

// AscensionRelease implements Khâu 3: sacrifice the chosen rank-5 Gu of Đại Sư
// paths, receive 300 Dao Marks per Gu, and complete the ascension — Rank 6,
// the Tiên Khiếu opens at the grade locked during Dẫn Khí.
func AscensionRelease(c *gin.Context) {
	var req AscensionReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	session, err := loadAscensionSession(tx, &player)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn phiên Thăng Tiên"})
		return
	}
	if session == nil || session.Step != AscensionStepRelease {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa đủ điều kiện Thả Cổ — phải vượt Địa Tai bản yếu trước"})
		return
	}
	if len(req.GuInstanceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phải chọn ít nhất một cổ 5C để thả"})
		return
	}

	// Validate and destroy each chosen Gu, accumulating Dao Marks per path.
	daoGain := models.DaoMarksMap{}
	for _, idStr := range req.GuInstanceIDs {
		instID, errParse := uuid.Parse(idStr)
		if errParse != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mã cổ không hợp lệ: " + idStr})
			return
		}
		var guID string
		var guRank int
		var isVital bool
		err := tx.QueryRow(`
			SELECT gu_id, gu_rank, is_vital FROM player_gu
			WHERE id = $1 AND player_id = $2 AND status <> 'DEAD'
		`, instID, player.ID).Scan(&guID, &guRank, &isVital)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không tồn tại trong túi đồ"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn cổ trùng"})
			return
		}
		if guRank != 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể thả cổ Ngũ Chuyển (5C)"})
			return
		}
		if isVital {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thả Cổ Bản Mệnh — nó vừa trấn giữ không khiếu cho đạo hữu!"})
			return
		}
		spec, ok := models.GetGu(guID, guRank)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không có trong điển tịch"})
			return
		}
		path := ""
		for _, p := range spec.Path {
			clean := gamefx.CleanPath(p)
			if clean != "" && clean != heavenDaoPath {
				path = clean
				break
			}
		}
		if path == "" || gamefx.PathMasteryLevel(player.PathMasteries[path]) < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Lưu phái của %s chưa đạt cảnh giới Đại Sư (10.000 Chân Ý)", spec.Name)})
			return
		}
		if _, err := tx.Exec(`DELETE FROM player_gu WHERE id = $1 AND player_id = $2 AND is_vital = false`, instID, player.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hủy cổ trùng"})
			return
		}
		daoGain[path] += AscensionDaoMarksPerGu
	}
	if len(daoGain) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có lưu phái hợp lệ để thả cổ"})
		return
	}

	// Complete the ascension: Rank 6, Nhân Khí consumed, Tiên Khiếu opens.
	grade := AscensionGradeForHumanQi(session.AscensionQi)
	if grade == "" {
		if err := failAscensionSession(tx, &player, session, "Nhân Khí bất túc"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên Thăng Tiên"})
			return
		}
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
			return
		}
		ClearPendingPlayerQueues(player.ID.String())
		PublishPlayerSnapshot(&player, PlayerEventUpdated)
		c.JSON(http.StatusOK, gin.H{"success": false, "session": session, "message": "Thăng tiên thất bại!"})
		return
	}

	player.Rank = 6
	player.SubRank = 1
	player.HumanQi = 0.0
	nextTrib := time.Now().Add(10 * 24 * time.Hour)
	player.NextTribulationTime = &nextTrib
	player.ImmortalStones += 10

	ap, apErr := createAperture(tx, player.ID.String(), session.AscensionQi, player.CurrentMana)
	if apErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Thăng tiên thành công nhưng không thể khai mở Tiên Khiếu: " + apErr.Error()})
		return
	}
	// Merge the released-path Dao Marks into the fresh aperture.
	if ap.DaoMarks == nil {
		ap.DaoMarks = models.DaoMarksMap{}
	}
	for path, amount := range daoGain {
		ap.DaoMarks[path] += amount
	}
	if _, err := tx.Exec(`
		UPDATE immortal_apertures SET dao_marks = $1, updated_at = NOW() WHERE id = $2
	`, ap.DaoMarks, ap.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi ghi Đạo Ngân vào Tiên Khiếu"})
		return
	}

	if _, err := tx.Exec(`
		UPDATE players SET rank = $1, sub_rank = $2, human_qi = $3, next_tribulation_time = $4, immortal_stones = $5
		WHERE id = $6
	`, player.Rank, player.SubRank, player.HumanQi, player.NextTribulationTime, player.ImmortalStones, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cổ sư"})
		return
	}
	if _, err := tx.Exec(`UPDATE ascension_sessions SET step = $1, updated_at = NOW() WHERE id = $2`, AscensionStepCompleted, session.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên Thăng Tiên"})
		return
	}
	session.Step = AscensionStepCompleted

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	pathsStr := ""
	for path := range daoGain {
		if pathsStr != "" {
			pathsStr += ", "
		}
		pathsStr += strings.ToUpper(path[:1]) + path[1:]
	}
	SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Đạo hữu %s đã THĂNG TIÊN thành công — Lục Chuyển Cổ Tiên, Phúc Địa %s! Người người ngưỡng mộ", player.Account.Username, AscensionGradeName(grade)))

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "ascension.release",
		EntityType:     "player",
		EntityID:       player.ID.String(),
		RequestSummary: fmt.Sprintf("Thả %d cổ 5C (paths: %s) → %d Đạo Ngân; Thăng Tiên thành công, Phúc Địa %s", len(req.GuInstanceIDs), pathsStr, len(req.GuInstanceIDs)*AscensionDaoMarksPerGu, grade),
		Metadata: map[string]interface{}{
			"released":     len(req.GuInstanceIDs),
			"dao_gain":     daoGain,
			"grade":        grade,
			"ascension_qi": session.AscensionQi,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"session": session,
		"message": fmt.Sprintf("THĂNG TIÊN THÀNH CÔNG! Tiêu hủy %d cổ 5C, nhận %s Đạo Ngân. Phúc Địa %s đã khai mở — chính thức Lục Chuyển Cổ Tiên!", len(req.GuInstanceIDs), pathsStr, AscensionGradeName(grade)),
	})
}

// choicesJSON marshals temper option indices for the JSONB column.
func choicesJSON(choices []int) []byte {
	if choices == nil {
		choices = []int{}
	}
	b, _ := json.Marshal(choices)
	return b
}
