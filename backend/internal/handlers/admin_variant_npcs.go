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
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminVariantNPCInput struct {
	TemplateSource  string          `json:"template_source"`
	TemplateID      string          `json:"template_id"`
	Species         string          `json:"species"`
	Race            string          `json:"race"`
	Name            string          `json:"name"`
	Rank            int             `json:"rank"`
	SubRank         int             `json:"sub_rank"`
	Experience      int64           `json:"experience"`
	Talent          float64         `json:"talent"`
	Loyalty         float64         `json:"loyalty"`
	HP              float64         `json:"hp"`
	MaxHP           float64         `json:"max_hp"`
	BaseAttack      float64         `json:"base_attack"`
	BaseDefense     float64         `json:"base_defense"`
	Speed           float64         `json:"speed"`
	Count           int             `json:"count"`
	Hunger          float64         `json:"hunger"`
	MaxHunger       float64         `json:"max_hunger"`
	UpkeepRate      float64         `json:"upkeep_rate"`
	HungerUpdatedAt *time.Time      `json:"hunger_updated_at"`
	LifecycleState  string          `json:"lifecycle_state"`
	ExecutionState  string          `json:"execution_state"`
	BehaviorProfile string          `json:"behavior_profile"`
	AIProfile       string          `json:"ai_profile"`
	Capabilities    json.RawMessage `json:"capabilities"`
	Traits          json.RawMessage `json:"traits"`
	Progression     json.RawMessage `json:"progression"`
	FormationSlot   int             `json:"formation_slot"`
	Path            string          `json:"path"`
	Role            string          `json:"role"`
	EquippedGu      []struct {
		PlayerGuID string `json:"player_gu_id"`
		Slot       int    `json:"slot"`
	} `json:"equipped_gu"`
}

func AdminListVariantNPCs(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT to_jsonb(n) || jsonb_build_object('owner_username',COALESCE(p.username,''),'equipped_gu',COALESCE((SELECT jsonb_agg(jsonb_build_object('player_gu_id',pg.id,'gu_id',pg.gu_id,'rank',pg.gu_rank,'slot',e.slot,'state',pg.state) ORDER BY e.slot) FROM actor_gu_equipment e JOIN player_gu pg ON pg.id=e.player_gu_id WHERE e.actor_id=n.id),'[]'::jsonb)) FROM non_player_actors n LEFT JOIN players p ON p.id=n.owner_player_id WHERE n.aperture_id=$1 AND n.actor_kind='VARIANT_NPC' ORDER BY n.lifecycle_state,n.race,n.rank DESC,n.name`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách cổ sư dị nhân"})
		return
	}
	defer rows.Close()
	list := make([]json.RawMessage, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu cổ sư dị nhân"})
			return
		}
		list = append(list, json.RawMessage(raw))
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách cổ sư dị nhân"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func validateAdminVariantNPC(c *gin.Context, r *AdminVariantNPCInput) bool {
	r.TemplateSource = strings.ToUpper(strings.TrimSpace(r.TemplateSource))
	r.TemplateID = strings.TrimSpace(r.TemplateID)
	r.Species = strings.TrimSpace(r.Species)
	r.Race = strings.TrimSpace(r.Race)
	r.Name = strings.TrimSpace(r.Name)
	r.LifecycleState = strings.ToUpper(strings.TrimSpace(r.LifecycleState))
	r.ExecutionState = strings.ToUpper(strings.TrimSpace(r.ExecutionState))
	r.BehaviorProfile = strings.TrimSpace(r.BehaviorProfile)
	r.AIProfile = strings.TrimSpace(r.AIProfile)
	r.Path = strings.TrimSpace(r.Path)
	r.Role = strings.ToUpper(strings.TrimSpace(r.Role))
	if r.TemplateSource != "VARIANT_RACE" && r.TemplateSource != "NPC_TEMPLATE" && r.TemplateSource != "CUSTOM" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nguồn mẫu cổ sư không hợp lệ"})
		return false
	}
	if r.TemplateID == "" || r.Race == "" || r.Name == "" || len(r.TemplateID) > 100 || len(r.Species) > 100 || len(r.Race) > 50 || len(r.Name) > 100 || len(r.Path) > 50 || len(r.Role) > 50 || len(r.BehaviorProfile) > 50 || len(r.AIProfile) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mẫu, chủng tộc, tên hoặc các trường văn bản không hợp lệ"})
		return false
	}
	if r.Rank < 1 || r.Rank > 9 || r.SubRank < 0 || r.Experience < 0 || r.Count < 1 || r.FormationSlot < 0 || r.FormationSlot > 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cấp bậc, kinh nghiệm, số lượng hoặc vị trí đội hình không hợp lệ"})
		return false
	}
	if r.LifecycleState != "ACTIVE" && r.FormationSlot != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ sư không hoạt động phải rời vị trí đội hình"})
		return false
	}
	if len(r.EquippedGu) > r.Rank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số Cổ trang bị vượt quá số ô của cấp bậc"})
		return false
	}
	seenGu := map[string]bool{}
	seenSlots := map[int]bool{}
	for _, g := range r.EquippedGu {
		if strings.TrimSpace(g.PlayerGuID) == "" || g.Slot < 1 || g.Slot > r.Rank || seenGu[g.PlayerGuID] || seenSlots[g.Slot] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách Cổ trang bị có ID hoặc ô trùng/không hợp lệ"})
			return false
		}
		seenGu[g.PlayerGuID] = true
		seenSlots[g.Slot] = true
	}
	for _, v := range []float64{r.Talent, r.Loyalty, r.HP, r.MaxHP, r.BaseAttack, r.BaseDefense, r.Speed, r.Hunger, r.MaxHunger, r.UpkeepRate} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thông số cổ sư phải là số hữu hạn"})
			return false
		}
	}
	if r.Talent < 0 || r.Talent > 100 || r.Loyalty < 0 || r.Loyalty > 100 || r.MaxHP <= 0 || r.HP < 0 || r.HP > r.MaxHP || r.BaseAttack < 0 || r.BaseDefense < 0 || r.Speed < 0 || r.MaxHunger <= 0 || r.Hunger < 0 || r.Hunger > r.MaxHunger || r.UpkeepRate < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sinh lực, thiên phú, trung thành hoặc chỉ số chiến đấu ngoài giới hạn"})
		return false
	}
	switch r.LifecycleState {
	case "ACTIVE", "DEAD", "DISMISSED":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vòng đời cổ sư không hợp lệ"})
		return false
	}
	switch r.ExecutionState {
	case "STORED", "DEPLOYING", "TRAINING", "WORKING", "RECOVERING", "DEAD", "DISMISSED", "WORLD_IDLE", "MOVING", "FOLLOWING", "GUARDING", "PATROLLING", "GATHERING", "COMBAT", "RETURNING", "INJURED":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái hoạt động cổ sư không hợp lệ"})
		return false
	}
	if (r.LifecycleState == "ACTIVE" && (r.ExecutionState == "DEAD" || r.ExecutionState == "DISMISSED")) || (r.LifecycleState == "DEAD" && r.ExecutionState != "DEAD") || (r.LifecycleState == "DISMISSED" && r.ExecutionState != "DISMISSED") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vòng đời và trạng thái hoạt động không đồng nhất"})
		return false
	}
	if len(r.Capabilities) == 0 {
		r.Capabilities = json.RawMessage(`{}`)
	}
	if len(r.Traits) == 0 {
		r.Traits = json.RawMessage(`{}`)
	}
	if len(r.Progression) == 0 {
		r.Progression = json.RawMessage(`{}`)
	}
	for _, raw := range []json.RawMessage{r.Capabilities, r.Traits, r.Progression} {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil || obj == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Capabilities, traits và progression phải là JSON object"})
			return false
		}
	}
	var raceExists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM aperture_variant_humans WHERE aperture_id=$1 AND race=$2)`, c.Param("id"), r.Race).Scan(&raceExists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh chủng tộc cổ sư"})
		return false
	}
	if !raceExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chủng tộc cổ sư chưa có trong tiên khiếu"})
		return false
	}
	return true
}

func replaceAdminVariantNPCGu(c *gin.Context, tx *sql.Tx, actorID, ownerID string, actorRank int, requested []struct {
	PlayerGuID string `json:"player_gu_id"`
	Slot       int    `json:"slot"`
}) error {
	if err := releaseActorGu(tx, actorID); err != nil {
		return err
	}
	for _, g := range requested {
		var guRank int
		err := tx.QueryRowContext(c.Request.Context(), `SELECT gu_rank FROM player_gu WHERE id=$1 AND player_id=$2 AND location='INVENTORY' AND state='ACTIVE' AND hunger>0 AND NOT equipped AND NOT is_vital AND NOT is_locked FOR UPDATE`, g.PlayerGuID, ownerID).Scan(&guRank)
		if err != nil {
			return fmt.Errorf("Cổ %s không thuộc kho của chủ nhân hoặc không khả dụng", g.PlayerGuID)
		}
		if guRank > actorRank {
			return fmt.Errorf("Cổ %s có cấp cao hơn cổ sư", g.PlayerGuID)
		}
		if _, err = tx.ExecContext(c.Request.Context(), `UPDATE player_gu SET location='ACTOR',equipped=false,is_vital=false,updated_at=NOW() WHERE id=$1`, g.PlayerGuID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(c.Request.Context(), `INSERT INTO actor_gu_equipment(actor_id,player_gu_id,slot) VALUES($1,$2,$3)`, actorID, g.PlayerGuID, g.Slot); err != nil {
			return err
		}
	}
	return nil
}

func AdminCreateVariantNPC(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var r AdminVariantNPCInput
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu cổ sư dị nhân không hợp lệ"})
		return
	}
	if r.LifecycleState == "" {
		r.LifecycleState = "ACTIVE"
	}
	if r.ExecutionState == "" {
		r.ExecutionState = "STORED"
	}
	if r.BehaviorProfile == "" {
		r.BehaviorProfile = "WORKER"
	}
	if r.HungerUpdatedAt == nil {
		now := time.Now()
		r.HungerUpdatedAt = &now
	}
	if !validateAdminVariantNPC(c, &r) {
		return
	}
	if len(r.EquippedGu) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hãy tạo cổ sư trước, sau đó giao Cổ vào trang bị"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu tạo cổ sư"})
		return
	}
	defer tx.Rollback()
	var ownerID string
	var state string
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT player_id,state FROM immortal_apertures WHERE id=$1 FOR UPDATE`, c.Param("id")).Scan(&ownerID, &state); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tiên khiếu"})
		return
	}
	if state == "DESTROYED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thêm cổ sư vào tiên khiếu đã hủy"})
		return
	}
	_, err = tx.ExecContext(c.Request.Context(), `INSERT INTO non_player_actors(actor_kind,owner_player_id,aperture_id,template_source,template_id,species,race,name,rank,sub_rank,experience,talent,loyalty,hp,max_hp,base_attack,base_defense,speed,count,hunger,max_hunger,upkeep_rate,hunger_updated_at,lifecycle_state,execution_state,behavior_profile,ai_profile,capabilities,traits,progression,formation_slot,path,role) VALUES('VARIANT_NPC',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32)`, ownerID, c.Param("id"), r.TemplateSource, r.TemplateID, r.Species, r.Race, r.Name, r.Rank, r.SubRank, r.Experience, r.Talent, r.Loyalty, r.HP, r.MaxHP, r.BaseAttack, r.BaseDefense, r.Speed, r.Count, r.Hunger, r.MaxHunger, r.UpkeepRate, r.HungerUpdatedAt, r.LifecycleState, r.ExecutionState, r.BehaviorProfile, r.AIProfile, r.Capabilities, r.Traits, r.Progression, r.FormationSlot, r.Path, r.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo cổ sư dị nhân: " + err.Error()})
		return
	}
	if r.LifecycleState == "ACTIVE" {
		if _, err = tx.ExecContext(c.Request.Context(), `UPDATE aperture_variant_humans SET gu_master_count=gu_master_count+1,updated_at=NOW() WHERE aperture_id=$1 AND race=$2`, c.Param("id"), r.Race); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bộ đếm cổ sư"})
			return
		}
	}
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận cổ sư đã tạo"})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_variant_npc", "aperture=%s race=%s name=%s", c.Param("id"), r.Race, r.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo cổ sư dị nhân"})
}

func AdminUpdateVariantNPC(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var r AdminVariantNPCInput
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu cổ sư dị nhân không hợp lệ"})
		return
	}
	if !validateAdminVariantNPC(c, &r) {
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu cập nhật cổ sư"})
		return
	}
	defer tx.Rollback()
	var apertureState string
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT state FROM immortal_apertures WHERE id=$1 FOR UPDATE`, c.Param("id")).Scan(&apertureState); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tiên khiếu"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa tiên khiếu"})
		}
		return
	}
	if apertureState == "DESTROYED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể quản lý cổ sư trong tiên khiếu đã hủy"})
		return
	}
	var oldRace, oldLifecycle, ownerID string
	var oldRank int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT race,lifecycle_state,owner_player_id,rank FROM non_player_actors WHERE id=$1 AND aperture_id=$2 AND actor_kind='VARIANT_NPC' FOR UPDATE`, c.Param("npc_id"), c.Param("id")).Scan(&oldRace, &oldLifecycle, &ownerID, &oldRank); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ sư dị nhân"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa cổ sư dị nhân"})
		}
		return
	}
	_, err = tx.ExecContext(c.Request.Context(), `UPDATE non_player_actors SET template_source=$1,template_id=$2,species=$3,race=$4,name=$5,rank=$6,sub_rank=$7,experience=$8,talent=$9,loyalty=$10,hp=$11,max_hp=$12,base_attack=$13,base_defense=$14,speed=$15,count=$16,hunger=$17,max_hunger=$18,upkeep_rate=$19,hunger_updated_at=$20,lifecycle_state=$21,execution_state=$22,behavior_profile=$23,ai_profile=$24,capabilities=$25,traits=$26,progression=$27,formation_slot=$28,path=$29,role=$30,updated_at=NOW(),version=version+1 WHERE id=$31 AND aperture_id=$32 AND actor_kind='VARIANT_NPC'`, r.TemplateSource, r.TemplateID, r.Species, r.Race, r.Name, r.Rank, r.SubRank, r.Experience, r.Talent, r.Loyalty, r.HP, r.MaxHP, r.BaseAttack, r.BaseDefense, r.Speed, r.Count, r.Hunger, r.MaxHunger, r.UpkeepRate, r.HungerUpdatedAt, r.LifecycleState, r.ExecutionState, r.BehaviorProfile, r.AIProfile, r.Capabilities, r.Traits, r.Progression, r.FormationSlot, r.Path, r.Role, c.Param("npc_id"), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ sư dị nhân: " + err.Error()})
		return
	}
	if r.LifecycleState != "ACTIVE" {
		if err := releaseActorGu(tx, c.Param("npc_id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tháo cổ trùng đang trang bị"})
			return
		}
	}
	if r.LifecycleState == "ACTIVE" && r.EquippedGu != nil {
		if err := replaceAdminVariantNPCGu(c, tx, c.Param("npc_id"), ownerID, r.Rank, r.EquippedGu); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
	} else if oldLifecycle == "ACTIVE" && r.LifecycleState == "ACTIVE" && r.Rank < oldRank {
		var invalid bool
		if err := tx.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM actor_gu_equipment WHERE actor_id=$1 AND slot>$2)`, c.Param("npc_id"), r.Rank).Scan(&invalid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra ô Cổ đang dùng"})
			return
		}
		if invalid {
			c.JSON(http.StatusConflict, gin.H{"error": "Hãy tháo Cổ ở các ô vượt quá cấp bậc mới trước khi giảm cấp"})
			return
		}
	}
	if oldRace != r.Race {
		if oldLifecycle == "ACTIVE" {
			_, err = tx.ExecContext(c.Request.Context(), `UPDATE aperture_variant_humans SET gu_master_count=GREATEST(0,gu_master_count-1),updated_at=NOW() WHERE aperture_id=$1 AND race=$2`, c.Param("id"), oldRace)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bộ đếm chủng tộc cũ"})
				return
			}
		}
		if r.LifecycleState == "ACTIVE" {
			_, err = tx.ExecContext(c.Request.Context(), `UPDATE aperture_variant_humans SET gu_master_count=gu_master_count+1,updated_at=NOW() WHERE aperture_id=$1 AND race=$2`, c.Param("id"), r.Race)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bộ đếm chủng tộc mới"})
				return
			}
		}
	} else if oldLifecycle != "ACTIVE" && r.LifecycleState == "ACTIVE" {
		_, err = tx.ExecContext(c.Request.Context(), `UPDATE aperture_variant_humans SET gu_master_count=gu_master_count+1,updated_at=NOW() WHERE aperture_id=$1 AND race=$2`, c.Param("id"), r.Race)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bộ đếm cổ sư"})
			return
		}
	} else if oldLifecycle == "ACTIVE" && r.LifecycleState != "ACTIVE" {
		_, err = tx.ExecContext(c.Request.Context(), `UPDATE aperture_variant_humans SET gu_master_count=GREATEST(0,gu_master_count-1),updated_at=NOW() WHERE aperture_id=$1 AND race=$2`, c.Param("id"), oldRace)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bộ đếm cổ sư"})
			return
		}
	}
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận cổ sư đã cập nhật"})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_variant_npc", "npc=%s", c.Param("npc_id"))
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật cổ sư dị nhân"})
}

func AdminDismissVariantNPC(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu giải tán cổ sư"})
		return
	}
	defer tx.Rollback()
	var race string
	if err := tx.QueryRowContext(c.Request.Context(), `UPDATE non_player_actors SET lifecycle_state='DISMISSED',execution_state='DISMISSED',formation_slot=0,updated_at=NOW(),version=version+1 WHERE id=$1 AND aperture_id=$2 AND actor_kind='VARIANT_NPC' AND lifecycle_state='ACTIVE' RETURNING race`, c.Param("npc_id"), c.Param("id")).Scan(&race); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ sư đang hoạt động"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể giải tán cổ sư"})
		}
		return
	}
	if err := releaseActorGu(tx, c.Param("npc_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tháo cổ trùng đang trang bị"})
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `UPDATE aperture_variant_humans SET gu_master_count=GREATEST(0,gu_master_count-1),updated_at=NOW() WHERE aperture_id=$1 AND race=$2`, c.Param("id"), race); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đồng bộ bộ đếm cổ sư"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận giải tán cổ sư"})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.dismiss_variant_npc", "npc=%s", c.Param("npc_id"))
	c.JSON(http.StatusOK, gin.H{"message": "Đã giải tán cổ sư dị nhân"})
}
