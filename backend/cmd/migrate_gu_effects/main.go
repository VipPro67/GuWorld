package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func main() {
	csvPath := "data/Gu.csv"
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		csvPath = "backend/data/Gu.csv"
	}
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		csvPath = "../../backend/data/Gu.csv"
	}

	file, err := os.Open(csvPath)
	if err != nil {
		log.Fatalf("Không thể mở file CSV %s: %v", csvPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Lỗi đọc header CSV: %v", err)
	}

	idIdx, rankIdx, effectsIdx := -1, -1, -1
	for i, col := range header {
		switch col {
		case "id":
			idIdx = i
		case "rank":
			rankIdx = i
		case "effects":
			effectsIdx = i
		}
	}

	if idIdx < 0 || rankIdx < 0 || effectsIdx < 0 {
		log.Fatalf("CSV thiếu cột bắt buộc (id, rank, effects)")
	}

	var allRows [][]string
	allRows = append(allRows, header)

	migratedCount := 0
	lineNum := 1

	for {
		lineNum++
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Lỗi đọc dòng %d: %v", lineNum, err)
		}

		if len(row) <= effectsIdx || row[idIdx] == "" {
			allRows = append(allRows, row)
			continue
		}

		effectsStr := row[effectsIdx]
		var rawRoot map[string]json.RawMessage
		if err := json.Unmarshal([]byte(effectsStr), &rawRoot); err != nil {
			log.Fatalf("Dòng %d (Gu: %s): Lỗi parse effects JSON: %v", lineNum, row[idIdx], err)
		}

		var typedEffects models.GuEffects
		if err := json.Unmarshal([]byte(effectsStr), &typedEffects); err != nil {
			log.Fatalf("Dòng %d (Gu: %s): Lỗi parse GuEffects: %v", lineNum, row[idIdx], err)
		}

		// Compile legacy active, passive, other into modular blocks
		blocks := models.CompileLegacyToBlocks(typedEffects.Active, typedEffects.Passive, typedEffects.Other)
		blocksJSON, err := json.Marshal(blocks)
		if err != nil {
			log.Fatalf("Dòng %d (Gu: %s): Lỗi marshal blocks: %v", lineNum, row[idIdx], err)
		}

		rawRoot["blocks"] = json.RawMessage(blocksJSON)

		combinedJSON, err := json.Marshal(rawRoot)
		if err != nil {
			log.Fatalf("Dòng %d (Gu: %s): Lỗi marshal combined effects: %v", lineNum, row[idIdx], err)
		}

		normalized, report, err := effectcatalog.NormalizeAndValidate(combinedJSON)
		if err != nil || report.HasErrors() {
			log.Fatalf("Dòng %d (Gu: %s, Rank: %s): NormalizeAndValidate thất bại: %v (issues: %+v)",
				lineNum, row[idIdx], row[rankIdx], err, report.Issues)
		}

		row[effectsIdx] = string(normalized)
		allRows = append(allRows, row)
		migratedCount++
	}

	file.Close()

	// Write migrated rows back to CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	for _, r := range allRows {
		if err := writer.Write(r); err != nil {
			log.Fatalf("Lỗi ghi dòng vào buffer: %v", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Fatalf("Lỗi flush CSV writer: %v", err)
	}

	if err := os.WriteFile(csvPath, buf.Bytes(), 0644); err != nil {
		log.Fatalf("Lỗi ghi đè file CSV %s: %v", csvPath, err)
	}

	fmt.Printf("Migration hoàn tất thành công! Đã xử lý và chuẩn hóa %d bản ghi Cổ trùng vào %s\n", migratedCount, csvPath)
}
