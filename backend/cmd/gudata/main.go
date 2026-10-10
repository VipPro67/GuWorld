// Command gudata is a set of offline tools for authoring the CSV game data in
// backend/data. It is dry-run by default; every mutating subcommand requires
// -write and a distinct -output path and never overwrites its source CSV.
//
// Subcommands:
//
//	gudata check-effects  validate Gu/Item effect blocks against effectcatalog
//	gudata balance        compare each Gu's power vs its rank peers (band + tol)
//	gudata match-food     audit/remap Gu food_types against Item.csv
//	gudata gen-recipe     generate missing refinement recipes (configurable)
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "check-effects":
		err = cmdCheckEffects(os.Args[2:])
	case "balance":
		err = cmdBalance(os.Args[2:])
	case "match-food":
		err = cmdMatchFood(os.Args[2:])
	case "gen-recipe":
		err = cmdGenRecipe(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gudata:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: gudata <check-effects|balance|match-food|gen-recipe> [flags]`)
}

// dataDir returns the -data flag value defaulting to "data".
func dataDir(fs *flag.FlagSet) string {
	d := fs.Lookup("data").Value.String()
	if d == "" {
		return "data"
	}
	return d
}

// requireWriteOutput enforces the shared contract: -write needs -output, and
// the output path must differ from the source it would overwrite.
func requireWriteOutput(fs *flag.FlagSet, source string) (string, error) {
	write := fs.Lookup("write").Value.String() == "true"
	output := fs.Lookup("output").Value.String()
	if !write {
		return "", nil
	}
	if output == "" {
		return "", fmt.Errorf("-output is required with -write")
	}
	if samePath(source, output) {
		return "", fmt.Errorf("-output must differ from source %s", source)
	}
	return output, nil
}

func samePath(a, b string) bool {
	aa, ea := filepath.Abs(a)
	bb, eb := filepath.Abs(b)
	if ea != nil || eb != nil {
		return a == b
	}
	return aa == bb
}

// ---------------------------------------------------------------------------
// CSV table helpers
// ---------------------------------------------------------------------------

type table struct {
	header []string
	col    map[string]int
	rows   [][]string
}

func readCSV(path string) (*table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	// Strip UTF-8 BOM from the first header cell (the data files carry one).
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\uFEFF")
	}
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	return &table{header: header, col: col, rows: rows}, nil
}

func (t *table) get(row []string, name string) string {
	i, ok := t.col[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

func (t *table) set(row []string, name, val string) {
	i, ok := t.col[name]
	if !ok || i >= len(row) {
		return
	}
	row[i] = val
}

func (t *table) write(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(t.header); err != nil {
		return err
	}
	if err := w.WriteAll(t.rows); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

// printJSON marshals v to stdout (machine-readable report).
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// jsonArray parses a JSON array of strings, tolerating empty/malformed input.
func jsonArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func jsonObject(raw string) map[string]json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "null" {
		return map[string]json.RawMessage{}
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]json.RawMessage{}
	}
	return out
}

// toFloat extracts a number from a JSON raw value that may be a float, an
// integer, or a quoted numeric string.
func toFloat(raw json.RawMessage) (float64, bool) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		var g float64
		if _, err := fmt.Sscanf(s, "%g", &g); err == nil {
			return g, true
		}
	}
	return 0, false
}
