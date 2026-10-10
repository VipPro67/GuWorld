package main

import (
	"encoding/json"
	"flag"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Power components of a Gu's effects. Values are summed flat magnitudes:
//
//	damage   = Σ DAMAGE FLAT
//	shield   = Σ SHIELD FLAT
//	heal     = Σ HEAL FLAT
//	direct_hp= Σ DIRECT_HP FLAT
//	stat     = Σ STAT_MOD ADD_FLAT
//	status   = Σ STATUS value×turns
//
// This is a coarse heuristic (ponytail: multipliers/percent/UNIQUE hooks are
// deliberately ignored); it exists to find same-rank outliers, not to rank Gu
// absolutely.
type powerComponents struct {
	Damage   float64 `json:"damage"`
	Shield   float64 `json:"shield"`
	Heal     float64 `json:"heal"`
	DirectHP float64 `json:"direct_hp"`
	Stat     float64 `json:"stat"`
	Status   float64 `json:"status"`
}

func (p powerComponents) total(w map[string]float64) float64 {
	return p.Damage*wv(w, "damage") +
		p.Shield*wv(w, "shield") +
		p.Heal*wv(w, "heal") +
		p.DirectHP*wv(w, "direct_hp") +
		p.Stat*wv(w, "stat") +
		p.Status*wv(w, "status")
}

func wv(w map[string]float64, k string) float64 {
	if v, ok := w[k]; ok {
		return v
	}
	return 1
}

type parsedOp struct {
	op      string
	subType string
	value   float64
	hasVal  bool
	turns   float64
}

// parseEffects walks the block array (or {blocks:[...]} object) and returns the
// flat operation values, ignoring UNIQUE hooks and non-numeric fields.
func parseEffects(raw string) ([]parsedOp, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" || trimmed == "{}" {
		return nil, nil
	}
	var blocks []json.RawMessage
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &blocks); err != nil {
			return nil, err
		}
	} else {
		var root struct {
			Blocks []json.RawMessage `json:"blocks"`
		}
		if err := json.Unmarshal([]byte(trimmed), &root); err != nil {
			return nil, err
		}
		blocks = root.Blocks
	}
	var out []parsedOp
	for _, b := range blocks {
		var block struct {
			Operations []struct {
				Op      string          `json:"op"`
				SubType string          `json:"sub_type"`
				Value   json.RawMessage `json:"value"`
				Turns   json.RawMessage `json:"turns"`
			} `json:"operations"`
		}
		if err := json.Unmarshal(b, &block); err != nil {
			continue
		}
		for _, o := range block.Operations {
			v, has := toFloat(o.Value)
			turns := 1.0
			if tv, ok := toFloat(o.Turns); ok && tv > 0 {
				turns = tv
			}
			out = append(out, parsedOp{op: strings.ToUpper(o.Op), subType: strings.ToUpper(o.SubType), value: v, hasVal: has, turns: turns})
		}
	}
	return out, nil
}

func powerOf(raw string) (powerComponents, error) {
	ops, err := parseEffects(raw)
	if err != nil {
		return powerComponents{}, err
	}
	var p powerComponents
	for _, o := range ops {
		if !o.hasVal {
			continue
		}
		switch o.op {
		case "DAMAGE":
			if o.subType == "FLAT" {
				p.Damage += o.value
			}
		case "SHIELD":
			if o.subType == "FLAT" {
				p.Shield += o.value
			}
		case "HEAL":
			if o.subType == "FLAT" {
				p.Heal += o.value
			}
		case "DIRECT_HP":
			if o.subType == "FLAT" {
				p.DirectHP += o.value
			}
		case "STAT_MOD":
			if o.subType == "ADD_FLAT" {
				p.Stat += o.value
			}
		case "STATUS":
			p.Status += o.value * o.turns
		}
	}
	return p, nil
}

// scaleCombatFlats multiplies every FLAT combat value (DAMAGE/SHIELD/HEAL/
// DIRECT_HP) by factor, leaving stat/status/UNIQUE untouched. It preserves the
// root shape (bare array vs {schema_version,blocks}) of the input.
func scaleCombatFlats(raw string, factor float64) (string, error) {
	trimmed := strings.TrimSpace(raw)
	isObj := strings.HasPrefix(trimmed, "{")
	var blocks []map[string]json.RawMessage
	var schemaVersion json.RawMessage
	if isObj {
		var root struct {
			SchemaVersion json.RawMessage              `json:"schema_version"`
			Blocks        []map[string]json.RawMessage `json:"blocks"`
		}
		if err := json.Unmarshal([]byte(trimmed), &root); err != nil {
			return raw, err
		}
		blocks, schemaVersion = root.Blocks, root.SchemaVersion
	} else {
		if err := json.Unmarshal([]byte(trimmed), &blocks); err != nil {
			return raw, err
		}
	}

	for _, b := range blocks {
		opsRaw, ok := b["operations"]
		if !ok {
			continue
		}
		var ops []map[string]json.RawMessage
		if json.Unmarshal(opsRaw, &ops) != nil {
			continue
		}
		changed := false
		for _, op := range ops {
			if !isCombatFlat(op) {
				continue
			}
			if v, ok := toFloat(op["value"]); ok {
				nv := math.Round(v*factor*1000) / 1000
				op["value"] = json.RawMessage(strconv.FormatFloat(nv, 'f', -1, 64))
				changed = true
			}
		}
		if changed {
			nb, err := json.Marshal(ops)
			if err == nil {
				b["operations"] = nb
			}
		}
	}

	var out []byte
	var err error
	if isObj {
		root := map[string]json.RawMessage{"blocks": json.RawMessage(mustJSON(blocks))}
		if len(schemaVersion) > 0 {
			root["schema_version"] = schemaVersion
		}
		out, err = json.Marshal(root)
	} else {
		out, err = json.Marshal(blocks)
	}
	if err != nil {
		return raw, err
	}
	return string(out), nil
}

func isCombatFlat(op map[string]json.RawMessage) bool {
	opName := rawStr(op["op"])
	sub := rawStr(op["sub_type"])
	switch opName {
	case "DAMAGE", "SHIELD", "HEAL", "DIRECT_HP":
		return sub == "FLAT"
	}
	return false
}

func rawStr(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.ToUpper(s)
	}
	return ""
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ---------------------------------------------------------------------------

type balanceRow struct {
	ID         string          `json:"id"`
	Rank       int             `json:"rank"`
	Score      float64         `json:"score"`
	Components powerComponents `json:"components"`
	Median     float64         `json:"median"`
	Band       [2]float64      `json:"band"`
	Status     string          `json:"status"` // ok | high | low | utility
	Factor     float64         `json:"factor,omitempty"`
	idx        int             `json:"-"`
}

func cmdBalance(args []string) error {
	fs := flag.NewFlagSet("balance", flag.ExitOnError)
	data := fs.String("data", "data", "data directory")
	cfgPath := fs.String("config", "", "optional JSON config")
	write := fs.Bool("write", false, "write corrected Gu.csv to -output")
	_ = fs.String("output", "", "output path (required with -write)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		return err
	}
	src := *data + "/Gu.csv"
	outPath, err := requireWriteOutput(fs, src)
	if err != nil {
		return err
	}

	t, err := readCSV(src)
	if err != nil {
		return err
	}

	type rec struct {
		idx   int
		id    string
		rank  int
		score float64
		comp  powerComponents
	}
	var recs []rec
	for i, row := range t.rows {
		id := t.get(row, "id")
		rank, _ := strconv.Atoi(t.get(row, "rank"))
		comp, err := powerOf(t.get(row, "effects"))
		if err != nil {
			comp = powerComponents{}
		}
		score := comp.total(cfg.Balance.Weights)
		recs = append(recs, rec{idx: i, id: id, rank: rank, score: score, comp: comp})
	}

	// Per-rank median.
	byRank := map[int][]float64{}
	for _, r := range recs {
		byRank[r.rank] = append(byRank[r.rank], r.score)
	}
	medians := map[int]float64{}
	for rank, scores := range byRank {
		sorted := append([]float64(nil), scores...)
		sort.Float64s(sorted)
		medians[rank] = sorted[len(sorted)/2]
	}

	rows := make([]balanceRow, 0, len(recs))
	high, low, utility := 0, 0, 0
	fixed := 0
	for _, r := range recs {
		med := medians[r.rank]
		lowBand := med * (1 - cfg.Balance.Tol)
		highBand := med * (1 + cfg.Balance.Tol)
		status := "ok"
		factor := 0.0
		switch {
		case r.score > highBand:
			status = "high"
			factor = highBand / r.score
		case r.score < lowBand && r.score < med*cfg.Balance.MinScoreFrac:
			status = "utility"
		case r.score < lowBand:
			status = "low"
			factor = lowBand / r.score
		}
		if status == "high" {
			high++
		} else if status == "low" {
			low++
		} else if status == "utility" {
			utility++
		}
		if status == "high" || status == "low" {
			rows = append(rows, balanceRow{
				ID: r.id, Rank: r.rank, Score: r.score, Components: r.comp,
				Median: med, Band: [2]float64{lowBand, highBand}, Status: status, Factor: factor,
				idx: r.idx,
			})
		}
	}

	if *write {
		for _, row := range rows {
			orig := t.get(t.rows[row.idx], "effects")
			scaled, err := scaleCombatFlats(orig, row.Factor)
			if err != nil {
				continue
			}
			if scaled != orig {
				t.set(t.rows[row.idx], "effects", scaled)
				fixed++
			}
		}
		if err := t.write(outPath); err != nil {
			return err
		}
	}

	report := struct {
		Summary struct {
			Total   int `json:"total"`
			Flagged int `json:"flagged"`
			High    int `json:"high"`
			Low     int `json:"low"`
			Utility int `json:"utility"`
			Fixed   int `json:"fixed"`
		} `json:"summary"`
		Bands map[int]struct {
			N      int     `json:"n"`
			Median float64 `json:"median"`
			Low    float64 `json:"low"`
			High   float64 `json:"high"`
		} `json:"bands"`
		Rows []balanceRow `json:"rows"`
	}{}
	report.Summary.Total = len(recs)
	report.Summary.Flagged = len(rows)
	report.Summary.High = high
	report.Summary.Low = low
	report.Summary.Utility = utility
	report.Summary.Fixed = fixed
	report.Bands = map[int]struct {
		N      int     `json:"n"`
		Median float64 `json:"median"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
	}{}
	for rank, med := range medians {
		report.Bands[rank] = struct {
			N      int     `json:"n"`
			Median float64 `json:"median"`
			Low    float64 `json:"low"`
			High   float64 `json:"high"`
		}{N: len(byRank[rank]), Median: med, Low: med * (1 - cfg.Balance.Tol), High: med * (1 + cfg.Balance.Tol)}
	}
	report.Rows = rows
	return printJSON(report)
}
