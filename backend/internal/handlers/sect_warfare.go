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
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/sectmechanics"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

const (
	sectArrayRallyWindow = 10 * time.Minute
	sectArrayCooldown    = 12 * time.Hour
)

func sectArrayCanIntercept(now time.Time, readyAt sql.NullTime) bool {
	return !readyAt.Valid || !now.Before(readyAt.Time)
}

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
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Level             int            `json:"level"`
	Region            string         `json:"region"`
	MapX              int            `json:"map_x"`
	MapY              int            `json:"map_y"`
	ShieldUntil       *time.Time     `json:"shield_until"`
	Defenders         []SectDefender `json:"defenders"`
	TotalDefensePower float64        `json:"total_defense_power"`
	Buildings         map[string]int `json:"buildings"`
	DefenderCount     int            `json:"defender_count"`
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
	return memberPowerFromStats(player, stats)
}

func computeMemberPowerWithTerritory(player *models.Player, territory SectTerritoryInfo) float64 {
	if player == nil {
		return 0
	}
	stats := calculatePlayerStatsWithTerritory(player, territory)
	return memberPowerFromStats(player, stats)
}

func memberPowerFromStats(player *models.Player, stats map[string]float64) float64 {
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
		ID:            fmt.Sprintf("sect_guardian_%s", uuid.New().String()[:8]),
		Name:          fmt.Sprintf("Hộ Sơn Thần Thú (%s)", sectName),
		Kind:          "Boss",
		Rank:          rank,
		SubRank:       arrLvl,
		HP:            baseHP,
		MaxHP:         baseHP,
		CurrentMana:   baseMana,
		MaxMana:       baseMana,
		BaseAttack:    baseAtk,
		BaseDefense:   baseDef,
		Speed:         baseSpeed,
		AIProfile:     "aggressive",
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
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Tông môn đã thuê %s làm thần thú hộ sơn. Tiêu hao %d Quỹ, còn lại %d Quỹ.", name, cost, funds-cost))
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
	var targetShield, arrayReadyAt sql.NullTime
	var targetBuildings []byte
	var targetFunds int64
	var targetName, targetRegion string
	var targetMapX, targetMapY, targetLevel int
	err = tx.QueryRow(`SELECT name, region, map_x, map_y, level, shield_until, buildings, funds, defensive_array_ready_at FROM sects WHERE id = $1 FOR UPDATE`, req.TargetSectID).Scan(
		&targetName, &targetRegion, &targetMapX, &targetMapY, &targetLevel, &targetShield, &targetBuildings, &targetFunds, &arrayReadyAt)
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

	distX := math.Abs(float64(attacker.X - targetMapX))
	distY := math.Abs(float64(attacker.Y - targetMapY))
	dist := int(math.Max(distX, distY))
	const maxSectAttackRange = 2
	if dist > maxSectAttackRange {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Khoảng cách quá xa (%d ô)! Cần tiếp cận Tông Môn trong vòng %d ô mới có thể khiêu chiến!", dist, maxSectAttackRange)})
		return
	}

	if attacker.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong trận chiến khác!"})
		return
	}

	arrLvl := getBuildingLvl(targetBuildings, "defensive_array")
	now := time.Now()
	if arrLvl > 0 && sectArrayCanIntercept(now, arrayReadyAt) {
		shieldUntil := now.Add(sectArrayRallyWindow)
		readyAt := now.Add(sectArrayCooldown)
		if _, err = tx.Exec(`UPDATE sects SET shield_until = $1, defensive_array_ready_at = $2 WHERE id = $3`, shieldUntil, readyAt, req.TargetSectID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kích hoạt Hộ Sơn Đại Trận"})
			return
		}
		var memberIDs []uuid.UUID
		rows, queryErr := tx.Query(`SELECT player_id FROM sect_members WHERE sect_id = $1`, req.TargetSectID)
		if queryErr == nil {
			for rows.Next() {
				var id uuid.UUID
				if rows.Scan(&id) == nil {
					memberIDs = append(memberIDs, id)
				}
			}
			rows.Close()
		}
		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
			return
		}
		alert := fmt.Sprintf("[Hộ Sơn Đại Trận] Đã chặn đợt tập kích của %s! Toàn tông có 10 phút tập hợp trấn thủ trước khi sơn môn có thể bị công kích lại.", attacker.Username)
		_ = SendSystemMessage("sect", sql.NullString{String: req.TargetSectID, Valid: true}, alert)
		for _, id := range memberIDs {
			_ = notify.Send(db.DB, id, "sect_attack", "Hộ Sơn Đại Trận cảnh báo", alert)
		}
		c.JSON(http.StatusConflict, gin.H{"error": alert, "formation_cancelled": true, "shield_until": shieldUntil, "cooldown_until": readyAt})
		return
	}

	// Build Guardian Beast as main enemy
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

	_ = SendSystemMessage("sect", sql.NullString{String: req.TargetSectID, Valid: true},
		fmt.Sprintf("[Chiến Báo] Kẻ địch %s đã phát động tấn công Sơn Môn!", attacker.Username))

	respondActionOnly(c, http.StatusOK, attacker, gin.H{"message": msg})
}

// sectPlunderVoidPct — Thiên Đạo Tiêu Hủy (Void Destruction) khi cướp kho Tông Môn,
// tương tự deathPenaltyConfig.VoidDestructionPct: một phần chiến lợi phẩm bị tiêu
// hủy vĩnh viễn, chỉ phần còn lại được chia cho đội tấn công.
const sectPlunderVoidPct = 40.0

// splitVoidDestruction chia số chiến lợi phẩm bị cướp thành phần bị tiêu hủy vĩnh
// viễn và phần chuyển cho đội, theo đúng cơ chế death penalty: khi mất đúng 1 đơn
// vị thì 50% cơ hội bị tiêu hủy; ngược lại luôn giữ lại ít nhất 1 đơn vị cho đội.
func splitVoidDestruction(stolen int64, voidPct float64) (destroyed, delivered int64) {
	if stolen <= 0 {
		return 0, 0
	}
	destroyed = int64(math.Round(float64(stolen) * voidPct))
	if stolen == 1 {
		if rand.Float64() < 0.5 {
			destroyed = 1
		} else {
			destroyed = 0
		}
	} else if destroyed >= stolen {
		destroyed = stolen - 1
	}
	return destroyed, stolen - destroyed
}

// ApplySectWarVictory is called by auto-combat resolution when attacker wins a SECT_WAR combat.
// It plunders funds, vault_materials, vault_gu, and vault_recipes from the target sect.
// Một phần chiến lợi phẩm bị Thiên Đạo tiêu hủy vĩnh viễn (Void Destruction),
// phần còn lại chia cho đội tấn công. Mọi thao tác xóa kho / lưu đồ cho đội chạy
// theo batch (một câu SQL mỗi bảng / mỗi thành viên) để nhanh và chính xác.
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
	var targetName, targetRegion string
	err := tx.QueryRow(`
		SELECT name, region, funds, buildings
		FROM sects WHERE id = $1 FOR UPDATE
	`, combat.TargetSectID).Scan(&targetName, &targetRegion, &targetFunds, &targetBuildings)
	if err != nil {
		return logs
	}

	treasuryLvl := getBuildingLvl(targetBuildings, "treasury")

	// Calculate plunder percentage (higher treasury = less plunder)
	plunderPercent := sectmechanics.PlunderChance(treasuryLvl)

	voidPct := sectPlunderVoidPct / 100.0

	// 1. Plunder funds — một phần bị Thiên Đạo tiêu hủy vĩnh viễn
	stolenFunds := int64(float64(targetFunds) * plunderPercent)
	destroyedFunds := int64(float64(stolenFunds) * voidPct)
	creditedFunds := stolenFunds - destroyedFunds
	if creditedFunds > 0 && mySectID != "" {
		// Credit attacker's sect (if attacker has no sect, funds disappear into the void)
		if _, err := tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, creditedFunds, mySectID); err != nil {
			logger.S.Errorf("ApplySectWarVictory: failed to credit funds to sect %s: %v", mySectID, err)
		}
	}

	// 2. Plunder vault_materials (items)
	type stolenMat struct {
		itemID    string
		vaultQty  int64
		stolen    int64
		destroyed int64
	}
	var matsList []stolenMat
	matRows, err := tx.Query(`SELECT item_id, quantity FROM sect_vault_materials WHERE sect_id = $1 AND quantity > 0 ORDER BY item_id ASC FOR UPDATE`, combat.TargetSectID)
	if err != nil {
		logger.S.Errorf("ApplySectWarVictory: failed to load vault materials for sect %s: %v", combat.TargetSectID, err)
	} else {
		for matRows.Next() {
			var itemID string
			var qty int64
			if err := matRows.Scan(&itemID, &qty); err != nil {
				continue
			}
			stolen := sectmechanics.RollPlunderQuantity(qty, plunderPercent, rand.Float64())
			if stolen <= 0 {
				continue
			}
			destroyed, _ := splitVoidDestruction(stolen, voidPct)
			matsList = append(matsList, stolenMat{itemID: itemID, vaultQty: qty, stolen: stolen, destroyed: destroyed})
		}
		matRows.Close()
	}

	// Batch update vault: whole rows removed in one DELETE, partial rows in one UPDATE
	if len(matsList) > 0 {
		fullRemoveIDs := make([]string, 0, len(matsList))
		partialIDs := make([]string, 0, len(matsList))
		partialAmts := make([]int64, 0, len(matsList))
		for _, m := range matsList {
			if m.stolen >= m.vaultQty {
				fullRemoveIDs = append(fullRemoveIDs, m.itemID)
			} else {
				partialIDs = append(partialIDs, m.itemID)
				partialAmts = append(partialAmts, m.stolen)
			}
		}
		if len(fullRemoveIDs) > 0 {
			if _, err := tx.Exec(`DELETE FROM sect_vault_materials WHERE sect_id = $1 AND item_id = ANY($2::text[])`, combat.TargetSectID, pq.Array(fullRemoveIDs)); err != nil {
				logger.S.Errorf("ApplySectWarVictory: failed to batch delete vault materials for sect %s: %v", combat.TargetSectID, err)
			}
		}
		if len(partialIDs) > 0 {
			if _, err := tx.Exec(`
				UPDATE sect_vault_materials AS sv
				SET quantity = sv.quantity - v.amt, updated_at = NOW()
				FROM unnest($1::text[], $2::bigint[]) AS v(item_id, amt)
				WHERE sv.sect_id = $3 AND sv.item_id = v.item_id
			`, pq.Array(partialIDs), pq.Array(partialAmts), combat.TargetSectID); err != nil {
				logger.S.Errorf("ApplySectWarVictory: failed to batch update vault materials for sect %s: %v", combat.TargetSectID, err)
			}
		}
	}

	// Distribute delivered materials among team members (one batched insert per member)
	deliveredMats := make(map[string]int64, len(matsList))
	for _, m := range matsList {
		if d := m.stolen - m.destroyed; d > 0 {
			deliveredMats[m.itemID] = d
		}
	}
	if len(deliveredMats) > 0 {
		numMembers := int64(len(teamMembers))
		for i, mID := range teamMembers {
			entries := make([]services.ItemBatchEntry, 0, len(deliveredMats))
			for itemID, totalQty := range deliveredMats {
				qty := totalQty / numMembers
				if int64(i) < totalQty%numMembers {
					qty++
				}
				if qty > 0 {
					entries = append(entries, services.ItemBatchEntry{ItemID: itemID, Category: "materials", Quantity: qty})
				}
			}
			if len(entries) > 0 {
				if err := InventorySvc.AddItemsBatch(tx, mID.String(), entries); err != nil {
					logger.S.Errorf("ApplySectWarVictory: failed to deliver materials to player %s: %v", mID, err)
				}
			}
		}
	}

	// 3. Plunder vault_gu
	type stolenGu struct {
		rowID      string
		guID       string
		rank       int
		vaultQty   int64
		instanceID string
		stolen     int64
		destroyed  int64
	}
	var gusList []stolenGu
	guRows, err := tx.Query(`SELECT id, gu_id, rank, quantity, COALESCE(instance_id, '') FROM sect_vault_gu WHERE sect_id = $1 AND quantity > 0 ORDER BY id ASC FOR UPDATE`, combat.TargetSectID)
	if err != nil {
		logger.S.Errorf("ApplySectWarVictory: failed to load vault Gus for sect %s: %v", combat.TargetSectID, err)
	} else {
		for guRows.Next() {
			var tg stolenGu
			if err := guRows.Scan(&tg.rowID, &tg.guID, &tg.rank, &tg.vaultQty, &tg.instanceID); err != nil {
				continue
			}
			stolen := sectmechanics.RollPlunderQuantity(tg.vaultQty, plunderPercent, rand.Float64())
			if stolen > tg.vaultQty {
				stolen = tg.vaultQty
			}
			if stolen <= 0 {
				continue
			}
			tg.stolen = stolen
			tg.destroyed, _ = splitVoidDestruction(stolen, voidPct)
			gusList = append(gusList, tg)
		}
		guRows.Close()
	}

	// Batch update vault: whole rows removed in one DELETE, partial rows in one UPDATE
	if len(gusList) > 0 {
		fullRemoveRowIDs := make([]string, 0, len(gusList))
		partialRowIDs := make([]string, 0, len(gusList))
		partialAmts := make([]int64, 0, len(gusList))
		for _, tg := range gusList {
			if tg.stolen >= tg.vaultQty {
				fullRemoveRowIDs = append(fullRemoveRowIDs, tg.rowID)
			} else {
				partialRowIDs = append(partialRowIDs, tg.rowID)
				partialAmts = append(partialAmts, tg.stolen)
			}
		}
		if len(fullRemoveRowIDs) > 0 {
			if _, err := tx.Exec(`DELETE FROM sect_vault_gu WHERE id = ANY($1::text[])`, pq.Array(fullRemoveRowIDs)); err != nil {
				logger.S.Errorf("ApplySectWarVictory: failed to batch delete vault Gus for sect %s: %v", combat.TargetSectID, err)
			}
		}
		if len(partialRowIDs) > 0 {
			if _, err := tx.Exec(`
				UPDATE sect_vault_gu AS sv
				SET quantity = sv.quantity - v.amt, updated_at = NOW()
				FROM unnest($1::text[], $2::bigint[]) AS v(id, amt)
				WHERE sv.id = v.id
			`, pq.Array(partialRowIDs), pq.Array(partialAmts)); err != nil {
				logger.S.Errorf("ApplySectWarVictory: failed to batch update vault Gus for sect %s: %v", combat.TargetSectID, err)
			}
		}
	}

	// Distribute delivered Gus to team members (round-robin, one bulk insert per member)
	memberBuckets := make([][]models.GuInstance, len(teamMembers))
	memberIdx := 0
	for _, tg := range gusList {
		for q := int64(0); q < tg.stolen-tg.destroyed; q++ {
			recipient := memberIdx % len(teamMembers)
			memberIdx++
			instID := tg.instanceID
			if instID == "" || q > 0 {
				instID = uuid.New().String()
			}
			r := tg.rank
			if r <= 0 {
				r = 1
			}
			guSpec, ok := models.GetGu(tg.guID, r)
			if !ok {
				guSpec, _ = models.GetGuAny(tg.guID)
			}
			memberBuckets[recipient] = append(memberBuckets[recipient], models.GuInstance{
				InstanceID: instID,
				GuID:       tg.guID,
				Rank:       r,
				Hunger:     vaultWithdrawnHunger(guSpec),
				Status:     "NORMAL",
				Equipped:   false,
				IsVital:    false,
			})
		}
	}
	for i, bucket := range memberBuckets {
		if len(bucket) == 0 {
			continue
		}
		if err := db.AddPlayerGusBulk(tx, teamMembers[i].String(), bucket); err != nil {
			logger.S.Errorf("ApplySectWarVictory: failed to deliver %d Gus to player %s: %v", len(bucket), teamMembers[i], err)
		}
	}

	// 4. Plunder vault_recipes
	recRows, err := tx.Query(`SELECT recipe_id, completion_rate FROM sect_vault_recipes WHERE sect_id = $1 ORDER BY recipe_id ASC FOR UPDATE`, combat.TargetSectID)
	var targetVaultRecipes []models.SectVaultRecipe
	if err != nil {
		logger.S.Errorf("ApplySectWarVictory: failed to load vault recipes for sect %s: %v", combat.TargetSectID, err)
	} else {
		for recRows.Next() {
			var vr models.SectVaultRecipe
			if err := recRows.Scan(&vr.RecipeID, &vr.CompletionRate); err == nil {
				targetVaultRecipes = append(targetVaultRecipes, vr)
			}
		}
		recRows.Close()
	}

	var stolenRecipes []models.SectVaultRecipe
	for _, vr := range targetVaultRecipes {
		if rand.Float64() < plunderPercent {
			stolenRecipes = append(stolenRecipes, vr)
		}
	}

	// Split stolen recipes between Void Destruction and the team (same as death penalty)
	destroyedRecCount := int(math.Round(float64(len(stolenRecipes)) * voidPct))
	if len(stolenRecipes) == 1 {
		if rand.Float64() < 0.5 {
			destroyedRecCount = 1
		} else {
			destroyedRecCount = 0
		}
	} else if destroyedRecCount >= len(stolenRecipes) {
		destroyedRecCount = len(stolenRecipes) - 1
		if destroyedRecCount < 0 {
			destroyedRecCount = 0
		}
	}
	destroyedRecipes := stolenRecipes[:destroyedRecCount]
	deliveredRecipes := stolenRecipes[destroyedRecCount:]

	// Remove every stolen recipe from the vault in one batched delete
	if len(stolenRecipes) > 0 {
		stolenRecIDs := make([]string, 0, len(stolenRecipes))
		for _, sr := range stolenRecipes {
			stolenRecIDs = append(stolenRecIDs, sr.RecipeID)
		}
		if _, err := tx.Exec(`DELETE FROM sect_vault_recipes WHERE sect_id = $1 AND recipe_id = ANY($2::text[])`, combat.TargetSectID, pq.Array(stolenRecIDs)); err != nil {
			logger.S.Errorf("ApplySectWarVictory: failed to batch delete vault recipes for sect %s: %v", combat.TargetSectID, err)
		}
	}

	// Unlock delivered recipes for team members in one batched upsert (preserves
	// existing progress: completion_rate stays the GREATEST, deduce/favorite untouched)
	if len(deliveredRecipes) > 0 {
		recIDs := make([]string, 0, len(deliveredRecipes))
		recRates := make([]float64, 0, len(deliveredRecipes))
		for _, sr := range deliveredRecipes {
			recIDs = append(recIDs, sr.RecipeID)
			recRates = append(recRates, sr.CompletionRate)
		}
		memberIDs := make([]string, 0, len(teamMembers))
		for _, mID := range teamMembers {
			memberIDs = append(memberIDs, mID.String())
		}
		if _, err := tx.Exec(`
			INSERT INTO player_recipe_progress (player_id, recipe_id, completion_rate, is_unlocked, created_at, updated_at)
			SELECT m, r.rec_id, r.rate, true, NOW(), NOW()
			FROM unnest($1::uuid[]) AS m
			CROSS JOIN unnest($2::text[], $3::double precision[]) AS r(rec_id, rate)
			ON CONFLICT (player_id, recipe_id)
			DO UPDATE SET is_unlocked = true, completion_rate = GREATEST(player_recipe_progress.completion_rate, EXCLUDED.completion_rate), updated_at = NOW()
		`, pq.Array(memberIDs), pq.Array(recIDs), pq.Array(recRates)); err != nil {
			logger.S.Errorf("ApplySectWarVictory: failed to batch unlock recipes for team: %v", err)
		}
	}

	now := time.Now()
	graceShield := now.Add(1 * time.Hour)

	// Forced exile teleport: find a valid random outdoor location that does not overlap any sect
	targetMainHallLvl := getBuildingLvl(targetBuildings, "main_hall")
	targetRadius := 5 + targetMainHallLvl
	newExileX, newExileY, errExile := FindValidExileCoordinates(tx, targetRegion, targetRadius, combat.TargetSectID)

	// Update target sect in DB: deduct plundered funds, grant 1h grace shield, and exile if new coords found
	if errExile == nil {
		_, _ = tx.Exec(`
			UPDATE sects SET
				funds = GREATEST(0, funds - $1),
				shield_until = $2,
				map_x = $3,
				map_y = $4
			WHERE id = $5
		`, stolenFunds, graceShield, newExileX, newExileY, combat.TargetSectID)
	} else {
		logger.S.Warnf("ApplySectWarVictory: failed to find valid exile coordinates for sect %s: %v", combat.TargetSectID, errExile)
		_, _ = tx.Exec(`
			UPDATE sects SET
				funds = GREATEST(0, funds - $1),
				shield_until = $2
			WHERE id = $3
		`, stolenFunds, graceShield, combat.TargetSectID)
	}

	_ = cache.Delete(sectsListCacheKey)

	// Reset defending status for all target sect members (members stay at their current location)
	_, _ = tx.Exec(`UPDATE sect_members SET is_defending = false WHERE sect_id = $1`, combat.TargetSectID)

	// Award attacker
	_, _ = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if mySectID != "" {
		_, _ = tx.Exec(`UPDATE sect_members SET contribution_points = contribution_points + 200 WHERE player_id = $1`, player.ID)
	}

	logs = append(logs, fmt.Sprintf("Công phá Tông Môn %s thành công!", targetName))
	if stolenFunds > 0 {
		if mySectID == "" {
			logs = append(logs, fmt.Sprintf("Cướp phá tiêu hủy %d Quỹ Tông Môn (quỹ bị tiêu biến vào hư vô).", stolenFunds))
		} else if destroyedFunds > 0 {
			logs = append(logs, fmt.Sprintf("Cướp được %d Quỹ Tông Môn: %d chuyển vào Tông Môn ta, %d bị Thiên Đạo tiêu hủy vĩnh viễn.", stolenFunds, creditedFunds, destroyedFunds))
		} else {
			logs = append(logs, fmt.Sprintf("Cướp được %d Quỹ Tông Môn chuyển vào Tông Môn ta.", stolenFunds))
		}
	}
	var matDelivered, matDestroyed []string
	for _, m := range matsList {
		name := m.itemID
		if spec, ok := models.ItemRegistry[m.itemID]; ok && spec.Name != "" {
			name = spec.Name
		}
		if d := m.stolen - m.destroyed; d > 0 {
			matDelivered = append(matDelivered, fmt.Sprintf("%s x%d", name, d))
		}
		if m.destroyed > 0 {
			matDestroyed = append(matDestroyed, fmt.Sprintf("%s x%d", name, m.destroyed))
		}
	}
	if len(matDelivered) > 0 {
		logs = append(logs, "Cướp linh tài kho tông (chia đều cho đội): "+joinDropList(matDelivered))
	}
	if len(matDestroyed) > 0 {
		logs = append(logs, "Thiên Đạo tiêu hủy vĩnh viễn linh tài: "+joinDropList(matDestroyed))
	}
	var guDelivered, guDestroyed []string
	for _, tg := range gusList {
		name := tg.guID
		if guDef, ok := models.GetGuAny(tg.guID); ok && guDef.Name != "" {
			name = guDef.Name
		}
		if d := tg.stolen - tg.destroyed; d > 0 {
			guDelivered = append(guDelivered, fmt.Sprintf("%s R%d x%d", name, tg.rank, d))
		}
		if tg.destroyed > 0 {
			guDestroyed = append(guDestroyed, fmt.Sprintf("%s R%d x%d", name, tg.rank, tg.destroyed))
		}
	}
	if len(guDelivered) > 0 {
		logs = append(logs, "Cướp Cổ trùng kho tông (chia cho đội): "+joinDropList(guDelivered))
	}
	if len(guDestroyed) > 0 {
		logs = append(logs, "Thiên Đạo chấn nát vĩnh viễn Cổ trùng kho tông: "+joinDropList(guDestroyed))
	}
	var recDelivered, recDestroyed []string
	for _, sr := range destroyedRecipes {
		name := sr.RecipeID
		if guDef, ok := models.GetGuAny(sr.RecipeID); ok && guDef.Name != "" {
			name = guDef.Name
		}
		recDestroyed = append(recDestroyed, name)
	}
	for _, sr := range deliveredRecipes {
		name := sr.RecipeID
		if guDef, ok := models.GetGuAny(sr.RecipeID); ok && guDef.Name != "" {
			name = guDef.Name
		}
		recDelivered = append(recDelivered, name)
	}
	if len(recDelivered) > 0 {
		logs = append(logs, "Cướp bí phương kho tông (mở khóa cho đội): "+joinDropList(recDelivered))
	}
	if len(recDestroyed) > 0 {
		logs = append(logs, "Thiên Đạo tiêu hủy vĩnh viễn bí phương kho tông: "+joinDropList(recDestroyed))
	}

	// Notifications — full losses summary for victim sect channel
	var totalStolenMats int64
	for _, m := range matsList {
		totalStolenMats += m.stolen
	}
	if mySectID != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: mySectID, Valid: true},
			fmt.Sprintf("[Chiến Báo] Tông môn ta đã đại phá sơn môn %s! Cướp được %d Quỹ Tông Môn.", targetName, stolenFunds))
	}
	exileNotice := ""
	if errExile == nil {
		exileNotice = fmt.Sprintf(" Tông môn buộc phải dời đến tọa độ mới (%d, %d) thuộc %s!", newExileX, newExileY, targetRegion)
	}
	_ = SendSystemMessage("sect", sql.NullString{String: combat.TargetSectID, Valid: true},
		fmt.Sprintf("[Chiến Báo] Sơn Môn bị %s công phá! Tổn thất %d Quỹ, %d Linh tài.%s Toàn tông bế môn chấn chỉnh sơn môn trong 1 giờ.", player.Username, stolenFunds, totalStolenMats, exileNotice))

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
		fmt.Sprintf("[Chiến Báo] Sơn Môn đã đánh lui cuộc tập kích của %s! Môn nhân trấn thủ được ban thưởng cống hiến.", player.Username))

	logs = append(logs, fmt.Sprintf("Khiêu chiến Tông Môn %s thất bại! Hộ Sơn Đại Trận quá vững chãi.", targetName))
	return logs
}
