package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// --- Monster Spawns Handlers ---

type MonsterSpawnRow struct {
	ID             string  `json:"id"`
	RegionID       string  `json:"region_id"`
	MonsterID      string  `json:"monster_id"`
	X              int     `json:"x"`
	Y              int     `json:"y"`
	SpawnMode      string  `json:"spawn_mode"`
	Chance         float64 `json:"chance"`
	RespawnSeconds int     `json:"respawn_seconds"`
	MinRank        int     `json:"min_rank"`
	MaxRank        int     `json:"max_rank"`
	IsBossSpawn    bool    `json:"is_boss_spawn"`
	IsActive       bool    `json:"is_active"`
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
		SELECT id, region_id, monster_id, COALESCE(x,0), COALESCE(y,0), COALESCE(spawn_mode,'Random'), COALESCE(chance,1.0), COALESCE(respawn_seconds,300), COALESCE(min_rank,1), COALESCE(max_rank,9), COALESCE(is_boss_spawn,false), COALESCE(is_active,true)
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ điểm xuất hiện quái"})
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách điểm xuất hiện quái"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func validateMonsterSpawnInput(c *gin.Context, req *MonsterSpawnRow) bool {
	req.RegionID, req.MonsterID, req.SpawnMode = strings.TrimSpace(req.RegionID), strings.TrimSpace(req.MonsterID), strings.TrimSpace(req.SpawnMode)
	if req.RegionID == "" || len(req.RegionID) > 50 || req.MonsterID == "" || len(req.MonsterID) > 50 || req.SpawnMode == "" || len(req.SpawnMode) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khu vực, mẫu quái và chế độ xuất hiện là bắt buộc"})
		return false
	}
	if math.IsNaN(req.Chance) || math.IsInf(req.Chance, 0) || req.Chance < 0 || req.Chance > 1 || req.RespawnSeconds < 0 || req.MinRank < 1 || req.MaxRank < req.MinRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tỉ lệ phải từ 0 đến 1, thời gian hồi không âm và khoảng rank hợp lệ"})
		return false
	}
	return true
}

func AdminCreateMonsterSpawn(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req MonsterSpawnRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if !validateMonsterSpawnInput(c, &req) {
		return
	}
	if req.SpawnMode == "normal" {
		req.SpawnMode = "Random"
	}
	var err error
	if req.ID == "" {
		err = db.DB.QueryRowContext(c.Request.Context(), `
		INSERT INTO monster_spawns (region_id, monster_id, x, y, spawn_mode, chance, respawn_seconds, min_rank, max_rank, is_boss_spawn, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
		req.RegionID, req.MonsterID, req.X, req.Y, req.SpawnMode, req.Chance, req.RespawnSeconds, req.MinRank, req.MaxRank, req.IsBossSpawn, req.IsActive).Scan(&req.ID)
	} else {
		_, err = db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO monster_spawns (id, region_id, monster_id, x, y, spawn_mode, chance, respawn_seconds, min_rank, max_rank, is_boss_spawn, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		req.ID, req.RegionID, req.MonsterID, req.X, req.Y, req.SpawnMode, req.Chance, req.RespawnSeconds, req.MinRank, req.MaxRank, req.IsBossSpawn, req.IsActive)
	}
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
	if req.ID != "" && req.ID != id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi ID điểm xuất hiện"})
		return
	}
	if !validateMonsterSpawnInput(c, &req) {
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE monster_spawns SET region_id=$1, monster_id=$2, x=$3, y=$4, spawn_mode=$5, chance=$6, respawn_seconds=$7, min_rank=$8, max_rank=$9, is_boss_spawn=$10, is_active=$11, updated_at=NOW()
		WHERE id=$12`,
		req.RegionID, req.MonsterID, req.X, req.Y, req.SpawnMode, req.Chance, req.RespawnSeconds, req.MinRank, req.MaxRank, req.IsBossSpawn, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận điểm xuất hiện đã được cập nhật"})
		return
	}
	if updated == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy điểm xuất hiện"})
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
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM monster_spawns WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận điểm xuất hiện đã được xóa"})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy điểm xuất hiện"})
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

func validateQuestRewards(c *gin.Context, raw *json.RawMessage) bool {
	if len(*raw) == 0 || string(*raw) == "{}" {
		*raw = json.RawMessage(`[]`)
	}
	var rewards []json.RawMessage
	if json.Unmarshal(*raw, &rewards) != nil || rewards == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phần thưởng nhiệm vụ phải là mảng JSON"})
		return false
	}
	return true
}

func AdminListQuests(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, COALESCE(title,''), COALESCE(description,''), COALESCE(req_desc,''), COALESCE(message,''), COALESCE(rewards,'[]'::jsonb) FROM quests ORDER BY id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []QuestRow{}
	for rows.Next() {
		var q QuestRow
		if err := rows.Scan(&q.ID, &q.Title, &q.Description, &q.ReqDesc, &q.Message, &q.Rewards); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ nhiệm vụ thế giới"})
			return
		}
		list = append(list, q)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách nhiệm vụ thế giới"})
		return
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
	if !validateQuestRewards(c, &req.Rewards) { return }
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO quests (id, title, description, req_desc, message, rewards) VALUES ($1, $2, $3, $4, $5, $6)`,
		req.ID, req.Title, req.Description, req.ReqDesc, req.Message, req.Rewards)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := LoadQuestRegistry(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhiệm vụ nhưng không thể tải lại catalog: " + err.Error()})
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
	if req.ID != "" && req.ID != id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi ID nhiệm vụ"})
		return
	}
	if req.Title == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "Tiêu đề nhiệm vụ là bắt buộc"}); return }
	if !validateQuestRewards(c, &req.Rewards) { return }
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE quests SET title=$1, description=$2, req_desc=$3, message=$4, rewards=$5 WHERE id=$6`,
		req.Title, req.Description, req.ReqDesc, req.Message, req.Rewards, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận nhiệm vụ đã cập nhật"}); return }
	if updated == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhiệm vụ"}); return }
	if err := LoadQuestRegistry(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhiệm vụ nhưng không thể tải lại catalog: " + err.Error()})
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
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM quests WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận nhiệm vụ đã xóa"}); return }
	if deleted == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhiệm vụ"}); return }
	if err := LoadQuestRegistry(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa nhiệm vụ nhưng không thể tải lại catalog: " + err.Error()})
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
	SpatialRegionID string        `json:"spatial_region_id"`
	DurationMinutes int           `json:"duration_minutes"`
}

func AdminListDungeons(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, COALESCE(name,''), COALESCE(type,'normal'), COALESCE(required_rank,1), COALESCE(entry_item,''), COALESCE(description,''), COALESCE(rewards,'{}'::jsonb), COALESCE(map_data,'[]'::jsonb), COALESCE(cooldown_hours,24), COALESCE(spatial_region_id,''), COALESCE(duration_minutes,120) FROM instance_templates ORDER BY required_rank ASC, id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []InstanceTemplateRow{}
	for rows.Next() {
		var r InstanceTemplateRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.RequiredRank, &r.EntryItem, &r.Description, &r.Rewards, &r.MapData, &r.CooldownHours, &r.SpatialRegionID, &r.DurationMinutes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ phụ bản"})
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách phụ bản"}); return }
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
	if req.Type == "" || req.RequiredRank < 1 || req.CooldownHours < 0 || req.DurationMinutes < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại, rank hoặc thời gian hồi phụ bản không hợp lệ"})
		return
	}
	if strings.EqualFold(req.Type,"SPATIAL_REALM") && (req.SpatialRegionID == "" || req.DurationMinutes < 1) { c.JSON(http.StatusBadRequest,gin.H{"error":"Bí cảnh bản đồ cần vùng bản đồ và thời hạn lớn hơn 0"}); return }
	if strings.EqualFold(req.Type,"SPATIAL_REALM") {
		var valid bool
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM regions WHERE id=$1 AND is_active=TRUE AND UPPER(zone_type)='SECRET_REALM')`, req.SpatialRegionID).Scan(&valid); err != nil || !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Region bí cảnh phải tồn tại, đang hoạt động và có zone_type SECRET_REALM"})
			return
		}
	}
	if req.DurationMinutes == 0 { req.DurationMinutes = 120 }
	if len(req.Rewards) == 0 {
		req.Rewards = json.RawMessage(`{}`)
	}
	if len(req.MapData) == 0 { req.MapData = json.RawMessage(`[]`) }
	var rewards map[string]json.RawMessage
	var stages []json.RawMessage
	if json.Unmarshal(req.Rewards, &rewards) != nil || rewards == nil || json.Unmarshal(req.MapData, &stages) != nil || stages == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu bản đồ phụ bản phải là mảng JSON"})
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO instance_templates (id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours, spatial_region_id, duration_minutes) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10,''), $11)`,
		req.ID, req.Name, req.Type, req.RequiredRank, req.EntryItem, req.Description, req.Rewards, req.MapData, req.CooldownHours, req.SpatialRegionID, req.DurationMinutes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := InitInstanceCache(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã tạo phụ bản nhưng không thể tải lại catalog: " + err.Error()}); return }
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
	if req.ID != "" && req.ID != id { c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi ID phụ bản"}); return }
	if req.Name == "" || req.Type == "" || req.RequiredRank < 1 || req.CooldownHours < 0 || req.DurationMinutes < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên, loại, rank hoặc thời gian hồi phụ bản không hợp lệ"})
		return
	}
	if strings.EqualFold(req.Type,"SPATIAL_REALM") && (req.SpatialRegionID == "" || req.DurationMinutes < 1) { c.JSON(http.StatusBadRequest,gin.H{"error":"Bí cảnh bản đồ cần vùng bản đồ và thời hạn lớn hơn 0"}); return }
	if strings.EqualFold(req.Type,"SPATIAL_REALM") {
		var valid bool
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM regions WHERE id=$1 AND is_active=TRUE AND UPPER(zone_type)='SECRET_REALM')`, req.SpatialRegionID).Scan(&valid); err != nil || !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Region bí cảnh phải tồn tại, đang hoạt động và có zone_type SECRET_REALM"})
			return
		}
	}
	if req.DurationMinutes == 0 { req.DurationMinutes = 120 }
	if len(req.Rewards) == 0 { req.Rewards = json.RawMessage(`{}`) }
	if len(req.MapData) == 0 { req.MapData = json.RawMessage(`[]`) }
	var rewards map[string]json.RawMessage
	var stages []json.RawMessage
	if json.Unmarshal(req.Rewards, &rewards) != nil || rewards == nil || json.Unmarshal(req.MapData, &stages) != nil || stages == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu bản đồ phụ bản phải là mảng JSON"})
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE instance_templates SET name=$1, type=$2, required_rank=$3, entry_item=$4, description=$5, rewards=$6, map_data=$7, cooldown_hours=$8, spatial_region_id=NULLIF($9,''), duration_minutes=$10 WHERE id=$11`,
		req.Name, req.Type, req.RequiredRank, req.EntryItem, req.Description, req.Rewards, req.MapData, req.CooldownHours, req.SpatialRegionID, req.DurationMinutes, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận phụ bản đã cập nhật"}); return }
	if updated == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phụ bản"}); return }
	if err := InitInstanceCache(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu phụ bản nhưng không thể tải lại catalog: " + err.Error()}); return }
	auditlog.Logf(db.DB, &adminID, "admin.update_dungeon", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật phụ bản"})
}

func AdminDeleteDungeon(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	id := c.Param("id")
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM instance_templates WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận phụ bản đã xóa"}); return }
	if deleted == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phụ bản"}); return }
	if err := InitInstanceCache(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa phụ bản nhưng không thể tải lại catalog: " + err.Error()}); return }
	auditlog.Logf(db.DB, &adminID, "admin.delete_dungeon", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa phụ bản"})
}

// --- Sect Mines Handlers ---

type SectMineRow struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Rarity           string  `json:"rarity"`
	OccupierSectID   string  `json:"occupier_sect_id"`
	OccupierSectName string  `json:"occupier_sect_name"`
	ShieldUntil      *string `json:"shield_until"`
	PendingFunds     int64   `json:"pending_funds"`
	PendingMaterials string  `json:"pending_materials"`
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách mỏ tông môn"})
			return
		}
		if shield.Valid {
			str := shield.Time.Format("2006-01-02 15:04:05")
			r.ShieldUntil = &str
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách mỏ tông môn"})
		return
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
		PendingMaterials json.RawMessage `json:"pending_materials"`
		ShieldUntil *string `json:"shield_until"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.PendingFunds < 0 || strings.TrimSpace(req.Rarity) == "" || len(req.Rarity) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phẩm chất hoặc quỹ tích lũy không hợp lệ"})
		return
	}
	var materials map[string]json.RawMessage
	if len(req.PendingMaterials) == 0 || json.Unmarshal(req.PendingMaterials, &materials) != nil || materials == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nguyên liệu tích lũy phải là object JSON"})
		return
	}
	var shieldUntil interface{}
	if req.ShieldUntil != nil && strings.TrimSpace(*req.ShieldUntil) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, *req.ShieldUntil)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thời điểm hết khiên không hợp lệ"})
			return
		}
		shieldUntil = parsed
	}
	var occupier interface{}
	if req.OccupierSectID != "" {
		occupier = req.OccupierSectID
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE sect_mines SET occupier_sect_id=$1, rarity=$2, pending_funds=$3, pending_materials=$4, shield_until=$5 WHERE id=$6`,
		occupier, strings.TrimSpace(req.Rarity), req.PendingFunds, req.PendingMaterials, shieldUntil, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mỏ tông môn"})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_sect_mine", "id=%s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mỏ tông môn"})
}

// --- Aperture Sub-entities (Nodes & Variant Humans) ---

type ApertureNodeRow struct {
	ID                      string  `json:"id"`
	ApertureID              string  `json:"aperture_id"`
	NodeType                string  `json:"node_type"`
	Name                    string  `json:"name"`
	Path                    string  `json:"path"`
	Level                   int     `json:"level"`
	HP                      int     `json:"hp"`
	MaxHP                   int     `json:"max_hp"`
	X                       int     `json:"x"`
	Y                       int     `json:"y"`
	ProductionType          string  `json:"production_type"`
	ProductionAmount        int64   `json:"production_amount"`
	ProductionIntervalHours int     `json:"production_interval_hours"`
	IsDestroyed             bool    `json:"is_destroyed"`
	NodeCategory            string  `json:"node_category"`
	DefensePower            int     `json:"defense_power"`
	AttackRange             int     `json:"attack_range"`
	HPBonus                 int     `json:"hp_bonus"`
	DamageReduction         int     `json:"damage_reduction"`
	BlocksMovement          bool    `json:"blocks_movement"`
	RequiredGuID            string  `json:"required_gu_id"`
	RequiredGuRank          int     `json:"required_gu_rank"`
	ZoneType                string  `json:"zone_type"`
	FoodPath                string  `json:"food_path"`
	ItemTier                string  `json:"item_tier"`
	ProductionKey           string  `json:"production_key"`
	LastCollectedAt         *time.Time `json:"last_collected_at"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func AdminListApertureNodes(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	apertureID := c.Param("id")
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, aperture_id, node_type, name, COALESCE(path,''), level, hp, max_hp, x, y, COALESCE(production_type,''), production_amount, production_interval_hours, is_destroyed, COALESCE(node_category,'RESOURCE'), defense_power, attack_range, hp_bonus, damage_reduction, blocks_movement, COALESCE(required_gu_id,''), COALESCE(required_gu_rank,0), COALESCE(zone_type,''), COALESCE(food_path,''), COALESCE(item_tier,''), COALESCE(production_key,''), last_collected_at, COALESCE(created_at,NOW()), COALESCE(updated_at,NOW())
		FROM aperture_resource_nodes WHERE aperture_id = $1 ORDER BY created_at DESC`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []ApertureNodeRow{}
	for rows.Next() {
		var r ApertureNodeRow
		if err := rows.Scan(&r.ID, &r.ApertureID, &r.NodeType, &r.Name, &r.Path, &r.Level, &r.HP, &r.MaxHP, &r.X, &r.Y, &r.ProductionType, &r.ProductionAmount, &r.ProductionIntervalHours, &r.IsDestroyed, &r.NodeCategory, &r.DefensePower, &r.AttackRange, &r.HPBonus, &r.DamageReduction, &r.BlocksMovement, &r.RequiredGuID, &r.RequiredGuRank, &r.ZoneType, &r.FoodPath, &r.ItemTier, &r.ProductionKey, &r.LastCollectedAt, &r.CreatedAt, &r.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu điểm tài nguyên"})
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"Không thể đọc danh sách điểm tài nguyên"}); return }
	c.JSON(http.StatusOK, list)
}

func AdminCreateApertureNode(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	apertureID := c.Param("id")
	var req ApertureNodeRow
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.NodeType) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên và loại điểm tài nguyên là bắt buộc"})
		return
	}
	if err := validateApertureNode(req); err != "" { c.JSON(http.StatusBadRequest, gin.H{"error":err}); return }
	if req.MaxHP <= 0 {
		req.MaxHP = 100
	}
	if req.HP <= 0 {
		req.HP = req.MaxHP
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO aperture_resource_nodes (aperture_id,node_type,name,path,level,hp,max_hp,x,y,production_type,production_amount,production_interval_hours,is_destroyed,node_category,defense_power,attack_range,hp_bonus,damage_reduction,blocks_movement,required_gu_id,required_gu_rank,zone_type,food_path,item_tier,production_key)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,NULLIF($10,''),$11,$12,$13,$14,$15,$16,$17,$18,$19,NULLIF($20,''),$21,NULLIF($22,''),NULLIF($23,''),NULLIF($24,''),NULLIF($25,''))`,
		apertureID, req.NodeType, req.Name, req.Path, req.Level, req.HP, req.MaxHP, req.X, req.Y, req.ProductionType, req.ProductionAmount, req.ProductionIntervalHours, req.IsDestroyed, req.NodeCategory, req.DefensePower, req.AttackRange, req.HPBonus, req.DamageReduction, req.BlocksMovement, req.RequiredGuID, req.RequiredGuRank, req.ZoneType, req.FoodPath, req.ItemTier, req.ProductionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_aperture_node", "aperture_id=%s, name=%s", apertureID, req.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo điểm tài nguyên tiên khiếu"})
}

func validateApertureNode(r ApertureNodeRow) string {
	if len(r.NodeType)>50 || len(r.Name)>100 || len(r.Path)>50 || len(r.ProductionType)>50 || len(r.NodeCategory)>20 || len(r.RequiredGuID)>50 || len(r.ZoneType)>50 || len(r.FoodPath)>50 || len(r.ItemTier)>10 || len(r.ProductionKey)>100 { return "Một hoặc nhiều trường văn bản vượt giới hạn độ dài" }
	if r.Level<1 || r.MaxHP<1 || r.HP<0 || r.HP>r.MaxHP || r.ProductionAmount<0 || r.ProductionIntervalHours<0 || r.DefensePower<0 || r.AttackRange<0 || r.HPBonus<0 || r.DamageReduction<0 || r.RequiredGuRank<0 { return "Thông số điểm tài nguyên không hợp lệ" }
	return ""
}

func AdminUpdateApertureNode(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	var req ApertureNodeRow
	if err:=c.ShouldBindJSON(&req); err!=nil { c.JSON(http.StatusBadRequest,gin.H{"error":"Dữ liệu điểm tài nguyên không hợp lệ"}); return }
	if strings.TrimSpace(req.Name)=="" || strings.TrimSpace(req.NodeType)=="" { c.JSON(http.StatusBadRequest,gin.H{"error":"Tên và loại điểm tài nguyên là bắt buộc"}); return }
	if message:=validateApertureNode(req); message!="" { c.JSON(http.StatusBadRequest,gin.H{"error":message}); return }
	result,err:=db.DB.ExecContext(c.Request.Context(),`UPDATE aperture_resource_nodes SET node_type=$1,name=$2,path=NULLIF($3,''),level=$4,hp=$5,max_hp=$6,x=$7,y=$8,production_type=NULLIF($9,''),production_amount=$10,production_interval_hours=$11,is_destroyed=$12,node_category=$13,defense_power=$14,attack_range=$15,hp_bonus=$16,damage_reduction=$17,blocks_movement=$18,required_gu_id=NULLIF($19,''),required_gu_rank=$20,zone_type=NULLIF($21,''),food_path=NULLIF($22,''),item_tier=NULLIF($23,''),production_key=NULLIF($24,''),updated_at=NOW() WHERE id=$25 AND aperture_id=$26`,req.NodeType,req.Name,req.Path,req.Level,req.HP,req.MaxHP,req.X,req.Y,req.ProductionType,req.ProductionAmount,req.ProductionIntervalHours,req.IsDestroyed,req.NodeCategory,req.DefensePower,req.AttackRange,req.HPBonus,req.DamageReduction,req.BlocksMovement,req.RequiredGuID,req.RequiredGuRank,req.ZoneType,req.FoodPath,req.ItemTier,req.ProductionKey,c.Param("node_id"),c.Param("id"))
	if err!=nil { c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể cập nhật điểm tài nguyên: "+err.Error()}); return }; if n,_:=result.RowsAffected();n==0 { c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy điểm tài nguyên"}); return }
	auditlog.Logf(db.DB,&adminID,"admin.update_aperture_node","node_id=%s",c.Param("node_id")); c.JSON(http.StatusOK,gin.H{"message":"Đã cập nhật điểm tài nguyên"})
}

func AdminDeleteApertureNode(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	nodeID := c.Param("node_id")
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM aperture_resource_nodes WHERE id=$1 AND aperture_id=$2`, nodeID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n,_:=result.RowsAffected();n==0 { c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy điểm tài nguyên"});return }
	auditlog.Logf(db.DB, &adminID, "admin.delete_aperture_node", "node_id=%s", nodeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa điểm tài nguyên"})
}

type VariantHumanRow struct {
	ID             string  `json:"id"`
	ApertureID     string  `json:"aperture_id"`
	Race           string  `json:"race"`
	Population     int     `json:"population"`
	Happiness      float64 `json:"happiness"`
	Oppression     float64 `json:"oppression"`
	RebellionRate  float64 `json:"rebellion_rate"`
	LaborIntensity string  `json:"labor_intensity"`
	Prosperity float64 `json:"prosperity"`
	GuMasterCount int `json:"gu_master_count"`
	GuImmortalCount int `json:"gu_immortal_count"`
	FoodStored float64 `json:"food_stored"`
	FoodPerCapita float64 `json:"food_per_capita"`
	PopulationGrowthRate float64 `json:"population_growth_rate"`
	MortalityRate float64 `json:"mortality_rate"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func AdminListApertureVariantHumans(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	apertureID := c.Param("id")
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, aperture_id, race, population, happiness, oppression, rebellion_rate, labor_intensity, prosperity, gu_master_count, gu_immortal_count, food_stored, food_per_capita, population_growth_rate, mortality_rate, created_at, updated_at
		FROM aperture_variant_humans WHERE aperture_id = $1 ORDER BY race ASC`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []VariantHumanRow{}
	for rows.Next() {
		var r VariantHumanRow
		if err := rows.Scan(&r.ID, &r.ApertureID, &r.Race, &r.Population, &r.Happiness, &r.Oppression, &r.RebellionRate, &r.LaborIntensity,&r.Prosperity,&r.GuMasterCount,&r.GuImmortalCount,&r.FoodStored,&r.FoodPerCapita,&r.PopulationGrowthRate,&r.MortalityRate,&r.CreatedAt,&r.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc dữ liệu chủng tộc tiên khiếu"});return
		}
		list = append(list, r)
	}
	if err:=rows.Err();err!=nil { c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc danh sách chủng tộc tiên khiếu"});return }
	c.JSON(http.StatusOK, list)
}

func AdminCreateApertureVariantHuman(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	apertureID := c.Param("id")
	var req VariantHumanRow
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Race) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chủng tộc là bắt buộc"})
		return
	}
	if message:=validateVariantHuman(req);message!="" { c.JSON(http.StatusBadRequest,gin.H{"error":message});return }
	_, err := db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO aperture_variant_humans (aperture_id,race,population,happiness,oppression,rebellion_rate,labor_intensity,prosperity,gu_master_count,gu_immortal_count,food_stored,food_per_capita,population_growth_rate,mortality_rate)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		apertureID, req.Race, req.Population, req.Happiness, req.Oppression, req.RebellionRate, req.LaborIntensity,req.Prosperity,req.GuMasterCount,req.GuImmortalCount,req.FoodStored,req.FoodPerCapita,req.PopulationGrowthRate,req.MortalityRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_variant_human", "aperture_id=%s, race=%s", apertureID, req.Race)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo nhóm dị nhân trong tiên khiếu"})
}

func validateVariantHuman(r VariantHumanRow) string {
	if len(strings.TrimSpace(r.Race))==0 || len(r.Race)>50 || len(r.LaborIntensity)>20 { return "Chủng tộc hoặc cường độ lao động không hợp lệ" }
	values:=[]float64{r.Happiness,r.Oppression,r.RebellionRate,r.Prosperity,r.FoodStored,r.FoodPerCapita,r.PopulationGrowthRate,r.MortalityRate}
	for _,v:=range values { if math.IsNaN(v)||math.IsInf(v,0) { return "Thông số dân cư phải là số hữu hạn" } }
	if r.Population<0 || r.Happiness<0 || r.Happiness>100 || r.Oppression<0 || r.Oppression>100 || r.RebellionRate<0 || r.RebellionRate>100 || r.Prosperity<0 || r.Prosperity>100 || r.GuMasterCount<0 || r.GuImmortalCount<0 || r.FoodStored<0 || r.FoodPerCapita<0 || r.PopulationGrowthRate<0 || r.MortalityRate<0 { return "Thông số dân cư ngoài phạm vi cho phép" }
	return ""
}

func AdminUpdateApertureVariantHuman(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};var req VariantHumanRow
	if err:=c.ShouldBindJSON(&req);err!=nil { c.JSON(http.StatusBadRequest,gin.H{"error":"Dữ liệu dân cư không hợp lệ"});return }
	if message:=validateVariantHuman(req);message!="" { c.JSON(http.StatusBadRequest,gin.H{"error":message});return }
	var assignedPopulation int
	if err:=db.DB.QueryRowContext(c.Request.Context(),`SELECT COALESCE(SUM(assigned_population),0) FROM aperture_node_workers WHERE variant_human_id=$1`,c.Param("vh_id")).Scan(&assignedPopulation);err!=nil { c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể kiểm tra lao động đã phân công"});return }
	if assignedPopulation>req.Population { c.JSON(http.StatusBadRequest,gin.H{"error":"Dân số mới thấp hơn số lao động đang phân công"});return }
	result,err:=db.DB.ExecContext(c.Request.Context(),`UPDATE aperture_variant_humans SET race=$1,population=$2,happiness=$3,oppression=$4,rebellion_rate=$5,labor_intensity=$6,prosperity=$7,gu_master_count=$8,gu_immortal_count=$9,food_stored=$10,food_per_capita=$11,population_growth_rate=$12,mortality_rate=$13,updated_at=NOW() WHERE id=$14 AND aperture_id=$15`,strings.TrimSpace(req.Race),req.Population,req.Happiness,req.Oppression,req.RebellionRate,req.LaborIntensity,req.Prosperity,req.GuMasterCount,req.GuImmortalCount,req.FoodStored,req.FoodPerCapita,req.PopulationGrowthRate,req.MortalityRate,c.Param("vh_id"),c.Param("id"))
	if err!=nil { c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể cập nhật dữ liệu dân cư: "+err.Error()});return };if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy nhóm dân cư"});return}
	auditlog.Logf(db.DB,&adminID,"admin.update_variant_human","id=%s",c.Param("vh_id"));c.JSON(http.StatusOK,gin.H{"message":"Đã cập nhật nhóm dân cư"})
}

func AdminDeleteApertureVariantHuman(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	vhID := c.Param("vh_id")
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM aperture_variant_humans WHERE id=$1 AND aperture_id=$2`, vhID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n,_:=result.RowsAffected();n==0 { c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy nhóm dân cư"});return }
	auditlog.Logf(db.DB, &adminID, "admin.delete_variant_human", "id=%s", vhID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa nhóm dị nhân"})
}

type AdminApertureWorker struct {
	ID string `json:"id"`
	NodeID string `json:"node_id"`
	VariantHumanID string `json:"variant_human_id"`
	Race string `json:"race"`
	Population int `json:"population"`
	AssignedPopulation int `json:"assigned_population"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func AdminListApertureNodeWorkers(c *gin.Context) {
	if _,ok:=requireAdmin(c);!ok{return}
	rows,err:=db.DB.QueryContext(c.Request.Context(),`SELECT w.id,w.node_id,w.variant_human_id,v.race,v.population,w.assigned_population,w.created_at,w.updated_at FROM aperture_node_workers w JOIN aperture_resource_nodes n ON n.id=w.node_id JOIN aperture_variant_humans v ON v.id=w.variant_human_id WHERE n.id=$1 AND n.aperture_id=$2 ORDER BY v.race`,c.Param("node_id"),c.Param("id"))
	if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tải phân công lao động"});return};defer rows.Close();list:=make([]AdminApertureWorker,0)
	for rows.Next(){var w AdminApertureWorker;if err:=rows.Scan(&w.ID,&w.NodeID,&w.VariantHumanID,&w.Race,&w.Population,&w.AssignedPopulation,&w.CreatedAt,&w.UpdatedAt);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc phân công lao động"});return};list=append(list,w)}
	if err:=rows.Err();err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể đọc phân công lao động"});return};c.JSON(http.StatusOK,list)
}

func AdminUpsertApertureNodeWorker(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};var req struct{VariantHumanID string `json:"variant_human_id"`;AssignedPopulation int `json:"assigned_population"`}
	if err:=c.ShouldBindJSON(&req);err!=nil||strings.TrimSpace(req.VariantHumanID)==""||req.AssignedPopulation<0{c.JSON(http.StatusBadRequest,gin.H{"error":"Nhóm dân cư và số lao động hợp lệ là bắt buộc"});return}
	tx,err:=db.DB.BeginTx(c.Request.Context(),nil);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể bắt đầu cập nhật phân công"});return};defer tx.Rollback()
	var population int;if err:=tx.QueryRowContext(c.Request.Context(),`SELECT v.population FROM aperture_variant_humans v JOIN aperture_resource_nodes n ON n.aperture_id=v.aperture_id WHERE v.id=$1 AND n.id=$2 AND n.aperture_id=$3 FOR UPDATE OF v`,req.VariantHumanID,c.Param("node_id"),c.Param("id")).Scan(&population);err!=nil{if err==sql.ErrNoRows{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy nhóm dân cư phù hợp với tiên khiếu và node"})}else{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xác thực dân số"})};return}
	var elsewhere int;if err:=tx.QueryRowContext(c.Request.Context(),`SELECT COALESCE(SUM(assigned_population),0) FROM aperture_node_workers WHERE variant_human_id=$1 AND node_id<>$2`,req.VariantHumanID,c.Param("node_id")).Scan(&elsewhere);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể tính lao động đã phân công"});return};if elsewhere+req.AssignedPopulation>population{c.JSON(http.StatusBadRequest,gin.H{"error":"Số lượng lao động vượt quá dân số chưa được phân công"});return}
	_,err=tx.ExecContext(c.Request.Context(),`INSERT INTO aperture_node_workers(node_id,variant_human_id,assigned_population) VALUES($1,$2,$3) ON CONFLICT(node_id,variant_human_id) DO UPDATE SET assigned_population=EXCLUDED.assigned_population,updated_at=NOW()`,c.Param("node_id"),req.VariantHumanID,req.AssignedPopulation);if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể lưu phân công lao động: "+err.Error()});return};if err:=tx.Commit();err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xác nhận phân công đã lưu"});return};auditlog.Logf(db.DB,&adminID,"admin.upsert_aperture_node_worker","node=%s variant_human=%s population=%d",c.Param("node_id"),req.VariantHumanID,req.AssignedPopulation);c.JSON(http.StatusOK,gin.H{"message":"Đã lưu phân công lao động"})
}

func AdminDeleteApertureNodeWorker(c *gin.Context) {
	adminID,ok:=requireAdmin(c);if !ok{return};result,err:=db.DB.ExecContext(c.Request.Context(),`DELETE FROM aperture_node_workers w USING aperture_resource_nodes n WHERE w.id=$1 AND w.node_id=n.id AND w.node_id=$2 AND n.aperture_id=$3`,c.Param("worker_id"),c.Param("node_id"),c.Param("id"));if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Không thể xóa phân công lao động"});return};if n,_:=result.RowsAffected();n==0{c.JSON(http.StatusNotFound,gin.H{"error":"Không tìm thấy phân công lao động"});return};auditlog.Logf(db.DB,&adminID,"admin.delete_aperture_node_worker","worker=%s node=%s",c.Param("worker_id"),c.Param("node_id"));c.JSON(http.StatusOK,gin.H{"message":"Đã gỡ phân công lao động"})
}
