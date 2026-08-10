package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type SectMineDefender struct {
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

type SectorOutput struct {
	FundsPerHour        int `json:"funds_per_hour"`
	ContributionPerHour int `json:"contribution_per_hour"`
}

type SectMineResponse struct {
	ID                int                `json:"id"`
	Name              string             `json:"name"`
	Rarity            string             `json:"rarity"`
	OccupierSectID    *string            `json:"occupier_sect_id"`
	OccupierName      *string            `json:"occupier_name"`
	OccupiedAt        *time.Time         `json:"occupied_at"`
	ShieldUntil       *time.Time         `json:"shield_until"`
	LastClaimedAt     *time.Time         `json:"last_claimed_at"`
	Defenders         []SectMineDefender `json:"defenders"`
	TotalDefensePower float64            `json:"total_defense_power"`
	OutputPerHour     *SectorOutput      `json:"output_per_hour"`
}

// sectMineBaseRate returns base Quỹ per minute by mine rarity.
func sectMineBaseRate(rarity string) float64 {
	switch rarity {
	case "Siêu Cấp":
		return 200.0
	case "Truyền Thuyết":
		return 100.0
	case "Sử Thi":
		return 50.0
	case "Hiếm":
		return 25.0
	default:
		return 10.0
	}
}

// sectMineDefenderSlots returns the max number of defenders a mine allows.
func sectMineDefenderSlots(rarity string) int {
	switch rarity {
	case "Siêu Cấp":
		return 4
	case "Truyền Thuyết", "Sử Thi":
		return 3
	case "Hiếm":
		return 2
	default:
		return 1
	}
}

// sectMineBaseDefense is the inherent defense power of an occupied-but-undefended mine.
func sectMineBaseDefense(rarity string) float64 {
	switch rarity {
	case "Siêu Cấp":
		return 700
	case "Truyền Thuyết":
		return 400
	case "Sử Thi":
		return 250
	case "Hiếm":
		return 120
	default:
		return 50
	}
}

// computeMemberPower estimates a member's combat power for territory defense.
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

// computeBeastPower estimates a guardian beast's combat power.
func computeBeastPower(mon models.MonsterTemplate, rank, subRank int) float64 {
	maxHP, atk, def := models.ScaleMonsterStats(mon.MaxHP, mon.BaseAttack, mon.BaseDefense, rank, subRank, mon.Type)
	return atk + def + maxHP/20.0
}

// loadMineDefenders loads all defenders of a mine plus their total power.
func loadMineDefenders(mineID int) ([]SectMineDefender, float64, error) {
	rows, err := db.DB.Query(`
		SELECT id, defender_type, COALESCE(player_id::text, ''), name, COALESCE(monster_id, ''),
		       monster_rank, monster_sub_rank, power, assigned_at
		FROM sect_mine_defenders
		WHERE mine_id = $1
		ORDER BY assigned_at ASC
	`, mineID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var defenders []SectMineDefender
	total := 0.0
	for rows.Next() {
		var d SectMineDefender
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

func GetSectMines(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT sm.id, sm.name, sm.rarity, sm.occupier_sect_id, s.name, sm.occupied_at, sm.shield_until, sm.last_claimed_at
		FROM sect_mines sm
		LEFT JOIN sects s ON sm.occupier_sect_id = s.id
		ORDER BY sm.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect mines"})
		return
	}
	defer rows.Close()

	var mines []SectMineResponse
	for rows.Next() {
		var m SectMineResponse
		var oName sql.NullString
		err = rows.Scan(&m.ID, &m.Name, &m.Rarity, &m.OccupierSectID, &oName, &m.OccupiedAt, &m.ShieldUntil, &m.LastClaimedAt)
		if err == nil {
			if oName.Valid {
				m.OccupierName = &oName.String
			}
			defenders, total, derr := loadMineDefenders(m.ID)
			if derr == nil {
				m.Defenders = defenders
				m.TotalDefensePower = total
			}
			if m.OccupierSectID != nil {
				var sectLevel, memberCount int
				_ = db.DB.QueryRow(`SELECT level FROM sects WHERE id = $1`, *m.OccupierSectID).Scan(&sectLevel)
				_ = db.DB.QueryRow(`SELECT COUNT(*) FROM sect_members WHERE sect_id = $1`, *m.OccupierSectID).Scan(&memberCount)
				defCount := len(m.Defenders)
				fundsPerHour := int(sectMineBaseRate(m.Rarity) * float64(sectLevel) * (1 + 0.2*float64(defCount)) * 60)
				contribPerHour := defCount*20 + (memberCount-defCount)*2
				if contribPerHour < 0 {
					contribPerHour = 0
				}
				m.OutputPerHour = &SectorOutput{FundsPerHour: fundsPerHour, ContributionPerHour: contribPerHour}
			}
			mines = append(mines, m)
		}
	}
	c.JSON(http.StatusOK, mines)
}

type DefendSectMineRequest struct {
	MineID       int    `json:"mine_id"`
	DefenderType string `json:"defender_type"` // member | beast
	PlayerID     string `json:"player_id"`
	MonsterID    string `json:"monster_id"`
	MonsterRank  int    `json:"monster_rank"`
}

// DefendSectMine assigns a member or hires a guardian beast to defend an owned mine.
func DefendSectMine(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DefendSectMineRequest
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

	var occupierSectID sql.NullString
	var rarity string
	err = tx.QueryRow(`SELECT occupier_sect_id, rarity FROM sect_mines WHERE id = $1 FOR UPDATE`, req.MineID).Scan(&occupierSectID, &rarity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mỏ khoáng!"})
		return
	}
	if !occupierSectID.Valid || occupierSectID.String != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tông môn của bạn không chiếm đóng mỏ khoáng này!"})
		return
	}

	var defCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM sect_mine_defenders WHERE mine_id = $1`, req.MineID).Scan(&defCount)
	if defCount >= sectMineDefenderSlots(rarity) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ đã đầy chỗ phòng thủ!"})
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

		var targetSect string
		var targetName string
		err = tx.QueryRow(`SELECT sm.sect_id, p.username FROM sect_members sm JOIN players p ON p.id = sm.player_id WHERE sm.player_id = $1`, pid).Scan(&targetSect, &targetName)
		if err != nil || targetSect != sectID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử này không thuộc tông môn của bạn!"})
			return
		}

		var existingID string
		err = tx.QueryRow(`SELECT id FROM sect_mine_defenders WHERE player_id = $1 AND defender_type = 'member'`, pid).Scan(&existingID)
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử này đang phòng thủ một mỏ khác!"})
			return
		}

		player, lerr := evaluateManaTxEquipped(tx, pid)
		if lerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
			return
		}
		power := computeMemberPower(&player)

		_, err = tx.Exec(`
			INSERT INTO sect_mine_defenders (mine_id, sect_id, defender_type, player_id, name, power, assigned_at)
			VALUES ($1, $2, 'member', $3, $4, $5, now())
		`, req.MineID, sectID, pid, targetName, power)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign defender: " + err.Error()})
			return
		}
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s đã được phong làm phòng thủ cho %s.", targetName, mineNameByID(req.MineID)))
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
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect funds"})
			return
		}
		if funds < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ để thuê thần thú hạng %d (cần %d Quỹ)!", rank, cost)})
			return
		}

		mon, merr := db.LoadMonsterTemplate(tx, req.MonsterID)
		if merr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy thần thú này!"})
			return
		}
		power := computeBeastPower(mon, rank, 1)
		name := fmt.Sprintf("%s (Hạng %d)", mon.Name, rank)

		_, err = tx.Exec(`
			INSERT INTO sect_mine_defenders (mine_id, sect_id, defender_type, name, monster_id, monster_rank, monster_sub_rank, power, assigned_at)
			VALUES ($1, $2, 'beast', $3, $4, $5, 1, $6, now())
		`, req.MineID, sectID, name, req.MonsterID, rank, power)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hire guardian beast: " + err.Error()})
			return
		}
		_, err = tx.Exec(`UPDATE sects SET funds = funds - $1 WHERE id = $2`, cost, sectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct funds"})
			return
		}
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Tông môn đã thuê %s canh giữ %s.", name, mineNameByID(req.MineID)))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại phòng thủ không hợp lệ (member hoặc beast)!"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã bố trí phòng thủ thành công!"})
}

func mineNameByID(mineID int) string {
	var name string
	if err := db.DB.QueryRow(`SELECT name FROM sect_mines WHERE id = $1`, mineID).Scan(&name); err != nil {
		return "Lãnh địa"
	}
	return name
}

// UndefendSectMine removes a defender from a mine.
func UndefendSectMine(c *gin.Context) {
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
	err = db.DB.QueryRow(`SELECT sect_id FROM sect_mine_defenders WHERE id = $1`, did).Scan(&dSectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phòng thủ này!"})
		return
	}
	if dSectID != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Phòng thủ này thuộc tông môn khác!"})
		return
	}

	_, err = db.DB.Exec(`DELETE FROM sect_mine_defenders WHERE id = $1`, did)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể rút phòng thủ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã rút phòng thủ!"})
}

func AttackSectMine(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		MineID     int    `json:"mine_id"`
		AttackerID string `json:"attacker_id"`
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

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	// Only leadership can launch territory attacks.
	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão/Hộ Pháp/Chưởng Môn mới được phát động công kích lãnh địa!"})
		return
	}

	// Resolve who actually fights: default = the leader launching, else the chosen member.
	attackerID := playerID
	if req.AttackerID != "" {
		aid, perr := uuid.Parse(req.AttackerID)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "attacker_id không hợp lệ"})
			return
		}
		var aSectID, aRank string
		err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, aid).Scan(&aSectID, &aRank)
		if err != nil || aSectID != sectID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử được chọn không thuộc tông môn của bạn!"})
			return
		}
		if aRank == "NGOAI_MON" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Ngoại môn đệ tử không thể tham gia tranh đoạt mỏ khoáng!"})
			return
		}
		attackerID = aid
	}

	var occupierSectID sql.NullString
	var shieldUntil sql.NullTime
	var rarity string
	err = tx.QueryRow(`SELECT occupier_sect_id, shield_until, rarity FROM sect_mines WHERE id = $1 FOR UPDATE`, req.MineID).Scan(&occupierSectID, &shieldUntil, &rarity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mỏ khoáng!"})
		return
	}

	if occupierSectID.Valid && occupierSectID.String == sectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn của bạn đã chiếm đóng mỏ khoáng này rồi!"})
		return
	}

	if shieldUntil.Valid && time.Now().Before(shieldUntil.Time) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ khoáng này đang trong thời gian bảo hộ!"})
		return
	}

	now := time.Now()

	// Unoccupied mine → capture directly (preserves legacy behavior).
	if !occupierSectID.Valid {
		newShieldUntil := now.Add(30 * time.Minute)
		_, err = tx.Exec(`UPDATE sect_mines SET occupier_sect_id = $1, occupied_at = $2, shield_until = $3, last_claimed_at = $2 WHERE id = $4`, sectID, now, newShieldUntil, req.MineID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mine occupier"})
			return
		}
		_, _ = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 5, $1) WHERE id = $2", MaxHumanQi, attackerID)
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Đã chiếm đóng mỏ khoáng thành công! Nhân Khí +5."})
		return
	}

	// Occupied mine → power showdown against the defenders.
	defenders, totalDefense, derr := loadMineDefendersTx(tx, req.MineID)
	if derr != nil {
		totalDefense = 0
	}
	if totalDefense < 1 {
		totalDefense = sectMineBaseDefense(rarity)
	}

	attacker, aerr := evaluateManaTxEquipped(tx, attackerID)
	if aerr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}
	attackerPower := computeMemberPower(&attacker)

	variance := 0.9 + rand.Float64()*0.2 // ±10%
	attackerEffective := attackerPower * variance

	if attackerEffective >= totalDefense {
		// Victory → capture
		newShieldUntil := now.Add(30 * time.Minute)
		_, err = tx.Exec(`UPDATE sect_mines SET occupier_sect_id = $1, occupied_at = $2, shield_until = $3, last_claimed_at = $2 WHERE id = $4`, sectID, now, newShieldUntil, req.MineID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mine occupier"})
			return
		}
		_, _ = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, attackerID)
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":        fmt.Sprintf("Đã phá vỡ phòng thủ (%.0f vs %.0f) và chiếm đóng mỏ khoáng! Nhân Khí +10.", attackerEffective, totalDefense),
			"won":            true,
			"attacker_power": attackerEffective,
			"defense_power":  totalDefense,
		})
		return
	}

	// Defeat → 15 min cooldown + reward defenders.
	cooldownUntil := now.Add(15 * time.Minute)
	_, err = tx.Exec(`UPDATE sect_mines SET shield_until = $1 WHERE id = $2`, cooldownUntil, req.MineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mine cooldown"})
		return
	}

	// Bonus contribution for member defenders who repelled the attack.
	if len(defenders) > 0 {
		ids := make([]uuid.UUID, 0, len(defenders))
		for _, d := range defenders {
			if d.PlayerID != nil {
				if pid, perr := uuid.Parse(*d.PlayerID); perr == nil {
					ids = append(ids, pid)
				}
			}
		}
		if len(ids) > 0 {
			for _, pid := range ids {
				_, _ = tx.Exec(`UPDATE sect_members SET contribution_points = contribution_points + 20 WHERE player_id = $1`, pid)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        fmt.Sprintf("Công kích thất bại! Phòng thủ quá mạnh (%.0f vs %.0f). Trận pháp hộ sơn khởi động 15 phút.", attackerEffective, totalDefense),
		"won":            false,
		"attacker_power": attackerEffective,
		"defense_power":  totalDefense,
	})
}

// loadMineDefendersTx is like loadMineDefenders but runs within the given tx.
func loadMineDefendersTx(tx *sql.Tx, mineID int) ([]SectMineDefender, float64, error) {
	rows, err := tx.Query(`
		SELECT id, defender_type, COALESCE(player_id::text, ''), name, COALESCE(monster_id, ''),
		       monster_rank, monster_sub_rank, power, assigned_at
		FROM sect_mine_defenders
		WHERE mine_id = $1
		ORDER BY assigned_at ASC
	`, mineID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var defenders []SectMineDefender
	total := 0.0
	for rows.Next() {
		var d SectMineDefender
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
