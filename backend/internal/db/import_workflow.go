package db

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const MaxImportRows = 50000

type ImportValidationIssue struct {
	Dataset  string `json:"dataset"`
	Row      int    `json:"row,omitempty"`
	Field    string `json:"field,omitempty"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type ImportRowDiff struct {
	Kind   string                 `json:"kind"`
	Before map[string]interface{} `json:"before,omitempty"`
	After  map[string]interface{} `json:"after,omitempty"`
}

type ImportDatasetResult struct {
	Key              string                  `json:"key"`
	Filename         string                  `json:"filename"`
	Rows             int                     `json:"rows"`
	Inserts          int                     `json:"inserts"`
	Updates          int                     `json:"updates"`
	Unchanged        int                     `json:"unchanged"`
	Diffs            []ImportRowDiff         `json:"diffs"`
	ValidationErrors []ImportValidationIssue `json:"validation_errors"`
	ReloadRequired   bool                    `json:"reload_required"`
	RestartRequired  bool                    `json:"restart_required"`
}

type ImportSummary struct {
	Rows      int `json:"rows"`
	Inserts   int `json:"inserts"`
	Updates   int `json:"updates"`
	Unchanged int `json:"unchanged"`
}

type ImportPlan struct {
	Scope            string                  `json:"scope"`
	Datasets         []ImportDatasetResult   `json:"datasets"`
	Totals           ImportSummary           `json:"totals"`
	ValidationErrors []ImportValidationIssue `json:"validation_errors"`
	CanCommit        bool                    `json:"can_commit"`
}

type importDataset struct {
	key, filename, table string
	reload, restart      bool
	keyColumns           []string
}

var importDatasets = []importDataset{
	{"item", "Item.csv", "item_dictionary", true, false, []string{"id"}},
	{"gu", "Gu.csv", "gu_dictionary", true, false, []string{"id", "rank"}},
	{"constitution", "Constitution.csv", "constitution_dictionary", true, false, []string{"key"}},
	{"race", "Race.csv", "race_dictionary", true, false, []string{"key"}},
	{"status", "Status.csv", "status_dictionary", true, false, []string{"key"}},
	{"recipe", "Recipe.csv", "refining_recipes", true, false, []string{"target_gu_id", "target_rank", "variant"}},
	{"shop_item", "ShopItem.csv", "caravan_shop_items", true, false, []string{"id", "rank"}},
	{"arena_shop_item", "ArenaShopItem.csv", "arena_shop_items", true, false, []string{"id"}},
	{"region", "Region.csv", "regions", true, false, []string{"id"}},
	{"loot_table", "LootTable.csv", "loot_tables", true, false, []string{"id"}},
	{"region_loot", "RegionLootTables.csv", "region_loot_tables", true, false, []string{"region_id", "loot_table_id", "monster_type"}},
	{"monster", "Monster.csv", "monster_templates", true, false, []string{"id"}},
	{"monster_spawn", "MonsterSpawn.csv", "monster_spawns", true, false, []string{"region_id", "monster_id"}},
	{"map_cell", "MapCell.csv", "map_cells", true, false, []string{"region_id", "x", "y"}},
	{"sect_mission", "SectMission.csv", "sect_missions", false, true, []string{"sect_id", "title"}},
	{"quest", "Quest.csv", "quests", true, false, []string{"id"}},
	{"dream_realm", "DreamRealm.csv", "instance_templates", true, false, []string{"id"}},
}

func ImportDatasetKeys() []string {
	keys := make([]string, 0, len(importDatasets))
	for _, dataset := range importDatasets {
		keys = append(keys, dataset.key)
	}
	return keys
}

// ImportSourceDigest fingerprints the server-side CSV selection so a commit
// cannot silently use files that changed after the administrator reviewed it.
func ReadImportSources(table string, upload []byte) (map[string][]byte, error) {
	datasets := importDatasets
	if table != "" {
		dataset, ok := findImportDataset(table)
		if !ok {
			return nil, fmt.Errorf("unknown table: %s", table)
		}
		datasets = []importDataset{dataset}
	}
	if len(upload) > 0 && len(datasets) != 1 {
		return nil, fmt.Errorf("upload chỉ hỗ trợ import một dataset")
	}
	sources := make(map[string][]byte, len(datasets))
	for _, dataset := range datasets {
		content := upload
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(csvPath(dataset.filename))
			if err != nil {
				return nil, err
			}
		}
		sources[dataset.filename] = append([]byte(nil), content...)
	}
	return sources, nil
}

func ImportSourcesDigest(sources map[string][]byte) string {
	hash := sha256.New()
	filenames := make([]string, 0, len(sources))
	for filename := range sources {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		hash.Write([]byte(filename))
		hash.Write([]byte{0})
		hash.Write(sources[filename])
		hash.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func findImportDataset(key string) (importDataset, bool) {
	for _, dataset := range importDatasets {
		if dataset.key == key {
			return dataset, true
		}
	}
	return importDataset{}, false
}

func validateImportCSV(dataset importDataset, content []byte) (int, []ImportValidationIssue) {
	r := csv.NewReader(bytes.NewReader(content))
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return 0, []ImportValidationIssue{{Dataset: dataset.key, Code: "invalid_csv", Message: "Không thể đọc header CSV: " + err.Error(), Severity: "error"}}
	}
	indexes := make([]int, len(dataset.keyColumns))
	for i, key := range dataset.keyColumns {
		indexes[i] = colIndex(header, key)
		if indexes[i] < 0 {
			return 0, []ImportValidationIssue{{Dataset: dataset.key, Field: key, Code: "missing_header", Message: "Thiếu cột khóa bắt buộc " + key, Severity: "error"}}
		}
	}
	seen := map[string]int{}
	issues := []ImportValidationIssue{}
	jsonFields := map[string]bool{"effects": true, "food_types": true, "required_gus": true, "required_materials": true, "pool_weights": true, "rules": true, "entries": true, "gu_inventory": true, "item_inventory": true, "dao_marks": true, "path_masteries": true, "metadata": true, "rewards": true, "map_data": true, "blocked_actions": true}
	if dataset.key != "dream_realm" {
		jsonFields["path"] = true
	}
	rows := 0
	for line := 2; ; line++ {
		row, readErr := r.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			issues = append(issues, ImportValidationIssue{Dataset: dataset.key, Row: line, Code: "invalid_csv", Message: readErr.Error(), Severity: "error"})
			break
		}
		if len(row) == 0 {
			continue
		}
		parts := make([]string, len(indexes))
		valid := true
		for i, idx := range indexes {
			if idx >= len(row) || strings.TrimSpace(row[idx]) == "" {
				issues = append(issues, ImportValidationIssue{Dataset: dataset.key, Row: line, Field: dataset.keyColumns[i], Code: "missing_key", Message: "Khóa dòng không được để trống", Severity: "error"})
				valid = false
				continue
			}
			parts[i] = strings.TrimSpace(row[idx])
		}
		if !valid {
			continue
		}
		for column, name := range header {
			if !jsonFields[name] || column >= len(row) || strings.TrimSpace(row[column]) == "" {
				continue
			}
			if !json.Valid([]byte(strings.TrimSpace(row[column]))) {
				issues = append(issues, ImportValidationIssue{Dataset: dataset.key, Row: line, Field: name, Code: "invalid_json", Message: "JSON không hợp lệ", Severity: "error"})
			}
		}
		rows++
		key := strings.Join(parts, "\x1f")
		if previous, exists := seen[key]; exists {
			issues = append(issues, ImportValidationIssue{Dataset: dataset.key, Row: line, Code: "duplicate_key", Message: fmt.Sprintf("Khóa trùng với dòng %d", previous), Severity: "error"})
		} else {
			seen[key] = line
		}
		if rows > MaxImportRows {
			issues = append(issues, ImportValidationIssue{Dataset: dataset.key, Row: line, Code: "too_many_rows", Message: fmt.Sprintf("Dataset vượt giới hạn %d dòng", MaxImportRows), Severity: "error"})
			break
		}
	}
	return rows, issues
}

func snapshotImportTable(ctx context.Context, tx *sql.Tx, dataset importDataset) (map[string]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT to_jsonb(t) FROM %s t", dataset.table)
	if dataset.key == "region" {
		query = `SELECT to_jsonb(t) || jsonb_build_object('world_map_id', COALESCE(rwm.world_map_id,'world_map'))
			FROM regions t LEFT JOIN region_world_maps rwm ON rwm.region_id=t.id`
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]interface{}{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var value map[string]interface{}
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		for key := range value {
			if strings.HasSuffix(key, "_at") {
				delete(value, key)
			}
		}
		keyParts := make([]string, len(dataset.keyColumns))
		for i, column := range dataset.keyColumns {
			keyParts[i] = fmt.Sprint(value[column])
		}
		result[strings.Join(keyParts, "\x1f")] = value
	}
	return result, rows.Err()
}

func compareImportSnapshots(before, after map[string]map[string]interface{}, rowCount int) (int, int, int, []ImportRowDiff) {
	insertCount, updateCount := 0, 0
	diffs := []ImportRowDiff{}
	for key, afterValue := range after {
		beforeValue, existed := before[key]
		if !existed {
			insertCount++
			diffs = append(diffs, ImportRowDiff{Kind: "insert", After: afterValue})
			continue
		}
		beforeJSON, _ := json.Marshal(beforeValue)
		afterJSON, _ := json.Marshal(afterValue)
		if !bytes.Equal(beforeJSON, afterJSON) {
			updateCount++
			diffs = append(diffs, ImportRowDiff{Kind: "update", Before: beforeValue, After: afterValue})
		}
	}
	if len(diffs) > 200 {
		diffs = diffs[:200]
	}
	unchanged := rowCount - insertCount - updateCount
	if unchanged < 0 {
		unchanged = 0
	}
	return insertCount, updateCount, unchanged, diffs
}

// RunImportPlan executes the exact import inside a transaction. dryRun always
// rolls back; commit callers receive the live transaction so audit can be
// written atomically before Commit.
func RunImportPlan(ctx context.Context, table string, upload []byte, dryRun bool) (ImportPlan, *sql.Tx, error) {
	sources, err := ReadImportSources(table, upload)
	if err != nil {
		return ImportPlan{}, nil, err
	}
	return RunImportPlanSources(ctx, table, sources, dryRun)
}

func RunImportPlanSources(ctx context.Context, table string, sources map[string][]byte, dryRun bool) (ImportPlan, *sql.Tx, error) {
	csvImportState.Lock()
	defer csvImportState.Unlock()
	if dryRun {
		previousBaseRanks := make(map[string]int, len(baseRankCache))
		for key, value := range baseRankCache {
			previousBaseRanks[key] = value
		}
		defer func() { baseRankCache = previousBaseRanks }()
	}
	datasets := importDatasets
	scope := "all"
	if table != "" {
		dataset, ok := findImportDataset(table)
		if !ok {
			return ImportPlan{}, nil, fmt.Errorf("unknown table: %s", table)
		}
		datasets = []importDataset{dataset}
		scope = "dataset"
	}
	tempPaths := []string{}
	defer func() {
		for _, path := range tempPaths {
			os.Remove(path)
		}
	}()
	overrides := map[string]string{}
	for filename, content := range sources {
		file, err := os.CreateTemp("", "guworld-import-*.csv")
		if err != nil {
			return ImportPlan{}, nil, err
		}
		tempPaths = append(tempPaths, file.Name())
		if _, err := file.Write(content); err != nil {
			file.Close()
			return ImportPlan{}, nil, err
		}
		file.Close()
		overrides[filename] = file.Name()
	}
	plan := ImportPlan{Scope: scope, CanCommit: true}
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return plan, nil, err
	}
	csvImportState.executor = tx
	csvImportState.override = overrides
	defer func() { csvImportState.executor = nil; csvImportState.override = nil }()

	for _, dataset := range datasets {
		content, exists := sources[dataset.filename]
		if !exists {
			tx.Rollback()
			return plan, nil, fmt.Errorf("missing staged source %s", dataset.filename)
		}
		rowCount, issues := validateImportCSV(dataset, content)
		if issues == nil {
			issues = []ImportValidationIssue{}
		}
		result := ImportDatasetResult{Key: dataset.key, Filename: dataset.filename, Rows: rowCount, ValidationErrors: issues, Diffs: []ImportRowDiff{}, ReloadRequired: dataset.reload, RestartRequired: dataset.restart}
		if len(issues) > 0 {
			plan.CanCommit = false
			plan.ValidationErrors = append(plan.ValidationErrors, issues...)
			plan.Datasets = append(plan.Datasets, result)
			continue
		}
		before, snapErr := snapshotImportTable(ctx, tx, dataset)
		if snapErr != nil {
			tx.Rollback()
			return plan, nil, snapErr
		}
		if importErr := TableImport(dataset.key); importErr != nil {
			issue := ImportValidationIssue{Dataset: dataset.key, Code: "import_validation", Message: importErr.Error(), Severity: "error"}
			result.ValidationErrors = append(result.ValidationErrors, issue)
			plan.ValidationErrors = append(plan.ValidationErrors, issue)
			plan.CanCommit = false
			plan.Datasets = append(plan.Datasets, result)
			tx.Rollback()
			return plan, nil, nil
		}
		after, snapErr := snapshotImportTable(ctx, tx, dataset)
		if snapErr != nil {
			tx.Rollback()
			return plan, nil, snapErr
		}
		result.Inserts, result.Updates, result.Unchanged, result.Diffs = compareImportSnapshots(before, after, rowCount)
		plan.Totals.Rows += result.Rows
		plan.Totals.Inserts += result.Inserts
		plan.Totals.Updates += result.Updates
		plan.Totals.Unchanged += result.Unchanged
		plan.Datasets = append(plan.Datasets, result)
	}
	sort.SliceStable(plan.Datasets, func(i, j int) bool { return plan.Datasets[i].Key < plan.Datasets[j].Key })
	if plan.Datasets == nil {
		plan.Datasets = []ImportDatasetResult{}
	}
	if plan.ValidationErrors == nil {
		plan.ValidationErrors = []ImportValidationIssue{}
	}
	if dryRun || !plan.CanCommit {
		tx.Rollback()
		return plan, nil, nil
	}
	return plan, tx, nil
}
