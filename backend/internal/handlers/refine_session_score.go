package handlers

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Refine session step scoring (WF-REF-04 §5). All scores are server-side only
// computations from the fixed plan + the submitted payload + server NOW().
// Scores are rounded to 2 digits and clamped into [0, 100].

// refineSubmitWindowGraceNs is the network grace after window_end (WF-REF-04 §3.4).
const refineSubmitWindowGraceNs = int64(1000 * time.Millisecond)

func refineClampScore(score float64) float64 {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return math.Round(score*100) / 100
}

// scoreRefineStep grades one submitted step. payload is the already-decoded
// JSON object (nil for missing/empty). Returns REFINE_PAYLOAD_INVALID-shaped
// errors for malformed payloads.
func scoreRefineStep(step RefinePlanStep, now time.Time, payload map[string]interface{}) (float64, error) {
	// Grace clamping (WF-REF-04 §3.4): a submit accepted inside the 1s grace
	// window is graded at window_end, never after.
	if now.After(step.WindowEnd) {
		now = step.WindowEnd
	}
	switch step.Type {
	case RefineStepTap:
		return scoreRefineTap(step, now), nil
	case RefineStepSequence:
		return scoreRefineSequence(step, now, payload)
	case RefineStepChoice:
		return scoreRefineChoice(step, payload)
	case RefineStepOrder:
		return scoreRefineOrder(step, payload)
	}
	return 0, fmt.Errorf("unknown step type %s", step.Type)
}

// scoreRefineTap grades Canh Nhịp from the server receive time.
func scoreRefineTap(step RefinePlanStep, now time.Time) float64 {
	periodMs := step.Config.MarkerPeriodMs
	if periodMs <= 0 {
		periodMs = 2000
	}
	elapsed := now.Sub(step.WindowStart).Milliseconds()
	if elapsed < 0 {
		elapsed = 0
	}
	u := float64(elapsed%int64(periodMs)) / float64(periodMs)
	pos := 1.0 - math.Abs(2*u-1)
	center := step.Config.Center
	halfWidth := step.Config.HalfWidth
	if halfWidth <= 0 {
		halfWidth = 0.12
	}
	d := math.Abs(pos - center)
	return refineClampScore(100 * math.Max(0, 1-d/halfWidth))
}

// scoreRefineSequence grades Dãy Ấn: match ratio base, full-match gets a
// speed conditional of 100 down to 80.
func scoreRefineSequence(step RefinePlanStep, now time.Time, payload map[string]interface{}) (float64, error) {
	symbols := step.Config.Symbols
	if len(symbols) == 0 {
		return 0, fmt.Errorf("plan thiếu symbols")
	}
	raw, ok := payload["sequence"]
	if !ok {
		return 0, fmt.Errorf("thiếu sequence")
	}
	list, ok := raw.([]interface{})
	if !ok {
		return 0, fmt.Errorf("sequence phải là mảng")
	}
	if len(list) > len(symbols) {
		return 0, fmt.Errorf("sequence quá dài")
	}
	correct := 0
	for i, v := range list {
		num, ok := toInt(v)
		if !ok || num < 0 || num > 3 {
			return 0, fmt.Errorf("sequence chứa ký tự không hợp lệ")
		}
		if num == symbols[i] {
			correct++
		}
	}
	base := refineClampScore(100 * float64(correct) / float64(len(symbols)))
	if correct == len(symbols) {
		displayTotal := int64(len(symbols)) * int64(step.Config.DisplayMs)
		target := displayTotal + int64(len(symbols))*int64(sequenceInputMs(step))
		used := now.Sub(step.WindowStart).Milliseconds()
		if used <= target {
			return 100, nil
		}
		if target <= 0 {
			return 80, nil
		}
		return refineClampScore(math.Max(80, 100-20*(float64(used)/float64(target)-1))), nil
	}
	return base, nil
}

func sequenceInputMs(step RefinePlanStep) int {
	// stored target_input_ms = L*(display+input); derive the per-symbol input
	// budget back out for scoring.
	if step.Config.DisplayMs <= 0 {
		return 1500
	}
	inputMs := int64(step.Config.TargetInputMs)/int64(len(step.Config.Symbols)) - int64(step.Config.DisplayMs)
	if inputMs <= 0 {
		return 1500
	}
	return int(inputMs)
}

// scoreRefineChoice grades Chọn Hành Động: 100 đúng / 50 trung tính / 0 sai.
func scoreRefineChoice(step RefinePlanStep, payload map[string]interface{}) (float64, error) {
	raw, ok := payload["choice_index"]
	if !ok {
		return 0, fmt.Errorf("thiếu choice_index")
	}
	idx, ok := toInt(raw)
	if !ok || idx < 0 {
		return 0, fmt.Errorf("choice_index không hợp lệ")
	}
	if len(step.Config.OptionTexts) > 0 && idx >= len(step.Config.OptionTexts) {
		return 0, fmt.Errorf("choice_index vượt số option")
	}
	switch idx {
	case step.Config.CorrectDisplayIndex:
		return 100, nil
	case step.Config.NeutralDisplayIndex:
		return 50, nil
	default:
		return 0, nil
	}
}

// scoreRefineOrder grades Ghép Liệu by adjacent-pairs correctness against the
// canonical order (rank asc, then id asc).
func scoreRefineOrder(step RefinePlanStep, payload map[string]interface{}) (float64, error) {
	items := step.Config.Items
	if len(items) < 2 {
		return 100, nil
	}
	raw, ok := payload["order"]
	if !ok {
		return 0, fmt.Errorf("thiếu order")
	}
	list, ok := raw.([]interface{})
	if !ok {
		return 0, fmt.Errorf("order phải là mảng")
	}
	sortedItems := append([]RefineOrderItem(nil), items...)
	sort.Slice(sortedItems, func(i, j int) bool {
		if sortedItems[i].Rank != sortedItems[j].Rank {
			return sortedItems[i].Rank < sortedItems[j].Rank
		}
		return sortedItems[i].ID < sortedItems[j].ID
	})
	// Canonical position (rank asc, then id asc) per item id.
	positionOf := make(map[string]int, len(sortedItems))
	for i, it := range sortedItems {
		positionOf[it.ID] = i
	}
	// Payload must be a permutation of the presented set.
	seen := map[string]bool{}
	submitted := make([]string, 0, len(list))
	for _, v := range list {
		s, ok := v.(string)
		if !ok || !knownOrderItem(items, s) {
			return 0, fmt.Errorf("order chứa giá trị không hợp lệ")
		}
		if seen[s] {
			return 0, fmt.Errorf("order trùng phần tử")
		}
		seen[s] = true
		submitted = append(submitted, s)
	}
	if len(submitted) != len(items) {
		return 0, fmt.Errorf("order sai số phần tử")
	}
	// "Đúng thứ tự" (WF-REF-04 §5.4): a submitted pair is correct when its two
	// items keep their canonical relative order.
	adjacentCorrect := 0
	for i := 0; i+1 < len(submitted); i++ {
		if positionOf[submitted[i]] < positionOf[submitted[i+1]] {
			adjacentCorrect++
		}
	}
	return refineClampScore(100 * float64(adjacentCorrect) / float64(len(items)-1)), nil
}

func knownOrderItem(items []RefineOrderItem, id string) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}
