package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// --- Monster Spawns Handlers ---

type MonsterSpawnRow struct {
	ID             string `json:"id"`
	RegionID       string `json:"region_id"`
	MonsterID      string `json:"monster_id"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	SpawnMode      string `json:"spawn_mode"`
	Chance         float64`json:"chance"`
	RespawnSeconds int    `json:"respawn_seconds"`
	MinRank        int    `json:"min_rank"`
	MaxRank        int    `json:"max_rank"`
	IsBossSpawn    bool   `json:"is_boss_spawn"`
	IsActive       bool   `json:"is_active"`
}

func AdminListMonsterSpawns(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	regionID := c.Query("region_id")
	where := ""
	args := []interface{}{}
	if regionID != "" {
		where = " WHERE region_id = $1"
		args = append(args, regionID)
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, region_id, monster_id, x, y, COALESCE(spawn_mode,'normal'), COALESCE(chance,1.0), COALESCE(respawn_seconds,300), COALESCE(min_rank,1), COALESCE(max_rank,9), COALESCE(is_boss_spawn,false), COALESCE(is_active,true)
		FROM monster_spawns `+where+` ORDER BY region_id ASC, id ASC`, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []MonsterSpawnRow{}
	for rows.Next() {
		var r MonsterSpawnRow
		if err := rows.Scan(&r.ID, &r.RegionID, &r.MonsterID, &r.X, &r.Y, &r.SpawnMode, &r.Chance, &r.RespawnSeconds, &r.MinRank, &r.MaxRank, &r.IsBossSpawn, &r.IsActive); err != nil {
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateMonsterSpawn(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req MonsterSpawnRow
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.RegionID == "" || req.MonsterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID, RegionID, MonsterID là bắt buộc"})
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO monster_spawns (id, region_id, monster_id, x, y, spawn_mode, chance, respawn_seconds, min_rank, max_rank, is_boss_spawn, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		req.ID, req.RegionID, req.MonsterID, req.X, req.Y, req.SpawnMode, req.Chance, req.RespawnSeconds, req.MinRank, req.MaxRank, req.IsBossSpawn, req.IsActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_monster_spawn", "id=%s", req.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo điểm xuất hiện quái"})
}

func AdminUpdateMonsterSpawn(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var req MonsterSpawnRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE monster_spawns SET region_id=$1, monster_id=$2, x=$3, y=$4, spawn_mode=$5, chance=$6, respawn_seconds=$7, min_rank=$8, max_rank=$9, is_boss_spawn=$10, is_active=$11, updated_at=NOW()
		WHERE id=$12`,
		req.RegionID, req.MonsterID, req.X, req.Y, req.SpawnMode, req.Chance, req.RespawnSeconds, req.MinRank, req.MaxRank, req.IsBossSpawn, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_monster_spawn", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật điểm xuất hiện quái"})
}

func AdminDeleteMonsterSpawn(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM monster_spawns WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_monster_spawn", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa điểm xuất hiện quái"})
}

// --- World Quests Handlers ---

type QuestRow struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	ReqDesc     string          `json:"req_desc"`
	Message     string          `json:"message"`
	Rewards     json.RawMessage `json:"rewards"`
}

func AdminListQuests(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, COALESCE(title,''), COALESCE(description,''), COALESCE(req_desc,''), COALESCE(message,''), COALESCE(rewards,'{}') FROM quests ORDER BY id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []QuestRow{}
	for rows.Next() {
		var q QuestRow
		if err := rows.Scan(&q.ID, &q.Title, &q.Description, &q.ReqDesc, &q.Message, &q.Rewards); err != nil {
			continue
		}
		list = append(list, q)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateQuest(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req QuestRow
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID và tiêu đề nhiệm vụ là bắt buộc"})
		return
	}
	if len(req.Rewards) == 0 {
		req.Rewards = json.RawMessage(`{}`)
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO quests (id, title, description, req_desc, message, rewards) VALUES ($1, $2, $3, $4, $5, $6)`,
		req.ID, req.Title, req.Description, req.ReqDesc, req.Message, req.Rewards)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_quest", "id=%s", req.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo nhiệm vụ thế giới"})
}

func AdminUpdateQuest(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var req QuestRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `UPDATE quests SET title=$1, description=$2, req_desc=$3, message=$4, rewards=$5 WHERE id=$6`,
		req.Title, req.Description, req.ReqDesc, req.Message, req.Rewards, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_quest", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật nhiệm vụ thế giới"})
}

func AdminDeleteQuest(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM quests WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_quest", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa nhiệm vụ thế giới"})
}

// --- Dungeons / Instance Templates Handlers ---

type InstanceTemplateRow struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	RequiredRank  int             `json:"required_rank"`
	EntryItem     string          `json:"entry_item"`
	Description   string          `json:"description"`
	Rewards       json.RawMessage `json:"rewards"`
	MapData       json.RawMessage `json:"map_data"`
	CooldownHours int             `json:"cooldown_hours"`
}

func AdminListDungeons(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, COALESCE(name,''), COALESCE(type,'normal'), COALESCE(required_rank,1), COALESCE(entry_item,''), COALESCE(description,''), COALESCE(rewards,'{}'), COALESCE(map_data,'{}'), COALESCE(cooldown_hours,24) FROM instance_templates ORDER BY required_rank ASC, id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []InstanceTemplateRow{}
	for rows.Next() {
		var r InstanceTemplateRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.RequiredRank, &r.EntryItem, &r.Description, &r.Rewards, &r.MapData, &r.CooldownHours); err != nil {
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateDungeon(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req InstanceTemplateRow
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID và Tên phụ bản là bắt buộc"})
		return
	}
	if len(req.Rewards) == 0 {
		req.Rewards = json.RawMessage(`{}`)
	}
	if len(req.MapData) == 0 {
		req.MapData = json.RawMessage(`{}`)
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO instance_templates (id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.ID, req.Name, req.Type, req.RequiredRank, req.EntryItem, req.Description, req.Rewards, req.MapData, req.CooldownHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_dungeon", "id=%s", req.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo phụ bản mới"})
}

func AdminUpdateDungeon(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var req InstanceTemplateRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `UPDATE instance_templates SET name=$1, type=$2, required_rank=$3, entry_item=$4, description=$5, rewards=$6, map_data=$7, cooldown_hours=$8 WHERE id=$9`,
		req.Name, req.Type, req.RequiredRank, req.EntryItem, req.Description, req.Rewards, req.MapData, req.CooldownHours, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_dungeon", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật phụ bản"})
}

func AdminDeleteDungeon(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM instance_templates WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_dungeon", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa phụ bản"})
}

// --- Sect Mines Handlers ---

type SectMineRow struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Rarity           string   `json:"rarity"`
	OccupierSectID   string   `json:"occupier_sect_id"`
	OccupierSectName string   `json:"occupier_sect_name"`
	ShieldUntil      *string  `json:"shield_until"`
	PendingFunds     int64    `json:"pending_funds"`
	PendingMaterials string   `json:"pending_materials"`
}

func AdminListSectMines(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT sm.id, sm.name, sm.rarity, COALESCE(sm.occupier_sect_id,''), COALESCE(s.name,''), sm.shield_until, COALESCE(sm.pending_funds,0), COALESCE(sm.pending_materials::text,'{}')
		FROM sect_mines sm
		LEFT JOIN sects s ON sm.occupier_sect_id = s.id
		ORDER BY sm.id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []SectMineRow{}
	for rows.Next() {
		var r SectMineRow
		var shield sql.NullTime
		if err := rows.Scan(&r.ID, &r.Name, &r.Rarity, &r.OccupierSectID, &r.OccupierSectName, &shield, &r.PendingFunds, &r.PendingMaterials); err != nil {
			continue
		}
		if shield.Valid {
			str := shield.Time.Format("2006-01-02 15:04:05")
			r.ShieldUntil = &str
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminUpdateSectMine(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var req struct {
		OccupierSectID string `json:"occupier_sect_id"`
		Rarity         string `json:"rarity"`
		PendingFunds   int64  `json:"pending_funds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var occupier interface{}
	if req.OccupierSectID != "" {
		occupier = req.OccupierSectID
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `UPDATE sect_mines SET occupier_sect_id=$1, rarity=COALESCE(NULLIF($2,''), rarity), pending_funds=$3 WHERE id=$4`,
		occupier, req.Rarity, req.PendingFunds, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_sect_mine", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mỏ tông môn"})
}

// --- Aperture Sub-entities (Nodes & Variant Humans) ---

type ApertureNodeRow struct {
	ID                       string  `json:"id"`
	ApertureID               string  `json:"aperture_id"`
	NodeType                 string  `json:"node_type"`
	Name                     string  `json:"name"`
	Path                     string  `json:"path"`
	Level                    int     `json:"level"`
	HP                       int     `json:"hp"`
	MaxHP                    int     `json:"max_hp"`
	X                        int     `json:"x"`
	Y                        int     `json:"y"`
	ProductionType           string  `json:"production_type"`
	ProductionAmount         float64 `json:"production_amount"`
	ProductionIntervalHours  float64 `json:"production_interval_hours"`
	IsDestroyed              bool    `json:"is_destroyed"`
	NodeCategory             string  `json:"node_category"`
	DefensePower             float64 `json:"defense_power"`
	AttackRange              float64 `json:"attack_range"`
}

func AdminListApertureNodes(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	apertureID := c.Param("id")
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, aperture_id, node_type, name, path, level, hp, max_hp, x, y, production_type, production_amount, production_interval_hours, is_destroyed, COALESCE(node_category,'resource'), COALESCE(defense_power,0), COALESCE(attack_range,0)
		FROM aperture_resource_nodes WHERE aperture_id = $1 ORDER BY created_at DESC`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []ApertureNodeRow{}
	for rows.Next() {
		var r ApertureNodeRow
		if err := rows.Scan(&r.ID, &r.ApertureID, &r.NodeType, &r.Name, &r.Path, &r.Level, &r.HP, &r.MaxHP, &r.X, &r.Y, &r.ProductionType, &r.ProductionAmount, &r.ProductionIntervalHours, &r.IsDestroyed, &r.NodeCategory, &r.DefensePower, &r.AttackRange); err != nil {
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateApertureNode(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	apertureID := c.Param("id")
	var req ApertureNodeRow
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên điểm tài nguyên là bắt buộc"})
		return
	}
	if req.MaxHP <= 0 {
		req.MaxHP = 100
	}
	if req.HP <= 0 {
		req.HP = req.MaxHP
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO aperture_resource_nodes (aperture_id, node_type, name, path, level, hp, max_hp, x, y, production_type, production_amount, production_interval_hours, is_destroyed, node_category, defense_power, attack_range)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		apertureID, req.NodeType, req.Name, req.Path, req.Level, req.HP, req.MaxHP, req.X, req.Y, req.ProductionType, req.ProductionAmount, req.ProductionIntervalHours, req.IsDestroyed, req.NodeCategory, req.DefensePower, req.AttackRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_aperture_node", "aperture_id=%s, name=%s", apertureID, req.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo điểm tài nguyên tiên khiếu"})
}

func AdminDeleteApertureNode(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	nodeID := c.Param("node_id")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM aperture_resource_nodes WHERE id=$1`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_aperture_node", "node_id=%s", nodeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa điểm tài nguyên"})
}

type VariantHumanRow struct {
	ID             string  `json:"id"`
	ApertureID     string  `json:"aperture_id"`
	Race           string  `json:"race"`
	Population     int64   `json:"population"`
	Happiness      float64 `json:"happiness"`
	Oppression     float64 `json:"oppression"`
	RebellionRate  float64 `json:"rebellion_rate"`
	LaborIntensity float64 `json:"labor_intensity"`
}

func AdminListApertureVariantHumans(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	apertureID := c.Param("id")
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, aperture_id, race, population, happiness, oppression, rebellion_rate, labor_intensity
		FROM aperture_variant_humans WHERE aperture_id = $1 ORDER BY race ASC`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []VariantHumanRow{}
	for rows.Next() {
		var r VariantHumanRow
		if err := rows.Scan(&r.ID, &r.ApertureID, &r.Race, &r.Population, &r.Happiness, &r.Oppression, &r.RebellionRate, &r.LaborIntensity); err != nil {
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateApertureVariantHuman(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	apertureID := c.Param("id")
	var req VariantHumanRow
	if err := c.ShouldBindJSON(&req); err != nil || req.Race == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chủng tộc là bắt buộc"})
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO aperture_variant_humans (aperture_id, race, population, happiness, oppression, rebellion_rate, labor_intensity)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		apertureID, req.Race, req.Population, req.Happiness, req.Oppression, req.RebellionRate, req.LaborIntensity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_variant_human", "aperture_id=%s, race=%s", apertureID, req.Race)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo nhóm dị nhân trong tiên khiếu"})
}

func AdminDeleteApertureVariantHuman(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	vhID := c.Param("vh_id")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM aperture_variant_humans WHERE id=$1`, vhID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_variant_human", "id=%s", vhID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa nhóm dị nhân"})
}
