package handlers

import (
	"strings"
	"testing"
)

func TestParseResourceNodeTemplates(t *testing.T) {
	csvData := "node_type,name,path,cost_stones,prod_type,prod_amount,prod_interval_hours,node_category,hp_max,attack_range,blocks_movement,cost_gu,cost_gu_rank,zone_type\n" +
		"WALL,Tường,Earth Path,16,,0,0,DEFENSE,3000,0,true,GU_BEAST_SKIN;GU_STONE_SKIN,1,khu_phong_thu\n" +
		"FARMING_ZONE,Khu Trồng Trọt,Wood Path,10,Material,10,50,,0,0,false,,0,khu_trong\n"

	got := parseResourceNodeTemplates(strings.NewReader(csvData))
	if len(got) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(got))
	}

	wall, ok := got["WALL"]
	if !ok {
		t.Fatal("WALL template missing")
	}
	if !wall.BlocksMovement || wall.HPMax != 3000 || wall.CostGuRank != 1 {
		t.Fatalf("WALL parsed wrong: %+v", wall)
	}
	if len(wall.CostGu) != 2 || wall.CostGu[0] != "GU_BEAST_SKIN" || wall.CostGu[1] != "GU_STONE_SKIN" {
		t.Fatalf("WALL CostGu parsed wrong: %+v", wall.CostGu)
	}

	farm, ok := got["FARMING_ZONE"]
	if !ok {
		t.Fatal("FARMING_ZONE template missing")
	}
	if farm.ProdType != "Material" || farm.ProdAmount != 10 || farm.ProdIntervalHours != 50 || farm.ZoneType != "khu_trong" {
		t.Fatalf("FARMING_ZONE parsed wrong: %+v", farm)
	}
	if farm.BlocksMovement {
		t.Fatal("FARMING_ZONE should not block movement")
	}
}

func TestLoadResourceNodeTemplatesFromFile(t *testing.T) {
	// Run from backend/ so csvPath("ApertureNodeTemplate.csv") resolves to data/.
	t.Chdir("../../")

	got := loadResourceNodeTemplates()
	if len(got) != 8 {
		t.Fatalf("expected 8 templates from data/ApertureNodeTemplate.csv, got %d", len(got))
	}
	cat, ok := got["GUARDIAN_STATUE"]
	if !ok {
		t.Fatal("GUARDIAN_STATUE missing")
	}
	if cat.HPBonus != 500 || cat.AttackRange != 2 || cat.NodeCategory != "DEFENSE" {
		t.Fatalf("GUARDIAN_STATUE parsed wrong: %+v", cat)
	}
}
