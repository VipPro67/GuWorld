package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"sort"
	"strconv"
)

type guSpec struct {
	id   string
	rank int
	path []string
}

type generatedRecipe struct {
	TargetGuID    string `json:"target_gu_id"`
	TargetRank    int    `json:"target_rank"`
	Variant       int    `json:"variant"`
	RequiredGus   string `json:"required_gus"`
	RequiredMatls string `json:"required_materials"`
	SuccessRate   string `json:"success_rate"`
	PrimevalCost  string `json:"primeval_stones_cost"`
	ImmortalCost  string `json:"immortal_essence_stones_cost"`
	NeedsGuID     bool   `json:"needs_gu_id,omitempty"`
}

// cmdGenRecipe generates missing refinement recipe rows. The unit is the
// (id, rank) pair: every Gu definition without a matching recipe row gets a
// draft. Success rate, costs, material quantities and the required-Gu quantity
// are all driven by -config.
func cmdGenRecipe(args []string) error {
	fs := flag.NewFlagSet("gen-recipe", flag.ExitOnError)
	data := fs.String("data", "data", "data directory")
	cfgPath := fs.String("config", "", "optional JSON config")
	write := fs.Bool("write", false, "write merged Recipe.csv to -output")
	_ = fs.String("output", "", "output path (required with -write)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		return err
	}
	dir := *data
	src := dir + "/Recipe.csv"
	outPath, err := requireWriteOutput(fs, src)
	if err != nil {
		return err
	}

	gus, err := loadGuSpecs(dir + "/Gu.csv")
	if err != nil {
		return err
	}
	items, err := loadItems(dir + "/Item.csv")
	if err != nil {
		return err
	}
	materials := map[string]itemRef{}
	for id, ref := range items {
		if ref.itemType == "Material" {
			materials[id] = ref
		}
	}

	recipeTable, err := readCSV(src)
	if err != nil {
		return err
	}
	existing := map[string]int{} // target "id\x00rank" -> max variant
	for _, row := range recipeTable.rows {
		id := recipeTable.get(row, "target_gu_id")
		rank := recipeTable.get(row, "target_rank")
		v, _ := strconv.Atoi(recipeTable.get(row, "variant"))
		key := id + "\x00" + rank
		if v > existing[key] {
			existing[key] = v
		}
	}

	// All (id, rank) pairs present in Gu.csv but missing a recipe row.
	type pair struct {
		id   string
		rank int
	}
	var missing []pair
	seen := map[string]bool{}
	for _, spec := range gus {
		key := spec.id + "\x00" + strconv.Itoa(spec.rank)
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, ok := existing[key]; !ok {
			missing = append(missing, pair{id: spec.id, rank: spec.rank})
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].rank != missing[j].rank {
			return missing[i].rank < missing[j].rank
		}
		return missing[i].id < missing[j].id
	})

	var generated []generatedRecipe
	for _, m := range missing {
		spec := gus[m.id+"\x00"+strconv.Itoa(m.rank)]
		variant := existing[m.id+"\x00"+strconv.Itoa(m.rank)] + 1

		requiredGus := "{}"
		needsGu := false
		if pred, ok := cfg.Recipe.RequiredGu[m.id]; ok && pred != "" {
			requiredGus = marshalGuReq(map[string]guReq{pred: {Qty: cfg.Recipe.RequiredGuQty, Rank: m.rank + cfg.Recipe.RequiredGuRankDelta}})
		} else if cfg.Recipe.RequiredGuQty > 0 {
			// Same-id predecessor at targetRank+delta, if it exists.
			predRank := m.rank + cfg.Recipe.RequiredGuRankDelta
			if _, ok := gus[m.id+"\x00"+strconv.Itoa(predRank)]; ok {
				requiredGus = marshalGuReq(map[string]guReq{m.id: {Qty: cfg.Recipe.RequiredGuQty, Rank: predRank}})
			} else if m.rank > 1 {
				// Rank 1 has no predecessor by design; only higher ranks need one.
				needsGu = true
			}
		}

		qty := math.Round(cfg.Recipe.MaterialQty * cfg.Recipe.MaterialQtyScale)
		if qty < 1 {
			qty = 1
		}
		requiredMatls := selectMaterials(spec.path, m.rank, materials, cfg.Recipe.MaterialCount, int(qty))

		generated = append(generated, generatedRecipe{
			TargetGuID:    m.id,
			TargetRank:    m.rank,
			Variant:       variant,
			RequiredGus:   requiredGus,
			RequiredMatls: marshalIntMap(requiredMatls),
			SuccessRate:   formatFloat(cfg.Recipe.SuccessRate[strconv.Itoa(m.rank)]),
			PrimevalCost:  fmt.Sprintf("%.3f", cfg.Recipe.PrimevalCost[strconv.Itoa(m.rank)]),
			ImmortalCost:  fmt.Sprintf("%.3f", cfg.Recipe.ImmortalCost[strconv.Itoa(m.rank)]),
			NeedsGuID:     needsGu,
		})
	}

	if *write {
		for _, g := range generated {
			recipeTable.rows = append(recipeTable.rows, []string{
				g.TargetGuID, strconv.Itoa(g.TargetRank), strconv.Itoa(g.Variant),
				g.RequiredGus, g.RequiredMatls, g.SuccessRate, g.PrimevalCost, g.ImmortalCost,
			})
		}
		if err := recipeTable.write(outPath); err != nil {
			return err
		}
	}

	report := struct {
		Summary struct {
			Missing   int `json:"missing"`
			Generated int `json:"generated"`
			NeedsGuID int `json:"needs_gu_id"`
		} `json:"summary"`
		Recipes []generatedRecipe `json:"recipes"`
	}{}
	report.Summary.Missing = len(missing)
	report.Summary.Generated = len(generated)
	for _, g := range generated {
		if g.NeedsGuID {
			report.Summary.NeedsGuID++
		}
	}
	report.Recipes = generated
	return printJSON(report)
}

func loadGuSpecs(path string) (map[string]guSpec, error) {
	t, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	out := map[string]guSpec{}
	for _, row := range t.rows {
		id := t.get(row, "id")
		rank, _ := strconv.Atoi(t.get(row, "rank"))
		key := id + "\x00" + strconv.Itoa(rank)
		out[key] = guSpec{id: id, rank: rank, path: jsonArray(t.get(row, "path"))}
	}
	return out, nil
}

type guReq struct {
	Qty  int `json:"qty"`
	Rank int `json:"rank"`
}

func marshalGuReq(m map[string]guReq) string {
	if len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func marshalIntMap(m map[string]int) string {
	if len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func selectMaterials(guPath []string, guRank int, materials map[string]itemRef, count int, qty int) map[string]int {
	type cand struct {
		id    string
		score int
	}
	var cands []cand
	for id, ref := range materials {
		score := 0
		if ref.rank == guRank {
			score += 10000
		}
		if overlap(guPath, ref.path) {
			score += 100
		}
		score -= abs(ref.rank-guRank) * 10
		if ref.rank >= guRank {
			score += 1
		}
		cands = append(cands, cand{id: id, score: score})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		return cands[i].id < cands[j].id
	})
	out := map[string]int{}
	for i := 0; i < count && i < len(cands); i++ {
		out[cands[i].id] = qty
	}
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
