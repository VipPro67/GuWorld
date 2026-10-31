package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// baseline stage rows mirror WF-REF-06 §3 for plan-shape tests.
func baselineStagesForTest() []*models.RefineStageTemplate {
	rows := []struct {
		rank, duration, steps, minigames, window int
	}{
		{1, 60, 3, 1, 20}, {2, 120, 3, 1, 15}, {3, 300, 4, 2, 12},
		{4, 600, 4, 2, 10}, {5, 1200, 5, 3, 8}, {6, 2700, 5, 3, 8},
		{7, 5400, 6, 4, 6}, {8, 7200, 6, 4, 6}, {9, 14400, 7, 4, 5},
	}
	out := make([]*models.RefineStageTemplate, 0, len(rows))
	for _, r := range rows {
		out = append(out, &models.RefineStageTemplate{
			Rank: r.rank, DurationSeconds: r.duration, StepCount: r.steps,
			MinigameCount: r.minigames, WindowSeconds: r.window,
			AccuracyMaxBonus: 0.08, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0,
			TapHalfWidth: 0.12, TapPeriodMs: 2000,
			SequenceLength: 3, SequenceDisplayMs: 600, SequenceInputMs: 1500,
			OrderItemCount: 3, ChoiceOptionCount: 2,
			MinigameWeights: map[string]float64{"TAP": 1, "SEQUENCE": 1, "ORDER": 1},
		})
	}
	return out
}

func TestBuildRefinePlanShapeAllRanks(t *testing.T) {
	startedAt := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	decisions := []refineDecisionRow{{
		ID: 1, Situation: "Hỏa bùng lên",
		Options:      []string{"Tăng hỏa", "Hạ hỏa", "Giữ nguyên"},
		CorrectIndex: 1, NeutralIndex: 2, Explanation: "x",
	}}
	for _, stage := range baselineStagesForTest() {
		endsAt := startedAt.Add(time.Duration(stage.DurationSeconds) * time.Second)
		for attempt := 0; attempt < 30; attempt++ {
			plan, err := buildRefinePlan(stage, nil, decisions, startedAt, endsAt)
			if err != nil {
				t.Fatalf("rank %d: buildRefinePlan: %v", stage.Rank, err)
			}
			n := len(plan.Steps)
			if n != stage.StepCount {
				t.Fatalf("rank %d: got %d steps, want %d", stage.Rank, n, stage.StepCount)
			}
			if plan.Steps[0].Type != RefineStepChoice {
				t.Fatalf("rank %d: step 1 = %s, want CHOICE", stage.Rank, plan.Steps[0].Type)
			}
			if plan.Steps[n-1].Type != RefineStepTap {
				t.Fatalf("rank %d: step N = %s, want TAP", stage.Rank, plan.Steps[n-1].Type)
			}
			if plan.Steps[n-2].Type == RefineStepTap {
				t.Fatalf("rank %d: step N-1 is TAP", stage.Rank)
			}
			physical := 0
			decisions := 0
			for i, st := range plan.Steps {
				switch st.Type {
				case RefineStepChoice:
					decisions++
				case RefineStepTap, RefineStepSequence, RefineStepOrder:
					physical++
				default:
					t.Fatalf("rank %d: unknown step type %s at %d", stage.Rank, st.Type, i)
				}
				if i > 0 && st.Type != RefineStepChoice && plan.Steps[i-1].Type == st.Type {
					t.Fatalf("rank %d: adjacent duplicate minigame type %s at %d", stage.Rank, st.Type, i)
				}
				// window sanity
				if !st.WindowEnd.After(st.WindowStart) {
					t.Fatalf("rank %d: step %d closed window", stage.Rank, st.I)
				}
				if st.Nonce == "" || st.Seed == "" {
					t.Fatalf("rank %d: step %d missing seed/nonce", stage.Rank, st.I)
				}
			}
			if physical != stage.MinigameCount {
				t.Fatalf("rank %d: physical minigames = %d, want %d", stage.Rank, physical, stage.MinigameCount)
			}
			planSeedCount := 0
			for _, st := range plan.Steps {
				if st.Seed != "" {
					planSeedCount++
				}
			}
			if planSeedCount != n {
				t.Fatalf("rank %d: seeds incomplete", stage.Rank)
			}
		}
	}
}

func TestBuildRefinePlanRank1Shape(t *testing.T) {
	stage := &models.RefineStageTemplate{Rank: 1, DurationSeconds: 60, StepCount: 3, MinigameCount: 1, WindowSeconds: 20, MinigameWeights: map[string]float64{"TAP": 1, "SEQUENCE": 1, "ORDER": 1}}
	startedAt := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	decisions := []refineDecisionRow{{
		ID: 1, Situation: "Hỏa bùng lên",
		Options:      []string{"Tăng hỏa", "Hạ hỏa", "Giữ nguyên"},
		CorrectIndex: 1, NeutralIndex: 2, Explanation: "x",
	}}
	plan, err := buildRefinePlan(stage, nil, decisions, startedAt, startedAt.Add(60*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	got := []string{plan.Steps[0].Type, plan.Steps[1].Type, plan.Steps[2].Type}
	want := []string{RefineStepChoice, RefineStepChoice, RefineStepTap}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rank 1 shape = %v, want %v", got, want)
		}
	}
	if plan.GapMs != 20000 {
		t.Fatalf("gap = %d, want 20000", plan.GapMs)
	}
}

func TestPublicSessionViewDoesNotLeakAnswers(t *testing.T) {
	stage := &models.RefineStageTemplate{Rank: 3, DurationSeconds: 300, StepCount: 4, MinigameCount: 2, WindowSeconds: 12, MinigameWeights: map[string]float64{"TAP": 1, "SEQUENCE": 1, "ORDER": 1}, ChoiceOptionCount: 3, SequenceLength: 4, SequenceDisplayMs: 500, SequenceInputMs: 1300, TapHalfWidth: 0.1, TapPeriodMs: 1700, OrderItemCount: 4}
	startedAt := time.Now().UTC().Add(-time.Minute)
	decisions := []refineDecisionRow{{
		ID: 1, Situation: "Hỏa bùng lên",
		Options: []string{"Tăng hỏa", "Hạ hỏa", "Giữ nguyên"},
		CorrectIndex: 1, NeutralIndex: 2, Explanation: "x",
	}}
	plan, err := buildRefinePlan(stage, nil, decisions, startedAt, startedAt.Add(300*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	sess := &refineSessionRow{
		ID: "s", PlayerID: "p", RecipeID: "r", TargetGuID: "GU_X", TargetRank: 3,
		Status: "ACTIVE", Plan: plan,
		SelectedGuIDs: []string{"a", "b"},
	}
	view := publicSessionView(sess, startedAt)
	raw, _ := json.Marshal(view)
	s := string(raw)
	for _, leaked := range []string{"\"seed\"", "correct_index", "neutral", "option_order", "CorrectDisplayIndex", "\"center\""} {
		// center for TAP is intentionally public; only forbid "seed"/answer keys
		if leaked == "\"seed\"" || leaked == "correct_index" || leaked == "neutral" || leaked == "option_order" {
			if strings.Contains(s, strings.ReplaceAll(leaked, "\"", "")) {
				t.Fatalf("public view leaks %s: %s", leaked, s)
			}
		}
	}
	if strings.Contains(s, "\"seed\"") || strings.Contains(s, "\"seed\":") {
		t.Fatalf("public view leaks plan seeds: %s", s)
	}
}

func TestAverageStepScores(t *testing.T) {
	results := map[string]refineStepResult{
		"1": {Score: 100}, "2": {Score: 50}, "3": {Score: 0},
	}
	missing := averageStepScores(results, 4) // step 4 not submitted → 0
	if missing != 37.5 {
		t.Fatalf("accuracy = %v, want 37.5", missing)
	}
	full := averageStepScores(map[string]refineStepResult{"1": {Score: 100}, "2": {Score: 100}, "3": {Score: 100}}, 3)
	if full != 100 {
		t.Fatalf("accuracy = %v, want 100", full)
	}
}

func TestChoiceFallbackToTapWhenRegistryEmpty(t *testing.T) {
	stage := &models.RefineStageTemplate{Rank: 3, DurationSeconds: 300, StepCount: 4, MinigameCount: 2, WindowSeconds: 12, MinigameWeights: map[string]float64{"TAP": 1, "SEQUENCE": 1, "ORDER": 1}, ChoiceOptionCount: 3, SequenceLength: 4, SequenceDisplayMs: 500, SequenceInputMs: 1300, TapHalfWidth: 0.1, TapPeriodMs: 1700}
	startedAt := time.Now().UTC()
	plan, err := buildRefinePlan(stage, nil, nil, startedAt, startedAt.Add(300*time.Second)) // no decisions
	if err != nil {
		t.Fatal(err)
	}
	choiceSlots := 0
	for _, st := range plan.Steps {
		if st.Type == RefineStepChoice {
			choiceSlots++
		}
	}
	// Step 1 is always CHOICE per spec... but fallback converts CHOICE→TAP
	// only for registry-dependent pick steps; step 1 leads the furnace and is
	// converted too per WF-REF-04 §7.3 ("Các bước CHOICE chuyển thành TAP").
	if choiceSlots != 0 {
		t.Fatalf("expected all CHOICE converted to TAP when registry is empty, got %d", choiceSlots)
	}
	for _, st := range plan.Steps {
		if st.Type != RefineStepTap && st.Type != RefineStepSequence && st.Type != RefineStepOrder {
			t.Fatalf("unexpected type %s", st.Type)
		}
	}
}
