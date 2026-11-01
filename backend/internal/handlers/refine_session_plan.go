package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Refine session plan generation (WF-REF-04 §2 + WF-REF-05 §4.2).
// The plan is server-only: it holds seeds, answers and nonces and is NEVER
// returned to the client raw — clients receive publicStepView instead.

const (
	RefineStepTap      = "TAP"
	RefineStepSequence = "SEQUENCE"
	RefineStepChoice   = "CHOICE"
	RefineStepOrder    = "ORDER"

	refinePhysicalMinigameTypes = 3 // TAP / SEQUENCE / ORDER
)

// RefineOrderItem is one Ghép Liệu token: recipe materials first, then
// same-path registry filler to reach stage.OrderItemCount.
type RefineOrderItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
	Rank int    `json:"rank"`
}

type RefinePlanStepConfig struct {
	// CHOICE
	DecisionID          int      `json:"decision_id,omitempty"`
	OptionOrder         []int    `json:"option_order,omitempty"` // display position -> original option index
	OptionTexts         []string `json:"option_texts,omitempty"`
	Situation           string   `json:"situation,omitempty"`
	CorrectDisplayIndex int      `json:"correct_display_index,omitempty"`
	NeutralDisplayIndex int      `json:"neutral_display_index,omitempty"`
	Explanation         string   `json:"explanation,omitempty"`

	// SEQUENCE
	Symbols       []int `json:"symbols,omitempty"`
	DisplayMs     int   `json:"display_ms,omitempty"`
	TargetInputMs int   `json:"target_input_ms,omitempty"`

	// TAP
	Center         float64 `json:"center,omitempty"`
	HalfWidth      float64 `json:"half_width,omitempty"`
	MarkerPeriodMs int     `json:"marker_period_ms,omitempty"`

	// ORDER (presentation order, shuffled)
	Items []RefineOrderItem `json:"items,omitempty"`
}

type RefinePlanStep struct {
	I           int                  `json:"i"`
	Type        string               `json:"type"`
	Seed        string               `json:"seed"`
	Nonce       string               `json:"nonce"`
	WindowStart time.Time            `json:"window_start"`
	WindowEnd   time.Time            `json:"window_end"`
	Config      RefinePlanStepConfig `json:"config"`
}

type RefinePlan struct {
	Seed     string           `json:"seed"`
	N        int              `json:"n"`
	K        int              `json:"k"`
	GapMs    int64            `json:"gap_ms"`
	WindowMs int64            `json:"window_ms"`
	Steps    []RefinePlanStep `json:"steps"`
}

// refineDecisionRow is one row of refine_decisions (RefineDecision.csv).
type refineDecisionRow struct {
	ID           int
	Situation    string
	Options      []string // 2 or 3 entries
	CorrectIndex int
	NeutralIndex int
	Explanation  string
}

// buildRefinePlan creates the fixed schedule derived from started_at
// (deterministic per session seed). gap = duration / N; missing a window
// scores 0 — the schedule never shifts for offline players.
func buildRefinePlan(stage *models.RefineStageTemplate, recipe *models.RefiningRecipe, decisions []refineDecisionRow, startedAt, endsAt time.Time) (RefinePlan, error) {
	seedBytes := make([]byte, 32)
	if _, err := rand.Read(seedBytes); err != nil {
		return RefinePlan{}, fmt.Errorf("sinh seed phiên luyện: %w", err)
	}
	plan := RefinePlan{
		Seed:     hex.EncodeToString(seedBytes),
		N:        stage.StepCount,
		K:        stage.MinigameCount,
		GapMs:    int64(stage.DurationSeconds) * 1000 / int64(stage.StepCount),
		WindowMs: int64(stage.WindowSeconds) * 1000,
	}

	src := int64(binary.BigEndian.Uint64(seedBytes[:8]))
	if src < 0 {
		src = -src
	}
	rng := mrand.New(mrand.NewSource(src))

	n := plan.N

	stepTypes := make([]string, n)
	stepTypes[0] = RefineStepChoice
	stepTypes[n-1] = RefineStepTap

	middleCount := n - 2
	physicalCount := stage.MinigameCount - 1 // step N "khai lò" TAP counts into K
	choiceCount := middleCount - physicalCount
	if physicalCount < 0 || choiceCount < 0 || physicalCount+choiceCount != middleCount {
		return RefinePlan{}, fmt.Errorf("cấu hình nhịp độ rank %d: minigame_count không hợp lệ (N=%d, K=%d)", stage.Rank, n, stage.MinigameCount)
	}

	middle := make([]string, 0, middleCount)
	// Physical picks avoiding immediate repeats (weighted by stage config).
	lastType := ""
	for i := 0; i < physicalCount; i++ {
		pick := refineWeightedPick(rng, stage.MinigameWeights, lastType)
		middle = append(middle, pick)
		lastType = pick
	}
	for i := 0; i < choiceCount; i++ {
		middle = append(middle, RefineStepChoice)
	}

	// Shuffle middle with server seed (fixed schedule, reproducible on same seed).
	rng.Shuffle(len(middle), func(i, j int) { middle[i], middle[j] = middle[j], middle[i] })

	// Constraint repair (two phases, no ping-pong):
	// 1) adjacent physical minigames never share a type — relabel the
	//    conflicting slot to a type matching neither neighbor;
	//    the last middle slot also avoids TAP preemptively.
	// 2) step N-1 must not be TAP (step N is the "khai lò" TAP) — final fix
	//    that only relabels the last slot, so it cannot create new conflicts.
	for i := 1; i < len(middle); i++ {
		middle[i] = repairMiddleSlot(middle, i, stage, rng)
	}
	if len(middle) > 0 && middle[len(middle)-1] == RefineStepTap {
		// Prefer swapping with a middle CHOICE slot (no type constraints), then
		// relabel as a fallback (excluding TAP and the neighbor).
		swapWith := -1
		for j := 0; j < len(middle)-1; j++ {
			if middle[j] == RefineStepChoice {
				swapWith = j
				break
			}
		}
		if swapWith >= 0 {
			middle[len(middle)-1], middle[swapWith] = middle[swapWith], middle[len(middle)-1]
			// The moved TAP cannot sit next to another TAP by phase 1, but the
			// moved CHOICE can expose a new pair — run one more safe pass on the
			// affected prefix (never the last slot, which is CHOICE now).
			for i := 1; i < len(middle)-1; i++ {
				middle[i] = repairMiddleSlot(middle, i, stage, rng)
			}
		} else {
			last := len(middle) - 1
			neighbor := middle[last-1]
			weights := map[string]float64{}
			for ty, w := range stage.MinigameWeights {
				if ty != RefineStepTap && ty != neighbor {
					weights[ty] = w
				}
			}
			middle[last] = refineWeightedPick(rng, weights, "")
		}
	}

	copy(stepTypes[1:n-1], middle)

	choiceUsedEmpty := len(decisions) == 0
	if choiceUsedEmpty && choiceCount > 0 {
		// Fallback (WF-REF-04 §7.3): registry trống → CHOICE chuyển thành TAP
		// cùng độ khó, không chặn người chơi.
		logger.S.Warnf("refine decisions registry is empty; falling back CHOICE -> TAP")
		for i := range stepTypes {
			if stepTypes[i] == RefineStepChoice {
				stepTypes[i] = RefineStepTap
			}
		}
	}

	plan.Steps = make([]RefinePlanStep, n)
	for i := 0; i < n; i++ {
		stepSeed := make([]byte, 16)
		_, _ = rand.Read(stepSeed)
		nonceBytes := make([]byte, 16)
		_, _ = rand.Read(nonceBytes)
		windowStart := startedAt.Add(time.Duration(int64(i)) * time.Duration(plan.GapMs) * time.Millisecond)
		windowEnd := windowStart.Add(time.Duration(plan.WindowMs) * time.Millisecond)
		if windowEnd.After(endsAt) {
			windowEnd = endsAt
		}
		step := RefinePlanStep{
			I:           i + 1,
			Type:        stepTypes[i],
			Seed:        hex.EncodeToString(stepSeed),
			Nonce:       hex.EncodeToString(nonceBytes),
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
		}
		switch step.Type {
		case RefineStepChoice:
			step.Config = buildChoiceConfig(rng, decisions)
		case RefineStepSequence:
			symbols := make([]int, stage.SequenceLength)
			for j := range symbols {
				symbols[j] = rng.Intn(4)
			}
			step.Config = RefinePlanStepConfig{
				Symbols:       symbols,
				DisplayMs:     stage.SequenceDisplayMs,
				TargetInputMs: stage.SequenceLength * (stage.SequenceDisplayMs + stage.SequenceInputMs),
			}
		case RefineStepTap:
			center := 0.2 + rng.Float64()*0.6
			step.Config = RefinePlanStepConfig{
				Center:         center,
				HalfWidth:      stage.TapHalfWidth,
				MarkerPeriodMs: stage.TapPeriodMs,
			}
		case RefineStepOrder:
			step.Config = RefinePlanStepConfig{Items: buildOrderItems(rng, recipe, stage.OrderItemCount)}
		}
		plan.Steps[i] = step
	}
	return plan, nil
}

// debugRefinePlan enables temporary instrumentation of plan building.
var debugRefinePlan = len(os.Getenv("REFINE_PLAN_DEBUG")) > 0

// repairMiddleSlot relabels middle[index] when it duplicates the previous
// slot's physical type. The new type matches neither neighbor; the last
// middle slot additionally avoids TAP (step N is already TAP). Returns the
// (possibly unchanged) slot type.
func repairMiddleSlot(middle []string, index int, stage *models.RefineStageTemplate, rng *mrand.Rand) string {
	if middle[index] == RefineStepChoice || middle[index] != middle[index-1] {
		return middle[index]
	}
	exclude := map[string]bool{middle[index]: true}
	if index+1 < len(middle) && middle[index+1] != RefineStepChoice {
		exclude[middle[index+1]] = true
	}
	if index == len(middle)-1 {
		exclude[RefineStepTap] = true
	}
	weights := map[string]float64{}
	for t, w := range stage.MinigameWeights {
		if !exclude[t] {
			weights[t] = w
		}
	}
	return refineWeightedPick(rng, weights, "")
}

// refineWeightedPick chooses one of TAP/SEQUENCE/ORDER weighted by the stage
// config, excluding the previous type to help the no-repeat constraint.
func refineWeightedPick(rng *mrand.Rand, weights map[string]float64, exclude string) string {
	types := []string{RefineStepTap, RefineStepSequence, RefineStepOrder}
	// Candidates: types with positive weight (when the map is populated) that
	// aren't excluded. Empty map = all types allowed.
	allowed := make([]string, 0, len(types))
	for _, t := range types {
		if t == exclude {
			continue
		}
		if len(weights) > 0 {
			if w, ok := weights[t]; !ok || w <= 0 {
				continue
			}
		}
		allowed = append(allowed, t)
	}
	if len(allowed) == 0 {
		// Degenerate: everything excluded — pick the weighted-default type.
		allowed = append(types[:0:0], types...)
		if len(allowed) > 0 {
			return allowed[rng.Intn(len(allowed))]
		}
		return RefineStepTap
	}
	total := 0.0
	for _, t := range allowed {
		total += weights[t]
	}
	if total <= 0 {
		return allowed[rng.Intn(len(allowed))]
	}
	roll := rng.Float64() * total
	for _, t := range allowed {
		roll -= weights[t]
		if roll <= 0 {
			return t
		}
	}
	return allowed[len(allowed)-1]
}

func buildChoiceConfig(rng *mrand.Rand, decisions []refineDecisionRow) RefinePlanStepConfig {
	if len(decisions) == 0 {
		return RefinePlanStepConfig{}
	}
	row := decisions[rng.Intn(len(decisions))]
	optCount := len(row.Options)
	if optCount > 3 {
		optCount = 3
	}
	displayPerm := make([]int, optCount)
	for i := range displayPerm {
		displayPerm[i] = i
	}
	rng.Shuffle(len(displayPerm), func(i, j int) { displayPerm[i], displayPerm[j] = displayPerm[j], displayPerm[i] })
	texts := make([]string, optCount)
	correctDisplay, neutralDisplay := 0, -1
	for pos, orig := range displayPerm {
		texts[pos] = row.Options[orig]
		if orig == row.CorrectIndex {
			correctDisplay = pos
		}
		if orig == row.NeutralIndex {
			neutralDisplay = pos
		}
	}
	return RefinePlanStepConfig{
		DecisionID:          row.ID,
		Situation:           row.Situation,
		OptionOrder:         displayPerm,
		OptionTexts:         texts,
		CorrectDisplayIndex: correctDisplay,
		NeutralDisplayIndex: neutralDisplay,
		Explanation:         row.Explanation,
	}
}

// buildOrderItems assembles M Ghép Liệu tokens: recipe materials first, then
// same-path registry items; presentation order is shuffled by seed. The
// correct answer (rank asc, id asc) is derived at scoring time.
func buildOrderItems(rng *mrand.Rand, recipe *models.RefiningRecipe, count int) []RefineOrderItem {
	items := make([]RefineOrderItem, 0, count)
	usedIDs := map[string]bool{}
	if recipe != nil {
		for matID := range recipe.RequiredMaterials {
			if len(items) >= count {
				break
			}
			if usedIDs[matID] {
				continue
			}
			spec, ok := models.ItemRegistry[matID]
			if !ok {
				continue
			}
			items = append(items, RefineOrderItem{ID: spec.ID, Name: spec.Name, Icon: spec.ImageURL, Rank: spec.Rank})
			usedIDs[matID] = true
		}
	}
	// Registry needs deterministic iteration for snapshot stability; sort ids.
	ids := make([]string, 0, len(models.ItemRegistry))
	for id := range models.ItemRegistry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if len(items) >= count {
			break
		}
		if usedIDs[id] {
			continue
		}
		spec := models.ItemRegistry[id]
		if spec.Name == "" {
			continue
		}
		items = append(items, RefineOrderItem{ID: spec.ID, Name: spec.Name, Icon: spec.ImageURL, Rank: spec.Rank})
		usedIDs[id] = true
	}
	rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	return items
}

// refineRowsQuerier accepts *sql.Tx / *sql.DB for multi-row loads.
type refineRowsQuerier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// loadRefineDecisionsDB reads the CHOICE scenario registry (refine_decisions)
// each time a session plan is built; an empty registry makes CHOICE steps fall
// back to TAP (WF-REF-04 §7.3).
func loadRefineDecisionsDB(q refineRowsQuerier) ([]refineDecisionRow, error) {
	if q == nil {
		return nil, nil
	}
	rows, err := q.Query(`SELECT id, situation, option_1, option_2, COALESCE(option_3,''),
		correct_index, neutral_index, COALESCE(explanation,'')
		FROM refine_decisions ORDER BY id`)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var out []refineDecisionRow
	for rows.Next() {
		var r refineDecisionRow
		var situation, opt1, opt2, opt3, explanation string
		if err := rows.Scan(&r.ID, &situation, &opt1, &opt2, &opt3, &r.CorrectIndex, &r.NeutralIndex, &explanation); err != nil {
			return nil, err
		}
		opts := []string{opt1, opt2}
		if strings.TrimSpace(opt3) != "" {
			opts = append(opts, opt3)
		}
		r = refineDecisionRow{
			ID:           r.ID,
			Situation:    situation,
			Options:      opts,
			CorrectIndex: r.CorrectIndex,
			NeutralIndex: r.NeutralIndex,
			Explanation:  explanation,
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
