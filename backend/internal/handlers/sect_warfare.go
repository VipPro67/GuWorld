package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type SectDefender struct {
	ID             string    `json:"id"`
	DefenderType   string    `json:"defender_type"` // member | beast
	PlayerID       *string   `json:"player_id"`
	Name           string    `json:"name"`
	MonsterID      *string   `json:"monster_id"`
	MonsterRank    int       `json:"monster_rank"`
	MonsterSubRank int       `json:"monster_sub_rank"`
	Power          float64   `json:"power"`
	AssignedAt     time.Time `json:"assigned_at"`
}

type SectWarfareResponse struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Level             int                `json:"level"`
	Region            string             `json:"region"`
	MapX              int                `json:"map_x"`
	MapY              int                `json:"map_y"`
	ShieldUntil       *time.Time         `json:"shield_until"`
	Defenders         []SectDefender     `json:"defenders"`
	TotalDefensePower float64            `json:"total_defense_power"`
	Buildings         map[string]int     `json:"buildings"`
	DefenderCount     int                `json:"defender_count"`
}

func getBuildingLvl(buildingsJSON []byte, buildingName string) int {
	var buildings map[string]int
	if err := json.Unmarshal(buildingsJSON, &buildings); err == nil {
		if lvl, ok := buildings[buildingName]; ok {
			return lvl
		}
	}
	return 1
}

func sectBaseDefense(defensiveArrayLevel int) float64 {
	return float64(defensiveArrayLevel) * 500.0
}

func computeMemberPower(player *models.Player) float64 {
	if player == nil {
		return 0
	}
	stats := CalculatePlayerStats(player)
	hp := player.BaseHP + stats["hp"]
	power := stats["attack"] + stats["defense"] + hp/20.0
	power *= 1.0 + 0.15*float64(player.Rank-1)
	return power
}

func computeBeastPower(mon models.MonsterTemplate, rank, subRank int) float64 {
	maxHP, atk, def := models.ScaleMonsterStats(mon.MaxHP, mon.BaseAttack, mon.BaseDefense, rank, subRank, mon.Type)
	return atk + def + maxHP/20.0
}

// sectGuardianCombatant generates the Hộ Sơn Thần Thú (Guardian Beast) Combatant
// whose stats scale with sect level and defensive_array building level.
func sectGuardianCombatant(sectName string, sectLevel int, defensiveArrayLvl int) models.Combatant {
	rank := sectLevel
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	arrLvl := defensiveArrayLvl
	if arrLvl < 1 {
		arrLvl = 1
	}
	baseHP := 500.0 * float64(arrLvl) * (1.0 + 0.5*float64(rank))
	baseAtk := 80.0 * float64(arrLvl) * (1.0 + 0.3*float64(rank))
	baseDef := 60.0 * float64(arrLvl) * (1.0 + 0.3*float64(rank))
	baseMana := 200.0 * float64(arrLvl)
	baseSpeed := 8.0 + float64(rank)*2.0

	return models.Combatant{
		ID:          fmt.Sprintf("sect_guardian_%s", uuid.New().String()[:8]),
		Name:        fmt.Sprintf("Hộ Sơn Thần Thú (%s)", sectName),
		Kind:        "Boss",
		Rank:        rank,
		SubRank:     arrLvl,
		HP:          baseHP,
		MaxHP:       baseHP,
		CurrentMana: baseMana,
		MaxMana:     baseMana,
		BaseAttack:  baseAtk,
		BaseDefense: baseDef,
		Speed:       baseSpeed,
		AIProfile:   "aggressive",
		StatusEffects: []models.StatusEffect{},
	}
}

func loadSectDefenders(sectID string) ([]SectDefender, float64, error) {
	rows, err := db.DB.Query(`
		SELECT id, defender_type, COALESCE(player_id::text, ''), name, COALESCE(monster_id, ''),
		       monster_rank, monster_sub_rank, power, assigned_at
		FROM sect_defenders
		WHERE sect_id = $1
		ORDER BY assigned_at ASC
	`, sectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var defenders []SectDefender
	total := 0.0
	for rows.Next() {
		var d SectDefender
		var playerID, monsterID string
		if err := rows.Scan(&d.ID, &d.DefenderType, &playerID, &d.Name, &monsterID, &d.MonsterRank, &d.MonsterSubRank, &d.Power, &d.AssignedAt); err != nil {
			continue
		}
		if playerID != "" {
			d.PlayerID = &playerID
		}
		if monsterID != "" {
			d.MonsterID = &monsterID
		}
		defenders = append(defenders, d)
		total += d.Power
	}
	return defenders, total, rows.Err()
}

func loadSectDefendersTx(tx *sql.Tx, sectID string) ([]SectDefender, float64, error) {
	rows, err := tx.Query(`
		SELECT id, defender_type, COALESCE(player_id::text, ''), name, COALESCE(monster_id, ''),
		       monster_rank, monster_sub_rank, power, assigned_at
		FROM sect_defenders
		WHERE sect_id = $1
		ORDER BY assigned_at ASC
	`, sectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var defenders []SectDefender
	total := 0.0
	for rows.Next() {
		var d SectDefender
		var playerID, monsterID string
		if err := rows.Scan(&d.ID, &d.DefenderType, &playerID, &d.Name, &monsterID, &d.MonsterRank, &d.MonsterSubRank, &d.Power, &d.AssignedAt); err != nil {
			continue
		}
		if playerID != "" {
			d.PlayerID = &playerID
		}
		if monsterID != "" {
			d.MonsterID = &monsterID
		}
		defenders = append(defenders, d)
		total += d.Power
	}
	return defenders, total, rows.Err()
}

func GetSectsMap(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT s.id, s.name, s.level, s.region, s.map_x, s.map_y, s.shield_until, s.buildings,
		       (SELECT COUNT(*) FROM sect_members sm
		        JOIN players p ON sm.player_id = p.id
		        WHERE sm.sect_id = s.id AND sm.is_defending = true
		          AND p.current_region = s.region AND p.x = s.map_x AND p.y = s.map_y AND p.hp > 0)
		FROM sects s
		ORDER BY s.level DESC, s.funds DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sects map"})
		return
	}
	defer rows.Close()

	var sects []SectWarfareResponse
	for rows.Next() {
		var s SectWarfareResponse
		var b []byte
		err = rows.Scan(&s.ID, &s.Name, &s.Level, &s.Region, &s.MapX, &s.MapY, &s.ShieldUntil, &b, &s.DefenderCount)
		if err == nil {
			_ = json.Unmarshal(b, &s.Buildings)
			defenders, total, derr := loadSectDefenders(s.ID)
			if derr == nil {
				s.Defenders = defenders
				arrLvl := s.Buildings["defensive_array"]
				if arrLvl == 0 {
					arrLvl = 1
				}
				s.TotalDefensePower = total + sectBaseDefense(arrLvl)
			}
			sects = append(sects, s)
		}
	}
	c.JSON(http.StatusOK, sects)
}

type DefendSectRequest struct {
	DefenderType string `json:"defender_type"` // member | beast
	PlayerID     string `json:"player_id"`
	MonsterID    string `json:"monster_id"`
	MonsterRank  int    `json:"monster_rank"`
}

func DefendSect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DefendSectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão/Hộ Pháp/Chưởng Môn mới có quyền bố trí phòng thủ!"})
		return
	}

	var buildings []byte
	var sectLevel int
	err = tx.QueryRow(`SELECT level, buildings FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&sectLevel, &buildings)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn!"})
		return
	}

	maxDefenders := sectLevel + 1 // Basic rule

	var defCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM sect_defenders WHERE sect_id = $1`, sectID).Scan(&defCount)
	if defCount >= maxDefenders {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn đã đầy chỗ phòng thủ!"})
		return
	}

	switch req.DefenderType {
	case "member":
		if req.PlayerID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu đệ tử cần bố trí!"})
			return
		}
		pid, perr := uuid.Parse(req.PlayerID)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Player ID không hợp lệ"})
			return
		}

		var targetSect, targetName string
		err = tx.QueryRow(`SELECT sm.sect_id, p.username FROM sect_members sm JOIN players p ON p.id = sm.player_id WHERE sm.player_id = $1`, pid).Scan(&targetSect, &targetName)
		if err != nil || targetSect != sectID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử này không thuộc tông môn của bạn!"})
			return
		}

		var existingID string
		err = tx.QueryRow(`SELECT id FROM sect_defenders WHERE player_id = $1 AND defender_type = 'member'`, pid).Scan(&existingID)
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử này đã đang phòng thủ!"})
			return
		}

		player, lerr := evaluateManaTxEquipped(tx, pid)
		if lerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
			return
		}
		power := computeMemberPower(&player)

		_, err = tx.Exec(`
			INSERT INTO sect_defenders (sect_id, defender_type, player_id, name, power, assigned_at)
			VALUES ($1, 'member', $2, $3, $4, now())
		`, sectID, pid, targetName, power)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign defender"})
			return
		}
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s đã được phong làm thủ hộ tông môn.", targetName))
	case "beast":
		if req.MonsterID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu thần thú cần thuê!"})
			return
		}
		rank := req.MonsterRank
		if rank < 1 {
			rank = 1
		}
		if rank > 9 {
			rank = 9
		}
		cost := int64(rank) * 500
		var funds int64
		err = tx.QueryRow(`SELECT funds FROM sects WHERE id = $1`, sectID).Scan(&funds)
		if err != nil || funds < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ (cần %d Quỹ)!", cost)})
			return
		}

		mon, merr := db.LoadMonsterTemplate(tx, req.MonsterID)
		if merr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy thần thú!"})
			return
		}
		power := computeBeastPower(mon, rank, 1)
		name := fmt.Sprintf("%s (Hạng %d)", mon.Name, rank)

		_, err = tx.Exec(`
			INSERT INTO sect_defenders (sect_id, defender_type, name, monster_id, monster_rank, monster_sub_rank, power, assigned_at)
			VALUES ($1, 'beast', $2, $3, $4, 1, $5, now())
		`, sectID, name, req.MonsterID, rank, power)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hire beast"})
			return
		}
		_, err = tx.Exec(`UPDATE sects SET funds = funds - $1 WHERE id = $2`, cost, sectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct funds"})
			return
		}
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Tông môn đã thuê %s làm thần thú hộ sơn.", name))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại phòng thủ không hợp lệ!"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã bố trí phòng thủ thành công!"})
}

func UndefendSect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		DefenderID string `json:"defender_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DefenderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu defender_id"})
		return
	}

	did, err := uuid.Parse(req.DefenderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Defender ID không hợp lệ"})
		return
	}

	var sectID, callerRank string
	err = db.DB.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}
	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão/Hộ Pháp/Chưởng Môn mới có quyền điều động phòng thủ!"})
		return
	}

	var dSectID string
	err = db.DB.QueryRow(`SELECT sect_id FROM sect_defenders WHERE id = $1`, did).Scan(&dSectID)
	if err != nil || dSectID != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Phòng thủ này thuộc tông môn khác hoặc không tồn tại!"})
		return
	}

	_, err = db.DB.Exec(`DELETE FROM sect_defenders WHERE id = $1`, did)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể rút phòng thủ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã rút phòng thủ!"})
}

// ToggleDefendSect lets a member toggle their is_defending flag.
// Requires the player to be standing at their sect's location.
func ToggleDefendSect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sectID string
	var isDefending bool
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id, is_defending FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &isDefending)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	// If turning ON, check player location == sect location
	if !isDefending {
		var sectRegion string
		var sectX, sectY int
		err = db.DB.QueryRowContext(c.Request.Context(), `SELECT region, map_x, map_y FROM sects WHERE id = $1`, sectID).Scan(&sectRegion, &sectX, &sectY)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy tông môn!"})
			return
		}

		var pRegion string
		var pX, pY int
		err = db.DB.QueryRowContext(c.Request.Context(), `SELECT current_region, x, y FROM players WHERE id = $1`, playerID).Scan(&pRegion, &pX, &pY)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin đạo hữu!"})
			return
		}

		if pRegion != sectRegion || pX != sectX || pY != sectY {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu phải đứng tại tọa độ Tông Môn mới có thể trấn thủ!"})
			return
		}
	}

	_, err = db.DB.ExecContext(c.Request.Context(), `UPDATE sect_members SET is_defending = NOT is_defending WHERE player_id = $1`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cập nhật trạng thái thất bại"})
		return
	}

	newState := !isDefending
	msg := "Đã tắt trạng thái Trấn Thủ Tông Môn."
	if newState {
		msg = "Đã bật Trấn Thủ Tông Môn! Đạo hữu sẽ tham chiến khi Tông Môn bị tấn công."
	}
	c.JSON(http.StatusOK, gin.H{"message": msg, "is_defending": newState})
}

// AttackSect initiates real turn-based combat against a target sect.
// Enemies = Guardian Beast + sect members with is_defending=true at sect location.
// Allies = attacker + party members + pets (loaded by StartCombat).
func AttackSect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		TargetSectID string `json:"target_sect_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Check attacker sect membership (optional: lone wolves / rogue cultivators can attack)
	var mySectID string
	_ = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&mySectID)

	if mySectID != "" && mySectID == req.TargetSectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đánh tông môn của chính mình!"})
		return
	}

	// Load target sect
	var targetShield sql.NullTime
	var targetBuildings []byte
	var targetFunds int64
	var targetName, targetRegion string
	var targetMapX, targetMapY, targetLevel int
	err = tx.QueryRow(`SELECT name, region, map_x, map_y, level, shield_until, buildings, funds FROM sects WHERE id = $1 FOR UPDATE`, req.TargetSectID).Scan(
		&targetName, &targetRegion, &targetMapX, &targetMapY, &targetLevel, &targetShield, &targetBuildings, &targetFunds)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tông môn mục tiêu không tồn tại!"})
		return
	}

	if targetShield.Valid && time.Now().Before(targetShield.Time) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn mục tiêu đang trong thời gian Hộ Sơn Đại Trận bảo hộ!"})
		return
	}

	// Load attacker and check position
	attacker, aerr := evaluateManaTxEquipped(tx, playerID)
	if aerr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	if attacker.CurrentRegion != targetRegion || attacker.X != targetMapX || attacker.Y != targetMapY {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu phải di chuyển đến Tông Môn mục tiêu mới có thể khiêu chiến!"})
		return
	}

	if attacker.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong trận chiến khác!"})
		return
	}

	// Build Guardian Beast as main enemy
	arrLvl := getBuildingLvl(targetBuildings, "defensive_array")
	guardian := sectGuardianCombatant(targetName, targetLevel, arrLvl)

	// Build combat state with guardian as main enemy
	sectWarCombat := models.CombatState{
		EnemyTemplateID: guardian.ID,
		EnemyName:       guardian.Name,
		EnemyType:       "SECT_WAR",
		CombatMode:      "real",
		EnemyHP:         guardian.HP,
		EnemyMaxHP:      guardian.MaxHP,
		EnemyMana:       guardian.CurrentMana,
		EnemyMaxMana:    guardian.MaxMana,
		EnemyDamage:     guardian.BaseAttack,
		EnemyDefense:    guardian.BaseDefense,
		EnemySpeed:      guardian.Speed,
		EnemyRank:       guardian.Rank,
		EnemySubRank:    guardian.SubRank,
		AIProfile:       "aggressive",
		TargetSectID:    req.TargetSectID,
	}

	// Initialize combat (this populates Allies from party + pets)
	attacker.ActiveCombat = StartCombat(tx, &sectWarCombat, &attacker)

	// Load defending members at sect location as additional enemies.
	// Collect all IDs and close the rows BEFORE running further queries on the
	// same tx — a still-open *sql.Rows pins the tx's connection and every
	// subsequent Exec/Query on the tx fails with conn-busy errors.
	var defenderIDs []uuid.UUID
	defRows, derr := tx.Query(`
		SELECT sm.player_id FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.sect_id = $1 AND sm.is_defending = true
		  AND p.current_region = $2 AND p.x = $3 AND p.y = $4 AND p.hp > 0
		LIMIT 8
	`, req.TargetSectID, targetRegion, targetMapX, targetMapY)
	if derr != nil {
		logger.S.Errorf("AttackSect: failed to load defenders for sect %s: %v", req.TargetSectID, derr)
	} else {
		for defRows.Next() {
			var defID uuid.UUID
			if defRows.Scan(&defID) != nil {
				continue
			}
			defenderIDs = append(defenderIDs, defID)
		}
		if rowsErr := defRows.Err(); rowsErr != nil {
			logger.S.Errorf("AttackSect: failed iterating defenders for sect %s: %v", req.TargetSectID, rowsErr)
		}
		defRows.Close()
	}
	for _, defID := range defenderIDs {
		defPlayer, errDef := evaluateManaTxEquippedReadonly(tx, defID)
		if errDef != nil || defPlayer.HP <= 0 {
			continue
		}
		defStats := CalculatePlayerStats(&defPlayer)
		defCombatant := models.CombatantFromPlayer(&defPlayer, defStats)
		defCombatant.Kind = "Player"
		defCombatant.CombatSettings = &defPlayer.CombatSettings
		defCombatant.GuInventory = defPlayer.GuInventory
		attacker.ActiveCombat.Enemies = append(attacker.ActiveCombat.Enemies, defCombatant)
	}

	// Re-assign grid positions for enemies
	enemySlots := [][2]int{
		{2, 3}, {2, 1}, {1, 1}, {3, 1}, {2, 2}, {1, 2}, {3, 2}, {1, 3}, {3, 3},
	}
	for idx := range attacker.ActiveCombat.Enemies {
		if idx < len(enemySlots) {
			attacker.ActiveCombat.Enemies[idx].GridX = enemySlots[idx][0]
			attacker.ActiveCombat.Enemies[idx].GridY = enemySlots[idx][1]
		}
		if attacker.ActiveCombat.Enemies[idx].Speed == 0 {
			attacker.ActiveCombat.Enemies[idx].Speed = 8.0 + float64(attacker.ActiveCombat.Enemies[idx].Rank)*2.0
		}
	}

	combatBytes, mErr := json.Marshal(attacker.ActiveCombat)
	if mErr != nil {
		// NaN/Inf in the combat state makes json.Marshal fail; an empty payload
		// would poison the jsonb column, so fail the request cleanly instead.
		logger.S.Errorf("AttackSect: failed to marshal combat state for player %s: %v", playerID, mErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat state"})
		return
	}
	_, err = tx.Exec("UPDATE players SET active_combat = $1, human_qi = LEAST(human_qi + 5, $2) WHERE id = $3", string(combatBytes), MaxHumanQi, playerID)
	if err != nil {
		logger.S.Errorf("AttackSect: failed to save combat state for player %s: %v", playerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat state"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	defCount := len(attacker.ActiveCombat.Enemies) - 1 // minus guardian
	msg := fmt.Sprintf("Khiêu chiến Tông Môn %s! Đối mặt Hộ Sơn Thần Thú", targetName)
	if defCount > 0 {
		msg += fmt.Sprintf(" cùng %d đệ tử trấn thủ", defCount)
	}
	msg += ". Nhân Khí +5."

	respondActionOnly(c, http.StatusOK, attacker, gin.H{"message": msg})
}

// ApplySectWarVictory is called by auto-combat resolution when attacker wins a SECT_WAR combat.
// It plunders funds, vault_materials, vault_gu, and vault_recipes from the target sect.
func ApplySectWarVictory(tx *sql.Tx, player *models.Player, combat *models.CombatState) []string {
	var logs []string
	if combat == nil || combat.TargetSectID == "" {
		return logs
	}

	// Gather all team members (attacker + allies/party)
	teamMembers := []uuid.UUID{player.ID}
	for _, ally := range combat.Allies {
		if ally.Kind == "Ally" || ally.Kind == "Player" {
			if aUUID, parseErr := uuid.Parse(ally.ID); parseErr == nil && aUUID != player.ID {
				teamMembers = append(teamMembers, aUUID)
			}
		}
	}

	var mySectID string
	_ = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, player.ID).Scan(&mySectID)

	var targetFunds int64
	var targetBuildings []byte
	var targetName string
	err := tx.QueryRow(`
		SELECT name, funds, buildings
		FROM sects WHERE id = $1 FOR UPDATE
	`, combat.TargetSectID).Scan(&targetName, &targetFunds, &targetBuildings)
	if err != nil {
		return logs
	}

	arrLvl := getBuildingLvl(targetBuildings, "defensive_array")
	treasuryLvl := getBuildingLvl(targetBuildings, "treasury")

	// Calculate plunder percentage (higher treasury = less plunder)
	plunderPercent := 0.20 - (float64(treasuryLvl) * 0.02)
	if plunderPercent < 0.05 {
		plunderPercent = 0.05
	}

	// 1. Plunder funds
	stolenFunds := int64(float64(targetFunds) * plunderPercent)
	if stolenFunds > 0 && mySectID != "" {
		// Credit attacker's sect (if attacker has no sect, funds disappear into the void)
		_, _ = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, stolenFunds, mySectID)
	}

	// 2. Plunder vault_materials (items)
	matRows, err := tx.Query(`SELECT item_id, quantity FROM sect_vault_materials WHERE sect_id = $1 AND quantity > 0 FOR UPDATE`, combat.TargetSectID)
	stolenMats := make(map[string]int64)
	if err == nil {
		type targetMatRow struct {
			itemID string
			qty    int64
		}
		var matsList []targetMatRow
		for matRows.Next() {
			var tm targetMatRow
			if err := matRows.Scan(&tm.itemID, &tm.qty); err == nil {
				matsList = append(matsList, tm)
			}
		}
		matRows.Close()

		for _, tm := range matsList {
			stolen := int64(float64(tm.qty) * plunderPercent)
			if stolen > 0 {
				stolenMats[tm.itemID] = stolen
				if stolen >= tm.qty {
					_, _ = tx.Exec(`DELETE FROM sect_vault_materials WHERE sect_id = $1 AND item_id = $2`, combat.TargetSectID, tm.itemID)
				} else {
					_, _ = tx.Exec(`UPDATE sect_vault_materials SET quantity = quantity - $1, updated_at = NOW() WHERE sect_id = $2 AND item_id = $3`, stolen, combat.TargetSectID, tm.itemID)
				}
			}
		}
	}

	// Distribute materials among team members
	if len(stolenMats) > 0 {
		numMembers := int64(len(teamMembers))
		for matID, totalQty := range stolenMats {
			perMember := totalQty / numMembers
			rem := totalQty % numMembers
			for i, mID := range teamMembers {
				qty := perMember
				if int64(i) < rem {
					qty++
				}
				if qty > 0 {
					_ = InventorySvc.AddItem(tx, mID.String(), matID, "materials", qty)
				}
			}
		}
	}

	// 3. Plunder vault_gu
	guRows, err := tx.Query(`SELECT id, gu_id, rank, cost, quantity, hunger, COALESCE(instance_id, '') FROM sect_vault_gu WHERE sect_id = $1 AND quantity > 0 FOR UPDATE`, combat.TargetSectID)
	var stolenGus []SectVaultGuItem
	if err == nil {
		type targetGuRow struct {
			id         string
			guID       string
			rank       int
			cost       int64
			quantity   int64
			hunger     float64
			instanceID string
		}
		var gusList []targetGuRow
		for guRows.Next() {
			var tg targetGuRow
			if err := guRows.Scan(&tg.id, &tg.guID, &tg.rank, &tg.cost, &tg.quantity, &tg.hunger, &tg.instanceID); err == nil {
				gusList = append(gusList, tg)
			}
		}
		guRows.Close()

		for _, tg := range gusList {
			stolenQty := int64(math.Ceil(float64(tg.quantity) * plunderPercent))
			if stolenQty > tg.quantity {
				stolenQty = tg.quantity
			}
			if stolenQty > 0 {
				stolenGus = append(stolenGus, SectVaultGuItem{
					ID:         tg.guID,
					InstanceID: tg.instanceID,
					Rank:       tg.rank,
					Cost:       tg.cost,
					Quantity:   stolenQty,
					Hunger:     tg.hunger,
				})
				if stolenQty >= tg.quantity {
					_, _ = tx.Exec(`DELETE FROM sect_vault_gu WHERE id = $1`, tg.id)
				} else {
					_, _ = tx.Exec(`UPDATE sect_vault_gu SET quantity = quantity - $1, updated_at = NOW() WHERE id = $2`, stolenQty, tg.id)
				}
			}
		}
	}

	// Distribute stolen Gus to team members
	if len(stolenGus) > 0 {
		memberIdx := 0
		for _, sg := range stolenGus {
			for q := int64(0); q < sg.Quantity; q++ {
				recipient := teamMembers[memberIdx%len(teamMembers)]
				memberIdx++
				instID := sg.InstanceID
				if instID == "" || q > 0 {
					instID = uuid.New().String()
				}
				r := sg.Rank
				if r <= 0 {
					r = 1
				}
				newGu := models.GuInstance{
					InstanceID: instID,
					GuID:       sg.ID,
					Rank:       r,
					Hunger:     int(sg.Hunger),
					Status:     "NORMAL",
					Equipped:   false,
					IsVital:    false,
				}
				_ = db.AddPlayerGu(tx, recipient.String(), newGu)
			}
		}
	}

	// 4. Plunder vault_recipes
	recRows, err := tx.Query(`SELECT recipe_id, completion_rate FROM sect_vault_recipes WHERE sect_id = $1 FOR UPDATE`, combat.TargetSectID)
	var stolenRecipes []models.SectVaultRecipe
	if err == nil {
		var targetVaultRecipes []models.SectVaultRecipe
		for recRows.Next() {
			var vr models.SectVaultRecipe
			if err := recRows.Scan(&vr.RecipeID, &vr.CompletionRate); err == nil {
				targetVaultRecipes = append(targetVaultRecipes, vr)
			}
		}
		recRows.Close()

		for _, vr := range targetVaultRecipes {
			if rand.Float64() < plunderPercent {
				stolenRecipes = append(stolenRecipes, vr)
			}
		}
		if len(targetVaultRecipes) > 0 && len(stolenRecipes) == 0 {
			stolenRecipes = append(stolenRecipes, targetVaultRecipes[0])
		}

		for _, sr := range stolenRecipes {
			_, _ = tx.Exec(`DELETE FROM sect_vault_recipes WHERE sect_id = $1 AND recipe_id = $2`, combat.TargetSectID, sr.RecipeID)
		}
	}

	// Unlock stolen recipes for team members
	for _, sr := range stolenRecipes {
		for _, mID := range teamMembers {
			_, _ = tx.Exec(`
				INSERT INTO player_recipe_progress (player_id, recipe_id, completion_rate, is_unlocked, created_at, updated_at)
				VALUES ($1, $2, $3, true, NOW(), NOW())
				ON CONFLICT (player_id, recipe_id)
				DO UPDATE SET is_unlocked = true, completion_rate = GREATEST(player_recipe_progress.completion_rate, EXCLUDED.completion_rate), updated_at = NOW()
			`, mID, sr.RecipeID, sr.CompletionRate)
		}
	}

	now := time.Now()
	shieldHours := time.Duration(12 + arrLvl*2)
	newShield := now.Add(shieldHours * time.Hour)

	// Update target sect in DB
	_, _ = tx.Exec(`
		UPDATE sects SET
			funds = GREATEST(0, funds - $1),
			shield_until = $2
		WHERE id = $3
	`, stolenFunds, newShield, combat.TargetSectID)

	// Reset defending status for all target sect members
	_, _ = tx.Exec(`UPDATE sect_members SET is_defending = false WHERE sect_id = $1`, combat.TargetSectID)

	// Award attacker
	_, _ = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if mySectID != "" {
		_, _ = tx.Exec(`UPDATE sect_members SET contribution_points = contribution_points + 200 WHERE player_id = $1`, player.ID)
	}

	logs = append(logs, fmt.Sprintf("Công phá Tông Môn %s thành công!", targetName))
	if stolenFunds > 0 {
		if mySectID != "" {
			logs = append(logs, fmt.Sprintf("Cướp được %d Quỹ Tông Môn chuyển vào Tông Môn ta.", stolenFunds))
		} else {
			logs = append(logs, fmt.Sprintf("Cướp phá tiêu hủy %d Quỹ Tông Môn (quỹ bị tiêu biến vào hư vô).", stolenFunds))
		}
	}
	if len(stolenMats) > 0 {
		matMsg := "Cướp linh tài kho tông (chia đều cho đội):"
		for key, val := range stolenMats {
			matMsg += fmt.Sprintf(" %s x%d,", key, val)
		}
		logs = append(logs, matMsg)
	}
	if len(stolenGus) > 0 {
		guMsg := "Cướp Cổ trùng kho tông (chia cho đội):"
		for _, sg := range stolenGus {
			guMsg += fmt.Sprintf(" %s R%d x%d,", sg.ID, sg.Rank, sg.Quantity)
		}
		logs = append(logs, guMsg)
	}
	if len(stolenRecipes) > 0 {
		recMsg := "Cướp bí phương kho tông (mở khóa cho đội):"
		for _, sr := range stolenRecipes {
			recMsg += fmt.Sprintf(" %s,", sr.RecipeID)
		}
		logs = append(logs, recMsg)
	}

	// Notifications
	if mySectID != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: mySectID, Valid: true},
			fmt.Sprintf("Tông môn ta đã đánh chiếm %s! Cướp được %d Quỹ.", targetName, stolenFunds))
	}
	_ = SendSystemMessage("sect", sql.NullString{String: combat.TargetSectID, Valid: true},
		fmt.Sprintf("Tông môn bị đạo hữu %s và đội ngũ tấn công và thất thủ! Mất %d Quỹ cùng Cổ trùng, linh tài kho. Hộ Sơn Đại Trận bảo hộ trong %d giờ.", player.Username, stolenFunds, shieldHours))

	return logs
}

// ApplySectWarDefeat is called when attacker loses a SECT_WAR combat.
// Awards defending members contribution points and grants a short shield.
func ApplySectWarDefeat(tx *sql.Tx, player *models.Player, combat *models.CombatState) []string {
	var logs []string
	if combat == nil || combat.TargetSectID == "" {
		return logs
	}

	now := time.Now()
	cooldownUntil := now.Add(4 * time.Hour)
	_, _ = tx.Exec(`UPDATE sects SET shield_until = $1 WHERE id = $2`, cooldownUntil, combat.TargetSectID)

	// Reward defending members
	_, _ = tx.Exec(`
		UPDATE sect_members SET contribution_points = contribution_points + 100
		WHERE sect_id = $1 AND is_defending = true
	`, combat.TargetSectID)

	var targetName string
	_ = tx.QueryRow(`SELECT name FROM sects WHERE id = $1`, combat.TargetSectID).Scan(&targetName)

	_ = SendSystemMessage("sect", sql.NullString{String: combat.TargetSectID, Valid: true},
		fmt.Sprintf("Tông môn đã chống lại cuộc tấn công của đạo hữu %s! Tất cả đệ tử trấn thủ nhận thưởng 100 Điểm Cống Hiến.", player.Username))

	logs = append(logs, fmt.Sprintf("Khiêu chiến Tông Môn %s thất bại! Hộ Sơn Đại Trận quá vững chãi.", targetName))
	return logs
}
