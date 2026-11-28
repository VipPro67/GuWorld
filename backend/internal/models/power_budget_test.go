package models

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

func TestEffectCatalogRankScaleParity(t *testing.T) {
	for rank := 1; rank <= 9; rank++ {
		if got, want := effectcatalog.RankScale(rank), RankScale(rank); got != want {
			t.Fatalf("rank %d scale mismatch: effectcatalog=%v, models=%v", rank, got, want)
		}
	}
}
