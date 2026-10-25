package ai

import "testing"

func TestDefaultStatsScaleWithMonsterType(t *testing.T) {
	normal := DefaultStats("Normal")
	elite := DefaultStats("Elite")
	boss := DefaultStats("Boss")
	if normal.AggroRadius <= 0 || normal.AttackRange <= 0 || normal.RespawnTicks == 0 {
		t.Fatalf("normal stats are incomplete: %+v", normal)
	}
	if elite.AggroRadius < normal.AggroRadius || elite.AttackCooldownTicks == 0 {
		t.Fatalf("elite stats regressed: %+v", elite)
	}
	if boss.AttackRange < elite.AttackRange || boss.Radius < elite.Radius {
		t.Fatalf("boss stats regressed: %+v", boss)
	}
	if DefaultStats("unknown").AttackRange != normal.AttackRange {
		t.Fatal("unknown type should use the normal archetype")
	}
}
