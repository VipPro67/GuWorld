package db

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
	"testing"
)

// TestGuMobaCSVReferencesRealGu guards the authored realtime skill cards. The
// table has a foreign key onto gu_dictionary, so a typo in gu_id or rank would
// otherwise only surface when an administrator commits the import.
func TestGuMobaCSVReferencesRealGu(t *testing.T) {
	guRanks, err := readGuRanks()
	if err != nil {
		t.Skipf("Gu.csv unavailable: %v", err)
	}

	file, reader, header, err := openCSV("GuMoba.csv")
	if err != nil {
		t.Fatalf("open GuMoba.csv: %v", err)
	}
	defer file.Close()

	idIdx := colIndex(header, "gu_id")
	rankIdx := colIndex(header, "rank")
	modeIdx := colIndex(header, "cast_mode")
	shapeIdx := colIndex(header, "shape")
	if idIdx < 0 || rankIdx < 0 || modeIdx < 0 || shapeIdx < 0 {
		t.Fatal("GuMoba.csv missing required columns (gu_id, rank, cast_mode, shape)")
	}

	rows := 0
	for {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatalf("read GuMoba.csv: %v", readErr)
		}
		if len(row) == 0 {
			continue
		}
		rows++
		id := strings.TrimSpace(row[idIdx])
		rank := strings.TrimSpace(row[rankIdx])
		mode := strings.TrimSpace(row[modeIdx])
		shape := strings.TrimSpace(row[shapeIdx])

		if !validMobaCastModes[mode] {
			t.Errorf("GuMoba.csv %s rank %s: invalid cast_mode %q", id, rank, mode)
		}
		if !validMobaShapes[shape] {
			t.Errorf("GuMoba.csv %s rank %s: invalid shape %q", id, rank, shape)
		}
		if !guRanks[id+"|"+rank] {
			t.Errorf("GuMoba.csv %s rank %s does not exist in Gu.csv", id, rank)
		}
	}
	if rows == 0 {
		t.Fatal("GuMoba.csv has no data rows")
	}
}

func readGuRanks() (map[string]bool, error) {
	file, err := os.Open(csvPath("Gu.csv"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	idIdx := colIndex(header, "id")
	rankIdx := colIndex(header, "rank")

	ranks := map[string]bool{}
	for {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
		if idIdx >= len(row) || rankIdx >= len(row) {
			continue
		}
		ranks[strings.TrimSpace(row[idIdx])+"|"+strings.TrimSpace(row[rankIdx])] = true
	}
	return ranks, nil
}
