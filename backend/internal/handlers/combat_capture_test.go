package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Đạo ngân Nô đạo khuếch đại tỉ lệ bắt và chỉ số thú bắt được; đạo ngân lưu phái
// khác không ảnh hưởng.
func TestCaptureDaoFactorScalesWithEnslavementMarks(t *testing.T) {
	cases := []struct {
		name  string
		marks models.DaoMarksMap
		want  float64
	}{
		{"no marks map", nil, 1.0},
		{"empty map", models.DaoMarksMap{}, 1.0},
		{"other path ignored", models.DaoMarksMap{"strength": 40000}, 1.0},
		{"zero marks", models.DaoMarksMap{"enslavement": 0}, 1.0},
		{"10k marks", models.DaoMarksMap{"enslavement": 10000}, 2.0},
		{"40k marks", models.DaoMarksMap{"enslavement": 40000}, 3.0},
	}
	for _, tc := range cases {
		if got := captureDaoFactor(tc.marks); math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("%s: captureDaoFactor = %.6f, want %.6f", tc.name, got, tc.want)
		}
	}
}
