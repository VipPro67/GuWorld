package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminRefinementJob struct {
	ID              string     `json:"id"`
	ApertureID      string     `json:"aperture_id"`
	RecipeID        string     `json:"recipe_id"`
	Mode            string     `json:"mode"`
	TargetQuantity  int        `json:"target_quantity"`
	CurrentQuantity int        `json:"current_quantity"`
	Priority        int        `json:"priority"`
	Enabled         bool       `json:"enabled"`
	LastError       string     `json:"last_error"`
	LastProcessedAt *time.Time `json:"last_processed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func AdminListApertureRefinementJobs(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id,aperture_id,recipe_id,mode,target_quantity,current_quantity,priority,enabled,last_error,last_processed_at,created_at,updated_at FROM aperture_refinement_queue WHERE aperture_id=$1 ORDER BY priority DESC,created_at`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải hàng đợi luyện cổ"})
		return
	}
	defer rows.Close()
	list := make([]AdminRefinementJob, 0)
	for rows.Next() {
		var r AdminRefinementJob
		if err := rows.Scan(&r.ID, &r.ApertureID, &r.RecipeID, &r.Mode, &r.TargetQuantity, &r.CurrentQuantity, &r.Priority, &r.Enabled, &r.LastError, &r.LastProcessedAt, &r.CreatedAt, &r.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc hàng đợi luyện cổ"})
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc hàng đợi luyện cổ"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func validateAdminRefinementJob(r *AdminRefinementJob) string {
	r.RecipeID = strings.TrimSpace(r.RecipeID)
	if r.RecipeID == "" || len(r.RecipeID) > 100 {
		return "Mã bí phương bắt buộc và tối đa 100 ký tự"
	}
	switch r.Mode {
	case "FIXED_COUNT", "UNTIL_SUCCESS", "UNTIL_MATERIAL_EMPTY", "MAINTAIN_STOCK":
	default:
		return "Chế độ luyện cổ không hợp lệ"
	}
	if r.TargetQuantity < 1 || r.CurrentQuantity < 0 || (r.Mode == "FIXED_COUNT" && r.CurrentQuantity > r.TargetQuantity) || r.Priority < -1000 || r.Priority > 1000 {
		return "Số lượng hoặc ưu tiên hàng đợi không hợp lệ"
	}
	if len(r.LastError) > 2000 {
		return "Thông báo lỗi tối đa 2000 ký tự"
	}
	return ""
}

func AdminCreateApertureRefinementJob(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminRefinementJob
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu hàng đợi luyện cổ không hợp lệ"})
		return
	}
	if message := validateAdminRefinementJob(&req); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM refining_recipes WHERE ('recipe_'||target_gu_id||'_'||target_rank::text||'_'||variant::text)=$1)`, req.RecipeID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh bí phương"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy bí phương phù hợp"})
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO aperture_refinement_queue(aperture_id,recipe_id,mode,target_quantity,current_quantity,priority,enabled,last_error,last_processed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, c.Param("id"), req.RecipeID, req.Mode, req.TargetQuantity, req.CurrentQuantity, req.Priority, req.Enabled, req.LastError, req.LastProcessedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo công việc luyện cổ: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_aperture_refinement_job", "aperture=%s recipe=%s", c.Param("id"), req.RecipeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo công việc luyện cổ"})
}

func AdminUpdateApertureRefinementJob(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminRefinementJob
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu hàng đợi luyện cổ không hợp lệ"})
		return
	}
	if message := validateAdminRefinementJob(&req); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM refining_recipes WHERE ('recipe_'||target_gu_id||'_'||target_rank::text||'_'||variant::text)=$1)`, req.RecipeID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh bí phương"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy bí phương phù hợp"})
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE aperture_refinement_queue SET recipe_id=$1,mode=$2,target_quantity=$3,current_quantity=$4,priority=$5,enabled=$6,last_error=$7,last_processed_at=$8,updated_at=NOW() WHERE id=$9 AND aperture_id=$10`, req.RecipeID, req.Mode, req.TargetQuantity, req.CurrentQuantity, req.Priority, req.Enabled, req.LastError, req.LastProcessedAt, c.Param("job_id"), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật công việc luyện cổ: " + err.Error()})
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy công việc luyện cổ"})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_aperture_refinement_job", "job=%s", c.Param("job_id"))
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật công việc luyện cổ"})
}

func AdminDeleteApertureRefinementJob(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM aperture_refinement_queue WHERE id=$1 AND aperture_id=$2`, c.Param("job_id"), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa công việc luyện cổ"})
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy công việc luyện cổ"})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_aperture_refinement_job", "job=%s", c.Param("job_id"))
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa công việc luyện cổ"})
}
