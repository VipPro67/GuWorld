package handlers

// Shared harvest-economics helpers for the Immortal Aperture resource-point
// economy (see docs/DESIGN_TIEN_KHAU.md §6) plus bulk defence repair. The old
// per-node harvest flow was retired together with the resource-point rebuild.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// parseItemPaths tolerates both a plain path string and a JSON array.
func parseItemPaths(pathRaw string) []string {
	pathRaw = strings.TrimSpace(pathRaw)
	if pathRaw == "" {
		return nil
	}
	var paths []string
	if strings.HasPrefix(pathRaw, "[") && strings.HasSuffix(pathRaw, "]") {
		_ = json.Unmarshal([]byte(pathRaw), &paths)
	}
	if len(paths) == 0 {
		paths = []string{pathRaw}
	}
	return paths
}

// normalizePathKey maps every authoring spelling of a path to the dao-mark
// key used across economia and combat ("Wood Path" → "wood", ...).
func normalizePathKey(path string) string {
	p := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(path)), " path")
	p = strings.TrimSuffix(p, "_path")
	switch p {
	case "vận", "vận đạo", "luck":
		return "luck"
	case "mộc", "mộc đạo", "wood":
		return "wood"
	case "hỏa", "hỏa đạo", "fire":
		return "fire"
	case "thủy", "thủy đạo", "water":
		return "water"
	case "kim", "kim đạo", "metal":
		return "metal"
	case "thổ", "thổ đạo", "earth":
		return "earth"
	case "lôi", "lôi đạo", "lightning":
		return "lightning"
	case "băng", "băng đạo", "ice":
		return "ice"
	case "phong", "phong đạo", "wind":
		return "wind"
	case "quang", "quang đạo", "light":
		return "light"
	case "ám", "ám đạo", "dark":
		return "dark"
	case "kiếm", "kiếm đạo", "sword":
		return "sword"
	case "đao", "đao đạo", "blade":
		return "blade"
	case "thực", "thực đạo", "food":
		return "food"
	case "biến hóa", "transformation":
		return "transformation"
	case "nô", "enslavement":
		return "enslavement"
	case "tín", "information":
		return "information"
	case "huyết", "blood":
		return "blood"
	case "độc", "poison":
		return "poison"
	case "nhân", "human":
		return "human"
	case "mộng", "dream":
		return "dream"
	case "huyễn", "phantom":
		return "phantom"
	case "sát", "killing":
		return "killing"
	case "lực", "strength":
		return "strength"
	case "trí", "wisdom":
		return "wisdom"
	default:
		return p
	}
}

func getPathDaoMultiplier(pathRaw string, daoMarks map[string]int64) float64 {
	paths := parseItemPaths(pathRaw)
	if len(paths) == 0 {
		return 1.0
	}
	var bestMarks int64
	for _, p := range paths {
		key := normalizePathKey(p)
		if marks, ok := daoMarks[key]; ok && marks > bestMarks {
			bestMarks = marks
		}
	}
	return 1.0 + float64(bestMarks)*0.0001
}

// nodeDaoPurity returns Dao purity (dominant / total). 0 = empty map, callers
// treat that as "no penalty".
func nodeDaoPurity(daoMarks map[string]int64) float64 {
	var total, dominant int64
	for _, v := range daoMarks {
		if v <= 0 {
			continue
		}
		total += v
		if v > dominant {
			dominant = v
		}
	}
	if total == 0 {
		return 0
	}
	return float64(dominant) / float64(total)
}

func loadApertureDaoCombatProfile(q apertureQueryRower, playerID string, daoMarks map[string]int64) (*gamefx.DaoCombatProfile, gamefx.Effects) {
	var bodyMods models.BodyModsList
	var guInventory models.GuInventory
	var race, constitution string
	var pathMasteries models.PathMasteriesMap
	err := q.QueryRow(`
		SELECT COALESCE(p.body_mods, '[]'::jsonb), COALESCE(p.race, 'human'), COALESCE(p.constitution, ''),
			COALESCE(p.path_masteries, '{}'::jsonb), `+db.GuInventoryAggregate("p.id")+`
		FROM players p WHERE p.id = $1
	`, playerID).Scan(&bodyMods, &race, &constitution, &pathMasteries, &guInventory)
	player := &models.Player{
		CharacterState: models.CharacterState{Race: race, Constitution: constitution},
		BodyState:      models.BodyState{BodyMods: bodyMods},
		DaoState:       models.DaoState{DaoMarks: daoMarks, PathMasteries: pathMasteries},
		GuInventory:    guInventory,
	}
	fx := gamefx.Get(player)
	if err != nil {
		return gamefx.BuildDaoCombatProfileWithReduction(daoMarks, nil, false, 0, nil), fx
	}
	return gamefx.BuildDaoCombatProfileWithReduction(daoMarks, pathMasteries, fx.NonConflictingDaoMarks, fx.DaoConflictReductionPct, nil), fx
}

func apertureDaoConflictMultiplierForProfile(profile *gamefx.DaoCombatProfile, pathRaw string) float64 {
	if profile == nil {
		return 1
	}
	paths := parseItemPaths(pathRaw)
	if len(paths) == 0 {
		return 1
	}
	_, _, multiplier := profile.GetOffensiveFactors(gamefx.CleanPath(paths[0]))
	return multiplier
}

func apertureDaoConflictMultiplier(profile *gamefx.DaoCombatProfile, pathRaw string) float64 {
	return apertureDaoConflictMultiplierForProfile(profile, pathRaw)
}

type apertureQueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

// computeBulkRepairCost returns the repair cost for a node after applying a
// bulk fee multiplier (rounded up). Destroyed nodes cannot be repaired.
func computeBulkRepairCost(hp, maxHP int, isDestroyed bool, feeMultiplier float64) (baseCost, totalCost int) {
	if isDestroyed {
		return 0, 0
	}
	baseCost = int(math.Ceil(float64(maxHP-hp) / 50.0))
	if baseCost <= 0 {
		return 0, 0
	}
	totalCost = int(math.Ceil(float64(baseCost) * feeMultiplier))
	return baseCost, totalCost
}

// RepairAllApertureNodes repairs every damaged (DEFENSE) node of the aperture
// in one go for a +30% convenience fee.
func RepairAllApertureNodes(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch aperture
	var apertureID string
	err = tx.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Tiên Khiếu"})
		return
	}

	// 2. Load damaged nodes. Fully destroyed nodes are deleted and cannot be rebuilt.
	rows, err := tx.Query(`
		SELECT id, name, hp, max_hp
		FROM aperture_resource_nodes
		WHERE aperture_id = $1 AND is_destroyed = false AND hp < max_hp
		FOR UPDATE
	`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải danh sách công trình"})
		return
	}

	type repairResult struct {
		NodeID string `json:"node_id"`
		Name   string `json:"name"`
		Cost   int    `json:"cost"`
	}
	var results []repairResult
	totalCost := 0

	for rows.Next() {
		var id, name string
		var hp, maxHP int
		if err := rows.Scan(&id, &name, &hp, &maxHP); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc dữ liệu công trình"})
			return
		}
		_, cost := computeBulkRepairCost(hp, maxHP, false, 1.3)
		if cost <= 0 {
			continue
		}
		totalCost += cost
		results = append(results, repairResult{NodeID: id, Name: name, Cost: cost})
	}
	rows.Close()

	if len(results) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có công trình nào cần sửa chữa."})
		return
	}

	// 3. Check player stones
	var stones int
	err = tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}

	if stones < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để sửa chữa toàn bộ (Yêu cầu: %d, Hiện có: %d).", totalCost, stones)})
		return
	}

	// 4. Deduct + repair
	var newStones float64
	if err := tx.QueryRow(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2 RETURNING immortal_stones`, totalCost, playerID).Scan(&newStones); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}

	nodeIDs := make([]string, len(results))
	for idx, r := range results {
		nodeIDs[idx] = r.NodeID
	}
	if len(nodeIDs) > 0 {
		res, err := tx.Exec(`
			UPDATE aperture_resource_nodes
			SET hp = max_hp, updated_at = NOW()
			WHERE id = ANY($1::uuid[]) AND is_destroyed = false
		`, pq.Array(nodeIDs))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái sửa chữa"})
			return
		}
		if repaired, rowsErr := res.RowsAffected(); rowsErr != nil || repaired != int64(len(nodeIDs)) {
			c.JSON(http.StatusConflict, gin.H{"error": "Một công trình đã bị phá hủy trong lúc sửa chữa; giao dịch đã được hoàn lại"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save repair"})
		return
	}

	repairAllMsg := fmt.Sprintf("Sửa chữa hàng loạt %d công trình thành công! Tiêu tốn tổng cộng %d Tiên Nguyên Thạch (Phí +30%%).", len(results), totalCost)
	c.JSON(http.StatusOK, gin.H{
		"message":        repairAllMsg,
		"repaired_count": len(results),
		"total_cost":     totalCost,
		"nodes":          results,
	})
}
