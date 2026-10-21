package db

import "testing"

func TestPathMaterialIDDreamFallback(t *testing.T) {
	// dream_path_material was removed from Item.csv; generated dream realm
	// content must not reference it as a required material.
	if got := pathMaterialID("dream"); got != "guts_gu" {
		t.Fatalf("pathMaterialID(dream) = %q, expected guts_gu", got)
	}
	if got := pathMaterialID("fire"); got != "fire_path_material" {
		t.Fatalf("pathMaterialID(fire) = %q, expected fire_path_material", got)
	}
}
