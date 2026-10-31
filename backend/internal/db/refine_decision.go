package db

import (
	"fmt"
	"io"
	"strings"
)

// RefineDecision.csv -> refine_decisions (WF-REF-04 §7): generic CHOICE
// scenario rows for refine sessions. Imported via Admin; empty registry falls
// back CHOICE -> TAP at plan generation.

type refineDecisionRowDB struct {
	ID         int
	Situation  string
	Option1    string
	Option2    string
	Option3    string
	CorrectIdx int
	NeutralIdx int
	Explain    string
}

func parseRefineDecisionCSV(filename string) ([]refineDecisionRowDB, error) {
	f, reader, header, err := openCSV(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\uFEFF")
	}
	idx := func(name string) int { return colIndex(header, name) }
	cols := map[string]int{
		"id": idx("id"), "situation": idx("situation"),
		"option_1": idx("option_1"), "option_2": idx("option_2"), "option_3": idx("option_3"),
		"correct_index": idx("correct_index"), "neutral_index": idx("neutral_index"),
		"explanation": idx("explanation"),
	}
	for _, name := range []string{"id", "situation", "option_1", "option_2", "correct_index", "neutral_index"} {
		if cols[name] < 0 {
			return nil, fmt.Errorf("RefineDecision.csv missing column %s", name)
		}
	}

	var rows []refineDecisionRowDB
	line := 1
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("RefineDecision.csv row %d: %w", line+1, readErr)
		}
		line++
		label := fmt.Sprintf("RefineDecision.csv line %d", line)
		row := refineDecisionRowDB{}
		row.ID, err = csvInt(record, cols["id"], label)
		if err != nil {
			return nil, err
		}
		get := func(name string) string {
			i := cols[name]
			if i < 0 || i >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[i])
		}
		row.Situation = get("situation")
		row.Option1 = get("option_1")
		row.Option2 = get("option_2")
		row.Option3 = get("option_3")
		row.CorrectIdx, err = csvInt(record, cols["correct_index"], label)
		if err != nil {
			return nil, err
		}
		if i := cols["neutral_index"]; i >= 0 && i < len(record) && strings.TrimSpace(record[i]) != "" {
			row.NeutralIdx, err = csvInt(record, i, label)
			if err != nil {
				return nil, err
			}
		} else {
			row.NeutralIdx = -1
		}
		row.Explain = get("explanation")
		rows = append(rows, row)
	}
	return rows, nil
}

func validateRefineDecisionRows(rows []refineDecisionRowDB) error {
	seen := map[int]bool{}
	for _, row := range rows {
		if seen[row.ID] {
			return fmt.Errorf("RefineDecision.csv: id %d bị trùng", row.ID)
		}
		seen[row.ID] = true
		if row.Situation == "" || row.Option1 == "" || row.Option2 == "" {
			return fmt.Errorf("RefineDecision.csv: id %d thiếu situation/option", row.ID)
		}
		optCount := 2
		if row.Option3 != "" {
			optCount = 3
		}
		if row.CorrectIdx < 0 || row.CorrectIdx >= optCount {
			return fmt.Errorf("RefineDecision.csv: id %d correct_index ngoài số option", row.ID)
		}
		if row.NeutralIdx != -1 && (row.NeutralIdx < 0 || row.NeutralIdx >= optCount) {
			return fmt.Errorf("RefineDecision.csv: id %d neutral_index ngoài số option", row.ID)
		}
	}
	return nil
}

func preflightRefineDecisions() error {
	rows, err := parseRefineDecisionCSV("RefineDecision.csv")
	if err != nil {
		return err
	}
	return validateRefineDecisionRows(rows)
}

func seedRefineDecisionsFromCSV(filename string) error {
	rows, err := parseRefineDecisionCSV(filename)
	if err != nil {
		return err
	}
	if err := validateRefineDecisionRows(rows); err != nil {
		return err
	}
	executor := activeCSVImportDB()
	for _, row := range rows {
		if _, err := executor.Exec(`
			INSERT INTO refine_decisions (id, situation, option_1, option_2, option_3, correct_index, neutral_index, explanation)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (id) DO UPDATE SET
				situation = EXCLUDED.situation,
				option_1 = EXCLUDED.option_1,
				option_2 = EXCLUDED.option_2,
				option_3 = EXCLUDED.option_3,
				correct_index = EXCLUDED.correct_index,
				neutral_index = EXCLUDED.neutral_index,
				explanation = EXCLUDED.explanation`,
			row.ID, row.Situation, row.Option1, row.Option2, row.Option3, row.CorrectIdx, row.NeutralIdx, row.Explain,
		); err != nil {
			return fmt.Errorf("upsert refine_decisions id %d: %w", row.ID, err)
		}
	}
	return nil
}
