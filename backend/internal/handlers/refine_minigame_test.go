package handlers

import (
	"testing"
	"time"
)

func makeStepForScore(stepType string, cfg RefinePlanStepConfig, windowStart time.Time, windowMs int64) RefinePlanStep {
	return RefinePlanStep{
		I: 1, Type: stepType, Seed: "seed", Nonce: "nonce",
		WindowStart: windowStart,
		WindowEnd:   windowStart.Add(time.Duration(windowMs) * time.Millisecond),
		Config:      cfg,
	}
}

func TestScoreRefineTap(t *testing.T) {
	windowStart := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	// period 2000ms; u=0.25 → pos=0.5; center 0.5 → score 100
	cfg := RefinePlanStepConfig{Center: 0.5, HalfWidth: 0.12, MarkerPeriodMs: 2000}
	step := makeStepForScore(RefineStepTap, cfg, windowStart, 20000)
	score := scoreRefineTap(step, windowStart.Add(500*time.Millisecond))
	if score != 100 {
		t.Fatalf("tap at center = %v, want 100", score)
	}

	// Miss far outside the half width → 0.
	score = scoreRefineTap(step, windowStart.Add(2000*time.Millisecond)) // u=1 → pos=0
	if score != 0 {
		t.Fatalf("tap far edge = %v, want 0", score)
	}

	// Half-width boundary edge: d == w → 0.
	edgeOffset := int64(0.12 * 2000 / 2) // pos changes 1 per half period (1000ms): d=|pos-center|
	score = scoreRefineTap(step, windowStart.Add(500*time.Millisecond+time.Duration(edgeOffset)*time.Millisecond))
	if score > 1 {
		t.Fatalf("tap beyond half width should be ~0, got %v", score)
	}
}

func TestScoreRefineSequence(t *testing.T) {
	windowStart := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	cfg := RefinePlanStepConfig{
		Symbols:       []int{0, 3, 1, 2},
		DisplayMs:     500,
		TargetInputMs: 4 * (500 + 1300),
	}
	step := makeStepForScore(RefineStepSequence, cfg, windowStart, 60000)

	// Full correct, fast → 100.
	score, err := scoreRefineStep(step, windowStart.Add(600*time.Millisecond), map[string]interface{}{
		"sequence": []interface{}{float64(0), float64(3), float64(1), float64(2)},
	})
	if err != nil || score != 100 {
		t.Fatalf("sequence fast correct = %v err %v, want 100", score, err)
	}

	// Full correct but slow (used = 2× target) → 80.
	score, err = scoreRefineStep(step, windowStart.Add(2*7200*time.Millisecond), map[string]interface{}{
		"sequence": []interface{}{float64(0), float64(3), float64(1), float64(2)},
	})
	if err != nil || score != 80 {
		t.Fatalf("sequence slow correct = %v err %v, want 80", score, err)
	}

	// Half correct (valid symbols, wrong at 2 positions) → 50.
	score, err = scoreRefineStep(step, windowStart.Add(600*time.Millisecond), map[string]interface{}{
		"sequence": []interface{}{float64(1), float64(0), float64(1), float64(2)},
	})
	if err != nil || score != 50 {
		t.Fatalf("sequence half = %v err %v, want 50", score, err)
	}

	// Invalid payload → error.
	if _, err := scoreRefineStep(step, windowStart, map[string]interface{}{}); err == nil {
		t.Fatalf("sequence missing payload must fail")
	}
	if _, err := scoreRefineStep(step, windowStart, map[string]interface{}{
		"sequence": []interface{}{float64(0), float64(3), float64(1), float64(9)},
	}); err == nil {
		t.Fatalf("sequence out-of-range symbol must fail")
	}
	// Shorter-than-L arrays count as partial matches (missing positions = 0).
	score, err = scoreRefineStep(step, windowStart.Add(600*time.Millisecond), map[string]interface{}{
		"sequence": []interface{}{float64(0), float64(3)},
	})
	if err != nil || score != 50 {
		t.Fatalf("sequence short = %v err %v, want 50", score, err)
	}
}

func TestScoreRefineChoice(t *testing.T) {
	cfg := RefinePlanStepConfig{
		Situation:   "Hỏa bùng lên",
		OptionTexts: []string{"Tăng hỏa", "Hạ hỏa", "Giữ nguyên"},
		// correct original index 1 displayed at position 1;
		// neutral original index 2 displayed at position 2.
		CorrectDisplayIndex: 1,
		NeutralDisplayIndex: 2,
	}
	step := makeStepForScore(RefineStepChoice, cfg, time.Now().UTC(), 12000)
	cases := []struct {
		idx  float64
		want float64
	}{
		{1, 100}, {2, 50}, {0, 0},
	}
	for _, tc := range cases {
		score, err := scoreRefineStep(step, time.Now().UTC(), map[string]interface{}{"choice_index": tc.idx})
		if err != nil {
			t.Fatalf("choice idx %v err %v", tc.idx, err)
		}
		if score != tc.want {
			t.Fatalf("choice idx %v = %v, want %v", tc.idx, score, tc.want)
		}
	}
	if _, err := scoreRefineStep(step, time.Now().UTC(), map[string]interface{}{"choice_index": 9.0}); err == nil {
		t.Fatalf("out-of-range choice must fail")
	}
	if _, err := scoreRefineStep(step, time.Now().UTC(), map[string]interface{}{}); err == nil {
		t.Fatalf("missing choice_index must fail")
	}
}

func TestScoreRefineOrder(t *testing.T) {
	items := []RefineOrderItem{
		{ID: "ITEM_C", Name: "C", Rank: 3},
		{ID: "ITEM_A", Name: "A", Rank: 1},
		{ID: "ITEM_B", Name: "B", Rank: 2},
	}
	cfg := RefinePlanStepConfig{Items: items}
	step := makeStepForScore(RefineStepOrder, cfg, time.Now().UTC(), 8000)
	correct := []interface{}{"ITEM_A", "ITEM_B", "ITEM_C"}

	score, err := scoreRefineStep(step, time.Now().UTC(), map[string]interface{}{"order": correct})
	if err != nil || score != 100 {
		t.Fatalf("order correct = %v err %v, want 100", score, err)
	}

	swapped := []interface{}{"ITEM_B", "ITEM_A", "ITEM_C"} // one adjacent pair inverted
	score, err = scoreRefineStep(step, time.Now().UTC(), map[string]interface{}{"order": swapped})
	if err != nil {
		t.Fatalf("order swapped err %v", err)
	}
	// 1 correct adjacent pair of 2 → 50
	if score != 50 {
		t.Fatalf("order swapped = %v, want 50", score)
	}

	reversed := []interface{}{"ITEM_C", "ITEM_B", "ITEM_A"}
	score, err = scoreRefineStep(step, time.Now().UTC(), map[string]interface{}{"order": reversed})
	if err != nil {
		t.Fatalf("order reversed err %v", err)
	}
	if score != 0 {
		t.Fatalf("order reversed = %v, want 0", score)
	}

	// Not a permutation → error.
	if _, err := scoreRefineStep(step, time.Now().UTC(), map[string]interface{}{
		"order": []interface{}{"ITEM_A", "ITEM_A", "ITEM_B"},
	}); err == nil {
		t.Fatalf("duplicate in order must fail")
	}
}
