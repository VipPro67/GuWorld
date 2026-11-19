package db

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func seedPublicApertureSitesFromCSV(filename string) error {
	f, reader, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	columns := map[string]int{}
	for _, key := range []string{"site_key", "region_id", "x", "y", "aperture_rank", "grade", "area_size", "dao_marks", "is_active"} {
		idx := colIndex(header, key)
		if idx < 0 {
			return fmt.Errorf("%s missing required column %q", filename, key)
		}
		columns[key] = idx
	}
	value := func(row []string, key string) string {
		idx := columns[key]
		if idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s line %d: %w", filename, line, err)
		}
		if len(row) == 0 {
			continue
		}
		key, region := value(row, "site_key"), value(row, "region_id")
		if key == "" || region == "" {
			return fmt.Errorf("%s line %d: site_key and region_id are required", filename, line)
		}
		x, err := strconv.ParseFloat(value(row, "x"), 64)
		if err != nil || x < 0 {
			return fmt.Errorf("%s line %d: x must be a non-negative number", filename, line)
		}
		y, err := strconv.ParseFloat(value(row, "y"), 64)
		if err != nil || y < 0 {
			return fmt.Errorf("%s line %d: y must be a non-negative number", filename, line)
		}
		rank, err := strconv.Atoi(value(row, "aperture_rank"))
		if err != nil || rank < 6 || rank > 9 {
			return fmt.Errorf("%s line %d: aperture_rank must be between 6 and 9", filename, line)
		}
		grade := strings.ToUpper(value(row, "grade"))
		if grade != "LOW" && grade != "MID" && grade != "HIGH" && grade != "EXTREME" {
			return fmt.Errorf("%s line %d: grade must be LOW, MID, HIGH, or EXTREME", filename, line)
		}
		area, err := strconv.ParseInt(value(row, "area_size"), 10, 64)
		if err != nil || area <= 0 {
			return fmt.Errorf("%s line %d: area_size must be a positive integer", filename, line)
		}
		daoMarks := value(row, "dao_marks")
		if daoMarks == "" {
			daoMarks = "{}"
		}
		var daoMarksValue map[string]float64
		if !json.Valid([]byte(daoMarks)) || json.Unmarshal([]byte(daoMarks), &daoMarksValue) != nil || daoMarksValue == nil {
			return fmt.Errorf("%s line %d: dao_marks must be a JSON object", filename, line)
		}
		active, err := strconv.ParseBool(value(row, "is_active"))
		if err != nil {
			return fmt.Errorf("%s line %d: is_active must be true or false", filename, line)
		}
		if _, err := activeCSVImportDB().Exec(`
			INSERT INTO public_aperture_site_configs
				(site_key, region_id, x, y, aperture_rank, grade, area_size, dao_marks, is_active, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, NOW())
			ON CONFLICT (site_key) DO UPDATE SET
				region_id = EXCLUDED.region_id, x = EXCLUDED.x, y = EXCLUDED.y,
				aperture_rank = EXCLUDED.aperture_rank, grade = EXCLUDED.grade,
				area_size = EXCLUDED.area_size, dao_marks = EXCLUDED.dao_marks,
				is_active = EXCLUDED.is_active, updated_at = NOW()
		`, key, region, x, y, rank, grade, area, daoMarks, active); err != nil {
			return fmt.Errorf("upsert public aperture site %q: %w", key, err)
		}
	}
}
