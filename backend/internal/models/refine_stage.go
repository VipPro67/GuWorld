package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// ErrRefineStageMissing is returned when refine_stage_templates has no row for
// the requested rank (Admin has not imported / deleted the config).
var ErrRefineStageMissing = fmt.Errorf("refine stage template missing")

// RefineStageTemplate is the data-driven pace/difficulty configuration for a
// refine session of a given target rank (WF-REF-06). Read fresh from DB on
// every session create; running sessions keep their snapshot.
type RefineStageTemplate struct {
	Rank                 int                `json:"rank"`
	DurationSeconds      int                `json:"duration_seconds"`
	StepCount            int                `json:"step_count"`
	MinigameCount        int                `json:"minigame_count"`
	WindowSeconds        int                `json:"window_seconds"`
	AccuracyMaxBonus     float64            `json:"accuracy_max_bonus"`
	DeathPenalty         float64            `json:"death_penalty"`
	MasteryMultiplierMax float64            `json:"mastery_multiplier_max"`
	TapHalfWidth         float64            `json:"tap_half_width"`
	TapPeriodMs          int                `json:"tap_period_ms"`
	SequenceLength       int                `json:"sequence_length"`
	SequenceDisplayMs    int                `json:"sequence_display_ms"`
	SequenceInputMs      int                `json:"sequence_input_ms"`
	OrderItemCount       int                `json:"order_item_count"`
	ChoiceOptionCount    int                `json:"choice_option_count"`
	MinigameWeights      map[string]float64 `json:"minigame_weights"`
}

type queryRower interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// LoadRefineStageTemplate queries one row by rank. No cache: sessions snapshot
// the values at create time, so late Admin imports only affect new sessions.
func LoadRefineStageTemplate(q queryRower, rank int) (*RefineStageTemplate, error) {
	if q == nil {
		return nil, fmt.Errorf("nil queryer")
	}
	var st RefineStageTemplate
	var weightsRaw []byte
	err := q.QueryRow(`
		SELECT rank, duration_seconds, step_count, minigame_count, window_seconds,
		       accuracy_max_bonus, death_penalty, mastery_multiplier_max,
		       tap_half_width, tap_period_ms, sequence_length, sequence_display_ms,
		       sequence_input_ms, order_item_count, choice_option_count,
		       COALESCE(minigame_weights, '{}'::jsonb)
		FROM refine_stage_templates
		WHERE rank = $1`, rank).Scan(
		&st.Rank, &st.DurationSeconds, &st.StepCount, &st.MinigameCount, &st.WindowSeconds,
		&st.AccuracyMaxBonus, &st.DeathPenalty, &st.MasteryMultiplierMax,
		&st.TapHalfWidth, &st.TapPeriodMs, &st.SequenceLength, &st.SequenceDisplayMs,
		&st.SequenceInputMs, &st.OrderItemCount, &st.ChoiceOptionCount,
		&weightsRaw,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRefineStageMissing
		}
		return nil, err
	}
	st.MinigameWeights = map[string]float64{"TAP": 1, "SEQUENCE": 1, "ORDER": 1}
	if len(weightsRaw) > 0 {
		weights := map[string]float64{}
		if err := json.Unmarshal(weightsRaw, &weights); err == nil && len(weights) > 0 {
			st.MinigameWeights = weights
		}
	}
	return &st, nil
}
