package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Refine timed sessions (WF-REF-05): /api/refine/sessions and friends.
// One ACTIVE session per player (partial unique index); everything is
// server-authoritative; the players row is only locked in create/resolve/abort
// transactions, never held across the waiting period.

const (
	refineCodeSessionActive    = "REFINE_SESSION_ACTIVE"
	refineCodeSessionNotFound  = "REFINE_SESSION_NOT_FOUND"
	refineCodeStepNotOpen      = "REFINE_STEP_NOT_OPEN"
	refineCodeStepClosed       = "REFINE_STEP_CLOSED"
	refineCodeStepDone         = "REFINE_STEP_ALREADY_DONE"
	refineCodeNonceInvalid     = "REFINE_NONCE_INVALID"
	refineCodePayloadInvalid   = "REFINE_PAYLOAD_INVALID"
	refineCodeGuLocked         = "REFINE_GU_LOCKED"
	refineCodeConfigMissing    = "REFINE_CONFIG_MISSING"
	refineCodeImmortalConflict = "REFINE_IMMORTAL_CONFLICT"
)

// refineStepResult is one graded step (WF-REF-05 §4.3).
type refineStepResult struct {
	Score           float64                `json:"score"`
	Action          string                 `json:"action"`
	Payload         map[string]interface{} `json:"payload,omitempty"`
	SubmittedAt     time.Time              `json:"submitted_at"`
	ReceiptOffsetMs int64                  `json:"receipt_offset_ms"`
}

// refineSessionRow mirrors one refine_sessions row.
type refineSessionRow struct {
	ID         string
	PlayerID   string
	RecipeID   string
	TargetGuID string
	TargetRank int
	Variant    int
	Status     string

	SelectedGuIDs     []string
	GuSnapshot        models.GuInventory
	MaterialsDeducted map[string]int64
	RateSnapshot      RefineRateSnapshot

	Plan        RefinePlan
	StepResults map[string]refineStepResult

	Accuracy     float64
	DeathPenalty float64

	ResultRaw []byte

	StartedAt      time.Time
	EndsAt         time.Time
	ResolvedAt     sql.NullTime
	AcknowledgedAt sql.NullTime
	DeathMarkedAt  sql.NullTime
}

const refineSessionSelect = `
	SELECT id, player_id, recipe_id, target_gu_id, target_rank, variant, status,
	       selected_gu_ids, gu_snapshot, materials_deducted, rate_snapshot,
	       plan, step_results, accuracy, death_penalty, COALESCE(result, '{}'::jsonb),
	       started_at, ends_at, resolved_at, acknowledged_at, death_marked_at
	FROM refine_sessions
`

const refineSessionSelectForUpdate = refineSessionSelect + ` WHERE id=$1 AND player_id=$2 FOR UPDATE`

func scanRefineSessionRow(scanner interface{ Scan(dest ...interface{}) error }) (*refineSessionRow, error) {
	var s refineSessionRow
	var selectedGuIDs []byte
	var guSnapshotRaw, materialsRaw, rateRaw, planRaw, stepResultsRaw []byte
	if err := scanner.Scan(
		&s.ID, &s.PlayerID, &s.RecipeID, &s.TargetGuID, &s.TargetRank, &s.Variant, &s.Status,
		&selectedGuIDs, &guSnapshotRaw, &materialsRaw, &rateRaw,
		&planRaw, &stepResultsRaw, &s.Accuracy, &s.DeathPenalty, &s.ResultRaw,
		&s.StartedAt, &s.EndsAt, &s.ResolvedAt, &s.AcknowledgedAt, &s.DeathMarkedAt,
	); err != nil {
		return nil, err
	}
	if err := pq.Array(&s.SelectedGuIDs).Scan(selectedGuIDs); err != nil {
		return nil, fmt.Errorf("scan selected_gu_ids: %w", err)
	}
	if len(guSnapshotRaw) > 0 {
		_ = json.Unmarshal(guSnapshotRaw, &s.GuSnapshot)
	}
	s.MaterialsDeducted = map[string]int64{}
	if len(materialsRaw) > 0 {
		_ = json.Unmarshal(materialsRaw, &s.MaterialsDeducted)
	}
	if len(rateRaw) > 0 {
		if err := json.Unmarshal(rateRaw, &s.RateSnapshot); err != nil {
			return nil, fmt.Errorf("scan rate_snapshot: %w", err)
		}
	}
	if len(planRaw) > 0 {
		if err := json.Unmarshal(planRaw, &s.Plan); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
	}
	s.StepResults = map[string]refineStepResult{}
	if len(stepResultsRaw) > 0 {
		if err := json.Unmarshal(stepResultsRaw, &s.StepResults); err != nil {
			return nil, fmt.Errorf("scan step_results: %w", err)
		}
	}
	return &s, nil
}

// ─── Public view (WF-REF-05 §9.1) ────────────────────────────────────────────

// publicSessionView builds the client-safe projection: no seed, no answers in
// CHOICE/plan config, no rate_snapshot — only the current step's public params.
func publicSessionView(s *refineSessionRow, now time.Time) gin.H {
	steps := make([]gin.H, 0, len(s.Plan.Steps))
	var stepToExpose *RefinePlanStep
	var nextStepAt *time.Time

	for _, st := range s.Plan.Steps {
		res, done := s.StepResults[strconv.Itoa(st.I)]
		state := "LOCKED"
		switch {
		case done:
			state = "DONE"
		case s.Status != "ACTIVE" || now.After(st.WindowEnd.Add(refineGraceDuration())):
			state = "MISSED"
		case now.Before(st.WindowStart):
			state = "LOCKED"
			if nextStepAt == nil {
				ws := st.WindowStart
				nextStepAt = &ws
			}
		default:
			state = "OPEN"
			if stepToExpose == nil {
				exposed := st
				stepToExpose = &exposed
			}
		}

		entry := gin.H{"index": st.I, "type": st.Type, "state": state}
		if done {
			entry["score"] = res.Score
		} else {
			entry["score"] = nil
		}
		steps = append(steps, entry)
	}

	view := gin.H{
		"id":          s.ID,
		"status":      s.Status,
		"recipe_id":   s.RecipeID,
		"target_gu_id": s.TargetGuID,
		"target_rank": s.TargetRank,
		"variant":     s.Variant,
		"started_at":  s.StartedAt,
		"ends_at":     s.EndsAt,
		"accuracy":    s.Accuracy,
		"server_now":  now,
		"steps":       steps,
		"step":        nil,
		"next_step_at": nextStepAt,
		"result":      nil,
	}

	if stepToExpose != nil {
		params := publicStepParams(*stepToExpose)
		view["step"] = gin.H{
			"index":        stepToExpose.I,
			"type":         stepToExpose.Type,
			"nonce":        stepToExpose.Nonce,
			"window_start": stepToExpose.WindowStart,
			"window_end":   stepToExpose.WindowEnd,
			"params":       params,
		}
	}

	if s.Status == "RESOLVED" || s.Status == "ABORTED" {
		var result map[string]interface{}
		if len(s.ResultRaw) > 0 {
			_ = json.Unmarshal(s.ResultRaw, &result)
		}
		view["result"] = result
		view["resolved_at"] = s.ResolvedAt
	}
	if s.AcknowledgedAt.Valid {
		view["acknowledged_at"] = s.AcknowledgedAt
	}
	return view
}

func refineGraceDuration() time.Duration {
	return time.Duration(refineSubmitWindowGraceNs)
}

// publicStepParams exposes only the params needed to render the minigame
// (WF-REF-04 §4). The client must know them to draw the UI — the accepted
// anti-bot trade-off of v1.
func publicStepParams(step RefinePlanStep) gin.H {
	switch step.Type {
	case RefineStepTap:
		return gin.H{
			"marker_period_ms": step.Config.MarkerPeriodMs,
			"center":           step.Config.Center,
			"half_width":       step.Config.HalfWidth,
		}
	case RefineStepSequence:
		return gin.H{
			"symbols":         step.Config.Symbols,
			"display_ms":      step.Config.DisplayMs,
			"target_input_ms": step.Config.TargetInputMs,
		}
	case RefineStepChoice:
		return gin.H{
			"situation": step.Config.Situation,
			"options":   step.Config.OptionTexts,
		}
	case RefineStepOrder:
		return gin.H{"items": step.Config.Items}
	}
	return gin.H{}
}

// publishRefineSession pushes the public session view on WS channel refine:.
func publishRefineSession(playerID string, view gin.H) {
	if cache.Client == nil {
		return
	}
	payload := gin.H{"type": "refine.session", "session": view, "revision": time.Now().UnixMilli()}
	if err := cache.PublishEventQuick("refine:"+playerID, payload); err != nil {
		logger.S.Warnf("publishRefineSession: %v", err)
	}
}

// ─── Create ──────────────────────────────────────────────────────────────────

type createRefineSessionRequest struct {
	RecipeID              string   `json:"recipe_id" binding:"required"`
	SelectedGuInstanceIDs []string `json:"selected_gu_instance_ids"`
}

// CreateRefineSession handles POST /api/refine/sessions: escrow resources,
// lock ingredient Gu and open a timed server-side session (WF-REF-05 §5).
func CreateRefineSession(c *gin.Context) {
	var req createRefineSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "recipe_id is required"})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Player not found"})
		} else {
			c.JSON(500, gin.H{"error": "Failed to lock player status"})
		}
		return
	}

	if gamefx.IsDeadState(&player) {
		c.JSON(403, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể luyện Cổ!"})
		return
	}

	// One ACTIVE session per player.
	var activeID string
	if err := tx.QueryRow(`SELECT id FROM refine_sessions WHERE player_id=$1 AND status='ACTIVE' FOR UPDATE`, player.ID).Scan(&activeID); err == nil {
		c.JSON(409, gin.H{"error": "Đạo hữu đang có một phiên luyện Cổ đang chạy!", "code": refineCodeSessionActive})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi kiểm tra phiên luyện"})
		return
	}

	recipeProgress, err := RecipeSvc.Get(tx, player.ID.String(), req.RecipeID)
	if err != nil {
		c.JSON(400, gin.H{"error": "Đạo hữu chưa khai mở công thức này!"})
		return
	}
	if recipeProgress.CompletionRate < 100.0 {
		c.JSON(400, gin.H{"error": "Bí phương chưa hoàn thiện 100%! Hãy thôi diễn cho hoàn chỉnh rồi mới luyện."})
		return
	}

	recipe, err := fetchRefiningRecipe(tx, req.RecipeID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Refining recipe not found"})
		} else {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	stage, err := models.LoadRefineStageTemplate(tx, recipe.TargetRank)
	if err != nil {
		c.JSON(503, gin.H{"error": "Chưa có cấu hình nhịp độ luyện cổ cho rank này. Hãy import RefineStage.csv qua Admin!", "code": refineCodeConfigMissing})
		return
	}

	manaCost, err := calculateRefineManaCost(player.Rank, recipe.TargetRank)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if player.CurrentMana < manaCost {
		unit := "Chân Nguyên"
		if player.Rank >= 6 {
			unit = "Tiên Nguyên"
		}
		c.JSON(400, gin.H{"error": fmt.Sprintf("Không đủ %s! Cần %.1f", unit, manaCost), "code": "REFINE_NOT_ENOUGH_MANA"})
		return
	}
	if player.PrimevalStones < recipe.PrimevalStonesCost {
		c.JSON(400, gin.H{"error": "Not enough Primeval Stones", "code": "REFINE_NOT_ENOUGH_STONES"})
		return
	}
	if recipe.ImmortalEssenceStonesCost > 0 && player.ImmortalStones < recipe.ImmortalEssenceStonesCost {
		c.JSON(400, gin.H{"error": "Not enough Immortal Essence Stones", "code": "REFINE_NOT_ENOUGH_STONES"})
		return
	}
	for matID, reqQty := range recipe.RequiredMaterials {
		currentQty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), matID)
		if currentQty < reqQty {
			matName := matID
			if itemSpec, ok := models.ItemRegistry[matID]; ok && itemSpec.Name != "" {
				matName = itemSpec.Name
			}
			c.JSON(400, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matName), "code": "REFINE_NOT_ENOUGH_MATERIALS"})
			return
		}
	}

	_, materialQualityBonus, guRequirementsNotMet, selErr := validateRefineSelection(&player, &recipe, req.SelectedGuInstanceIDs)
	if selErr != nil {
		if strings.HasPrefix(selErr.Error(), "Gu bug not found") {
			c.JSON(404, gin.H{"error": selErr.Error()})
			return
		}
		c.JSON(400, gin.H{"error": selErr.Error()})
		return
	}

	// Gu nguyên liệu phải không bị khóa bởi phiên khác.
	lockedCount := 0
	if err := tx.QueryRow(`SELECT COUNT(1) FROM player_gu WHERE player_id=$1 AND id = ANY($2::text[]) AND is_locked`,
		player.ID, pq.Array(req.SelectedGuInstanceIDs)).Scan(&lockedCount); err != nil {
		logger.S.Errorf("CreateRefineSession: locked check error: %v", err)
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi kiểm tra Gu bị khóa"})
		return
	}
	if lockedCount > 0 {
		c.JSON(409, gin.H{"error": "Có Cổ nguyên liệu đang bị giữ trong phiên luyện khác!", "code": refineCodeGuLocked})
		return
	}

	// Snapshot of selected Gu at escrow time (vital flags for failure handling).
	guSnapshot := models.GuInventory{}
	for _, instID := range req.SelectedGuInstanceIDs {
		if item, _, found := models.FindGuByInstanceID(player.GuInventory, instID); found && item != nil {
			guSnapshot = append(guSnapshot, *item)
		}
	}

	// ── Escrow: lock Gu + deduct everything, all inside this tx ──
	res, err := tx.Exec(`UPDATE player_gu SET is_locked = true, updated_at = NOW() WHERE player_id = $1 AND id = ANY($2::text[]) AND NOT is_locked`,
		player.ID, pq.Array(req.SelectedGuInstanceIDs))
	if err != nil {
		logger.S.Errorf("CreateRefineSession: lock Gus error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể khóa Cổ nguyên liệu"})
		return
	}
	if rows, _ := res.RowsAffected(); rows != int64(len(req.SelectedGuInstanceIDs)) {
		c.JSON(409, gin.H{"error": "Có Cổ nguyên liệu đang bị giữ trong phiên luyện khác!", "code": refineCodeGuLocked})
		return
	}

	materialsDeducted := map[string]int64{}
	for matID, reqQty := range recipe.RequiredMaterials {
		if err := InventorySvc.DeductItem(tx, player.ID.String(), matID, reqQty); err != nil {
			matName := matID
			if itemSpec, ok := models.ItemRegistry[matID]; ok && itemSpec.Name != "" {
				matName = itemSpec.Name
			}
			if err == sql.ErrNoRows {
				c.JSON(400, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matName), "code": "REFINE_NOT_ENOUGH_MATERIALS"})
			} else {
				logger.S.Errorf("CreateRefineSession: DeductItem error: %v", err)
				c.JSON(500, gin.H{"error": "Lỗi hệ thống khi trừ nguyên liệu"})
			}
			return
		}
		materialsDeducted[matID] = reqQty
	}

	player.CurrentMana -= manaCost
	player.PrimevalStones -= recipe.PrimevalStonesCost
	if recipe.ImmortalEssenceStonesCost > 0 {
		player.ImmortalStones -= recipe.ImmortalEssenceStonesCost
	}

	// Phiên luôn là lần luyện "tỉ mỉ": nguyên liệu khớp công thức khi tạo,
	// pity chỉ cộng khi mục tiêu hợp lệ (Tiên Cổ trùng được kiểm tra lại lúc resolve).
	rateSnapshot, err := computeRefineRate(tx, &player, recipe, stage, refineRateInput{
		materialQualityBonus: materialQualityBonus,
		pityEligible:         !guRequirementsNotMet,
		requirementsNotMet:   guRequirementsNotMet,
	})
	if err != nil {
		logger.S.Errorf("CreateRefineSession: computeRefineRate error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể tính tỉ lệ luyện"})
		return
	}

	decisions, err := loadRefineDecisionsDB(tx)
	if err != nil {
		logger.S.Errorf("CreateRefineSession: loadRefineDecisions error: %v", err)
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc kịch bản luyện"})
		return
	}

	startedAt := time.Now().UTC()
	endsAt := startedAt.Add(time.Duration(stage.DurationSeconds) * time.Second)
	plan, err := buildRefinePlan(stage, &recipe, decisions, startedAt, endsAt)
	if err != nil {
		logger.S.Errorf("CreateRefineSession: buildRefinePlan error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể sinh lịch luyện"})
		return
	}

	planRaw, err := json.Marshal(plan)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể mã hóa lịch luyện"})
		return
	}
	guSnapshotRaw, err := json.Marshal(guSnapshot)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể mã hóa snapshot Cổ"})
		return
	}
	materialsRaw, err := json.Marshal(materialsDeducted)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể mã hóa nguyên liệu"})
		return
	}
	rateRaw, err := json.Marshal(rateSnapshot)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể mã hóa tỉ lệ"})
		return
	}

	if _, err := tx.Exec(`
		UPDATE players SET primeval_stones = $1, immortal_stones = $2, current_mana = $3 WHERE id = $4`,
		player.PrimevalStones, player.ImmortalStones, player.CurrentMana, player.ID); err != nil {
		logger.S.Errorf("CreateRefineSession: update player error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể trừ tài nguyên"})
		return
	}

	var sessionID string
	if err := tx.QueryRow(`
		INSERT INTO refine_sessions (
			player_id, recipe_id, target_gu_id, target_rank, variant, status,
			selected_gu_ids, gu_snapshot, materials_deducted, rate_snapshot,
			plan, step_results, started_at, ends_at
		) VALUES ($1,$2,$3,$4,$5,'ACTIVE',$6,$7,$8,$9,$10,'{}',$11,$12)
		RETURNING id`,
		player.ID, req.RecipeID, recipe.TargetGuID, recipe.TargetRank, recipe.Variant,
		pq.Array(req.SelectedGuInstanceIDs), string(guSnapshotRaw), string(materialsRaw), string(rateRaw),
		string(planRaw), startedAt, endsAt,
	).Scan(&sessionID); err != nil {
		if db.IsImmortalConflict(err) || strings.Contains(err.Error(), "refine_sessions_one_active") {
			c.JSON(409, gin.H{"error": "Đạo hữu đang có một phiên luyện Cổ đang chạy!", "code": refineCodeSessionActive})
			return
		}
		logger.S.Errorf("CreateRefineSession: insert error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể tạo phiên luyện"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to commit refine session"})
		return
	}

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, ItemInventoryDomain, CombatStatsDomain)

	// Reload the session from DB to guarantee the stored view matches bytes on disk.
	session, err := getRefineSessionRow(db.DB, sessionID, player.ID.String())
	if err != nil {
		logger.S.Errorf("CreateRefineSession: reload error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể đọc lại phiên luyện"})
		return
	}
	view := publicSessionView(session, time.Now().UTC())
	publishRefineSession(player.ID.String(), view)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "refine.session.create",
		EntityType:     "refine_session",
		EntityID:       sessionID,
		RequestSummary: fmt.Sprintf("Mở phiên luyện cổ: %s -> %s R%d", req.RecipeID, recipe.TargetGuID, recipe.TargetRank),
		Metadata: map[string]interface{}{
			"recipe_id":    req.RecipeID,
			"target_gu_id": recipe.TargetGuID,
			"target_rank":  recipe.TargetRank,
			"gu_ids":       req.SelectedGuInstanceIDs,
			"ends_at":      endsAt,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	respondActionWithInventory(c, 200, player, gin.H{"session": view})
}

// ─── Read paths (with lazy resolve) ──────────────────────────────────────────

func getRefineSessionRow(q interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}, sessionID, playerID string) (*refineSessionRow, error) {
	return scanRefineSessionRow(q.QueryRow(refineSessionSelect+` WHERE id=$1 AND player_id=$2`, sessionID, playerID))
}

// getRefineSessionRowLocked is the mutating paths' read: row lock held until
// the surrounding transaction commits (WF-REF-05 §11).
func getRefineSessionRowLocked(q interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}, sessionID, playerID string) (*refineSessionRow, error) {
	return scanRefineSessionRow(q.QueryRow(refineSessionSelectForUpdate, sessionID, playerID))
}

// GetActiveRefineSession: GET /api/refine/sessions/active.
// Resolves an overdue ACTIVE session first; otherwise returns the latest
// RESOLVED/ABORTED session not acknowledged within 7 days, or {session: null}.
func GetActiveRefineSession(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	now := time.Now().UTC()

	var activeID string
	err := db.DB.QueryRow(`SELECT id FROM refine_sessions WHERE player_id=$1 AND status='ACTIVE' LIMIT 1`, authID).Scan(&activeID)
	if err == nil {
		// Lazy resolve when overdue (primary path if the worker is down).
		row, serr := scanRefineSessionRow(db.DB.QueryRow(refineSessionSelect+` WHERE id=$1`, activeID))
		if serr != nil {
			logger.S.Errorf("GetActiveRefineSession: scan error: %v", serr)
			c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc phiên luyện"})
			return
		}
		if !row.EndsAt.After(now) {
			resolved, rerr := resolveRefineSession(c.Request.Context(), row.ID, "expired")
			if rerr != nil {
				logger.S.Errorf("GetActiveRefineSession: resolve error: %v", rerr)
				c.JSON(500, gin.H{"error": "Lỗi hệ thống khi kết thúc phiên luyện"})
				return
			}
			c.JSON(200, gin.H{"session": publicSessionView(resolved, now)})
			return
		}
		c.JSON(200, gin.H{"session": publicSessionView(row, now)})
		return
	} else if err != sql.ErrNoRows {
		logger.S.Errorf("GetActiveRefineSession: query error: %v", err)
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi kiểm tra phiên luyện"})
		return
	}

	// Latest finished result not yet acknowledged (UI result overlay).
	var recentID string
	err = db.DB.QueryRow(`
		SELECT id FROM refine_sessions
		WHERE player_id=$1 AND status IN ('RESOLVED','ABORTED')
		  AND acknowledged_at IS NULL AND resolved_at > NOW() - INTERVAL '7 days'
		ORDER BY resolved_at DESC LIMIT 1`, authID).Scan(&recentID)
	if err == sql.ErrNoRows {
		c.JSON(200, gin.H{"session": nil})
		return
	}
	if err != nil {
		logger.S.Errorf("GetActiveRefineSession: recent query error: %v", err)
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc phiên luyện"})
		return
	}
	row, err := scanRefineSessionRow(db.DB.QueryRow(refineSessionSelect+` WHERE id=$1`, recentID))
	if err != nil {
		logger.S.Errorf("GetActiveRefineSession: scan error: %v", err)
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc phiên luyện"})
		return
	}
	c.JSON(200, gin.H{"session": publicSessionView(row, now)})
}

// GetRefineSession: GET /api/refine/sessions/:id (lazy resolve when overdue).
func GetRefineSession(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	sessionID := c.Param("id")
	row, err := getRefineSessionRow(db.DB, sessionID, authID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Không tìm thấy phiên luyện", "code": refineCodeSessionNotFound})
		} else {
			logger.S.Errorf("GetRefineSession: %v", err)
			c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc phiên luyện"})
		}
		return
	}
	now := time.Now().UTC()
	if row.Status == "ACTIVE" && !row.EndsAt.After(now) {
		resolved, rerr := resolveRefineSession(c.Request.Context(), row.ID, "expired")
		if rerr != nil {
			logger.S.Errorf("GetRefineSession: resolve error: %v", rerr)
			c.JSON(500, gin.H{"error": "Lỗi hệ thống khi kết thúc phiên luyện"})
			return
		}
		c.JSON(200, gin.H{"session": publicSessionView(resolved, now)})
		return
	}
	c.JSON(200, gin.H{"session": publicSessionView(row, now)})
}

// ─── Submit step ─────────────────────────────────────────────────────────────

type submitRefineStepRequest struct {
	StepIndex int         `json:"step_index" binding:"required"`
	Nonce     string      `json:"nonce" binding:"required"`
	Action    string      `json:"action" binding:"required"`
	Payload   interface{} `json:"payload"`
	ClientTS  int64       `json:"client_ts"`
}

// SubmitRefineSessionStep handles POST /api/refine/sessions/:id/steps.
// Short transaction, session row lock only — never locks players.
func SubmitRefineSessionStep(c *gin.Context) {
	var req submitRefineStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "step_index, nonce, action là bắt buộc"})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	sessionID := c.Param("id")
	now := time.Now().UTC()

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	session, err := getRefineSessionRowLocked(tx, sessionID, authID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Không tìm thấy phiên luyện", "code": refineCodeSessionNotFound})
		} else {
			logger.S.Errorf("SubmitRefineSessionStep: scan error: %v", err)
			c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc phiên luyện"})
		}
		return
	}

	// Resolve-only states: replaying the exact same submit with the same nonce
	// returns the stored result; anything else is rejected.
	if session.Status != "ACTIVE" {
		key := strconv.Itoa(req.StepIndex)
		if res, done := session.StepResults[key]; done && req.Nonce == findStepNonce(session, req.StepIndex) {
			c.JSON(200, gin.H{"session": publicSessionView(session, now), "last_step": stepResultView(key, res)})
			return
		}
		c.JSON(409, gin.H{"error": "Phiên luyện đã kết thúc", "code": refineCodeStepDone, "session": publicSessionView(session, now)})
		return
	}

	if req.StepIndex < 1 || req.StepIndex > len(session.Plan.Steps) {
		c.JSON(400, gin.H{"error": "step_index ngoài phạm vi", "code": refineCodePayloadInvalid})
		return
	}
	step := session.Plan.Steps[req.StepIndex-1]
	key := strconv.Itoa(req.StepIndex)

	if res, done := session.StepResults[key]; done {
		if req.Nonce == step.Nonce {
			// Idempotent replay: return the stored result, never re-grade.
			c.JSON(200, gin.H{"session": publicSessionView(session, now), "last_step": stepResultView(key, res)})
			return
		}
		c.JSON(409, gin.H{"error": "Bước này đã được gửi với nonce khác", "code": refineCodeStepDone})
		return
	}

	if now.Before(step.WindowStart) {
		c.JSON(409, gin.H{"error": "Bước chưa mở cửa sổ thao tác", "code": refineCodeStepNotOpen, "session": publicSessionView(session, now)})
		return
	}
	if now.After(step.WindowEnd.Add(refineGraceDuration())) {
		c.JSON(410, gin.H{"error": "Bước đã đóng, điểm của bước này là 0", "code": refineCodeStepClosed, "session": publicSessionView(session, now)})
		return
	}
	if req.Nonce != step.Nonce {
		c.JSON(409, gin.H{"error": "Nonce không hợp lệ cho bước này", "code": refineCodeNonceInvalid, "session": publicSessionView(session, now)})
		return
	}
	if req.Action != step.Type {
		c.JSON(400, gin.H{"error": "Loại thao tác không khớp bước hiện tại", "code": refineCodePayloadInvalid})
		return
	}

	payload, ok := req.Payload.(map[string]interface{})
	if req.Payload != nil && !ok {
		c.JSON(400, gin.H{"error": "payload sai định dạng", "code": refineCodePayloadInvalid})
		return
	}
	score, perr := scoreRefineStep(step, now, payload)
	if perr != nil {
		c.JSON(400, gin.H{"error": perr.Error(), "code": refineCodePayloadInvalid})
		return
	}

	result := refineStepResult{
		Score:           score,
		Action:          step.Type,
		SubmittedAt:     now,
		ReceiptOffsetMs: now.Sub(step.WindowStart).Milliseconds(),
	}
	if payload != nil {
		trimmed := payload
		if len(trimmed) > 0 {
			result.Payload = trimmed
		}
	}
	session.StepResults[key] = result
	accuracy := averageStepScores(session.StepResults, session.Plan.N)

	stepResultsRaw, err := json.Marshal(session.StepResults)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể lưu kết quả bước"})
		return
	}
	if _, err := tx.Exec(`UPDATE refine_sessions SET step_results=$1, accuracy=$2 WHERE id=$3`,
		string(stepResultsRaw), accuracy, session.ID); err != nil {
		logger.S.Errorf("SubmitRefineSessionStep: update error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể lưu kết quả bước"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to commit step"})
		return
	}

	session.Accuracy = accuracy
	view := publicSessionView(session, time.Now().UTC())
	publishRefineSession(session.PlayerID, view)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "refine.session.step",
		EntityType:     "refine_session",
		EntityID:       sessionID,
		RequestSummary: fmt.Sprintf("Bước %d/%d (%s) điểm %.2f", req.StepIndex, session.Plan.N, step.Type, score),
		Metadata: map[string]interface{}{
			"step_index": req.StepIndex, "score": score, "action": step.Type,
			"receipt_offset_ms": result.ReceiptOffsetMs, "client_ts": req.ClientTS,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	// Bước cuối (hoặc đã đủ tất cả) → resolve ngay trong request này.
	lastStep := req.StepIndex == session.Plan.N
	if lastStep || len(session.StepResults) == session.Plan.N {
		resolved, rerr := resolveRefineSession(c.Request.Context(), session.ID, "completed")
		if rerr != nil {
			logger.S.Errorf("SubmitRefineSessionStep: resolve error: %v", rerr)
			c.JSON(500, gin.H{"error": "Lỗi hệ thống khi chốt phiên luyện"})
			return
		}
		finalView := publicSessionView(resolved, time.Now().UTC())
		refreshAfterResolve := gin.H{"session": finalView, "last_step": stepResultView(key, result), "result": extractResultJSON(resolved.ResultRaw)}
		player, perr := PlayerSvc.Load(db.DB, authID, false)
		if perr == nil {
			refreshAfterResolve["player_ready"] = true
			respondActionWithInventory(c, 200, player, refreshAfterResolve)
			return
		}
		c.JSON(200, refreshAfterResolve)
		return
	}

	c.JSON(200, gin.H{"session": view, "last_step": stepResultView(key, result)})
}

func findStepNonce(session *refineSessionRow, stepIndex int) string {
	if session == nil || stepIndex < 1 || stepIndex > len(session.Plan.Steps) {
		return ""
	}
	return session.Plan.Steps[stepIndex-1].Nonce
}

func stepResultView(key string, res refineStepResult) gin.H {
	return gin.H{"step_key": key, "score": res.Score, "action": res.Action, "submitted_at": res.SubmittedAt}
}

func extractResultJSON(raw []byte) map[string]interface{} {
	var result map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}
	return result
}

// averageStepScores averages submitted step scores; missing steps count 0.
// Result 0..100 rounded to 2 digits.
func averageStepScores(results map[string]refineStepResult, n int) float64 {
	if n <= 0 {
		return 0
	}
	total := 0.0
	for key, res := range results {
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 1 || idx > n {
			continue
		}
		total += res.Score
	}
	return math.Round((total/float64(n))*100) / 100
}

// ─── Abort ───────────────────────────────────────────────────────────────────

// AbortRefineSession: POST /api/refine/sessions/:id/abort.
// Refunds floor(qty/2) of every material, sends every ingredient Gu into
// DYING (Hấp Hối), never refunds mana/stones.
func AbortRefineSession(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	sessionID := c.Param("id")

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	row, err := getRefineSessionRowLocked(tx, sessionID, authID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Không tìm thấy phiên luyện", "code": refineCodeSessionNotFound})
		} else {
			logger.S.Errorf("AbortRefineSession: scan error: %v", err)
			c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc phiên luyện"})
		}
		return
	}
	if row.Status != "ACTIVE" {
		c.JSON(200, gin.H{"session": publicSessionView(row, time.Now().UTC())})
		return
	}

	// Refund floor(qty/2) of each material (into whatever category stack
	// already holds the item so the deduction round-trips cleanly).
	for matID, qty := range row.MaterialsDeducted {
		refund := qty / 2
		if refund <= 0 {
			continue
		}
		category := "materials"
		var catDB string
		if err := tx.QueryRow(`SELECT category FROM player_inventory_items
			WHERE player_id=$1 AND item_id=$2 LIMIT 1`, authID, matID).Scan(&catDB); err == nil && catDB != "" {
			category = catDB
		}
		if err := InventorySvc.AddItem(tx, authID.String(), matID, category, refund); err != nil {
			logger.S.Errorf("AbortRefineSession: refund material %s error: %v", matID, err)
			c.JSON(500, gin.H{"error": "Không thể hoàn nguyên liệu"})
			return
		}
	}

	// All ingredient Gu go back starving (Hấp Hối), unlocked and unequipped.
	if _, err := tx.Exec(`UPDATE player_gu SET is_locked=false, hunger=1, state='DYING', equipped=false, updated_at=NOW()
		WHERE player_id=$1 AND id = ANY($2::text[])`, authID, pq.Array(row.SelectedGuIDs)); err != nil {
		logger.S.Errorf("AbortRefineSession: unlock/dying error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể hoàn trạng thái Cổ"})
		return
	}

	// Recompute equipped list from DB state.
	playerGu, err := db.LoadPlayerGus(tx, authID.String(), false)
	if err != nil {
		logger.S.Errorf("AbortRefineSession: load gus error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể đọc Cổ"})
		return
	}
	equipped := []string{}
	for _, gu := range models.FilterEquippedActive(playerGu) {
		equipped = append(equipped, gu.InstanceID)
	}
	if _, err := tx.Exec(`UPDATE players SET equipped_gus = $1 WHERE id = $2`, pq.Array(equipped), authID); err != nil {
		logger.S.Errorf("AbortRefineSession: update equipped error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể cập nhật Cổ trang bị"})
		return
	}

	result := gin.H{
		"reason":  "aborted",
		"message": "Đạo hữu chủ động hủy phiên luyện. Nguyên liệu hoàn lại 50%, các Cổ nguyên liệu rơi vào Hấp Hối.",
	}
	resultRaw, merr := json.Marshal(result)
	if merr != nil {
		logger.S.Errorf("AbortRefineSession: marshal result error: %v", merr)
	}
	if _, err := tx.Exec(`UPDATE refine_sessions SET status='ABORTED', resolved_at=NOW(), result=$1 WHERE id=$2 AND status='ACTIVE'`,
		string(resultRaw), row.ID); err != nil {
		logger.S.Errorf("AbortRefineSession: update session error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể hủy phiên luyện"})
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		logger.S.Errorf("AbortRefineSession: reload player error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể đọc trạng thái người chơi"})
		return
	}
	player.GuInventory = playerGu
	player.EquippedGus = equipped

	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to commit abort"})
		return
	}

	aborted, err := getRefineSessionRow(db.DB, row.ID, authID.String())
	if err == nil {
		row = aborted
	}
	view := publicSessionView(row, time.Now().UTC())
	publishRefineSession(authID.String(), view)
	ClearPendingPlayerQueues(authID.String())
	refreshActionPlayerState(&player)
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, ItemInventoryDomain, CombatStatsDomain)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "refine.session.abort",
		EntityType:     "refine_session",
		EntityID:       row.ID,
		RequestSummary: "Hủy phiên luyện cổ",
		Metadata:       map[string]interface{}{"recipe_id": row.RecipeID, "reason": "player_abort"},
		IPAddress:      c.ClientIP(), Success: true,
	})

	respondActionWithInventory(c, 200, player, gin.H{"session": view})
}

// ─── Ack ─────────────────────────────────────────────────────────────────────

// AckRefineSession: POST /api/refine/sessions/:id/ack marks the result seen
// (pure UI flag; rewards are already applied at resolve time).
func AckRefineSession(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	sessionID := c.Param("id")
	res, err := db.DB.Exec(`UPDATE refine_sessions SET acknowledged_at = NOW()
		WHERE id=$1 AND player_id=$2 AND acknowledged_at IS NULL`, sessionID, authID)
	if err != nil {
		logger.S.Errorf("AckRefineSession: %v", err)
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi xác nhận phiên luyện"})
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		// Already acknowledged or unknown session — idempotent success.
		var count int
		_ = db.DB.QueryRow(`SELECT COUNT(1) FROM refine_sessions WHERE id=$1 AND player_id=$2`, sessionID, authID).Scan(&count)
		if count == 0 {
			c.JSON(404, gin.H{"error": "Không tìm thấy phiên luyện", "code": refineCodeSessionNotFound})
			return
		}
	}
	c.JSON(200, gin.H{"ok": true})
}

// ─── Preview ─────────────────────────────────────────────────────────────────

// RefinePreview: GET /api/refine/preview?recipe_id=...
// Non-locking rate preview with accuracy band [rate_min, rate_max].
func RefinePreview(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	recipeID := c.Query("recipe_id")
	if recipeID == "" {
		c.JSON(400, gin.H{"error": "recipe_id là bắt buộc"})
		return
	}

	player, err := PlayerSvc.Load(db.DB, authID, false)
	if err != nil {
		logger.S.Errorf("RefinePreview: load player error: %v", err)
		c.JSON(500, gin.H{"error": "Lỗi hệ thống khi đọc trạng thái người chơi"})
		return
	}

	warnings := []string{}

	var activeID string
	if err := db.DB.QueryRow(`SELECT id FROM refine_sessions WHERE player_id=$1 AND status='ACTIVE'`, authID).Scan(&activeID); err == nil {
		warnings = append(warnings, "Đạo hữu đang có phiên luyện đang chạy.")
	} else if err != sql.ErrNoRows {
		logger.S.Errorf("RefinePreview: session check error: %v", err)
	}
	recipe, err := fetchRefiningRecipe(db.DB, recipeID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Refining recipe not found"})
		return
	}

	if progress, perr := RecipeSvc.Get(db.DB, authID.String(), recipeID); perr != nil || progress == nil || progress.CompletionRate < 100.0 {
		warnings = append(warnings, "Bí phương chưa hoàn thiện 100%.")
	}

	stage, err := models.LoadRefineStageTemplate(db.DB, recipe.TargetRank)
	if err != nil {
		stage = nil
		warnings = append(warnings, "Chưa có cấu hình nhịp độ luyện cổ cho rank này (import RefineStage.csv).")
	}

	manaCost, err := calculateRefineManaCost(player.Rank, recipe.TargetRank)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if player.CurrentMana < manaCost {
		warnings = append(warnings, "Không đủ Chân Nguyên.")
	}
	if player.PrimevalStones < recipe.PrimevalStonesCost {
		warnings = append(warnings, "Không đủ Nguyên Thạch.")
	}
	if recipe.ImmortalEssenceStonesCost > 0 && player.ImmortalStones < recipe.ImmortalEssenceStonesCost {
		warnings = append(warnings, "Không đủ Tiên Nguyên Thạch.")
	}
	for matID, reqQty := range recipe.RequiredMaterials {
		if qty, _ := InventorySvc.GetQuantity(db.DB, authID.String(), matID); qty < reqQty {
			matName := matID
			if itemSpec, ok := models.ItemRegistry[matID]; ok && itemSpec.Name != "" {
				matName = itemSpec.Name
			}
			warnings = append(warnings, fmt.Sprintf("Không đủ nguyên liệu %s.", matName))
		}
	}

	rateSnapshot, err := computeRefineRate(db.DB, &player, recipe, stage, refineRateInput{pityEligible: true})
	if err != nil {
		logger.S.Errorf("RefinePreview: computeRefineRate error: %v", err)
		c.JSON(500, gin.H{"error": "Không thể tính tỉ lệ luyện"})
		return
	}

	statusPenalty := rateSnapshot.StatusPenalty
	raw := rateSnapshot.RawPositiveRate
	if raw > 1 {
		raw = 1
	}
	rateMin := rateSnapshot.Breakdown.FinalRate
	rateMax := raw + rateSnapshot.AccuracyMaxBonus - statusPenalty
	if rateMax < 0 {
		rateMax = 0
	}
	if rateMax > 1 {
		rateMax = 1
	}

	resp := gin.H{
		"rate_breakdown":      rateSnapshot.Breakdown,
		"rate_min":            rateMin,
		"rate_max":            rateMax,
		"accuracy_max_bonus":  rateSnapshot.AccuracyMaxBonus,
		"death_penalty":       rateSnapshot.DeathPenalty,
		"warnings":            warnings,
		"mana_cost":           manaCost,
		"primeval_stones_cost": recipe.PrimevalStonesCost,
		"immortal_stones_cost": recipe.ImmortalEssenceStonesCost,
		"materials":           recipe.RequiredMaterials,
	}
	c.JSON(200, resp)
}
