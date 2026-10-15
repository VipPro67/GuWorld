package models

import (
	"testing"
)

func TestBuildCombatStateFromMonster_RankFilteringAndGuAssignment(t *testing.T) {
	// Setup dummy GuRegistry for testing
	GuRegistry = map[string]map[int]GuDictionaryItem{
		"fire_gu_weak": {
			6: GuDictionaryItem{
				ID:   "fire_gu_weak",
				Name: "Bạch Hỏa Cổ",
				Rank: 6,
				Path: []string{"fire"},
				Effects: GuEffects{
					Active: &ActiveEffect{
						HealFlat: 50,
					},
				},
			},
		},
		"fire_gu_strong": {
			6: GuDictionaryItem{
				ID:   "fire_gu_strong",
				Name: "Liệt Diễm Cổ",
				Rank: 6,
				Path: []string{"fire"},
				Effects: GuEffects{
					Active: &ActiveEffect{
						DamageFlat: 500,
					},
				},
			},
		},
		"fire_gu_passive": {
			6: GuDictionaryItem{
				ID:   "fire_gu_passive",
				Name: "Hỏa Thể Cổ",
				Rank: 6,
				Path: []string{"fire"},
				Type: "PASSIVE",
				Effects: GuEffects{
					Passive: &PassiveEffect{
						StatBaseAttack: 200,
						StatBaseHP:     1000,
					},
				},
			},
		},
		"fire_gu_passive_2": {
			6: GuDictionaryItem{
				ID:   "fire_gu_passive_2",
				Name: "Bão Hỏa Cổ",
				Rank: 6,
				Path: []string{"fire"},
				Type: "PASSIVE",
				Effects: GuEffects{
					Passive: &PassiveEffect{
						StatBaseSpeed: 50,
					},
				},
			},
		},
		"mortal_gu_5": {
			5: GuDictionaryItem{
				ID:   "mortal_gu_5",
				Name: "Ngũ Chuyển Hỏa Cổ",
				Rank: 5,
				Path: []string{"fire"},
				Effects: GuEffects{
					Active: &ActiveEffect{
						DamageFlat: 150,
					},
				},
			},
		},
		"mortal_gu_4": {
			4: GuDictionaryItem{
				ID:   "mortal_gu_4",
				Name: "Tứ Chuyển Hỏa Cổ",
				Rank: 4,
				Path: []string{"fire"},
				Type: "PASSIVE",
				Effects: GuEffects{
					Passive: &PassiveEffect{
						StatBaseAttack: 50,
					},
				},
			},
		},
		"mortal_gu_3": {
			3: GuDictionaryItem{
				ID:   "mortal_gu_3",
				Name: "Tam Chuyển Cổ",
				Rank: 3,
				Path: []string{"fire"},
				Effects: GuEffects{
					Active: &ActiveEffect{
						DamageFlat: 30,
					},
				},
			},
		},
		"mortal_gu_2": {
			2: GuDictionaryItem{
				ID:   "mortal_gu_2",
				Name: "Nhị Chuyển Cổ",
				Rank: 2,
				Path: []string{"fire"},
				Type: "PASSIVE",
				Effects: GuEffects{
					Passive: &PassiveEffect{
						StatBaseHP: 100,
					},
				},
			},
		},
	}

	tests := []struct {
		name             string
		template         MonsterTemplate
		rank             int
		expectedDaoMarks bool
		expectedGuCount  int
		checkBossGu      bool
	}{
		{
			name: "Normal monster rank 1 - no dao marks, no gu",
			template: MonsterTemplate{
				ID:       "mon_1",
				Name:     "Sói Đá",
				Type:     "Normal",
				DaoMarks: DaoMarksMap{"fire": 100},
			},
			rank:             1,
			expectedDaoMarks: false,
			expectedGuCount:  0,
		},
		{
			name: "Normal monster rank 6 - no gu despite rank 6 and dao marks",
			template: MonsterTemplate{
				ID:       "mon_2",
				Name:     "Điện Lang",
				Type:     "Normal",
				DaoMarks: DaoMarksMap{"fire": 100},
			},
			rank:             6,
			expectedDaoMarks: true,
			expectedGuCount:  0,
		},
		{
			name: "Elite monster rank 3 - no dao marks because rank < 6, but has Gu",
			template: MonsterTemplate{
				ID:       "mon_3",
				Name:     "Thạch Hầu Vương",
				Type:     "Elite",
				DaoMarks: DaoMarksMap{"fire": 100},
			},
			rank:             3,
			expectedDaoMarks: false,
			expectedGuCount:  2,
		},
		{
			name: "Elite monster rank 6 - has dao marks and active+passive gu",
			template: MonsterTemplate{
				ID:       "mon_4",
				Name:     "Thạch Hầu Tiên",
				Type:     "Elite",
				DaoMarks: DaoMarksMap{"fire": 500},
			},
			rank:             6,
			expectedDaoMarks: true,
			expectedGuCount:  3,
		},
		{
			name: "Boss monster rank 6 - has dao marks and 5 active+passive gu",
			template: MonsterTemplate{
				ID:       "mon_5",
				Name:     "Cuồng Lang Vương",
				Type:     "Boss",
				DaoMarks: DaoMarksMap{"fire": 1000},
			},
			rank:             6,
			expectedDaoMarks: true,
			expectedGuCount:  5,
			checkBossGu:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := BuildCombatStateFromMonster(tt.template, tt.rank, 1)

			// Check DaoMarks
			if tt.expectedDaoMarks {
				if len(cs.EnemyDaoMarks) == 0 {
					t.Errorf("expected EnemyDaoMarks to be populated for rank %d, got empty", tt.rank)
				}
			} else {
				if len(cs.EnemyDaoMarks) > 0 {
					t.Errorf("expected EnemyDaoMarks to be empty for rank %d, got %v", tt.rank, cs.EnemyDaoMarks)
				}
			}

			// Check GuInventory count
			if len(cs.EnemyGuInventory) != tt.expectedGuCount {
				t.Errorf("expected %d gu items, got %d", tt.expectedGuCount, len(cs.EnemyGuInventory))
			}

			// For Boss, check if assigned Gu is strong combat Gu
			if tt.checkBossGu && len(cs.EnemyGuInventory) > 0 {
				foundStrong := false
				for _, guInst := range cs.EnemyGuInventory {
					if guInst.GuID == "fire_gu_strong" || guInst.GuID == "fire_gu_weak" {
						foundStrong = true
						break
					}
				}
				if !foundStrong {
					t.Errorf("expected Boss to receive a Rank 6 active Gu, got %v", cs.EnemyGuInventory)
				}
			}
		})
	}
}

func TestScaleAndRandomizeMonsterDaoMarks(t *testing.T) {
	mon := MonsterTemplate{
		ID:       "boss_test",
		Type:     "Boss",
		DaoMarks: DaoMarksMap{"fire": 1000},
	}

	dm6 := ScaleAndRandomizeMonsterDaoMarks(mon, 6, 1)
	if dm6["fire"] < 8000 || dm6["fire"] > 50000 {
		t.Errorf("expected rank 6 fire dao marks to be within range [8k, 50k], got %d", dm6["fire"])
	}

	dm8 := ScaleAndRandomizeMonsterDaoMarks(mon, 8, 10)
	if dm8["fire"] < 1000000 {
		t.Errorf("expected rank 8 subrank 10 fire dao marks to be near 1.4M, got %d", dm8["fire"])
	}

	dm9Sub1 := ScaleAndRandomizeMonsterDaoMarks(mon, 9, 1)
	dm9Sub10 := ScaleAndRandomizeMonsterDaoMarks(mon, 9, 10)
	if dm9Sub10["fire"] <= dm9Sub1["fire"] {
		t.Errorf("expected Rank 9 subrank 10 (%d) to be exponentially larger than subrank 1 (%d)", dm9Sub10["fire"], dm9Sub1["fire"])
	}

	monEmpty := MonsterTemplate{
		ID:   "boss_random",
		Type: "Boss",
	}

	dmRandom := ScaleAndRandomizeMonsterDaoMarks(monEmpty, 6, 1)
	if len(dmRandom) == 0 {
		t.Errorf("expected random dao marks to be generated for rank 6 boss, got empty")
	}
}

func TestBuildMonsterGuInventoryFromPools(t *testing.T) {
	prevRegistry := GuRegistry
	prevChecker := ImmortalGuChecker
	defer func() {
		GuRegistry = prevRegistry
		ImmortalGuChecker = prevChecker
	}()
	GuRegistry = map[string]map[int]GuDictionaryItem{}
	ImmortalGuChecker = nil

	one := func(guID string, rank int) []GuPoolEntry {
		return []GuPoolEntry{{GuID: guID, Rank: rank, Weight: 1, MinCount: 1, MaxCount: 1}}
	}

	t.Run("per-monster pool equips configured Gu for a Normal monster", func(t *testing.T) {
		mon := MonsterTemplate{ID: "mon_pool", Type: "Normal", GuPool: one("gu_monster", 1)}
		cs := BuildCombatStateFromMonsterWithRegion(mon, 1, 1, nil)
		if len(cs.EnemyGuInventory) != 1 || cs.EnemyGuInventory[0].GuID != "gu_monster" {
			t.Fatalf("expected monster gu_pool to equip gu_monster, got %+v", cs.EnemyGuInventory)
		}
	})

	t.Run("per-region pool equips configured Gu for a Normal monster", func(t *testing.T) {
		mon := MonsterTemplate{ID: "mon_region", Type: "Normal"}
		cs := BuildCombatStateFromMonsterWithRegion(mon, 2, 1, one("gu_region", 2))
		if len(cs.EnemyGuInventory) != 1 || cs.EnemyGuInventory[0].GuID != "gu_region" {
			t.Fatalf("expected region pool to equip gu_region, got %+v", cs.EnemyGuInventory)
		}
	})

	t.Run("per-monster pool wins over the region pool", func(t *testing.T) {
		mon := MonsterTemplate{ID: "mon_both", Type: "Normal", GuPool: one("gu_monster", 1)}
		cs := BuildCombatStateFromMonsterWithRegion(mon, 2, 1, one("gu_region", 2))
		if len(cs.EnemyGuInventory) != 1 || cs.EnemyGuInventory[0].GuID != "gu_monster" {
			t.Fatalf("expected monster pool to win, got %+v", cs.EnemyGuInventory)
		}
	})

	t.Run("Normal monster without any pool carries no Gu", func(t *testing.T) {
		mon := MonsterTemplate{ID: "mon_plain", Type: "Normal"}
		cs := BuildCombatStateFromMonsterWithRegion(mon, 3, 1, nil)
		if len(cs.EnemyGuInventory) != 0 {
			t.Fatalf("expected no Gu, got %+v", cs.EnemyGuInventory)
		}
	})

	t.Run("pool entries above the monster rank are filtered out", func(t *testing.T) {
		mon := MonsterTemplate{ID: "mon_lowrank", Type: "Normal", GuPool: one("gu_high", 5)}
		cs := BuildCombatStateFromMonsterWithRegion(mon, 2, 1, nil)
		if len(cs.EnemyGuInventory) != 0 {
			t.Fatalf("expected rank-5 entry to be filtered at rank 2, got %+v", cs.EnemyGuInventory)
		}
	})

	t.Run("immortal Gu already alive in the world is skipped", func(t *testing.T) {
		ImmortalGuChecker = func(guID string, rank int) bool { return guID == "gu_immortal" }
		defer func() { ImmortalGuChecker = nil }()
		mon := MonsterTemplate{ID: "mon_immortal", Type: "Boss", GuPool: one("gu_immortal", 6)}
		cs := BuildCombatStateFromMonsterWithRegion(mon, 6, 1, nil)
		if len(cs.EnemyGuInventory) != 0 {
			t.Fatalf("expected existing immortal Gu to be skipped, got %+v", cs.EnemyGuInventory)
		}
	})

	t.Run("Boss pool equips up to its slot budget", func(t *testing.T) {
		mon := MonsterTemplate{ID: "mon_boss_pool", Type: "Boss", GuPool: []GuPoolEntry{
			{GuID: "gu_1", Rank: 1, Weight: 1, MinCount: 1, MaxCount: 1},
			{GuID: "gu_2", Rank: 1, Weight: 1, MinCount: 1, MaxCount: 1},
		}}
		cs := BuildCombatStateFromMonsterWithRegion(mon, 1, 1, nil)
		if len(cs.EnemyGuInventory) != 2 {
			t.Fatalf("expected 2 Gu from the pool, got %d: %+v", len(cs.EnemyGuInventory), cs.EnemyGuInventory)
		}
	})
}
