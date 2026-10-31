package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// Refine session resolve shared logic (WF-REF-05 §7.2, §8).
// resolveRefineSession is idempotent: guarded by a FOR UPDATE row lock and
// status check, callable concurrently from submit-last-step, worker and lazy GET.

// resolveRefineSession resolves an ACTIVE session exactly once. reason is
// "completed" (roll at last submitted step, accuracy counts) or "expired"
// (roll at ends_at with whatever accuracy the player managed).
func resolveRefineSession(ctx context.Context, sessionID, reason string) (*refineSessionRow, error) {
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout='15000'`); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout='5000'`); err != nil {
		return nil, err
	}

	session, err := scanRefineSessionRow(tx.QueryRow(refineSessionSelect+` WHERE id=$1 FOR UPDATE`, sessionID))
	if err != nil {
		return nil, err
	}
	if session.Status != "ACTIVE" {
		// Idempotent guard: another caller already resolved it.
		return session, nil
	}

	player, err := evaluateManaTx(tx, session.playerUUID())
	if err != nil {
		return nil, fmt.Errorf("lock player: %w", err)
	}

	accuracy := averageStepScores(session.StepResults, session.Plan.N)
	snap := session.RateSnapshot

	// WF-REF-05 §7.2 / WF-REF-06 §7.3:
	//   bonus  = AccuracyMaxBonus × clamp((accuracy−50)/50, 0, 1)
	//   capped = min(1, rawPositive + bonus)
	//   final  = max(0, capped − statusPenalty − deathPenalty)
	accuracyBonus := snap.AccuracyMaxBonus * clamp01((accuracy-50)/50)
	capped := snap.RawPositiveRate + accuracyBonus
	if capped > 1 {
		capped = 1
	}
	finalRate := capped - snap.StatusPenalty - session.DeathPenalty
	if finalRate < 0 {
		finalRate = 0
	}

	// Tiên cổ duy nhất: re-check at resolve; a conflicting immortal forces
	// FAIL_NORMAL with no accuracy bonus and no backfire.
	isUniqueImmortalFail := false
	if session.TargetRank >= 6 {
		exists, ierr := db.CheckImmortalGuExists(tx, session.TargetGuID, session.TargetRank, session.SelectedGuIDs)
		if ierr != nil {
			return nil, fmt.Errorf("check immortal: %w", ierr)
		}
		isUniqueImmortalFail = exists
		if exists {
			accuracyBonus = 0
			finalRate = snap.Breakdown.FinalRate - session.DeathPenalty
			if finalRate < 0 {
				finalRate = 0
			}
		}
	}

	roll := rand.Float64()
	outcome := determineRefineOutcome(roll, finalRate, snap.RefinementRealm, session.TargetRank)
	if isUniqueImmortalFail && outcome != OutcomeFailNormal {
		outcome = OutcomeFailNormal
	}

	// ── Classify selected Gu (vital survive DYING, others consumed) ──
	consumedGuIDs := []string{}
	vitalIDs := []string{}
	snapshotByID := map[string]models.GuInstance{}
	for _, inst := range session.GuSnapshot {
		snapshotByID[strings.TrimSpace(inst.InstanceID)] = inst
	}
	for _, instID := range session.SelectedGuIDs {
		trimmed := strings.TrimSpace(instID)
		if inst, ok := snapshotByID[trimmed]; ok && inst.IsVital {
			vitalIDs = append(vitalIDs, trimmed)
		} else {
			consumedGuIDs = append(consumedGuIDs, trimmed)
		}
	}

	var newGuInstance models.GuInstance
	var msg strings.Builder

	switch outcome {
	case OutcomeSuccess:
		msg.WriteString("Hợp luyện thành công! Đạo hữu luyện chế được Cổ trùng mới.")
		if len(consumedGuIDs) > 0 {
			if err := db.RemovePlayerGus(tx, consumedGuIDs); err != nil {
				return nil, fmt.Errorf("xóa Cổ nguyên liệu: %w", err)
			}
		}
		for _, inst := range session.GuSnapshot {
			if spec, ok := models.GetGu(inst.GuID, inst.Rank); ok {
				if emsg, cerr := HandleGuEvent(&player, spec, inst, "ON_REFINE_MATERIAL"); cerr == nil && emsg != "" {
					msg.WriteString(" " + emsg)
				}
			}
		}
		newGuInstance = models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       session.TargetGuID,
			Rank:       session.TargetRank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
			IsVital:    len(vitalIDs) > 0,
		}
		// Savepoint around the new-Gu insert for immortal-rank targets: a
		// concurrent immortal insert elsewhere must degrade this session to
		// FAIL_NORMAL instead of erroring out (WF-REF-05 §8.2).
		savepointStarted := false
		if session.TargetRank >= 6 {
			if _, err := tx.Exec(`SAVEPOINT sp_refine_gu_insert`); err == nil {
				savepointStarted = true
			}
		}
		if err := db.AddPlayerGu(tx, player.ID.String(), newGuInstance); err != nil {
			if db.IsImmortalConflict(err) {
				if savepointStarted {
					if _, rbErr := tx.Exec(`ROLLBACK TO SAVEPOINT sp_refine_gu_insert`); rbErr != nil {
						return nil, fmt.Errorf("rollback savepoint: %w", rbErr)
					}
				}
				outcome = OutcomeFailNormal
				isUniqueImmortalFail = true
				accuracyBonus = 0
				newGuInstance = models.GuInstance{}
				msg.Reset()
				msg.WriteString("Tiên cổ đã xuất thế — luyện chế thất bại!")
				if len(consumedGuIDs) > 0 {
					if rerr := db.RemovePlayerGus(tx, consumedGuIDs); rerr != nil {
						return nil, fmt.Errorf("xóa Cổ nguyên liệu: %w", rerr)
					}
				}
				if len(vitalIDs) > 0 {
					if verr := updateVitalGusDying(tx, vitalIDs); verr != nil {
						return nil, fmt.Errorf("cập nhật Cổ Bản Mệnh: %w", verr)
					}
				}
			} else {
				return nil, fmt.Errorf("insert new Gu: %w", err)
			}
		}
		if savepointStarted && outcome == OutcomeSuccess {
			if _, err := tx.Exec(`RELEASE SAVEPOINT sp_refine_gu_insert`); err != nil {
				return nil, fmt.Errorf("release savepoint: %w", err)
			}
		}
		if outcome == OutcomeSuccess {
			guName := newGuInstance.GuID
			if targetSpec, ok := models.GetGu(newGuInstance.GuID, newGuInstance.Rank); ok && targetSpec.Name != "" {
				guName = targetSpec.Name
			}
			refineSummary := fmt.Sprintf("Cổ Sư [%s] đã hợp luyện thành công Cổ trùng [%s] R%d!", player.Username, guName, newGuInstance.Rank)
			_ = db.LogGuHistory(tx, newGuInstance.InstanceID, newGuInstance.GuID, newGuInstance.Rank, "CREATED_REFINE", &player.ID, nil, refineSummary, map[string]interface{}{
				"recipe_id": session.RecipeID,
				"gu_name":   guName,
			})
			// Mastery: gain × (1 + (snapshot max − 1) × accuracy/100)
			refineGain := calculateRefinementPathGain(session.TargetRank, snap.RefinementRealm)
			masteryMult := 1.0 + (snap.MasteryMultiplierMax-1)*accuracy/100
			gain := int(float64(refineGain)*masteryMult + 0.5)
			if gain > 0 {
				if player.PathMasteries == nil {
					player.PathMasteries = make(models.PathMasteriesMap)
				}
				player.PathMasteries["refinement"] += gain
				newLevel := gamefx.PathMasteryLevel(player.PathMasteries["refinement"])
				msg.WriteString(fmt.Sprintf(" Ngộ Đạo tăng tiến! +%d điểm Chân Ý lưu phái Luyện Đạo (Cảnh giới: %s).", gain, gamefx.PathMasteryLevelName(newLevel)))
			}
			if newGuInstance.Rank >= 6 {
				guName := newGuInstance.GuID
				if targetSpec, ok := models.GetGu(newGuInstance.GuID, newGuInstance.Rank); ok {
					guName = targetSpec.Name
				}
				message := fmt.Sprintf("Đạo hữu %s đã luyện chế thành công Cổ Trùng [%s] R%d!", player.Username, guName, newGuInstance.Rank)
				_ = SendSystemMessage("world", sql.NullString{}, message)
			}
		}
	default: // FAIL_NORMAL / FAIL_SAFE / FAIL_BACKFIRE
		if isUniqueImmortalFail {
			msg.WriteString("Tiên cổ là duy nhất, đã tồn tại trong thiên hạ, luyện chế thất bại!")
		} else {
			msg.WriteString("Hợp luyện thất bại! Các nguyên liệu thường đã hóa thành tro bụi.")
		}
		if len(consumedGuIDs) > 0 {
			if err := db.RemovePlayerGus(tx, consumedGuIDs); err != nil {
				return nil, fmt.Errorf("xóa Cổ nguyên liệu: %w", err)
			}
		}
		for _, inst := range session.GuSnapshot {
			if spec, ok := models.GetGu(inst.GuID, inst.Rank); ok {
				if emsg, cerr := HandleGuEvent(&player, spec, inst, "ON_REFINE_MATERIAL"); cerr == nil && emsg != "" {
					msg.WriteString(" " + emsg)
				}
			}
		}
		if len(vitalIDs) > 0 {
			if err := updateVitalGusDying(tx, vitalIDs); err != nil {
				return nil, fmt.Errorf("cập nhật Cổ Bản Mệnh: %w", err)
			}
			for i := range player.GuInventory {
				for _, v := range vitalIDs {
					if strings.TrimSpace(player.GuInventory[i].InstanceID) == v {
						player.GuInventory[i].Hunger = 1
						player.GuInventory[i].Status = "DYING"
						player.GuInventory[i].Equipped = false
					}
				}
			}
			msg.WriteString(" Riêng Cổ Bản Mệnh vẫn sống sót nhưng rơi vào trạng thái Hấp Hối!")
		}
		if outcome == OutcomeFailBackfire {
			hpLoss := gamefx.EffectiveMaxHP(&player) * 0.50
			player.HP -= hpLoss
			if player.HP < 10.0 {
				player.HP = 10.0
			}
			soulLoss := int(math.Ceil(float64(player.MaxSoul) * 0.30))
			player.SoulHP -= soulLoss
			if player.SoulHP < 1 {
				player.SoulHP = 1
			}
			msg.WriteString(fmt.Sprintf(" Luyện cổ CHÁY NỔ (Backfire)! Phản phệ dữ dội, tổn thất %d Hồn Phách và %.0f HP.", soulLoss, hpLoss))
		}
	}

	msg.WriteString(fmt.Sprintf(" (Tỉ lệ: %.1f%%)", finalRate*100.0))
	if session.DeathPenalty > 0 {
		msg.WriteString(" Tâm thần chấn động: tỉ lệ bị giảm do tâm thần chấn động.")
	}
	if reason == "expired" {
		msg.WriteString(fmt.Sprintf(" Phiên luyện kết thúc khi đạo hữu vắng mặt (điểm chính xác: %.1f%%).", accuracy))
	}

	// ── Statuses: fatigue/backfire escalate (same chain as quick path) ──
	isTiringRefine := session.TargetRank >= 6 || session.TargetRank >= player.Rank
	activeStatuses := player.Statuses.Active()
	hasFatigue := activeStatuses.Has("refine_fatigue")
	hasBackfire := activeStatuses.Has("refine_backfire")
	if isTiringRefine || outcome == OutcomeFailBackfire {
		switch {
		case hasFatigue:
			player.Statuses.Remove("refine_fatigue")
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			msg.WriteString(" [Luyện Cổ Phản Phệ]: Tỉ lệ Luyện Cổ giảm 95% trong 20 phút do cố luyện khi tâm thần mệt mỏi!")
		case hasBackfire:
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			msg.WriteString(" Trạng thái [Luyện Cổ Phản Phệ] (giảm 95% tỉ lệ) đã được gia hạn thêm 20 phút.")
		case outcome == OutcomeFailBackfire:
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			msg.WriteString(" [Luyện Cổ Phản Phệ]: Thất bại cháy nổ khiến bạn bị Phản Phệ! Tỉ lệ Luyện Cổ giảm 95% trong 20 phút.")
		default:
			player.Statuses.Add("refine_fatigue", "refine", 10*time.Minute, 0)
			msg.WriteString(" [Tâm Thần Mệt Mỏi]: Tỉ lệ Luyện Cổ lượt tiếp theo giảm 50% trong 10 phút.")
		}
	}

	// ── Pity (uses CURRENT statuses per WF-REF-05 §7.3) ──
	insightKey := gamefx.RefineInsightKey(session.TargetRank)
	// Eligibility is the creation-time snapshot (đúng nguyên liệu/công thức):
	// yêu cầu sai nguyên liệu mất pity như đường nhanh (WF-REF-05 review #7).
	pityEligible := !isUniqueImmortalFail && snap.PityEligible
	currentStacks := refineInsightStacks(&player, session.TargetRank)
	if outcome == OutcomeSuccess && pityEligible {
		if currentStacks > 0 || player.Statuses.Has(insightKey) {
			player.Statuses.Remove(insightKey)
			msg.WriteString(" [Tâm Đắc viên mãn]: Luyện thành công, pity rank này reset về 0.")
		}
	} else if outcome != OutcomeSuccess && pityEligible {
		newStacks := currentStacks + 1
		player.Statuses.Add(insightKey, "refine", 0, float64(newStacks))
		msg.WriteString(fmt.Sprintf(" [Tâm Đắc R%d]: Tích %d tầng (+%d%% tỉ lệ luyện R%d, không giới hạn).", session.TargetRank, newStacks, newStacks, session.TargetRank))
	} else if isUniqueImmortalFail {
		msg.WriteString(" [Tâm Đắc]: Tiên cổ cùng bậc đã tồn tại; lần này không tăng tầng Tâm Đắc.")
	}

	// ── Persist ──
	// Rebuild the in-memory inventory: drop consumed ingredients, mark vitals
	// dying, append the new Gu — so equipped_gus has no dangling ids (#5).
	kept := make(models.GuInventory, 0, len(player.GuInventory))
	for _, inst := range player.GuInventory {
		trimmed := strings.TrimSpace(inst.InstanceID)
		if inst.IsVital {
			for _, v := range vitalIDs {
				if trimmed == v {
					inst.Hunger = 1
					inst.Status = "DYING"
					inst.Equipped = false
					break
				}
			}
			kept = append(kept, inst)
			continue
		}
		dropped := false
		for _, consumed := range consumedGuIDs {
			if trimmed == consumed {
				dropped = true
				break
			}
		}
		if !dropped {
			kept = append(kept, inst)
		}
	}
	player.GuInventory = kept
	if outcome == OutcomeSuccess && newGuInstance.InstanceID != "" {
		player.GuInventory = append(player.GuInventory, newGuInstance)
	}
	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}

	if outcome == OutcomeSuccess && newGuInstance.InstanceID != "" {
		if err := TrackDailyMissionProgress(tx, player.ID, player.Rank, "refine:"+newGuInstance.InstanceID, "refine", "", 1); err != nil {
			return nil, fmt.Errorf("track mission: %w", err)
		}
	}

	statusesRaw, err := player.Statuses.Value()
	if err != nil {
		return nil, fmt.Errorf("serialize statuses: %w", err)
	}

	// Safety net: nothing selected stays locked after resolve.
	if len(session.SelectedGuIDs) > 0 {
		if _, err := tx.Exec(`UPDATE player_gu SET is_locked=false, updated_at=NOW() WHERE id = ANY($1::text[])`,
			pq.Array(session.SelectedGuIDs)); err != nil {
			logger.S.Warnf("resolveRefineSession: final unlock error: %v", err)
		}
	}

	if _, err := tx.Exec(`
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, equipped_gus = $3, hp = $4, max_mana = $5, current_mana = $6, human_qi = $7, path_masteries = $8, soul_hp = $9, statuses = $10
		WHERE id = $11`,
		player.PrimevalStones, player.ImmortalStones, pq.Array(player.EquippedGus), player.HP, player.MaxMana,
		player.CurrentMana, player.HumanQi, player.PathMasteries, player.SoulHP, statusesRaw, player.ID); err != nil {
		return nil, fmt.Errorf("update player: %w", err)
	}

	stepScores := make([]float64, session.Plan.N)
	for key, res := range session.StepResults {
		if idx, aerr := strconv.Atoi(key); aerr == nil && idx >= 1 && idx <= session.Plan.N {
			stepScores[idx-1] = res.Score
		}
	}

	breakdown := map[string]interface{}{
		"base_rate":            snap.Breakdown.BaseRate,
		"completion_penalty":   snap.Breakdown.CompletionPenalty,
		"combo_accuracy":       snap.Breakdown.ComboAccuracy,
		"material_quality":     snap.Breakdown.MaterialQuality,
		"refinement_dao_marks": snap.Breakdown.RefinementDaoMarks,
		"refinement_mastery":   snap.Breakdown.RefinementMastery,
		"target_path_mastery":  snap.Breakdown.TargetPathMastery,
		"luck_bonus":           snap.Breakdown.LuckBonus,
		"support_bonus":        snap.Breakdown.SupportBonus,
		"status_penalty":       snap.StatusPenalty,
		"env_bonus":            snap.Breakdown.EnvBonus,
		"insight_bonus":        snap.Breakdown.InsightBonus,
		"insight_stacks":       snap.InsightStacks,
		"final_rate":           snap.Breakdown.FinalRate,
		"refinement_realm":     snap.RefinementRealm,
		"accuracy":             accuracy,
		"accuracy_bonus":       accuracyBonus,
		"accuracy_max_bonus":   snap.AccuracyMaxBonus,
		"death_penalty":        session.DeathPenalty,
		"step_scores":          stepScores,
	}
	resultPayload := map[string]interface{}{
		"reason":         reason,
		"outcome":        string(outcome),
		"message":        msg.String(),
		"rate_breakdown": breakdown,
	}
	if outcome == OutcomeSuccess {
		resultPayload["new_gu"] = newGuInstance
	}
	resultRaw, err := json.Marshal(resultPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	if _, err := tx.Exec(`UPDATE refine_sessions
		SET status='RESOLVED', accuracy=$1, result=$2, resolved_at=NOW()
		WHERE id=$3`, accuracy, string(resultRaw), session.ID); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// ── Post-commit (best effort) ──
	ClearPendingPlayerQueues(player.ID.String())
	refreshActionPlayerState(&player)
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, ItemInventoryDomain, CombatStatsDomain)

	notifyTitle := "Luyện Cổ Thất Bại"
	if outcome == OutcomeSuccess {
		notifyTitle = "Luyện Cổ Thành Công"
	}
	notify.Send(db.DB, player.ID, "SYSTEM", notifyTitle, msg.String())

	targetName := session.TargetGuID
	if targetSpec, ok := models.GetGu(session.TargetGuID, session.TargetRank); ok {
		targetName = targetSpec.Name
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &player.ID,
		Action:         "refine.session.resolve",
		EntityType:     "refine_session",
		EntityID:       sessionID,
		RequestSummary: fmt.Sprintf("Phiên luyện %s: %s R%d | Outcome: %s | Accuracy %.1f%%", reason, targetName, session.TargetRank, outcome, accuracy),
		Metadata: map[string]interface{}{
			"recipe_id": session.RecipeID, "outcome": string(outcome), "reason": reason,
			"accuracy": accuracy, "final_rate": finalRate, "death_penalty": session.DeathPenalty,
		},
		Success: outcome == OutcomeSuccess,
	})

	if resolved, err := getRefineSessionRow(db.DB, sessionID, session.PlayerID); err == nil {
		publishRefineSession(session.PlayerID, publicSessionView(resolved, time.Now().UTC()))
		return resolved, nil
	}
	// DB reload failed — still return the in-memory resolution.
	session.Status = "RESOLVED"
	session.ResultRaw = resultRaw
	session.Accuracy = accuracy
	session.ResolvedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	publishRefineSession(session.PlayerID, publicSessionView(session, time.Now().UTC()))
	return session, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (s *refineSessionRow) playerUUID() uuid.UUID {
	id, _ := uuid.Parse(s.PlayerID)
	return id
}

// updateVitalGusDying marks vital ingredient Gu into Hấp Hối (unlocked).
func updateVitalGusDying(tx *sql.Tx, instanceIDs []string) error {
	if len(instanceIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(`UPDATE player_gu
		SET hunger=1, state='DYING', equipped=false, is_locked=false, updated_at=NOW()
		WHERE id = ANY($1::text[])`, pq.Array(instanceIDs))
	return err
}

// StartRefineSessionWorker sweeps overdue ACTIVE sessions every 30s
// (WF-REF-05 §12). Each session resolves in its own short transaction; one
// failing session never blocks the others.
func StartRefineSessionWorker(ctx context.Context) {
	worker.RunEvery(ctx, 30*time.Second, "Refine Session Worker", func(ctx context.Context) {
		resolveDueRefineSessions(ctx)
	})
}

func resolveDueRefineSessions(ctx context.Context) {
	if db.DB == nil {
		return
	}
	rows, err := db.DB.Query(`SELECT id FROM refine_sessions WHERE status='ACTIVE' AND ends_at <= NOW() ORDER BY ends_at ASC LIMIT 50`)
	if err != nil {
		logger.S.Errorf("refine session worker: query due sessions: %v", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if _, err := resolveRefineSession(ctx, id, "expired"); err != nil {
			logger.S.Errorf("refine session worker: resolve %s: %v", id, err)
		}
	}
}
