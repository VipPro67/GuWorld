package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

func formatChangedFields(sets []string, args []interface{}) string {
	var fields []string
	for i, s := range sets {
		parts := strings.Split(s, "=")
		if len(parts) == 0 {
			continue
		}
		fieldName := strings.TrimSpace(parts[0])
		if fieldName == "updated_at" {
			continue
		}
		if i >= len(args) {
			continue
		}
		val := args[i]

		translated := fieldName
		valStr := fmt.Sprintf("%v", val)

		switch fieldName {
		case "rank":
			translated = "Cảnh giới"
		case "sub_rank":
			translated = "Tiểu cảnh giới"
		case "aptitude_type":
			translated = "Hệ tư chất"
		case "aptitude_percent":
			translated = "Tỉ lệ tư chất"
			valStr = fmt.Sprintf("%.1f%%", val)
		case "aptitude_grade":
			translated = "Phẩm cấp tư chất"
		case "current_mana":
			translated = "Mana hiện tại"
			valStr = fmt.Sprintf("%.0f", val)
		case "max_mana":
			translated = "Mana tối đa"
			valStr = fmt.Sprintf("%.0f", val)
		case "hp":
			translated = "HP"
			valStr = fmt.Sprintf("%.0f", val)
		case "base_hp":
			translated = "HP cơ bản"
			valStr = fmt.Sprintf("%.0f", val)
		case "wall_progress":
			translated = "Tiến trình phá bức tường"
			valStr = fmt.Sprintf("%.1f%%", val)
		case "human_qi":
			translated = "Nhân khí"
			valStr = fmt.Sprintf("%.0f", val)
		case "lifespan":
			translated = "Thọ nguyên"
			valStr = fmt.Sprintf("%.0f năm", val)
		case "current_region":
			translated = "Khu vực"
		case "soul_hp":
			translated = "HP linh hồn"
			valStr = fmt.Sprintf("%.0f", val)
		case "max_soul":
			translated = "Linh hồn tối đa"
			valStr = fmt.Sprintf("%.0f", val)
		case "soul_tier":
			translated = "Cấp bậc linh hồn"
		case "explore_count":
			translated = "Số lượt thám hiểm"
		case "aperture_stress":
			translated = "Áp lực tiên khiếu"
			valStr = fmt.Sprintf("%.1f%%", val)
		case "bio":
			translated = "Tiểu sử"
		case "avatar_url":
			translated = "Ảnh đại diện"
		case "race":
			translated = "Chủng tộc"
		case "gender":
			translated = "Giới tính"
		case "constitution":
			translated = "Thể chất"
		case "statuses":
			translated = "Trạng thái"
		case "is_admin":
			translated = "Quyền Admin"
		case "name":
			translated = "Tên"
		case "description":
			translated = "Mô tả"
		case "region":
			translated = "Khu vực"
		case "level":
			translated = "Cấp độ"
		case "funds":
			translated = "Ngân quỹ"
		case "require_approval":
			translated = "Yêu cầu phê duyệt"
			if b, ok := val.(bool); ok {
				if b {
					valStr = "Có"
				} else {
					valStr = "Không"
				}
			}
		case "require_promotion_approval":
			translated = "Duyệt lên cấp"
			if b, ok := val.(bool); ok {
				if b {
					valStr = "Có"
				} else {
					valStr = "Không"
				}
			}
		case "grade":
			translated = "Phẩm cấp"
			gradeMap := map[string]string{"LOW": "Hạ Phẩm", "MID": "Trung Phẩm", "HIGH": "Thượng Phẩm", "EXTREME": "Chí Tôn"}
			if str, ok := val.(string); ok {
				if mapped, found := gradeMap[str]; found {
					valStr = mapped
				}
			}
		case "time_rate":
			translated = "Tốc độ thời gian"
			valStr = fmt.Sprintf("%.1fx", val)
		case "aperture_hp":
			translated = "HP Tiên Khiếu"
		case "max_aperture_hp":
			translated = "HP tối đa"
		case "area_size":
			translated = "Diện tích"
			valStr = fmt.Sprintf("%v mẫu", val)
		case "immortal_essence":
			translated = "Tiên Nguyên"
		case "essence_production_per_year":
			translated = "Sản lượng Tiên Nguyên/năm"
		case "internal_years_passed":
			translated = "Thời gian trôi qua"
			valStr = fmt.Sprintf("%.3f năm", val)
		case "next_tribulation_year":
			translated = "Năm chịu kiếp tiếp theo"
			valStr = fmt.Sprintf("Năm %.1f", val)
		case "tribulation_count":
			translated = "Số lần độ kiếp"
		case "rank_start_year":
			translated = "Năm bắt đầu cảnh giới"
			valStr = fmt.Sprintf("Năm %.1f", val)
		case "earth_calamity_count":
			translated = "Số lần Địa Tai"
		case "heavenly_tribulation_count":
			translated = "Số lần Thiên Kiếp"
		case "grand_tribulation_count":
			translated = "Số lần Đại Kiếp"
		case "myriad_tribulation_count":
			translated = "Số lần Vạn Kiếp"
		case "chaos_calamity_count":
			translated = "Số lần Hỗn Độn Kiếp"
		case "state":
			translated = "Trạng thái"
			stateMap := map[string]string{"STABLE": "Ổn định", "TRIBULATION_PENDING": "Đợi Kiếp", "TRIBULATION_ACTIVE": "Đang Kiếp", "DAMAGED": "Tổn hại", "DESTROYED": "Phá huỷ"}
			if str, ok := val.(string); ok {
				if mapped, found := stateMap[str]; found {
					valStr = mapped
				}
			}
		case "dao_marks":
			translated = "Đạo ngân"
			if bytes, ok := val.([]byte); ok {
				var m map[string]interface{}
				if err := json.Unmarshal(bytes, &m); err == nil {
					pathToVietnamese := map[string]string{
						// Ngũ hành & Tự nhiên
						"metal":     "Kim đạo",
						"wood":      "Mộc đạo",
						"water":     "Thủy đạo",
						"fire":      "Hỏa đạo",
						"earth":     "Thổ đạo",
						"wind":      "Phong đạo",
						"lightning": "Lôi đạo",
						"ice":       "Băng Tuyết đạo",
						"cloud":     "Vân đạo",
						"light":     "Quang đạo",
						"dark":      "Ám đạo",
						"star":      "Tinh đạo",
						"sound":     "Âm đạo",

						// Không - Thời gian & Vũ trụ
						"space":  "Vũ đạo",
						"time":   "Trụ đạo",
						"heaven": "Thiên đạo",

						// Quy tắc & Kỹ nghệ
						"rule":       "Luật đạo",
						"refinement": "Luyện đạo",
						"formation":  "Trận đạo",
						"qi":         "Khí đạo",

						// Cơ thể, Tinh thần & Xã hội
						"strength":       "Lực đạo",
						"soul":           "Hồn đạo",
						"wisdom":         "Trí đạo",
						"emotion":        "Tình đạo",
						"transformation": "Biến hóa đạo",
						"enslavement":    "Nô đạo",
						"information":    "Tín đạo",
						"blood":          "Huyết đạo",
						"poison":         "Độc đạo",
						"bone":           "Cốt đạo",
						"human":          "Nhân đạo",
						"dream":          "Mộng đạo",
						"food":           "Thực đạo",
						"luck":           "Vận đạo",

						// Vũ khí, Sát phạt & Đặc thù
						"sword":       "Kiếm đạo",
						"blade":       "Đao đạo",
						"theft":       "Thâu đạo",
						"phantom":     "Huyễn đạo",
						"painting":    "Họa đạo",
						"weapon":      "Binh đạo",
						"killing":     "Sát đạo",
						"restriction": "Cấm đạo",
						"pill":        "Đan đạo",
					}
					parts := []string{}
					for k, v := range m {
						pathName := k
						if viName, found := pathToVietnamese[k]; found {
							pathName = viName
						}
						valFormatted := fmt.Sprintf("%v", v)
						if f, ok := v.(float64); ok {
							if f == float64(int64(f)) {
								valFormatted = fmt.Sprintf("%.0f", f)
							} else {
								valFormatted = fmt.Sprintf("%f", f)
								valFormatted = strings.TrimRight(strings.TrimRight(valFormatted, "0"), ".")
							}
						}
						parts = append(parts, fmt.Sprintf("%s: %s", pathName, valFormatted))
					}
					valStr = strings.Join(parts, ", ")
				}
			}
		}

		fields = append(fields, fmt.Sprintf("%s (%s)", translated, valStr))
	}
	return strings.Join(fields, ", ")
}

// GetBatchMetrics returns metrics about the batch persister worker.
func GetBatchMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total_flushes":      atomic.LoadInt64(&worker.Metrics.TotalFlushes),
		"successful_flushes": atomic.LoadInt64(&worker.Metrics.SuccessfulFlushes),
		"failed_flushes":     atomic.LoadInt64(&worker.Metrics.FailedFlushes),
		"items_processed":    atomic.LoadInt64(&worker.Metrics.ItemsProcessed),
		"items_retried":      atomic.LoadInt64(&worker.Metrics.ItemsRetried),
		"items_dropped":      atomic.LoadInt64(&worker.Metrics.ItemsDropped),
		"queue_depth":        atomic.LoadInt64(&worker.Metrics.QueueDepth),
	})
}
