package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Thanh Đạo Tẩy Thể (Dao Cleansing) — spec §6.
const (
	cleanseCooldown      = 12 * time.Hour
	cleanseMaxMarks      = 10000 // max marks stripped per cleanse
	cleanseMarksPerStone = 100   // 1 Tiên Nguyên Thạch per 100 tạp đạo ngân
	cleanseRefundRatio   = 0.30  // 30% refunded as Dao Materials
	// cleanseRefundMarks là số Đạo Ngân mà 1 món Đạo Tài hoàn trả bổ sung khi
	// dùng (khớp params.value của block ADD_DAO_MARKS trong Item.csv).
	cleanseRefundMarks = 1000
)

type cleanseDaoMarksRequest struct {
	Path  string `json:"path"`
	Marks int64  `json:"marks"`
}

// CleanseDaoMarks strips foreign Dao Marks from the player's immortal aperture,
// charging Tiên Nguyên Thạch and refunding 30% as Dao Materials of that path.
func CleanseDaoMarks(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req cleanseDaoMarksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	targetPath := gamefx.CleanPath(req.Path)
	if targetPath == "" || req.Marks <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lưu phái và số Đạo Ngân cần tẩy không hợp lệ"})
		return
	}
	if req.Marks > cleanseMaxMarks {
		req.Marks = cleanseMaxMarks
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu giao dịch"})
		return
	}
	defer tx.Rollback()

	var daoJSON []byte
	var lastCleanse *time.Time
	err = tx.QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(dao_marks, '{}'::jsonb), last_cleanse_at FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, playerID,
	).Scan(&daoJSON, &lastCleanse)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Tiên Khiếu"})
		return
	}

	if lastCleanse != nil {
		if remain := cleanseCooldown - time.Since(*lastCleanse); remain > 0 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("Tiên Khiếu chưa hồi phục, cần bế quan thêm %s.", formatCleanseRemain(remain))})
			return
		}
	}

	daoMarks := make(models.DaoMarksMap)
	if len(daoJSON) > 0 {
		_ = json.Unmarshal(daoJSON, &daoMarks)
	}

	// Never allow cleansing the dominant path.
	dominant := ""
	var maxMarks, available int64
	for p, v := range daoMarks {
		if v <= 0 {
			continue
		}
		clean := gamefx.CleanPath(p)
		if clean == targetPath {
			available += v
		}
		if v > maxMarks {
			maxMarks = v
			dominant = clean
		}
	}
	if dominant == targetPath {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tẩy Đạo Ngân của phái chủ đạo"})
		return
	}
	if available <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lưu phái này không có Đạo Ngân để tẩy"})
		return
	}
	cleansed := req.Marks
	if cleansed > available {
		cleansed = available
	}

	stoneCost := int64(math.Ceil(float64(cleansed) / cleanseMarksPerStone))

	var stones int64
	if err := tx.QueryRowContext(c.Request.Context(),
		`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1 FOR UPDATE`, playerID,
	).Scan(&stones); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra Tiên Nguyên Thạch"})
		return
	}
	if stones < stoneCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch (cần %d, đang có %d).", stoneCost, stones)})
		return
	}

	// Apply mark removal (aggregate across raw keys mapping to the same clean path).
	remaining := cleansed
	for p, v := range daoMarks {
		if remaining <= 0 || v <= 0 || gamefx.CleanPath(p) != targetPath {
			continue
		}
		if v <= remaining {
			remaining -= v
			delete(daoMarks, p)
		} else {
			daoMarks[p] = v - remaining
			remaining = 0
		}
	}

	newDaoJSON, err := json.Marshal(daoMarks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi mã hoá Đạo Ngân"})
		return
	}

	var newStones float64
	if err := tx.QueryRowContext(c.Request.Context(),
		`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2 RETURNING immortal_stones`,
		stoneCost, playerID,
	).Scan(&newStones); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ Tiên Nguyên Thạch"})
		return
	}

	if _, err := tx.ExecContext(c.Request.Context(),
		`UPDATE immortal_apertures SET dao_marks = $1, last_cleanse_at = NOW(), updated_at = NOW() WHERE player_id = $2`,
		newDaoJSON, playerID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Đạo Ngân"})
		return
	}

	// Hoàn trả 30% Đạo Ngân đã bóc tách dưới dạng Đạo Tài của lưu phái đó;
	// mỗi món bổ sung cleanseRefundMarks Đạo Ngân khi sử dụng.
	refundItem := "dao_material_" + targetPath
	refundQty := cleanseRefundQty(cleansed)
	if refundQty > 0 {
		if _, ok := models.ItemRegistry[refundItem]; !ok {
			// Chưa import Đạo Tài vào item_dictionary → bỏ hoàn trả, vẫn tẩy thành công.
			logger.S.Warnf("cleanse refund skipped: item %s not in registry", refundItem)
			refundQty = 0
		} else if err := InventorySvc.AddItem(tx, playerID.String(), refundItem, "materials", refundQty); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn trả Đạo Tài"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất tẩy thể"})
		return
	}

	respondActionWithInventory(c, http.StatusOK, models.Player{Account: models.Account{ID: playerID}}, gin.H{
		"message":      fmt.Sprintf("Thanh Đạo Tẩy Thể thành công! Bóc tách %d Đạo Ngân hệ %s, tiêu hao %d Tiên Nguyên Thạch, thu hồi %d Đạo Tài (+%d Đạo Ngân khi dùng).", cleansed, strings.ToUpper(targetPath), stoneCost, refundQty, refundQty*cleanseRefundMarks),
		"cleansed":     cleansed,
		"path":         targetPath,
		"stone_cost":   stoneCost,
		"refund_item":  refundItem,
		"refund_qty":   refundQty,
		"refund_marks": refundQty * cleanseRefundMarks,
		"dao_marks":    daoMarks,
	})
}

// cleanseRefundQty trả về số món Đạo Tài hoàn trả: floor(30% marks / 1000).
// Không làm tròn lên — tẩy nhỏ không được lợi (1 món luôn đáng 1000 Đạo Ngân).
func cleanseRefundQty(cleansed int64) int64 {
	if cleansed <= 0 {
		return 0
	}
	return int64(float64(cleansed)*cleanseRefundRatio) / cleanseRefundMarks
}

func formatCleanseRemain(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h <= 0 {
		return fmt.Sprintf("%d phút", m)
	}
	return fmt.Sprintf("%d giờ %d phút", h, m)
}
