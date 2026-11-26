package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// EffectiveRegionNaturalDaoMarks returns additive inherited natural Dao Marks
// from the root ancestor down to regionID.
func EffectiveRegionNaturalDaoMarks(regionID string) map[string]float64 {
	out := make(map[string]float64)
	if regionID == "" {
		return out
	}
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	var chain []*models.Region
	visited := make(map[string]bool)
	curID := regionID
	for curID != "" && !visited[curID] {
		visited[curID] = true
		r, ok := zoneReg.byID[curID]
		if !ok || r == nil {
			break
		}
		chain = append(chain, r)
		curID = r.ParentID
	}
	for i := len(chain) - 1; i >= 0; i-- {
		for k, v := range chain[i].NaturalDaoMarks {
			if v > 0 {
				out[gamefx.CleanPath(k)] += v
			}
		}
	}
	return out
}

// EffectiveEnvDaoMarks returns EffectiveRegionNaturalDaoMarks plus any
// persistent map_cell hotspot Dao Marks at (regionID, x, y).
func EffectiveEnvDaoMarks(regionID string, x, y int) map[string]float64 {
	out := EffectiveRegionNaturalDaoMarks(regionID)
	zoneReg.mu.RLock()
	if pc, ok := zoneReg.persistent[persistentCellKey{RegionID: regionID, X: x, Y: y}]; ok {
		for k, v := range pc.DaoMarks {
			if v > 0 {
				out[gamefx.CleanPath(k)] += v
			}
		}
	}
	zoneReg.mu.RUnlock()
	return out
}

// GetRegionDominion returns the cached Dao Domain for regionID if any.
func GetRegionDominion(regionID string) (models.RegionDaoDominion, bool) {
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	d, ok := zoneReg.dominions[regionID]
	return d, ok
}

// SetRegionDominionInCache updates or deletes a dominion entry in zoneReg.
func SetRegionDominionInCache(regionID string, dom *models.RegionDaoDominion) {
	zoneReg.mu.Lock()
	defer zoneReg.mu.Unlock()
	if zoneReg.dominions == nil {
		zoneReg.dominions = make(map[string]models.RegionDaoDominion)
	}
	if dom == nil {
		delete(zoneReg.dominions, regionID)
	} else {
		zoneReg.dominions[regionID] = *dom
	}
}

// buildEnvCombatContext resolves the combat environmental Dao Marks & Dao Domain
// for either aperture interior or world map combat.
func buildEnvCombatContext(tx *sql.Tx, player *models.Player) *gamefx.EnvCombatContext {
	if player == nil {
		return nil
	}
	if player.CurrentRegion == "aperture_interior" && player.MapInstanceID != "" {
		var ownerID sql.NullString
		var daoRaw string
		err := tx.QueryRow(`
			SELECT COALESCE(player_id::text,''), COALESCE(dao_marks::text,'{}')
			FROM immortal_apertures WHERE id = $1
		`, player.MapInstanceID).Scan(&ownerID, &daoRaw)
		if err != nil {
			return nil
		}
		marks := make(map[string]float64)
		_ = json.Unmarshal([]byte(daoRaw), &marks)
		domPath, _ := gamefx.DominantEnvPath(marks)
		ctx := &gamefx.EnvCombatContext{
			RegionID:        "aperture_interior",
			RegionName:      "Bên trong Tiên Khiếu",
			IsAperture:      true,
			NaturalDaoMarks: marks,
		}
		if ownerID.Valid && ownerID.String != "" {
			ctx.DomainOwnerID = ownerID.String
			ctx.DomainPath = domPath
			ctx.DomainControlPct = 100
		}
		return ctx
	}

	regID := player.CurrentRegion
	if regID == "" {
		return nil
	}
	regName := regID
	zoneReg.mu.RLock()
	if r, ok := zoneReg.byID[regID]; ok && r != nil {
		regName = r.Name
	}
	zoneReg.mu.RUnlock()

	ctx := &gamefx.EnvCombatContext{
		RegionID:        regID,
		RegionName:      regName,
		NaturalDaoMarks: EffectiveEnvDaoMarks(regID, player.X, player.Y),
	}
	if dom, ok := GetRegionDominion(regID); ok && dom.ControlPct > 0 {
		ctx.DomainOwnerID = dom.OwnerPlayerID
		ctx.DomainOwnerName = dom.OwnerName
		ctx.DomainPath = dom.Path
		ctx.DomainControlPct = dom.ControlPct
	}
	return ctx
}

// ApertureDominionQiMultiplier computes the Heaven/Earth Qi multiplier when
// unanchoring an aperture placed in destRegion.
func ApertureDominionQiMultiplier(playerID string, destRegion string) float64 {
	dom, ok := GetRegionDominion(destRegion)
	if !ok || dom.ControlPct < gamefx.DefaultEnvDaoConfig.DomainActivationPct {
		return 1.0
	}
	pct := math.Max(0, math.Min(1.0, dom.ControlPct/100.0))
	if dom.OwnerPlayerID == playerID {
		return 1.0 + gamefx.DefaultEnvDaoConfig.DomainQiOwnerBonus*pct
	}
	return 1.0 - (1.0-gamefx.DefaultEnvDaoConfig.DomainQiEnemyFloor)*pct
}

// GetRegionEnvDaoStatus returns the current region's environmental Dao Marks,
// active Dao Domain, and the player's own Venerable Dao Domain list.
func GetRegionEnvDaoStatus(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		httperr.Respond(c, httperr.Unauthorized("Chưa đăng nhập"))
		return
	}
	var curRegion string
	var x, y, rank int
	if err := db.DB.QueryRow(`SELECT current_region, x, y, rank FROM players WHERE id = $1`, playerID).Scan(&curRegion, &x, &y, &rank); err != nil {
		httperr.Respond(c, httperr.NotFound("Không tìm thấy người chơi", err))
		return
	}

	regionID := c.Query("region_id")
	if regionID == "" {
		regionID = curRegion
	}
	natural := EffectiveEnvDaoMarks(regionID, x, y)
	dom, hasDom := GetRegionDominion(regionID)
	var domPtr *models.RegionDaoDominion
	if hasDom {
		domCopy := dom
		domPtr = &domCopy
	}

	var myDomains []models.RegionDaoDominion
	rows, err := db.DB.Query(`
		SELECT region_id, owner_player_id::text, COALESCE(owner_name,''), path,
		       control_pct, refined_marks, last_reinforced_at, updated_at
		FROM region_dao_dominions WHERE owner_player_id = $1
		ORDER BY updated_at DESC
	`, playerID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d models.RegionDaoDominion
			if rows.Scan(&d.RegionID, &d.OwnerPlayerID, &d.OwnerName, &d.Path, &d.ControlPct, &d.RefinedMarks, &d.LastReinforcedAt, &d.UpdatedAt) == nil {
				myDomains = append(myDomains, d)
			}
		}
	}

	zoneType := ""
	regionName := regionID
	zoneReg.mu.RLock()
	if r, ok := zoneReg.byID[regionID]; ok && r != nil {
		zoneType = r.ZoneType
		regionName = r.Name
	}
	zoneReg.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"region_id":           regionID,
		"region_name":         regionName,
		"zone_type":           zoneType,
		"is_exempt":           zoneType == ZoneTypeHub || zoneType == ZoneTypeSafe || regionID == "aperture_interior",
		"natural_dao_marks":   natural,
		"dominion":            domPtr,
		"my_domains":          myDomains,
		"max_domains":         gamefx.DefaultEnvDaoConfig.MaxConcurrentDomains,
		"player_rank":         rank,
		"domain_activation":   gamefx.DefaultEnvDaoConfig.DomainActivationPct,
	})
}

type refineRegionEnvRequest struct {
	Path string `json:"path"`
}

// RefineRegionEnvDao handles Rank 9 Venerable refinement or erosion of a region's
// environmental Dao Marks.
func RefineRegionEnvDao(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		httperr.Respond(c, httperr.Unauthorized("Chưa đăng nhập"))
		return
	}
	var req refineRegionEnvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Respond(c, httperr.BadRequest("Dữ liệu không hợp lệ"))
		return
	}
	cleanPath := gamefx.CleanPath(req.Path)

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi mở giao dịch", err))
		return
	}
	defer tx.Rollback()

	var username, curRegion, masteriesRaw string
	var rank int
	var mana float64
	if err := tx.QueryRow(`
		SELECT username, rank, current_region, COALESCE(path_masteries::text,'{}'), current_mana
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&username, &rank, &curRegion, &masteriesRaw, &mana); err != nil {
		httperr.Respond(c, httperr.NotFound("Không tìm thấy người chơi", err))
		return
	}
	if rank < 9 {
		httperr.Respond(c, httperr.Forbidden("Chỉ Cửu Chuyển Cổ Tôn mới có thể luyện hóa Đạo Ngân môi trường"))
		return
	}
	if curRegion == "" || curRegion == "aperture_interior" {
		httperr.Respond(c, httperr.BadRequest("Không thể luyện hóa Đạo Ngân bên trong Tiên Khiếu"))
		return
	}
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	reg, exists := zoneReg.byID[curRegion]
	zoneReg.mu.RUnlock()
	if !exists || reg == nil {
		httperr.Respond(c, httperr.BadRequest("Khu vực hiện tại không hợp lệ"))
		return
	}
	if reg.ZoneType == ZoneTypeHub || reg.ZoneType == ZoneTypeSafe {
		httperr.Respond(c, httperr.Forbidden("Khu vực an toàn được miễn trừ luyện hóa Đạo Vực"))
		return
	}

	masteries := make(map[string]int)
	_ = json.Unmarshal([]byte(masteriesRaw), &masteries)
	if cleanPath == "" {
		// Pick highest Level 4 path
		for p, pts := range masteries {
			if gamefx.PathMasteryLevel(pts) >= 4 {
				cleanPath = gamefx.CleanPath(p)
				break
			}
		}
	}
	if cleanPath == "" || gamefx.PathMasteryLevel(masteries[cleanPath]) < 4 {
		httperr.Respond(c, httperr.Forbidden("Cần đạt cảnh giới Vô Thượng Đại Tông Sư (Cấp 4) ở lưu phái này để luyện hóa"))
		return
	}

	var curDom models.RegionDaoDominion
	var hasDom bool
	err = tx.QueryRow(`
		SELECT region_id, owner_player_id::text, COALESCE(owner_name,''), path,
		       control_pct, refined_marks, last_reinforced_at, updated_at
		FROM region_dao_dominions WHERE region_id = $1 FOR UPDATE
	`, curRegion).Scan(&curDom.RegionID, &curDom.OwnerPlayerID, &curDom.OwnerName, &curDom.Path,
		&curDom.ControlPct, &curDom.RefinedMarks, &curDom.LastReinforcedAt, &curDom.UpdatedAt)
	if err == nil {
		hasDom = true
	} else if err != sql.ErrNoRows {
		httperr.Respond(c, httperr.Internal("Lỗi đọc trạng thái Đạo Vực", err))
		return
	}

	// Check concurrent domain cap if claiming an unowned region
	if !hasDom {
		var ownedCnt int
		_ = tx.QueryRow(`SELECT COUNT(*) FROM region_dao_dominions WHERE owner_player_id = $1`, playerID).Scan(&ownedCnt)
		if ownedCnt >= gamefx.DefaultEnvDaoConfig.MaxConcurrentDomains {
			httperr.Respond(c, httperr.Conflict(fmt.Sprintf("Đã đạt giới hạn %d Đạo Vực đồng thời; hãy giải phóng một Đạo Vực trước", gamefx.DefaultEnvDaoConfig.MaxConcurrentDomains)))
			return
		}
	}

	manaCost := 50000.0
	if hasDom && curDom.OwnerPlayerID != playerID.String() {
		manaCost = 85000.0 // Higher cost to erode rival Venerable's Dao Domain
	}
	if mana < manaCost {
		httperr.Respond(c, httperr.BadRequest(fmt.Sprintf("Không đủ Tiên Nguyên (cần %.0f)", manaCost)))
		return
	}
	if _, err := tx.Exec(`UPDATE players SET current_mana = current_mana - $1 WHERE id = $2`, manaCost, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi trừ Tiên Nguyên", err))
		return
	}

	var updatedDom *models.RegionDaoDominion
	var actionMsg string

	if hasDom && curDom.OwnerPlayerID != playerID.String() {
		// Phase 1: Erode rival's control_pct toward 0
		newPct := math.Max(0, curDom.ControlPct-25.0)
		if newPct <= 0 {
			_, err = tx.Exec(`DELETE FROM region_dao_dominions WHERE region_id = $1`, curRegion)
			actionMsg = fmt.Sprintf("💥 %s đã phá hủy Đạo Vực của %s tại [%s], khu vực trở về vô chủ!", username, curDom.OwnerName, reg.Name)
		} else {
			_, err = tx.Exec(`UPDATE region_dao_dominions SET control_pct = $1, updated_at = NOW() WHERE region_id = $2`, newPct, curRegion)
			curDom.ControlPct = newPct
			curDom.UpdatedAt = time.Now()
			updatedDom = &curDom
			actionMsg = fmt.Sprintf("⚠️ %s đang xâm thực Đạo Vực [%s] của %s (còn %.0f%%)!", username, reg.Name, curDom.OwnerName, newPct)
		}
	} else {
		// Claim or reinforce own Dao Domain
		newPct := 25.0
		newMarks := 10000.0
		if hasDom {
			if curDom.ControlPct >= 100 {
				httperr.Respond(c, httperr.BadRequest("Đạo Vực này đã đạt 100% kiểm soát"))
				return
			}
			newPct = math.Min(100.0, curDom.ControlPct+25.0)
			newMarks = curDom.RefinedMarks + 10000.0
		}
		_, err = tx.Exec(`
			INSERT INTO region_dao_dominions (region_id, owner_player_id, owner_name, path, control_pct, refined_marks, last_reinforced_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			ON CONFLICT (region_id) DO UPDATE SET
				owner_player_id = EXCLUDED.owner_player_id,
				owner_name = EXCLUDED.owner_name,
				path = EXCLUDED.path,
				control_pct = EXCLUDED.control_pct,
				refined_marks = EXCLUDED.refined_marks,
				last_reinforced_at = NOW(),
				updated_at = NOW()
		`, curRegion, playerID, username, cleanPath, newPct, newMarks)
		updatedDom = &models.RegionDaoDominion{
			RegionID:         curRegion,
			OwnerPlayerID:    playerID.String(),
			OwnerName:        username,
			Path:             cleanPath,
			ControlPct:       newPct,
			RefinedMarks:     newMarks,
			LastReinforcedAt: time.Now(),
			UpdatedAt:        time.Now(),
		}
		actionMsg = fmt.Sprintf("✨ %s luyện hóa Đạo Ngân [%s] (%s) đạt %.0f%% kiểm soát!", username, reg.Name, strings.Title(cleanPath), newPct)
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật Đạo Vực", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lưu giao dịch", err))
		return
	}

	SetRegionDominionInCache(curRegion, updatedDom)
	publishWorldChange("region_dao_dominion", "update", curRegion, "", 0, 0, map[string]any{
		"region_id": curRegion,
		"dominion":  updatedDom,
		"message":   actionMsg,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":  actionMsg,
		"dominion": updatedDom,
		"mana":     mana - manaCost,
	})
}

// ReleaseRegionEnvDao allows a Venerable to voluntarily release one of their Dao Domains.
func ReleaseRegionEnvDao(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		httperr.Respond(c, httperr.Unauthorized("Chưa đăng nhập"))
		return
	}
	regionID := c.Param("regionId")
	res, err := db.DB.Exec(`DELETE FROM region_dao_dominions WHERE region_id = $1 AND owner_player_id = $2`, regionID, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi giải phóng Đạo Vực", err))
		return
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		httperr.Respond(c, httperr.NotFound("Bạn không sở hữu Đạo Vực tại khu vực này", nil))
		return
	}
	SetRegionDominionInCache(regionID, nil)
	publishWorldChange("region_dao_dominion", "release", regionID, "", 0, 0, map[string]any{
		"region_id": regionID,
		"dominion":  nil,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã giải phóng Đạo Vực", "region_id": regionID})
}
