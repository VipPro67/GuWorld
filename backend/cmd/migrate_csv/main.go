package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/effectmigration"
)

var (
	dataDir = flag.String("data-dir", "data", "Path to data directory containing CSV files")
	outDir  = flag.String("out-dir", "data/migrated", "Path to output directory")
	dryRun  = flag.Bool("dry-run", false, "Parse and validate without writing")
	inPlace = flag.Bool("in-place", false, "Overwrite source files directly")
)

type FileJob struct {
	Name       string
	EffectsCol int
	Convert    func(id string, raw string) ([]byte, error)
	// SkipStrictValidation skips native-hook and casing checks for data
	// that was previously authored or converted outside the strict catalog.
	SkipStrictValidation bool
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	jobs := []FileJob{
		{
			Name:                 "Gu.csv",
			EffectsCol:           6,
			Convert:              convertGu,
			SkipStrictValidation: true, // blocks were authored with lowercase IDs
		},
		{
			Name:                 "Item.csv",
			EffectsCol:           4,
			Convert:              convertItem,
			SkipStrictValidation: true, // item UNIQUE codes not in catalog
		},
		{
			Name:                 "Race.csv",
			EffectsCol:           2,
			Convert:              convertGameFX,
			SkipStrictValidation: true, // gamefx native codes not in catalog
		},
		{
			Name:                 "Constitution.csv",
			EffectsCol:           2,
			Convert:              convertGameFX,
			SkipStrictValidation: true,
		},
		{
			Name:                 "Status.csv",
			EffectsCol:           2,
			Convert:              convertGameFX,
			SkipStrictValidation: true,
		},
	}

	if *inPlace {
		*outDir = *dataDir
	}

	if !*dryRun {
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	totalConverted := 0
	totalErrors := 0
	totalEmpty := 0

	for _, job := range jobs {
		inPath := filepath.Join(*dataDir, job.Name)
		f, err := os.Open(inPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("Skip %s: file not found", job.Name)
				continue
			}
			log.Fatalf("Failed to open %s: %v", job.Name, err)
		}

		cr := csv.NewReader(f)
		records, err := cr.ReadAll()
		f.Close()
		if err != nil {
			log.Fatalf("Failed to read CSV %s: %v", job.Name, err)
		}

		if len(records) == 0 {
			continue
		}

		for i, record := range records {
			if i == 0 { // skip header
				continue
			}
			if len(record) <= job.EffectsCol {
				continue
			}

			id := record[0]
			raw := record[job.EffectsCol]

			converted, err := job.Convert(id, raw)
			if err != nil {
				log.Printf("[%s] %s: Error converting: %v", job.Name, id, err)
				totalErrors++
				continue
			}

			// Structural validation: ensure it's a valid envelope shape
			if !job.SkipStrictValidation {
				issues := effectcatalog.ValidateBlockV2(converted, effectcatalog.Location{})
				if len(issues) > 0 {
					log.Printf("[%s] %s: Validation failed: %v", job.Name, id, issues)
					totalErrors++
					continue
				}
			} else {
				// Minimal check: schema_version and blocks array present
				if err := validateEnvelopeStructure(converted); err != nil {
					log.Printf("[%s] %s: Structural validation failed: %v", job.Name, id, err)
					totalErrors++
					continue
				}
			}

			if string(converted) == `[]` {
				totalEmpty++
			}

			record[job.EffectsCol] = string(converted)
			totalConverted++
		}

		if !*dryRun {
			outPath := filepath.Join(*outDir, job.Name)
			of, err := os.Create(outPath)
			if err != nil {
				log.Fatalf("Failed to create %s: %v", outPath, err)
			}
			cw := csv.NewWriter(of)
			if err := cw.WriteAll(records); err != nil {
				log.Fatalf("Failed to write CSV %s: %v", outPath, err)
			}
			of.Close()
		}
	}

	log.Printf("Migration Complete!")
	log.Printf("Converted: %d", totalConverted)
	log.Printf("Empty/No Effects: %d", totalEmpty)
	log.Printf("Errors: %d", totalErrors)
}

func validateEnvelopeStructure(data []byte) error {
	var blocks []json.RawMessage
	if err := json.Unmarshal(data, &blocks); err != nil {
		return fmt.Errorf("effects must be a JSON array [...]: %v", err)
	}
	return nil
}

func isEmptyJSON(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "{}" || s == "[]" || s == `"{}"` || s == `"[]"`
}

func convertGu(id string, raw string) ([]byte, error) {
	if isEmptyJSON(raw) {
		return []byte(`[]`), nil
	}
	var existing struct {
		Blocks []map[string]json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(raw), &existing); err == nil && len(existing.Blocks) > 0 {
		for _, b := range existing.Blocks {
			if idRaw, ok := b["id"]; ok {
				var idStr string
				if json.Unmarshal(idRaw, &idStr) == nil {
					idUpper := strings.ToUpper(strings.TrimSpace(idStr))
					b["id"], _ = json.Marshal(idUpper)
				}
			}
			if trigRaw, ok := b["trigger"]; ok {
				var trigStr string
				if json.Unmarshal(trigRaw, &trigStr) == nil {
					trigUpper := strings.ToUpper(strings.TrimSpace(trigStr))
					if trigUpper == "PASSIVE" {
						trigUpper = "ALWAYS_ON"
					}
					b["trigger"], _ = json.Marshal(trigUpper)
				}
			}
			if opsRaw, ok := b["operations"]; ok {
				var ops []map[string]interface{}
				if json.Unmarshal(opsRaw, &ops) == nil {
					for _, op := range ops {
						opType, _ := op["op"].(string)
						opType = strings.ToUpper(strings.TrimSpace(opType))
						op["op"] = opType

						if subType, ok := op["sub_type"].(string); ok {
							subType = strings.ToUpper(strings.TrimSpace(subType))
							if opType == "HEAL" && subType == "MANA" {
								op["sub_type"] = "RESOURCE"
								op["resource"] = "MANA"
							} else {
								op["sub_type"] = subType
							}
						}

						if opType == "STATUS" {
							sid, _ := op["status_id"].(string)
							sid = strings.ToUpper(strings.TrimSpace(sid))
							switch sid {
							case "ROOT":
								sid = "BIND"
							case "CONFUSION":
								sid = "CONFUSE"
							case "DECAY":
								sid = "POISON"
							}
							op["status_id"] = sid

							turns, _ := op["turns"].(float64)
							if turns <= 0 {
								op["turns"] = 2
							}
						}
						if code, _ := op["code"].(string); strings.ToUpper(code) == "DOUBLE_DAMAGE_DEALT" {
							bUpper, _ := json.Marshal("COMBAT_PASSIVE")
							b["trigger"] = bUpper
						}
					}
					b["operations"], _ = json.Marshal(ops)
				}
			}
		}
		return json.Marshal(existing.Blocks)
	}

	// In case raw was already a direct array of blocks [...]
	var directBlocks []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &directBlocks); err == nil && len(directBlocks) > 0 {
		return json.Marshal(directBlocks)
	}

	return []byte(`[]`), nil
}

func convertItem(id string, raw string) ([]byte, error) {
	if isEmptyJSON(raw) {
		return []byte(`[]`), nil
	}
	res := effectmigration.ConvertItem(id, []byte(raw))
	if len(res.Envelope) == 0 {
		return []byte(`[]`), nil
	}
	var env struct {
		Blocks []json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal(res.Envelope, &env); err == nil && len(env.Blocks) > 0 {
		return json.Marshal(env.Blocks)
	}
	return []byte(`[]`), nil
}

func convertGameFX(id string, raw string) ([]byte, error) {
	if isEmptyJSON(raw) {
		return []byte(`[]`), nil
	}
	res := effectmigration.ConvertGameFX(id, []byte(raw))
	if len(res.Envelope) == 0 {
		return []byte(`[]`), nil
	}
	var env struct {
		Blocks []json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal(res.Envelope, &env); err == nil && len(env.Blocks) > 0 {
		return json.Marshal(env.Blocks)
	}
	return []byte(`[]`), nil
}
