package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestFillApertureEnvironment_TerrainTranslation(t *testing.T) {
	tests := []struct {
		name         string
		daoMarks     models.DaoMarksMap
		expectedName string
	}{
		{
			name: "Fire Leyline (> 20000)",
			daoMarks: models.DaoMarksMap{
				"fire": 25000,
			},
			expectedName: "Địa Mạch Viêm Đạo",
		},
		{
			name: "Water Grotto (> 100000)",
			daoMarks: models.DaoMarksMap{
				"water": 120000,
			},
			expectedName: "Bí Cảnh Thủy Đạo",
		},
		{
			name: "Time Special Region (> 5000 and not special case)",
			daoMarks: models.DaoMarksMap{
				"time": 6000,
			},
			expectedName: "Vùng Đặc Thù Trụ Đạo",
		},
		{
			name: "Lightning Special Case (> 5000)",
			daoMarks: models.DaoMarksMap{
				"lightning": 6000,
			},
			expectedName: "Lôi Điện Hải Vực",
		},
		{
			name: "Low amount defaults to meadow",
			daoMarks: models.DaoMarksMap{
				"fire": 100,
			},
			expectedName: "Thảo Nguyên Trù Phú",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := &models.ImmortalAperture{}
			FillApertureEnvironment(ap, tt.daoMarks)
			if ap.Terrain != tt.expectedName {
				t.Errorf("expected Terrain %q, got %q", tt.expectedName, ap.Terrain)
			}
		})
	}
}
