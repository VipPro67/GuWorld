package main

import (
	"encoding/json"
	"flag"
	"sort"
	"strconv"
	"strings"
)

var pseudoFoods = map[string]bool{
	"primeval_stone": true,
	"immortal_stone": true,
	"blood":          true,
	"thought":        true,
	"soul":           true,
	"lifespan":       true,
}

type itemRef struct {
	rank     int
	path     []string
	itemType string
}

// cmdMatchFood audits Gu food_types against Item.csv and Gu.csv: it flags
// dangling references (food that no longer exists) and low-nutrition food
// (rank below the Gu's rank). With -write it remaps dangling references to a
// path/rank-matched Food (or Material) and writes the corrected Gu.csv.
func cmdMatchFood(args []string) error {
	fs := flag.NewFlagSet("match-food", flag.ExitOnError)
	data := fs.String("data", "data", "data directory")
	write := fs.Bool("write", false, "write corrected Gu.csv to -output")
	_ = fs.String("output", "", "output path (required with -write)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := *data
	src := dir + "/Gu.csv"
	outPath, err := requireWriteOutput(fs, src)
	if err != nil {
		return err
	}

	items, err := loadItems(dir + "/Item.csv")
	if err != nil {
		return err
	}
	guIDs, err := loadGuIDSet(dir + "/Gu.csv")
	if err != nil {
		return err
	}

	gu, err := readCSV(src)
	if err != nil {
		return err
	}

	type dangling struct {
		GuID       string `json:"gu_id"`
		Rank       int    `json:"rank"`
		Food       string `json:"food"`
		Suggestion string `json:"suggestion,omitempty"`
	}

	var danglingRows []dangling
	lowCount := 0
	replaced, unresolved := 0, 0

	for i, row := range gu.rows {
		id := gu.get(row, "id")
		rank, _ := strconv.Atoi(gu.get(row, "rank"))
		path := jsonArray(gu.get(row, "path"))
		foods := jsonArray(gu.get(row, "food_types"))
		if len(foods) == 0 {
			continue
		}
		newFoods := make([]string, 0, len(foods))
		for _, food := range foods {
			kind, foodRank := classifyFood(food, guIDs, items)
			switch kind {
			case "ok":
				newFoods = append(newFoods, food)
				if foodRank > 0 && foodRank < rank {
					lowCount++
				}
			case "dangling":
				suggestion := suggestFood(path, rank, items)
				danglingRows = append(danglingRows, dangling{GuID: id, Rank: rank, Food: food, Suggestion: suggestion})
				if *write && suggestion != "" && !contains(newFoods, suggestion) {
					newFoods = append(newFoods, suggestion)
					replaced++
				} else if *write && suggestion == "" {
					unresolved++
				}
			}
		}
		if *write {
			gu.set(gu.rows[i], "food_types", marshalStringArray(newFoods))
		}
	}

	if *write {
		if err := gu.write(outPath); err != nil {
			return err
		}
	}

	report := struct {
		Summary struct {
			Gu           int `json:"gu"`
			Dangling     int `json:"dangling"`
			LowNutrition int `json:"low_nutrition"`
			Replaced     int `json:"replaced"`
			Unresolved   int `json:"unresolved"`
		} `json:"summary"`
		Dangling []dangling `json:"dangling"`
	}{}
	report.Summary.Gu = len(gu.rows)
	report.Summary.Dangling = len(danglingRows)
	report.Summary.LowNutrition = lowCount
	report.Summary.Replaced = replaced
	report.Summary.Unresolved = unresolved
	report.Dangling = danglingRows
	return printJSON(report)
}

func loadItems(path string) (map[string]itemRef, error) {
	t, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	out := make(map[string]itemRef, len(t.rows))
	for _, row := range t.rows {
		id := t.get(row, "id")
		rank, _ := strconv.Atoi(t.get(row, "rank"))
		out[id] = itemRef{rank: rank, path: jsonArray(t.get(row, "path")), itemType: t.get(row, "item_type")}
	}
	return out, nil
}

func loadGuIDSet(path string) (map[string]bool, error) {
	t, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(t.rows))
	for _, row := range t.rows {
		out[t.get(row, "id")] = true
	}
	return out, nil
}

func classifyFood(food string, guIDs map[string]bool, items map[string]itemRef) (kind string, rank int) {
	if pseudoFoods[food] {
		return "ok", 0
	}
	if strings.HasPrefix(food, "GU_") {
		if guIDs[food] {
			return "ok", 0
		}
		return "dangling", 0
	}
	if it, ok := items[food]; ok {
		return "ok", it.rank
	}
	return "dangling", 0
}

// suggestFood picks a replacement for a dangling food reference: prefer a Food
// item of the Gu's rank with a shared path, then any Food at >= rank, then the
// highest-rank Food. Returns "" when no candidate exists. Ties break on id so
// the result is deterministic.
func suggestFood(guPath []string, guRank int, items map[string]itemRef) string {
	type cand struct {
		id    string
		ref   itemRef
		score int
	}
	var cands []cand
	for id, ref := range items {
		if ref.itemType != "Food" && ref.itemType != "Material" {
			continue
		}
		score := 0
		if ref.itemType == "Food" {
			score += 1000
		}
		if overlap(guPath, ref.path) {
			score += 100
		}
		score -= abs(ref.rank - guRank)
		cands = append(cands, cand{id: id, ref: ref, score: score})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		return cands[i].id < cands[j].id
	})
	for _, c := range cands {
		if c.ref.rank >= guRank {
			return c.id
		}
	}
	if len(cands) == 0 {
		return ""
	}
	return cands[0].id
}

func overlap(a, b []string) bool {
	set := map[string]bool{}
	for _, s := range a {
		set[s] = true
	}
	for _, s := range b {
		if set[s] {
			return true
		}
	}
	return false
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func marshalStringArray(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	sort.Strings(ss)
	b, _ := json.Marshal(ss)
	return string(b)
}
