package handlers

import (
	"fmt"
)

// LayerQiMultipliers returns the rate multipliers for Heaven Qi and Earth Qi
// based on the vertical world layer where the immortal aperture is anchored.
// Canon rules (Reverend Insanity / Cổ Chân Nhân):
// - Tầng 0 (Địa Uyên - dia_uyen): Dồi dào Địa Khí (350%), nhưng cực kỳ hiếm Thiên Khí (20%).
// - Tầng 1 (Ngũ Vực - world_map): Cân bằng tự nhiên chuẩn mực (100% Thiên Khí, 100% Địa Khí).
// - Tầng 2 (Bạch Thiên - bai_thien): Dồi dào Thiên Khí (300%), nhưng thiếu hụt Địa Khí (30%).
// - Tầng 3 (Hắc Thiên - hac_thien): Thiên Khí dị biến (100% pha trộn Ám/Tinh), Địa Khí thấp (50%).
func LayerQiMultipliers(layer string) (heavenMultiplier float64, earthMultiplier float64) {
	switch layer {
	case "dia_uyen":
		return 0.20, 3.50
	case "bai_thien":
		return 3.00, 0.30
	case "hac_thien":
		return 1.00, 0.50
	default:
		return 1.00, 1.00
	}
}

// LayerTribulationTimeRateMultiplier returns the time flow factor affecting tribulation
// progression when an aperture is anchored in a vertical world layer.
//   - In Thái Cổ Hắc Thiên (hac_thien), Thiên Ý (Heaven's Will) is distorted and obscured by
//     the immemorial darkness and soul/star clouds, lengthening the tribulation interval by 50%
//     (effective time progression is 1.0 / 1.5 ~ 0.6667).
//   - Other layers maintain standard synchronized world time progression (1.0).
func LayerTribulationTimeRateMultiplier(layer string) float64 {
	if layer == "hac_thien" {
		return 1.0 / 1.5
	}
	return 1.0
}

// LayerPlacementDescription returns a lore-accurate description of the layer environment
// where the aperture was anchored.
func LayerPlacementDescription(layer string, darkMarksGained, starMarksGained int64) string {
	switch layer {
	case "dia_uyen":
		return "Vạn Trượng Địa Uyên (Địa Khí dồi dào +350%, Thiên Khí hiếm hoi +20%)"
	case "bai_thien":
		return "Thái Cổ Bạch Thiên (Thiên Khí dồi dào +300%, Địa Khí thiếu hụt +30%)"
	case "hac_thien":
		if darkMarksGained > 0 || starMarksGained > 0 {
			return fmt.Sprintf("Thái Cổ Hắc Thiên (Thiên Khí dị biến: +%d Đạo Ngân Ám Đạo, +%d Đạo Ngân Tinh Đạo; chu kỳ tai kiếp kéo dài +50%%)", darkMarksGained, starMarksGained)
		}
		return "Thái Cổ Hắc Thiên (Thiên Khí dị biến pha trộn Ám/Tinh Đạo; chu kỳ tai kiếp kéo dài +50%)"
	default:
		return "Ngũ Vực Phàm Trần (Thiên Địa Nhị Khí cân bằng tiêu chuẩn)"
	}
}
