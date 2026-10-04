package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// deathPenaltyConfig — mức phạt khi chết (áp vào túi thật).
// Chết PvE rơi vào hư không; chết PvP rơi cho kẻ thắng.
// Chống lợi dụng chết chuyển đồ: Tăng tỷ lệ hao hụt tài nguyên và áp dụng
// Thiên Đạo Tiêu Hủy (Void Destruction) để tiêu hủy vĩnh viễn một phần tài sản.
var deathPenaltyConfig = struct {
	// Tiền (Nguyên Thạch / Tiên Nguyên Thạch) — % hao hụt khi chết (tăng lên 40%)
	PrimevalStonePct float64
	ImmortalStonePct float64

	// Vật phẩm (nguyên liệu trong túi thật) (tăng lên 40%)
	ItemsEnabled    bool
	ItemsTypePct    float64
	ItemsMaxPerType int

	// Cổ trùng (luôn loại trừ Cổ Bản Mệnh; Cổ trang bị chỉ rơi khi Sát Khí > 5) (tăng lên 30%)
	GusEnabled  bool
	GusPct      float64
	GusMaxCount int

	// Bí phương (player_recipe_progress) (tăng lên 30%)
	RecipesEnabled  bool
	RecipesPct      float64
	RecipesMaxCount int

	// Tỷ lệ tiêu hủy vĩnh viễn (Void Destruction Ratio): 40% lượng mất bị Thiên Đạo xóa sổ vĩnh viễn
	VoidDestructionPct float64
}{
	PrimevalStonePct: 40,
	ImmortalStonePct: 40,

	ItemsEnabled:    true,
	ItemsTypePct:    40,
	ItemsMaxPerType: 30,

	GusEnabled:  true,
	GusPct:      30,
	GusMaxCount: 20,

	RecipesEnabled:  true,
	RecipesPct:      30,
	RecipesMaxCount: 20,

	VoidDestructionPct: 40,
}

// satKhiGainForRanks returns how many Sát Khí layers a killer gains from a PvP kill.
// The gain is based on the rank difference between the killer and the victim.
// If the killer has a higher rank than the victim, the gain is multiplied by the difference.

func satKhiGainForRanks(killerRank, victimRank int) int {
	diff := killerRank - victimRank

	if diff <= 0 {
		return 1
	}

	return diff * diff
}

// AddSatKhiStatus adds or increments Sát Khí status (sat_khi) for a PvP killer.
// The status is permanent — it no longer decays over time, only removed/reduced on death.
func AddSatKhiStatus(tx *sql.Tx, killerID *uuid.UUID, gain int) {
	if killerID == nil {
		return
	}
	if gain < 1 {
		gain = 1
	}
	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1 FOR UPDATE`, *killerID).Scan(&rawStatuses)
	if err != nil {
		return
	}

	var sl models.StatusList
	if len(rawStatuses) > 0 {
		_ = json.Unmarshal(rawStatuses, &sl)
	}

	currentLevel := 0
	for _, st := range sl {
		if st.ID == "sat_khi" {
			if st.Value > 0 {
				currentLevel = int(st.Value)
			} else {
				currentLevel = 1
			}
		}
	}

	newLevel := currentLevel + gain // Sát Khí tăng theo chênh lệch cảnh giới kẻ hạ vs nạn nhân

	sl.Add("sat_khi", "pvp_kill", 0, float64(newLevel)) // vĩnh viễn, không tụt theo thời gian
	newBytes, _ := json.Marshal(sl)
	if _, spErr := tx.Exec("SAVEPOINT sp_sat_khi_add"); spErr == nil {
		if _, err := tx.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, newBytes, *killerID); err != nil {
			logger.S.Warnf("AddSatKhiStatus: failed to update statuses for player %s: %v", *killerID, err)
			_, _ = tx.Exec("ROLLBACK TO SAVEPOINT sp_sat_khi_add")
		} else {
			_, _ = tx.Exec("RELEASE SAVEPOINT sp_sat_khi_add")
		}
	}

	notify.Send(tx, *killerID, "sat_khi_increase", "Sát Khí gia tăng!",
		fmt.Sprintf("Tích tụ oán khí đồ sát! Sát Khí của bạn tăng %d tầng (lên Tầng %d). Nếu bị đánh bại, lượng tài sản đánh rơi sẽ gia tăng x%.1f!", gain, newLevel, 1.0+float64(newLevel)*0.5))

	// Broadcast wanted / hunt notice when killer has slaughtered multiple players (>= 3 kills / Sát Khí level >= 3)
	if newLevel >= 3 {
		var killerName, killerRegion string
		var killerX, killerY int
		_ = tx.QueryRow(`SELECT username, COALESCE(current_region, ''), x, y FROM players WHERE id = $1`, *killerID).Scan(&killerName, &killerRegion, &killerX, &killerY)

		regionName := killerRegion
		if z := ZoneByID(killerRegion); z != nil && z.Name != "" {
			regionName = z.Name
		}
		var alertTitle string
		switch {
		case newLevel >= 10:
			alertTitle = "🔴 [ĐẠI LỆNH TRUY SÁT - MA ĐẦU ĐẠI NẠN]"
		case newLevel >= 5:
			alertTitle = "⚠️ [LỆNH TRUY SÁT - SÁT NHÂN CUỒNG MA]"
		default:
			alertTitle = "⚡ [CẢNH BÁO TRUY SÁT]"
		}
		msg := fmt.Sprintf("%s Cổ Sư [%s] đã đồ sát liên tiếp %d đồng đạo, oán khí ngút trời (Sát Khí Tầng %d) tại %s (%d, %d)! Chư vị đồng đạo hãy liên thủ vây bắt tru diệt ma đầu!",
			alertTitle, killerName, newLevel, newLevel, regionName, killerX, killerY)
		_ = SendSystemMessage("world", sql.NullString{}, msg)
	}
}

// getSatKhiMultiplier checks if player has sat_khi status and returns (multiplier, level).
func getSatKhiMultiplier(tx *sql.Tx, playerID uuid.UUID) (float64, int) {
	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1`, playerID).Scan(&rawStatuses)
	if err != nil || len(rawStatuses) == 0 {
		return 1.0, 0
	}

	var sl models.StatusList
	if err := json.Unmarshal(rawStatuses, &sl); err != nil {
		return 1.0, 0
	}

	active := sl.Active()
	for _, st := range active {
		if st.ID == "sat_khi" {
			level := int(st.Value)
			if level <= 0 {
				level = 1
			}
			mult := 1.0 + float64(level)*0.5
			return mult, level
		}
	}

	return 1.0, 0
}

// reduceSatKhiOnDeath decrements or removes sat_khi status after dying.
func reduceSatKhiOnDeath(tx *sql.Tx, playerID uuid.UUID) {
	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1 FOR UPDATE`, playerID).Scan(&rawStatuses)
	if err != nil || len(rawStatuses) == 0 {
		return
	}

	var sl models.StatusList
	if err := json.Unmarshal(rawStatuses, &sl); err != nil {
		return
	}

	hasSatKhi := false
	currentLevel := 0
	for _, st := range sl {
		if st.ID == "sat_khi" {
			hasSatKhi = true
			currentLevel = int(st.Value)
			if currentLevel <= 0 {
				currentLevel = 1
			}
		}
	}

	if !hasSatKhi {
		return
	}

	if currentLevel > 1 {
		sl.Add("sat_khi", "pvp_death", 0, float64(currentLevel-1)) // vĩnh viễn, không tụt theo thời gian
	} else {
		sl.Remove("sat_khi")
	}

	newBytes, _ := json.Marshal(sl)
	if _, spErr := tx.Exec("SAVEPOINT sp_sat_khi_reduce"); spErr == nil {
		if _, err := tx.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, newBytes, playerID); err != nil {
			logger.S.Warnf("reduceSatKhiOnDeath: failed to update statuses for player %s: %v", playerID, err)
			_, _ = tx.Exec("ROLLBACK TO SAVEPOINT sp_sat_khi_reduce")
		} else {
			_, _ = tx.Exec("RELEASE SAVEPOINT sp_sat_khi_reduce")
		}
	}
}

type DeathPos struct {
	Region string
	X      int
	Y      int
}

// applyDeathDrop applies the death penalty to a fully-loaded in-memory loser.
func applyDeathDrop(tx *sql.Tx, loser *models.Player, winnerID *uuid.UUID, isAggressor bool, deathPos ...DeathPos) string {
	mult, satKhiLevel := getSatKhiMultiplier(tx, loser.ID)
	remainingGus, droppedGus := selectEligibleGus(loser.GuInventory, mult, satKhiLevel)
	loser.GuInventory = remainingGus
	if len(droppedGus) > 0 && len(loser.EquippedGus) > 0 {
		droppedMap := make(map[string]bool)
		for _, g := range droppedGus {
			droppedMap[g.InstanceID] = true
		}
		newEquipped := make(models.EquippedGusList, 0, len(loser.EquippedGus))
		for _, id := range loser.EquippedGus {
			if !droppedMap[id] {
				newEquipped = append(newEquipped, id)
			}
		}
		loser.EquippedGus = newEquipped
	}

	effectiveStonePct := math.Min(100.0, deathPenaltyConfig.PrimevalStonePct*mult)
	primevalLoss := math.Floor(loser.PrimevalStones*effectiveStonePct/100*1000) / 1000
	effectiveImmortalPct := math.Min(100.0, deathPenaltyConfig.ImmortalStonePct*mult)
	immortalLoss := math.Floor(loser.ImmortalStones*effectiveImmortalPct/100*1000) / 1000
	if primevalLoss < 0 {
		primevalLoss = 0
	}
	if immortalLoss < 0 {
		immortalLoss = 0
	}
	if primevalLoss > loser.PrimevalStones {
		primevalLoss = loser.PrimevalStones
	}
	if immortalLoss > loser.ImmortalStones {
		immortalLoss = loser.ImmortalStones
	}
	loser.PrimevalStones -= primevalLoss
	loser.ImmortalStones -= immortalLoss

	dp := deathPos
	if len(dp) == 0 && loser.CurrentRegion != "" {
		dp = []DeathPos{{Region: loser.CurrentRegion, X: loser.X, Y: loser.Y}}
	}

	return persistDeathDrop(tx, loser.ID, winnerID, isAggressor, primevalLoss, immortalLoss, droppedGus, mult, satKhiLevel, dp...)
}

// applyDeathDropRecord applies the death penalty to a player record that is not fully loaded in memory.
func applyDeathDropRecord(tx *sql.Tx, loserID, winnerID *uuid.UUID, isAggressor bool, deathPos ...DeathPos) string {
	var primevalStones, immortalStones float64
	err := tx.QueryRow(`
		SELECT primeval_stones, immortal_stones
		FROM players WHERE id = $1 FOR UPDATE
	`, *loserID).Scan(&primevalStones, &immortalStones)
	if err != nil {
		return ""
	}

	mult, satKhiLevel := getSatKhiMultiplier(tx, *loserID)

	effectiveStonePct := math.Min(100.0, deathPenaltyConfig.PrimevalStonePct*mult)
	primevalLoss := math.Floor(primevalStones*effectiveStonePct/100*1000) / 1000
	effectiveImmortalPct := math.Min(100.0, deathPenaltyConfig.ImmortalStonePct*mult)
	immortalLoss := math.Floor(immortalStones*effectiveImmortalPct/100*1000) / 1000
	if primevalLoss < 0 {
		primevalLoss = 0
	}
	if immortalLoss < 0 {
		immortalLoss = 0
	}
	if primevalLoss > primevalStones {
		primevalLoss = primevalStones
	}
	if immortalLoss > immortalStones {
		immortalLoss = immortalStones
	}

	allGus, _ := db.LoadPlayerGus(tx, loserID.String(), false)
	_, droppedGus := selectEligibleGus(allGus, mult, satKhiLevel)

	return persistDeathDrop(tx, *loserID, winnerID, isAggressor, primevalLoss, immortalLoss, droppedGus, mult, satKhiLevel, deathPos...)
}

type DroppedItem struct {
	ItemID   string `json:"item_id"`
	Category string `json:"category"`
	Quantity int    `json:"quantity"`
}

type GroundLootData struct {
	PrimevalStones float64            `json:"primeval_stones,omitempty"`
	ImmortalStones float64            `json:"immortal_stones,omitempty"`
	Items          []DroppedItem      `json:"items,omitempty"`
	Gus            models.GuInventory `json:"gus,omitempty"`
	Recipes        []string           `json:"recipes,omitempty"`
}

// persistDeathDrop writes the computed losses to DB, creates a map_ground_drops record, and builds summary.
func persistDeathDrop(tx *sql.Tx, loserID uuid.UUID, winnerID *uuid.UUID, isAggressor bool, primevalLoss, immortalLoss float64, droppedGus models.GuInventory, mult float64, satKhiLevel int, deathPos ...DeathPos) string {
	// Split stones: Void Destruction vs Ground Loot
	voidPct := deathPenaltyConfig.VoidDestructionPct / 100.0

	primevalDestroyed := math.Floor(primevalLoss*voidPct*1000) / 1000
	primevalLoot := math.Floor((primevalLoss-primevalDestroyed)*1000) / 1000

	immortalDestroyed := math.Floor(immortalLoss*voidPct*1000) / 1000
	immortalLoot := math.Floor((immortalLoss-immortalDestroyed)*1000) / 1000

	lootData := GroundLootData{
		PrimevalStones: primevalLoot,
		ImmortalStones: immortalLoot,
	}
	var lost []string

	// Tiền — trừ bên thua
	if primevalLoss >= 1 || immortalLoss >= 1 {
		if _, err := tx.Exec(`
			UPDATE players SET
				primeval_stones = GREATEST(0, primeval_stones - $1),
				immortal_stones = GREATEST(0, immortal_stones - $2)
			WHERE id = $3
		`, primevalLoss, immortalLoss, loserID); err != nil {
			return ""
		}
		if primevalLoss >= 1 {
			if primevalDestroyed > 0 && primevalLoot > 0 {
				lost = append(lost, fmt.Sprintf("%.3f Nguyên Thạch (%.3f rơi tại chỗ, %.3f bị tiêu hủy vĩnh viễn)", primevalLoss, primevalLoot, primevalDestroyed))
			} else if primevalDestroyed > 0 {
				lost = append(lost, fmt.Sprintf("%.3f Nguyên Thạch (bị tiêu hủy vĩnh viễn)", primevalLoss))
			} else {
				lost = append(lost, fmt.Sprintf("%.3f Nguyên Thạch (rơi tại chỗ)", primevalLoss))
			}
		}
		if immortalLoss >= 1 {
			if immortalDestroyed > 0 && immortalLoot > 0 {
				lost = append(lost, fmt.Sprintf("%.3f Tiên Nguyên Thạch (%.3f rơi tại chỗ, %.3f bị tiêu hủy vĩnh viễn)", immortalLoss, immortalLoot, immortalDestroyed))
			} else if immortalDestroyed > 0 {
				lost = append(lost, fmt.Sprintf("%.3f Tiên Nguyên Thạch (bị tiêu hủy vĩnh viễn)", immortalLoss))
			} else {
				lost = append(lost, fmt.Sprintf("%.3f Tiên Nguyên Thạch (rơi tại chỗ)", immortalLoss))
			}
		}
	}

	// Vật phẩm (nguyên liệu túi thật)
	if deathPenaltyConfig.ItemsEnabled {
		matItems, matDestroyed, matLogs := dropMaterials(tx, loserID, mult, satKhiLevel)
		_ = matDestroyed
		lootData.Items = matItems
		lost = append(lost, matLogs...)
	}

	var loserRegion sql.NullString
	var loserX, loserY int
	var loserName string

	if len(deathPos) > 0 && deathPos[0].Region != "" {
		loserRegion.String = deathPos[0].Region
		loserRegion.Valid = true
		loserX = deathPos[0].X
		loserY = deathPos[0].Y
		_ = tx.QueryRow(`SELECT username FROM players WHERE id = $1`, loserID).Scan(&loserName)
	} else {
		_ = tx.QueryRow(`SELECT current_region, x, y, username FROM players WHERE id = $1`, loserID).Scan(&loserRegion, &loserX, &loserY, &loserName)
	}

	// Cổ trùng
	if deathPenaltyConfig.GusEnabled && len(droppedGus) > 0 {
		ids := make([]string, 0, len(droppedGus))
		guDestroyCount := int(math.Round(float64(len(droppedGus)) * voidPct))
		if len(droppedGus) == 1 {
			if rand.Float64() < 0.5 {
				guDestroyCount = 1
			} else {
				guDestroyCount = 0
			}
		} else if guDestroyCount >= len(droppedGus) {
			guDestroyCount = len(droppedGus) - 1
			if guDestroyCount < 0 {
				guDestroyCount = 0
			}
		}

		destroyedGus := droppedGus[:guDestroyCount]
		groundGus := droppedGus[guDestroyCount:]
		lootData.Gus = groundGus

		for _, g := range destroyedGus {
			ids = append(ids, g.InstanceID)
			name := g.GuID
			if guDef, ok := models.GetGuAny(g.GuID); ok && guDef.Name != "" {
				name = guDef.Name
			}
			lost = append(lost, fmt.Sprintf("Cổ trùng %s (bị chấn nát vĩnh viễn)", name))
			deathSummary := fmt.Sprintf("Cổ Sư [%s] tử đạo, Cổ trùng [%s] R%d bị chấn nát vĩnh viễn vào cõi hư vô.", loserName, name, g.Rank)
			_ = db.LogGuHistory(tx, g.InstanceID, g.GuID, g.Rank, "DEATH_DESTROYED", &loserID, winnerID, deathSummary, map[string]interface{}{
				"gu_name": name,
			})
		}

		for _, g := range groundGus {
			ids = append(ids, g.InstanceID)
			name := g.GuID
			if guDef, ok := models.GetGuAny(g.GuID); ok && guDef.Name != "" {
				name = guDef.Name
			}
			lost = append(lost, fmt.Sprintf("Cổ trùng %s (rơi tại chỗ)", name))
			deathSummary := fmt.Sprintf("Cổ Sư [%s] tử đạo, đánh rơi Cổ trùng [%s] R%d xuống đất.", loserName, name, g.Rank)
			_ = db.LogGuHistory(tx, g.InstanceID, g.GuID, g.Rank, "DEATH_DROPPED", &loserID, winnerID, deathSummary, map[string]interface{}{
				"gu_name": name,
			})
		}
		_ = db.RemovePlayerGus(tx, ids)
	}

	// Bí phương
	if deathPenaltyConfig.RecipesEnabled {
		recipeIDs, recipeDestroyed, recipeLogs := dropRecipes(tx, loserID, mult, satKhiLevel)
		_ = recipeDestroyed
		lootData.Recipes = recipeIDs
		lost = append(lost, recipeLogs...)
	}

	// Nếu người thắng là kẻ chủ động tấn công (aggressor), tăng Sát Khí cho kẻ chiến thắng.
	// Lượng sát khí tăng theo chênh lệch cảnh giới: rank càng cao hạ rank càng thấp → càng nhiều.
	if winnerID != nil && isAggressor {
		var winnerRank, loserRank int
		_ = tx.QueryRow(`SELECT COALESCE(rank, 0) FROM players WHERE id = $1`, *winnerID).Scan(&winnerRank)
		_ = tx.QueryRow(`SELECT COALESCE(rank, 0) FROM players WHERE id = $1`, loserID).Scan(&loserRank)
		AddSatKhiStatus(tx, winnerID, satKhiGainForRanks(winnerRank, loserRank))
	}

	// Nếu người thua có Sát Khí, giảm 1 tầng Sát Khí sau khi trả nợ máu bằng tử đạo
	if satKhiLevel > 0 {
		reduceSatKhiOnDeath(tx, loserID)
	}

	var killerName string
	if winnerID != nil {
		_ = tx.QueryRow(`SELECT username FROM players WHERE id = $1`, *winnerID).Scan(&killerName)
	}

	summary := ""
	hasGroundLoot := lootData.PrimevalStones > 0 || lootData.ImmortalStones > 0 || len(lootData.Items) > 0 || len(lootData.Gus) > 0 || len(lootData.Recipes) > 0

	if len(lost) > 0 {
		summary = joinDropList(lost)
		if satKhiLevel > 0 {
			summary += fmt.Sprintf(" [Oán Khí Tầng %d: Mất đồ x%.1f]", satKhiLevel, mult)
		}

		if hasGroundLoot && loserRegion.Valid && loserRegion.String != "" && !IsSafeZone(loserRegion.String, loserX, loserY) {
			lootBytes, _ := json.Marshal(lootData)
			if _, spErr := tx.Exec("SAVEPOINT sp_ground_drop"); spErr == nil {
				_, errDrop := tx.Exec(`
					INSERT INTO map_ground_drops (
						region_id, x, y, loser_id, loser_name, killer_id, killer_name,
						loot_data, protected_until, expires_at
					) VALUES (
						$1, $2, $3, $4, $5, $6, $7,
						$8, NOW() + INTERVAL '3 minutes', NOW() + INTERVAL '10 minutes'
					)
				`, loserRegion.String, loserX, loserY, loserID, loserName, winnerID, killerName, lootBytes)
				if errDrop != nil {
					logger.S.Warnf("persistDeathDrop: failed to insert map_ground_drops for loser %s: %v", loserID, errDrop)
					_, _ = tx.Exec("ROLLBACK TO SAVEPOINT sp_ground_drop")
				} else {
					_, _ = tx.Exec("RELEASE SAVEPOINT sp_ground_drop")
				}
			}
		}
	}

	// Notify loser and (PvP) winner — best-effort, non-fatal.
	if winnerID != nil {
		attackerName := killerName
		if attackerName == "" {
			attackerName = "Kẻ thù"
		}
		if summary != "" {
			notify.Send(tx, loserID, "death_drop", "Tử đạo - Bị hạ gục (Tổn thất tài sản)",
				fmt.Sprintf("Bạn đã bị Cổ Sư [%s] hạ gục tại (%d, %d)! Tổn thất: %s. (Lưu ý: Thiên Đạo chấn nát tiêu hủy vĩnh viễn một phần tài nguyên; phần còn lại rơi tại chỗ và sẽ tan biến sau 10 phút). Bạn đã được đưa về nơi an dưỡng và hồi sinh!", attackerName, loserX, loserY, summary))
			notify.Send(tx, *winnerID, "loot_gained", "Kẻ thù bị hạ gục",
				fmt.Sprintf("Bạn đã hạ gục Cổ Sư [%s]! Chiến lợi phẩm của đối phương rơi tại (%d, %d): %s. (Lưu ý: Một phần tài sản đối phương đã bị tiêu hủy vĩnh viễn trong trận chiến).", loserName, loserX, loserY, summary))
		} else {
			notify.Send(tx, loserID, "death_drop", "Tử đạo - Bị hạ gục",
				fmt.Sprintf("Bạn đã bị Cổ Sư [%s] hạ gục tại (%d, %d). May mắn không bị rơi rớt tài sản! Bạn đã được đưa về nơi an dưỡng và hồi sinh!", attackerName, loserX, loserY))
			notify.Send(tx, *winnerID, "loot_gained", "Kẻ thù bị hạ gục",
				fmt.Sprintf("Bạn đã hạ gục Cổ Sư [%s] tại (%d, %d)! Đối phương không có tài sản rơi rớt.", loserName, loserX, loserY))
		}
	} else if summary != "" {
		notify.Send(tx, loserID, "death_drop", "Tử đạo - Mất tài sản",
			fmt.Sprintf("Bạn đã tử đạo tại (%d, %d)! Tổn thất: %s. (Lưu ý: Thiên Đạo chấn nát tiêu hủy vĩnh viễn một phần tài nguyên; phần còn lại rơi tại chỗ và sẽ tan biến sau 10 phút).", loserX, loserY, summary))
	}

	// Broadcast justice announcement to world chat when a wanted player (Sát Khí >= 3) is defeated
	if satKhiLevel >= 3 && winnerID != nil {
		regionName := loserRegion.String
		if z := ZoneByID(loserRegion.String); z != nil && z.Name != "" {
			regionName = z.Name
		}
		killerDisplayName := killerName
		if killerDisplayName == "" {
			killerDisplayName = "Nghĩa sĩ Cổ Sư"
		}
		justiceMsg := fmt.Sprintf("⚔️ [THIÊN ĐẠO TRỪ MA] Đại ma đầu [%s] (Sát Khí Tầng %d) đã bị nghĩa sĩ [%s] trảm sát tại %s (%d, %d)! Trừ đi một mối đại họa cho Cổ Giới!",
			loserName, satKhiLevel, killerDisplayName, regionName, loserX, loserY)
		_ = SendSystemMessage("world", sql.NullString{}, justiceMsg)
	}

	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &loserID,
		Action:         "combat.death_penalty",
		EntityType:     "player",
		EntityID:       loserID.String(),
		RequestSummary: fmt.Sprintf("Tử đạo đánh rơi tài sản: %s", summary),
		Metadata: map[string]interface{}{
			"lost_items": lost,
			"winner_id":  winnerID,
		},
		Success: true,
	})

	return summary
}

// selectEligibleGus returns the remaining inventory and the randomly chosen Gus to drop.
// Vital Gu (Cổ Bản Mệnh) is NEVER dropped.
// Equipped Gus can only be dropped if satKhiLevel > 0.
func selectEligibleGus(gus models.GuInventory, mult float64, satKhiLevel int) (models.GuInventory, models.GuInventory) {
	eligible := make(models.GuInventory, 0, len(gus))
	for _, g := range gus {
		if g.IsVital {
			continue
		}
		if g.Equipped && satKhiLevel <= 0 {
			continue
		}
		eligible = append(eligible, g)
	}
	if !deathPenaltyConfig.GusEnabled || len(eligible) == 0 {
		return gus, models.GuInventory{}
	}

	effectivePct := math.Min(100.0, deathPenaltyConfig.GusPct*mult)

	count := int(math.Ceil(float64(len(eligible)) * effectivePct / 100.0))
	if count > len(eligible) {
		count = len(eligible)
	}

	rand.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })
	dropped := eligible[:count]

	dropIDs := make(map[string]bool, len(dropped))
	for _, g := range dropped {
		dropIDs[g.InstanceID] = true
	}
	remaining := make(models.GuInventory, 0, len(gus))
	for _, g := range gus {
		if dropIDs[g.InstanceID] {
			continue
		}
		remaining = append(remaining, g)
	}
	return remaining, dropped
}

// dropMaterials samples a portion of the loser's real inventory by percentage and removes it in batch.
// Splitting lost items: a portion is destroyed forever (Void Destruction), and the rest is dropped on the ground.
// Returns ground dropped items, destroyed items, and human-readable lost lines.
func dropMaterials(tx *sql.Tx, loserID uuid.UUID, mult float64, satKhiLevel int) ([]DroppedItem, []DroppedItem, []string) {
	rows, err := tx.Query(`
		SELECT item_id, category, quantity
		FROM player_inventory_items
		WHERE player_id = $1 AND quantity > 0
	`, loserID)
	if err != nil {
		return nil, nil, nil
	}
	defer rows.Close()

	type row struct {
		itemID   string
		category string
		quantity int
	}
	var items []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.itemID, &r.category, &r.quantity); err != nil {
			continue
		}
		items = append(items, r)
	}

	if len(items) == 0 {
		return nil, nil, nil
	}

	effectivePct := math.Min(100.0, deathPenaltyConfig.ItemsTypePct*mult)

	pickCount := int(math.Ceil(float64(len(items)) * effectivePct / 100.0))
	if pickCount > len(items) {
		pickCount = len(items)
	}

	rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })

	var groundDropped []DroppedItem
	var destroyed []DroppedItem
	var lost []string
	itemsByCat := make(map[string]map[string]int64)

	voidPct := deathPenaltyConfig.VoidDestructionPct / 100.0

	for _, r := range items[:pickCount] {
		// Calculated percentage of stack quantity
		qty := int(math.Ceil(float64(r.quantity) * effectivePct / 100.0))
		if qty > r.quantity {
			qty = r.quantity
		}
		if qty <= 0 {
			qty = 1
		}

		if itemsByCat[r.category] == nil {
			itemsByCat[r.category] = make(map[string]int64)
		}
		itemsByCat[r.category][r.itemID] = int64(qty)

		// Split between void destruction and ground drop
		destroyCount := int(math.Round(float64(qty) * voidPct))
		if qty == 1 {
			// If only 1 item lost, 50% chance destroyed forever
			if rand.Float64() < 0.5 {
				destroyCount = 1
			} else {
				destroyCount = 0
			}
		} else if destroyCount >= qty {
			destroyCount = qty - 1
			if destroyCount < 0 {
				destroyCount = 0
			}
		}
		dropCount := qty - destroyCount

		if dropCount > 0 {
			groundDropped = append(groundDropped, DroppedItem{
				ItemID:   r.itemID,
				Category: r.category,
				Quantity: dropCount,
			})
		}
		if destroyCount > 0 {
			destroyed = append(destroyed, DroppedItem{
				ItemID:   r.itemID,
				Category: r.category,
				Quantity: destroyCount,
			})
		}

		name := r.itemID
		if spec, ok := models.ItemRegistry[r.itemID]; ok && spec.Name != "" {
			name = spec.Name
		}
		if destroyCount > 0 && dropCount > 0 {
			lost = append(lost, fmt.Sprintf("%dx %s (%dx rơi tại chỗ, %dx bị chấn nát vĩnh viễn)", qty, name, dropCount, destroyCount))
		} else if destroyCount > 0 {
			lost = append(lost, fmt.Sprintf("%dx %s (bị chấn nát vĩnh viễn)", qty, name))
		} else {
			lost = append(lost, fmt.Sprintf("%dx %s (rơi tại chỗ)", qty, name))
		}
	}

	// Batch remove inventory items by category
	for cat, itemMap := range itemsByCat {
		_ = InventorySvc.BulkRemoveItems(tx, loserID.String(), cat, itemMap)
	}

	return groundDropped, destroyed, lost
}

// dropRecipes samples a portion of the loser's unlocked recipes by percentage and removes them in batch.
// Splitting lost recipes: a portion is destroyed forever (Void Destruction), and the rest is dropped on the ground.
// Returns ground dropped recipe IDs, destroyed recipe IDs, and human-readable lost lines.
func dropRecipes(tx *sql.Tx, loserID uuid.UUID, mult float64, satKhiLevel int) ([]string, []string, []string) {
	rows, err := tx.Query(`
		SELECT recipe_id FROM player_recipe_progress
		WHERE player_id = $1 AND is_unlocked = true
	`, loserID)
	if err != nil {
		return nil, nil, nil
	}
	defer rows.Close()

	var recipeIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		recipeIDs = append(recipeIDs, id)
	}
	if len(recipeIDs) == 0 {
		return nil, nil, nil
	}

	effectivePct := math.Min(100.0, deathPenaltyConfig.RecipesPct*mult)

	pickCount := int(math.Ceil(float64(len(recipeIDs)) * effectivePct / 100.0))
	if pickCount > len(recipeIDs) {
		pickCount = len(recipeIDs)
	}

	rand.Shuffle(len(recipeIDs), func(i, j int) { recipeIDs[i], recipeIDs[j] = recipeIDs[j], recipeIDs[i] })

	droppedIDs := recipeIDs[:pickCount]

	// Batch delete recipes in a single query
	if len(droppedIDs) > 0 {
		_ = RecipeSvc.DeleteBatch(tx, loserID.String(), droppedIDs)
	}

	voidPct := deathPenaltyConfig.VoidDestructionPct / 100.0
	destroyCount := int(math.Round(float64(len(droppedIDs)) * voidPct))
	if len(droppedIDs) == 1 {
		if rand.Float64() < 0.5 {
			destroyCount = 1
		} else {
			destroyCount = 0
		}
	} else if destroyCount >= len(droppedIDs) {
		destroyCount = len(droppedIDs) - 1
		if destroyCount < 0 {
			destroyCount = 0
		}
	}

	destroyedIDs := droppedIDs[:destroyCount]
	groundIDs := droppedIDs[destroyCount:]

	var lost []string
	for _, id := range destroyedIDs {
		name := id
		if guDef, ok := models.GetGuAny(id); ok && guDef.Name != "" {
			name = guDef.Name
		}
		lost = append(lost, fmt.Sprintf("Bí phương %s (bị tiêu hủy vĩnh viễn)", name))
	}
	for _, id := range groundIDs {
		name := id
		if guDef, ok := models.GetGuAny(id); ok && guDef.Name != "" {
			name = guDef.Name
		}
		lost = append(lost, fmt.Sprintf("Bí phương %s (rơi tại chỗ)", name))
	}

	return groundIDs, destroyedIDs, lost
}

func joinDropList(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
