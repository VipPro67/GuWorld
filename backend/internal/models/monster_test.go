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
				ID:         "fire_gu_passive_2",
				Name:       "Bão Hỏa Cổ",
				Rank:       6,
				IsImmortal: true,
				Path:       []string{"fire"},
				Type:       "PASSIVE",
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
				DaoMarks: map[string]int{"fire": 100},
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
				DaoMarks: map[string]int{"fire": 100},
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
				DaoMarks: map[string]int{"fire": 100},
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
				DaoMarks: map[string]int{"fire": 500},
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
				DaoMarks: map[string]int{"fire": 1000},
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
		DaoMarks: map[string]int{"fire": 1000},
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

