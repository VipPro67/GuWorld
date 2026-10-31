package db

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Data-driven refine session pacing config (WF-REF-06): RefineStage.csv ->
// refine_stage_templates. Baseline is seeded by Go in schemaRefine(); Admin
// import overrides. Sessions snapshot config at create time.

type refineStageRow struct {
	Rank                 int
	DurationSeconds      int
	StepCount            int
	MinigameCount        int
	WindowSeconds        int
	AccuracyMaxBonus     float64
	DeathPenalty         float64
	MasteryMultiplierMax float64
	TapHalfWidth         float64
	TapPeriodMs          int
	SequenceLength       int
	SequenceDisplayMs    int
	SequenceInputMs      int
	OrderItemCount       int
	ChoiceOptionCount    int
	MinigameWeights      map[string]float64
}

// defaultMinigameWeights is the baseline JSON shape for minigame_weights.
func defaultMinigameWeights() map[string]float64 {
	return map[string]float64{"TAP": 1, "SEQUENCE": 1, "ORDER": 1}
}

func parseRefineStageCSV(filename string) ([]refineStageRow, error) {
	f, reader, header, err := openCSV(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\uFEFF")
	}

	cols := map[string]int{
		"rank": colIndex(header, "rank"), "duration_seconds": colIndex(header, "duration_seconds"),
		"step_count": colIndex(header, "step_count"), "minigame_count": colIndex(header, "minigame_count"),
		"window_seconds": colIndex(header, "window_seconds"), "accuracy_max_bonus": colIndex(header, "accuracy_max_bonus"),
		"death_penalty": colIndex(header, "death_penalty"), "mastery_multiplier_max": colIndex(header, "mastery_multiplier_max"),
		"tap_half_width": colIndex(header, "tap_half_width"), "tap_period_ms": colIndex(header, "tap_period_ms"),
		"sequence_length": colIndex(header, "sequence_length"), "sequence_display_ms": colIndex(header, "sequence_display_ms"),
		"sequence_input_ms": colIndex(header, "sequence_input_ms"), "order_item_count": colIndex(header, "order_item_count"),
		"choice_option_count": colIndex(header, "choice_option_count"), "minigame_weights": colIndex(header, "minigame_weights"),
	}
	for name, idx := range cols {
		if name == "minigame_weights" {
			continue // optional column; empty -> baseline weights
		}
		if idx < 0 {
			return nil, fmt.Errorf("RefineStage.csv missing column %s", name)
		}
	}

	var rows []refineStageRow
	line := 1
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("RefineStage.csv row %d: %w", line+1, readErr)
		}
		line++
		rowLabel := fmt.Sprintf("RefineStage.csv line %d", line)
		row := refineStageRow{}
		if row.Rank, err = csvInt(record, cols["rank"], rowLabel); err != nil {
			return nil, err
		}
		if row.DurationSeconds, err = csvInt(record, cols["duration_seconds"], rowLabel); err != nil {
			return nil, err
		}
		if row.StepCount, err = csvInt(record, cols["step_count"], rowLabel); err != nil {
			return nil, err
		}
		if row.MinigameCount, err = csvInt(record, cols["minigame_count"], rowLabel); err != nil {
			return nil, err
		}
		if row.WindowSeconds, err = csvInt(record, cols["window_seconds"], rowLabel); err != nil {
			return nil, err
		}
		if row.AccuracyMaxBonus, err = csvFloat(record, cols["accuracy_max_bonus"], rowLabel); err != nil {
			return nil, err
		}
		if row.DeathPenalty, err = csvFloat(record, cols["death_penalty"], rowLabel); err != nil {
			return nil, err
		}
		if row.MasteryMultiplierMax, err = csvFloat(record, cols["mastery_multiplier_max"], rowLabel); err != nil {
			return nil, err
		}
		if row.TapHalfWidth, err = csvFloat(record, cols["tap_half_width"], rowLabel); err != nil {
			return nil, err
		}
		if row.TapPeriodMs, err = csvInt(record, cols["tap_period_ms"], rowLabel); err != nil {
			return nil, err
		}
		if row.SequenceLength, err = csvInt(record, cols["sequence_length"], rowLabel); err != nil {
			return nil, err
		}
		if row.SequenceDisplayMs, err = csvInt(record, cols["sequence_display_ms"], rowLabel); err != nil {
			return nil, err
		}
		if row.SequenceInputMs, err = csvInt(record, cols["sequence_input_ms"], rowLabel); err != nil {
			return nil, err
		}
		if row.OrderItemCount, err = csvInt(record, cols["order_item_count"], rowLabel); err != nil {
			return nil, err
		}
		if row.ChoiceOptionCount, err = csvInt(record, cols["choice_option_count"], rowLabel); err != nil {
			return nil, err
		}
		row.MinigameWeights = defaultMinigameWeights()
		if idx := cols["minigame_weights"]; idx >= 0 && idx < len(record) && strings.TrimSpace(record[idx]) != "" {
			weights := map[string]float64{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(record[idx])), &weights); err != nil {
				return nil, fmt.Errorf("%s: invalid minigame_weights JSON: %w", rowLabel, err)
			}
			row.MinigameWeights = weights
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func csvInt(record []string, idx int, rowLabel string) (int, error) {
	if idx < 0 || idx >= len(record) {
		return 0, fmt.Errorf("%s: missing column value", rowLabel)
	}
	v, err := strconv.Atoi(strings.TrimSpace(record[idx]))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", rowLabel, err)
	}
	return v, nil
}

func csvFloat(record []string, idx int, rowLabel string) (float64, error) {
	if idx < 0 || idx >= len(record) {
		return 0, fmt.Errorf("%s: missing column value", rowLabel)
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(record[idx]), 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", rowLabel, err)
	}
	return v, nil
}

// validateRefineStageRows enforces the WF-REF-06 §6.2 business rules.
func validateRefineStageRows(rows []refineStageRow) error {
	seen := map[int]bool{}
	for _, row := range rows {
		if row.Rank < 1 || row.Rank > 9 {
			return fmt.Errorf("RefineStage.csv: rank %d ngoài dải 1–9", row.Rank)
		}
		if seen[row.Rank] {
			return fmt.Errorf("RefineStage.csv: rank %d bị trùng", row.Rank)
		}
		seen[row.Rank] = true
		label := fmt.Sprintf("rank %d", row.Rank)
		if row.DurationSeconds <= 0 {
			return fmt.Errorf("RefineStage.csv %s: duration_seconds phải > 0", label)
		}
		if row.StepCount < 3 || row.StepCount > 12 {
			return fmt.Errorf("RefineStage.csv %s: step_count phải trong 3..12", label)
		}
		if row.MinigameCount < 1 || row.MinigameCount > row.StepCount-1 {
			return fmt.Errorf("RefineStage.csv %s: minigame_count phải trong 1..step_count-1", label)
		}
		if row.WindowSeconds <= 0 {
			return fmt.Errorf("RefineStage.csv %s: window_seconds phải > 0", label)
		}
		if float64(row.WindowSeconds) > float64(row.DurationSeconds)/float64(row.StepCount) {
			return fmt.Errorf("RefineStage.csv %s: window_seconds chồng lên bước kế tiếp (window > duration/step)", label)
		}
		if row.AccuracyMaxBonus < 0 || row.AccuracyMaxBonus > 1 {
			return fmt.Errorf("RefineStage.csv %s: accuracy_max_bonus phải trong 0..1", label)
		}
		if row.DeathPenalty < 0 || row.DeathPenalty > 1 {
			return fmt.Errorf("RefineStage.csv %s: death_penalty phải trong 0..1", label)
		}
		if row.MasteryMultiplierMax < 1 {
			return fmt.Errorf("RefineStage.csv %s: mastery_multiplier_max phải >= 1", label)
		}
		if row.TapHalfWidth <= 0 || row.TapHalfWidth > 0.5 {
			return fmt.Errorf("RefineStage.csv %s: tap_half_width phải trong (0..0.5]", label)
		}
		if row.TapPeriodMs < 300 {
			return fmt.Errorf("RefineStage.csv %s: tap_period_ms phải >= 300", label)
		}
		if row.SequenceLength < 2 || row.SequenceLength > 10 {
			return fmt.Errorf("RefineStage.csv %s: sequence_length phải trong 2..10", label)
		}
		if row.SequenceDisplayMs < 100 || row.SequenceDisplayMs > 2000 {
			return fmt.Errorf("RefineStage.csv %s: sequence_display_ms phải trong 100..2000", label)
		}
		if row.SequenceInputMs < 200 || row.SequenceInputMs > 5000 {
			return fmt.Errorf("RefineStage.csv %s: sequence_input_ms phải trong 200..5000", label)
		}
		if row.OrderItemCount < 2 || row.OrderItemCount > 10 {
			return fmt.Errorf("RefineStage.csv %s: order_item_count phải trong 2..10", label)
		}
		if row.ChoiceOptionCount < 2 || row.ChoiceOptionCount > 3 {
			return fmt.Errorf("RefineStage.csv %s: choice_option_count phải trong 2..3", label)
		}
		total := 0.0
		for key, weight := range row.MinigameWeights {
			switch key {
			case "TAP", "SEQUENCE", "ORDER":
			default:
				return fmt.Errorf("RefineStage.csv %s: minigame_weights có khóa lạ %q", label, key)
			}
			if weight < 0 {
				return fmt.Errorf("RefineStage.csv %s: minigame_weights[%s] âm", label, key)
			}
			total += weight
		}
		if total <= 0 {
			return fmt.Errorf("RefineStage.csv %s: tổng minigame_weights phải > 0", label)
		}
	}
	for rank := 1; rank <= 9; rank++ {
		if !seen[rank] {
			return fmt.Errorf("RefineStage.csv: thiếu rank %d", rank)
		}
	}
	return nil
}

// preflightRefineStages validates the current RefineStage.csv without writing.
func preflightRefineStages() error {
	rows, err := parseRefineStageCSV("RefineStage.csv")
	if err != nil {
		return err
	}
	return validateRefineStageRows(rows)
}

// seedRefineStagesFromCSV upserts every rank row from RefineStage.csv.
// Runs inside the admin import transaction when one is active.
func seedRefineStagesFromCSV(filename string) error {
	rows, err := parseRefineStageCSV(filename)
	if err != nil {
		return err
	}
	if err := validateRefineStageRows(rows); err != nil {
		return err
	}
	executor := activeCSVImportDB()
	for _, row := range rows {
		weightsRaw, err := json.Marshal(row.MinigameWeights)
		if err != nil {
			return fmt.Errorf("marshal minigame_weights rank %d: %w", row.Rank, err)
		}
		if _, err := executor.Exec(`
			INSERT INTO refine_stage_templates (
				rank, duration_seconds, step_count, minigame_count, window_seconds,
				accuracy_max_bonus, death_penalty, mastery_multiplier_max,
				tap_half_width, tap_period_ms, sequence_length, sequence_display_ms,
				sequence_input_ms, order_item_count, choice_option_count, minigame_weights,
				updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NOW())
			ON CONFLICT (rank) DO UPDATE SET
				duration_seconds = EXCLUDED.duration_seconds,
				step_count = EXCLUDED.step_count,
				minigame_count = EXCLUDED.minigame_count,
				window_seconds = EXCLUDED.window_seconds,
				accuracy_max_bonus = EXCLUDED.accuracy_max_bonus,
				death_penalty = EXCLUDED.death_penalty,
				mastery_multiplier_max = EXCLUDED.mastery_multiplier_max,
				tap_half_width = EXCLUDED.tap_half_width,
				tap_period_ms = EXCLUDED.tap_period_ms,
				sequence_length = EXCLUDED.sequence_length,
				sequence_display_ms = EXCLUDED.sequence_display_ms,
				sequence_input_ms = EXCLUDED.sequence_input_ms,
				order_item_count = EXCLUDED.order_item_count,
				choice_option_count = EXCLUDED.choice_option_count,
				minigame_weights = EXCLUDED.minigame_weights,
				updated_at = NOW()`,
			row.Rank, row.DurationSeconds, row.StepCount, row.MinigameCount, row.WindowSeconds,
			row.AccuracyMaxBonus, row.DeathPenalty, row.MasteryMultiplierMax,
			row.TapHalfWidth, row.TapPeriodMs, row.SequenceLength, row.SequenceDisplayMs,
			row.SequenceInputMs, row.OrderItemCount, row.ChoiceOptionCount, string(weightsRaw),
		); err != nil {
			return fmt.Errorf("upsert refine_stage_templates rank %d: %w", row.Rank, err)
		}
	}
	return nil
}

// seedRefineStageBaseline seeds the 9 baseline rows (ON CONFLICT DO NOTHING)
// so the game runs before any Admin import.
func seedRefineStageBaseline() error {
	baseline := []refineStageRow{
		{Rank: 1, DurationSeconds: 60, StepCount: 3, MinigameCount: 1, WindowSeconds: 20, AccuracyMaxBonus: 0.08, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.12, TapPeriodMs: 2000, SequenceLength: 3, SequenceDisplayMs: 600, SequenceInputMs: 1500, OrderItemCount: 3, ChoiceOptionCount: 2},
		{Rank: 2, DurationSeconds: 120, StepCount: 3, MinigameCount: 1, WindowSeconds: 15, AccuracyMaxBonus: 0.08, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.12, TapPeriodMs: 2000, SequenceLength: 3, SequenceDisplayMs: 600, SequenceInputMs: 1500, OrderItemCount: 3, ChoiceOptionCount: 2},
		{Rank: 3, DurationSeconds: 300, StepCount: 4, MinigameCount: 2, WindowSeconds: 12, AccuracyMaxBonus: 0.07, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.10, TapPeriodMs: 1700, SequenceLength: 4, SequenceDisplayMs: 500, SequenceInputMs: 1300, OrderItemCount: 4, ChoiceOptionCount: 3},
		{Rank: 4, DurationSeconds: 600, StepCount: 4, MinigameCount: 2, WindowSeconds: 10, AccuracyMaxBonus: 0.07, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.10, TapPeriodMs: 1700, SequenceLength: 4, SequenceDisplayMs: 500, SequenceInputMs: 1300, OrderItemCount: 4, ChoiceOptionCount: 3},
		{Rank: 5, DurationSeconds: 1200, StepCount: 5, MinigameCount: 3, WindowSeconds: 8, AccuracyMaxBonus: 0.06, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.08, TapPeriodMs: 1400, SequenceLength: 5, SequenceDisplayMs: 450, SequenceInputMs: 1200, OrderItemCount: 5, ChoiceOptionCount: 3},
		{Rank: 6, DurationSeconds: 2700, StepCount: 5, MinigameCount: 3, WindowSeconds: 8, AccuracyMaxBonus: 0.05, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.08, TapPeriodMs: 1400, SequenceLength: 5, SequenceDisplayMs: 450, SequenceInputMs: 1200, OrderItemCount: 5, ChoiceOptionCount: 3},
		{Rank: 7, DurationSeconds: 5400, StepCount: 6, MinigameCount: 4, WindowSeconds: 6, AccuracyMaxBonus: 0.05, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.065, TapPeriodMs: 1100, SequenceLength: 6, SequenceDisplayMs: 380, SequenceInputMs: 1000, OrderItemCount: 6, ChoiceOptionCount: 3},
		{Rank: 8, DurationSeconds: 7200, StepCount: 6, MinigameCount: 4, WindowSeconds: 6, AccuracyMaxBonus: 0.04, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.065, TapPeriodMs: 1100, SequenceLength: 6, SequenceDisplayMs: 380, SequenceInputMs: 1000, OrderItemCount: 6, ChoiceOptionCount: 3},
		{Rank: 9, DurationSeconds: 14400, StepCount: 7, MinigameCount: 4, WindowSeconds: 5, AccuracyMaxBonus: 0.04, DeathPenalty: 0.25, MasteryMultiplierMax: 2.0, TapHalfWidth: 0.05, TapPeriodMs: 900, SequenceLength: 6, SequenceDisplayMs: 320, SequenceInputMs: 900, OrderItemCount: 6, ChoiceOptionCount: 3},
	}
	for _, row := range baseline {
		weightsRaw, err := json.Marshal(defaultMinigameWeights())
		if err != nil {
			return err
		}
		if _, err := schemaDB().Exec(`
			INSERT INTO refine_stage_templates (
				rank, duration_seconds, step_count, minigame_count, window_seconds,
				accuracy_max_bonus, death_penalty, mastery_multiplier_max,
				tap_half_width, tap_period_ms, sequence_length, sequence_display_ms,
				sequence_input_ms, order_item_count, choice_option_count, minigame_weights,
				updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NOW())
			ON CONFLICT (rank) DO NOTHING`,
			row.Rank, row.DurationSeconds, row.StepCount, row.MinigameCount, row.WindowSeconds,
			row.AccuracyMaxBonus, row.DeathPenalty, row.MasteryMultiplierMax,
			row.TapHalfWidth, row.TapPeriodMs, row.SequenceLength, row.SequenceDisplayMs,
			row.SequenceInputMs, row.OrderItemCount, row.ChoiceOptionCount, string(weightsRaw),
		); err != nil {
			return fmt.Errorf("seed refine_stage_templates rank %d: %w", row.Rank, err)
		}
	}
	return nil
}
