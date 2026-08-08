package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// AdminListApertures lists all players' immortal apertures with pagination and search
func AdminListApertures(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if search != "" {
		like := "%" + search + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(p.username ILIKE $%d OR p.email ILIKE $%d OR ia.id::text ILIKE $%d OR p.id::text ILIKE $%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, like, like, like, like)
		argIdx += 4
	}

	where := ""
	if len(whereClauses) > 0 {
		where = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM immortal_apertures ia JOIN players p ON ia.player_id = p.id` + where
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số lượng tiên khiếu: " + err.Error()})
		return
	}

	query := `
		SELECT 
			ia.id, ia.player_id, p.username, ia.grade, ia.time_rate, 
			ia.aperture_hp, ia.max_aperture_hp, ia.area_size, ia.immortal_essence, 
			ia.essence_production_per_year, ia.internal_years_passed, 
			ia.next_tribulation_year, ia.tribulation_count, ia.state, 
			ia.created_at, ia.updated_at
		FROM immortal_apertures ia
		JOIN players p ON ia.player_id = p.id
	` + where + ` ORDER BY ia.created_at DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)

	dataArgs := append(append([]interface{}{}, args...), limit, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách tiên khiếu: " + err.Error()})
		return
	}
	defer rows.Close()

	apertures := []gin.H{}
	for rows.Next() {
		var id, playerID, username, grade, state string
		var timeRate, nextTribulationYear, internalYearsPassed float64
		var apertureHP, maxApertureHP, immortalEssence, areaSize int64
		var essenceProductionPerYear, tribulationCount int
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &playerID, &username, &grade, &timeRate,
			&apertureHP, &maxApertureHP, &areaSize, &immortalEssence,
			&essenceProductionPerYear, &internalYearsPassed,
			&nextTribulationYear, &tribulationCount, &state,
			&createdAt, &updatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu tiên khiếu: " + err.Error()})
			return
		}

		apertures = append(apertures, gin.H{
			"id":                          id,
			"player_id":                   playerID,
			"username":                    username,
			"grade":                       grade,
			"time_rate":                   timeRate,
			"aperture_hp":                 apertureHP,
			"max_aperture_hp":             maxApertureHP,
			"area_size":                   areaSize,
			"immortal_essence":            immortalEssence,
			"essence_production_per_year": essenceProductionPerYear,
			"internal_years_passed":       internalYearsPassed,
			"next_tribulation_year":       nextTribulationYear,
			"tribulation_count":           tribulationCount,
			"state":                       state,
			"created_at":                  createdAt,
			"updated_at":                  updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  apertures,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// AdminGetApertureDetail retrieves the details of a single immortal aperture
func AdminGetApertureDetail(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	apertureID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tiên khiếu không hợp lệ"})
		return
	}

	var ap models.ImmortalAperture
	var username string
	var daoJSON []byte

	query := `
		SELECT 
			ia.id, ia.player_id, p.username, ia.grade, ia.time_rate, 
			ia.aperture_hp, ia.max_aperture_hp, ia.area_size, ia.immortal_essence, 
			ia.essence_production_per_year, ia.internal_years_passed, 
			ia.next_tribulation_year, ia.tribulation_count, ia.rank_start_year,
			ia.earth_calamity_count, ia.heavenly_tribulation_count, 
			ia.grand_tribulation_count, ia.myriad_tribulation_count, 
			ia.chaos_calamity_count, ia.state, ia.dao_marks,
			ia.created_at, ia.updated_at
		FROM immortal_apertures ia
		JOIN players p ON ia.player_id = p.id
		WHERE ia.id = $1
	`

	var hp, maxHp, immortalEssence int64
	err = db.DB.QueryRowContext(c.Request.Context(), query, apertureID).Scan(
		&ap.ID, &ap.PlayerID, &username, &ap.Grade, &ap.TimeRate,
		&hp, &maxHp, &ap.AreaSize, &immortalEssence,
		&ap.EssenceProductionPerYear, &ap.InternalYearsPassed,
		&ap.NextTribulationYear, &ap.TribulationCount, &ap.RankStartYear,
		&ap.EarthCalamityCount, &ap.HeavenlyTribulationCount,
		&ap.GrandTribulationCount, &ap.MyriadTribulationCount,
		&ap.ChaosCalamityCount, &ap.State, &daoJSON,
		&ap.CreatedAt, &ap.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tiên khiếu"})
		return
	}

	ap.ApertureHP = int(hp)
	ap.MaxApertureHP = int(maxHp)
	ap.ImmortalEssence = int(immortalEssence)

	var daoMarks models.DaoMarksMap
	if len(daoJSON) > 0 {
		if err := json.Unmarshal(daoJSON, &daoMarks); err != nil {
			daoMarks = make(models.DaoMarksMap)
		}
	} else {
		daoMarks = make(models.DaoMarksMap)
	}
	ap.DaoMarks = daoMarks

	c.JSON(http.StatusOK, gin.H{
		"aperture": ap,
		"username": username,
	})
}

// AdminUpdateApertureRequest defines fields that can be updated on an immortal aperture
type AdminUpdateApertureRequest struct {
	Grade                    *string             `json:"grade"`
	TimeRate                 *float64            `json:"time_rate"`
	ApertureHP               *int64              `json:"aperture_hp"`
	MaxApertureHP            *int64              `json:"max_aperture_hp"`
	AreaSize                 *int64              `json:"area_size"`
	ImmortalEssence          *int64              `json:"immortal_essence"`
	EssenceProductionPerYear *int                `json:"essence_production_per_year"`
	InternalYearsPassed      *float64            `json:"internal_years_passed"`
	NextTribulationYear      *float64            `json:"next_tribulation_year"`
	TribulationCount         *int                `json:"tribulation_count"`
	RankStartYear            *float64            `json:"rank_start_year"`
	EarthCalamityCount       *int                `json:"earth_calamity_count"`
	HeavenlyTribulationCount *int                `json:"heavenly_tribulation_count"`
	GrandTribulationCount    *int                `json:"grand_tribulation_count"`
	MyriadTribulationCount   *int                `json:"myriad_tribulation_count"`
	ChaosCalamityCount       *int                `json:"chaos_calamity_count"`
	State                    *string             `json:"state"`
	DaoMarks                 *models.DaoMarksMap `json:"dao_marks"`
}

// AdminUpdateAperture handles updating player's immortal aperture properties
func AdminUpdateAperture(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	apertureID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tiên khiếu không hợp lệ"})
		return
	}

	var req AdminUpdateApertureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	// Verify aperture exists and load player ID & old dao marks
	var playerID string
	var oldDaoJSON []byte
	err = tx.QueryRow(`SELECT player_id, dao_marks FROM immortal_apertures WHERE id = $1`, apertureID).Scan(&playerID, &oldDaoJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tiên khiếu"})
		return
	}

	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Grade != nil {
		sets = append(sets, "grade=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Grade)
		argIdx++
	}
	if req.TimeRate != nil {
		sets = append(sets, "time_rate=$"+strconv.Itoa(argIdx))
		args = append(args, *req.TimeRate)
		argIdx++
	}
	if req.ApertureHP != nil {
		sets = append(sets, "aperture_hp=$"+strconv.Itoa(argIdx))
		args = append(args, *req.ApertureHP)
		argIdx++
	}
	if req.MaxApertureHP != nil {
		sets = append(sets, "max_aperture_hp=$"+strconv.Itoa(argIdx))
		args = append(args, *req.MaxApertureHP)
		argIdx++
	}
	if req.AreaSize != nil {
		sets = append(sets, "area_size=$"+strconv.Itoa(argIdx))
		args = append(args, *req.AreaSize)
		argIdx++
	}
	if req.ImmortalEssence != nil {
		sets = append(sets, "immortal_essence=$"+strconv.Itoa(argIdx))
		args = append(args, *req.ImmortalEssence)
		argIdx++
	}
	if req.EssenceProductionPerYear != nil {
		sets = append(sets, "essence_production_per_year=$"+strconv.Itoa(argIdx))
		args = append(args, *req.EssenceProductionPerYear)
		argIdx++
	}
	if req.InternalYearsPassed != nil {
		sets = append(sets, "internal_years_passed=$"+strconv.Itoa(argIdx))
		args = append(args, *req.InternalYearsPassed)
		argIdx++
	}
	if req.NextTribulationYear != nil {
		sets = append(sets, "next_tribulation_year=$"+strconv.Itoa(argIdx))
		args = append(args, *req.NextTribulationYear)
		argIdx++
	}
	if req.TribulationCount != nil {
		sets = append(sets, "tribulation_count=$"+strconv.Itoa(argIdx))
		args = append(args, *req.TribulationCount)
		argIdx++
	}
	if req.RankStartYear != nil {
		sets = append(sets, "rank_start_year=$"+strconv.Itoa(argIdx))
		args = append(args, *req.RankStartYear)
		argIdx++
	}
	if req.EarthCalamityCount != nil {
		sets = append(sets, "earth_calamity_count=$"+strconv.Itoa(argIdx))
		args = append(args, *req.EarthCalamityCount)
		argIdx++
	}
	if req.HeavenlyTribulationCount != nil {
		sets = append(sets, "heavenly_tribulation_count=$"+strconv.Itoa(argIdx))
		args = append(args, *req.HeavenlyTribulationCount)
		argIdx++
	}
	if req.GrandTribulationCount != nil {
		sets = append(sets, "grand_tribulation_count=$"+strconv.Itoa(argIdx))
		args = append(args, *req.GrandTribulationCount)
		argIdx++
	}
	if req.MyriadTribulationCount != nil {
		sets = append(sets, "myriad_tribulation_count=$"+strconv.Itoa(argIdx))
		args = append(args, *req.MyriadTribulationCount)
		argIdx++
	}
	if req.ChaosCalamityCount != nil {
		sets = append(sets, "chaos_calamity_count=$"+strconv.Itoa(argIdx))
		args = append(args, *req.ChaosCalamityCount)
		argIdx++
	}
	if req.State != nil {
		sets = append(sets, "state=$"+strconv.Itoa(argIdx))
		args = append(args, *req.State)
		argIdx++
	}
	if req.DaoMarks != nil {
		daoJSON, err := json.Marshal(*req.DaoMarks)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dao marks JSON không hợp lệ"})
			return
		}
		sets = append(sets, "dao_marks=$"+strconv.Itoa(argIdx))
		args = append(args, daoJSON)
		argIdx++
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có trường nào để cập nhật"})
		return
	}

	sets = append(sets, "updated_at=now()")

	args = append(args, apertureID)
	updateQuery := fmt.Sprintf("UPDATE immortal_apertures SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	if _, err := tx.Exec(updateQuery, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tiên khiếu: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể commit transaction"})
		return
	}

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)

	var sysSets []string
	var sysArgs []interface{}

	for idx, s := range sets {
		if strings.HasPrefix(s, "dao_marks=") {
			var oldDao models.DaoMarksMap
			if len(oldDaoJSON) > 0 {
				_ = json.Unmarshal(oldDaoJSON, &oldDao)
			}
			if oldDao == nil {
				oldDao = make(models.DaoMarksMap)
			}
			changedDao := make(models.DaoMarksMap)
			if req.DaoMarks != nil {
				for k, newVal := range *req.DaoMarks {
					if oldDao[k] != newVal {
						changedDao[k] = newVal
					}
				}
				for k := range oldDao {
					if _, ok := (*req.DaoMarks)[k]; !ok {
						changedDao[k] = 0
					}
				}
			}
			if len(changedDao) > 0 {
				changedJSON, _ := json.Marshal(changedDao)
				sysSets = append(sysSets, s)
				sysArgs = append(sysArgs, changedJSON)
			}
		} else {
			sysSets = append(sysSets, s)
			sysArgs = append(sysArgs, args[idx])
		}
	}

	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã cập nhật Tiên Khiếu của đạo hữu %s: %s", username, formatChangedFields(sysSets, sysArgs)))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.update_aperture",
		EntityType:     "aperture",
		EntityID:       apertureID.String(),
		RequestSummary: strings.Join(sets, ", "),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật tiên khiếu thành công"})
}
