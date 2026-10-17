package handlers

import (
	"math/rand"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestLineCellsHorizontal(t *testing.T) {
	cells := lineCells(5, 5, 8, 5)
	want := [][2]int{{6, 5}, {7, 5}, {8, 5}}
	if len(cells) != len(want) {
		t.Fatalf("expected %d cells, got %d: %v", len(want), len(cells), cells)
	}
	for i, c := range want {
		if cells[i] != c {
			t.Fatalf("cell[%d] = %v, want %v", i, cells[i], c)
		}
	}
}

func TestLineCellsVertical(t *testing.T) {
	cells := lineCells(3, 9, 3, 6)
	want := [][2]int{{3, 8}, {3, 7}, {3, 6}}
	if len(cells) != len(want) {
		t.Fatalf("expected %d cells, got %d: %v", len(want), len(cells), cells)
	}
	for i, c := range want {
		if cells[i] != c {
			t.Fatalf("cell[%d] = %v, want %v", i, cells[i], c)
		}
	}
}

func TestProceduralMonsterCampsAreStableAndDenseNearCenter(t *testing.T) {
	zoneReg.mu.Lock()
	previousZones, previousPools, previousByID := zoneReg.zones, zoneReg.spawnPools, zoneReg.byID
	zoneReg.zones = []models.Region{{ID: "camp_test", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999, MinRank: 1, MaxRank: 4}}
	zoneReg.byID = map[string]*models.Region{"camp_test": &zoneReg.zones[0]}
	zoneReg.spawnPools = map[string][]models.MonsterSpawn{
		"camp_test": {
			{RegionID: "camp_test", MonsterID: "monster_test_pack", Chance: 1, MinRank: 2, MaxRank: 3},
			{RegionID: "camp_test", MonsterID: "monster_test_elite", Chance: 1, MinRank: 4, MaxRank: 5},
		},
	}
	zoneReg.mu.Unlock()
	previousNormal, hadNormal := models.MonsterRegistry["monster_test_pack"]
	previousElite, hadElite := models.MonsterRegistry["monster_test_elite"]
	models.MonsterRegistry["monster_test_pack"] = models.MonsterTemplate{ID: "monster_test_pack", Type: "Normal"}
	models.MonsterRegistry["monster_test_elite"] = models.MonsterTemplate{ID: "monster_test_elite", Type: "Elite"}
	t.Cleanup(func() {
		zoneReg.mu.Lock()
		zoneReg.zones, zoneReg.spawnPools, zoneReg.byID = previousZones, previousPools, previousByID
		zoneReg.mu.Unlock()
		if hadNormal {
			models.MonsterRegistry["monster_test_pack"] = previousNormal
		} else {
			delete(models.MonsterRegistry, "monster_test_pack")
		}
		if hadElite {
			models.MonsterRegistry["monster_test_elite"] = previousElite
		} else {
			delete(models.MonsterRegistry, "monster_test_elite")
		}
	})

	var center proceduralMonsterCamp
	found := false
	for gy := 2; gy < 14 && !found; gy++ {
		for gx := 2; gx < 14; gx++ {
			rng := rand.New(rand.NewSource(GetCellSeed(gx, gy, "camp_test:monster-camp")))
			cx := gx*monsterCampSpacing + 8 + rng.Intn(monsterCampSpacing-16)
			cy := gy*monsterCampSpacing + 8 + rng.Intn(monsterCampSpacing-16)
			camp, ok := findProceduralMonsterCamp(cx, cy, ZoneByID("camp_test"))
			if ok && camp.distance == 0 {
				center, found = camp, true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected a deterministic camp in the sample area")
	}
	if center.spawn.MonsterID != "monster_test_pack" && center.spawn.MonsterID != "monster_test_elite" {
		t.Fatalf("camp spawn should come from the region pool: %+v", center.spawn)
	}
	if center.spawn.MinRank < 2 || center.spawn.MinRank > 5 {
		t.Fatalf("camp rank out of pool band: %+v", center.spawn)
	}
	got, ok := findProceduralMonsterCamp(center.x, center.y, ZoneByID("camp_test"))
	if !ok || got.x != center.x || got.y != center.y || got.spawn.MonsterID != center.spawn.MonsterID || got.spawn.MinRank != center.spawn.MinRank {
		t.Fatalf("camp center changed across resolutions: first=%+v second=%+v", center, got)
	}
	neighbor, ok := findProceduralMonsterCamp(center.x+1, center.y, ZoneByID("camp_test"))
	if !ok || neighbor.distance != 1 || neighbor.spawn.MonsterID != center.spawn.MonsterID || neighbor.spawn.MinRank != center.spawn.MinRank {
		t.Fatalf("cells within a camp should share its monster and rank: center=%+v neighbor=%+v", center, neighbor)
	}
	monstersInCamp := 0
	for y := center.y - monsterCampRadius; y <= center.y+monsterCampRadius; y++ {
		for x := center.x - monsterCampRadius; x <= center.x+monsterCampRadius; x++ {
			cell := generateCellForZone(x, y, ZoneByID("camp_test"))
			if cell.Type == "Monster" {
				if cell.Content != "monster_test_pack" && cell.Content != "monster_test_elite" {
					t.Fatalf("camp cell (%d,%d) has unexpected monster %q", x, y, cell.Content)
				}
				monstersInCamp++
			}
		}
	}
	if monstersInCamp < 12 {
		t.Fatalf("camp generated only %d monster cells around its center", monstersInCamp)
	}
	if monsterCampSpawnChance(0) <= monsterCampSpawnChance(monsterCampRadius) {
		t.Fatal("monster spawn chance should taper toward the camp edge")
	}
}

// TestZoneWithoutSpawnRowsSpawnsNoMonster guards the rule that a region only
// spawns monsters listed in monster_spawns; it must never fall back to a
// template from another region.
func TestZoneWithoutSpawnRowsSpawnsNoMonster(t *testing.T) {
	zoneReg.mu.Lock()
	previousZones, previousPools, previousByID := zoneReg.zones, zoneReg.spawnPools, zoneReg.byID
	zoneReg.zones = []models.Region{{ID: "empty_zone_test", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999, MinRank: 1, MaxRank: 4,
		PoolWeights: models.ZonePools{CellWeights: map[string]int{"Monster": 100, "Empty": 0}}}}
	zoneReg.byID = map[string]*models.Region{"empty_zone_test": &zoneReg.zones[0]}
	zoneReg.spawnPools = map[string][]models.MonsterSpawn{}
	zoneReg.mu.Unlock()
	t.Cleanup(func() {
		zoneReg.mu.Lock()
		zoneReg.zones, zoneReg.spawnPools, zoneReg.byID = previousZones, previousPools, previousByID
		zoneReg.mu.Unlock()
	})

	for x := 0; x < 80; x++ {
		for y := 0; y < 80; y++ {
			if cell := generateCellForZone(x, y, ZoneByID("empty_zone_test")); cell.Type == "Monster" {
				t.Fatalf("zone without monster_spawns rows spawned %q at (%d,%d)", cell.Content, x, y)
			}
		}
	}
}

// TestEliteShareIsDampedInAmbientPool verifies Elite templates now share the
// ambient pool but are made rarer by eliteSpawnWeightFactor, and that an
// Elite-only pool still spawns its elite.
func TestEliteShareIsDampedInAmbientPool(t *testing.T) {
	zoneReg.mu.Lock()
	previousPools := zoneReg.spawnPools
	zoneReg.spawnPools = map[string][]models.MonsterSpawn{
		"elite_pool_test": {
			{RegionID: "elite_pool_test", MonsterID: "monster_pool_normal", Chance: 1},
			{RegionID: "elite_pool_test", MonsterID: "monster_pool_elite", Chance: 1},
		},
	}
	zoneReg.mu.Unlock()
	prevN, hadN := models.MonsterRegistry["monster_pool_normal"]
	prevE, hadE := models.MonsterRegistry["monster_pool_elite"]
	models.MonsterRegistry["monster_pool_normal"] = models.MonsterTemplate{ID: "monster_pool_normal", Type: "Normal"}
	models.MonsterRegistry["monster_pool_elite"] = models.MonsterTemplate{ID: "monster_pool_elite", Type: "Elite"}
	t.Cleanup(func() {
		zoneReg.mu.Lock()
		zoneReg.spawnPools = previousPools
		zoneReg.mu.Unlock()
		if hadN {
			models.MonsterRegistry["monster_pool_normal"] = prevN
		} else {
			delete(models.MonsterRegistry, "monster_pool_normal")
		}
		if hadE {
			models.MonsterRegistry["monster_pool_elite"] = prevE
		} else {
			delete(models.MonsterRegistry, "monster_pool_elite")
		}
	})

	eliteHits, normalHits := 0, 0
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 10000; i++ {
		s, ok := pickMonsterFromPool("elite_pool_test", rng, false)
		if !ok {
			t.Fatal("ambient pool pick failed")
		}
		switch s.MonsterID {
		case "monster_pool_elite":
			eliteHits++
		case "monster_pool_normal":
			normalHits++
		}
	}
	if eliteHits == 0 || normalHits == 0 {
		t.Fatalf("both templates should appear: elite=%d normal=%d", eliteHits, normalHits)
	}
	if eliteHits*2 >= normalHits {
		t.Fatalf("elite not damped by eliteSpawnWeightFactor: elite=%d normal=%d", eliteHits, normalHits)
	}

	zoneReg.mu.Lock()
	zoneReg.spawnPools["elite_pool_test"] = []models.MonsterSpawn{{RegionID: "elite_pool_test", MonsterID: "monster_pool_elite", Chance: 1}}
	zoneReg.mu.Unlock()
	if s, ok := pickMonsterFromPool("elite_pool_test", rand.New(rand.NewSource(2)), false); !ok || s.MonsterID != "monster_pool_elite" {
		t.Fatalf("elite-only pool should still spawn the elite: %+v ok=%v", s, ok)
	}
}

func TestLineCellsDiagonal(t *testing.T) {
	cells := lineCells(0, 0, 3, 3)
	want := [][2]int{{1, 1}, {2, 2}, {3, 3}}
	if len(cells) != len(want) {
		t.Fatalf("expected %d cells, got %d: %v", len(want), len(cells), cells)
	}
	for i, c := range want {
		if cells[i] != c {
			t.Fatalf("cell[%d] = %v, want %v", i, cells[i], c)
		}
	}
}

func TestLineCellsSamePosition(t *testing.T) {
	if cells := lineCells(4, 4, 4, 4); len(cells) != 0 {
		t.Fatalf("expected no cells for same start/end, got %v", cells)
	}
}

func TestLineCellsEndsAtTarget(t *testing.T) {
	cells := lineCells(10, 20, 14, 22)
	last := cells[len(cells)-1]
	if last[0] != 14 || last[1] != 22 {
		t.Fatalf("expected last cell to be target, got %v", last)
	}
	// Every step must be adjacent to the previous one (Chebyshev distance 1).
	prevX, prevY := 10, 20
	for _, c := range cells {
		dx := c[0] - prevX
		dy := c[1] - prevY
		if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
			t.Fatalf("non-adjacent step %v from %d,%d", c, prevX, prevY)
		}
		prevX, prevY = c[0], c[1]
	}
}

// TestGenerateCellStableAcrossCalls verifies cell content is deterministic and
// does not re-randomize between calls (no time-window shuffle, no map-order
// dependence). Regression for "monsters keep resetting" on the world map.
func TestGenerateCellStableAcrossCalls(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_test", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999,
			PoolWeights: models.ZonePools{
				CellWeights: map[string]int{"Monster": 25, "Resource": 15, "WildGu": 5, "Empty": 55},
				Resources:   []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone"},
				WildGus:     []string{"GU_LITTLE_LIGHT", "GU_JADE_SKIN"},
			}},
	}
	zoneReg.spawnPools = map[string][]models.MonsterSpawn{
		"dai_vuc_test": {
			{RegionID: "dai_vuc_test", MonsterID: "monster_wild_boar", Chance: 1.0, SpawnMode: "Random", MinRank: 1, MaxRank: 3, IsActive: true},
		},
	}
	zoneReg.mu.Unlock()

	// Compare many cells across repeated resolutions — content must be identical.
	for i := 0; i < 200; i++ {
		x := 100 + i%50
		y := 200 + i%50
		c1 := ResolveCell(x, y)
		c2 := ResolveCell(x, y)
		if c1.Type != c2.Type || c1.Content != c2.Content {
			t.Fatalf("cell (%d,%d) not stable: %+v vs %+v", x, y, c1, c2)
		}
	}
}

func TestIsSafeZone(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "Southern_Border", ZoneType: ZoneTypeHub, MinX: 14000, MinY: 9000, MaxX: 16000, MaxY: 11000},
		{ID: "hub_trung_chau", ZoneType: ZoneTypeHub, MinX: 14950, MinY: 4950, MaxX: 15050, MaxY: 5050},
		{ID: "tan_thu_coc_trung_chau", ZoneType: ZoneTypeSafe, MinX: 14500, MinY: 4500, MaxX: 15500, MaxY: 5500},
		{ID: "dai_vuc_nam_cuong", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999},
	}
	zoneReg.byID = make(map[string]*models.Region)
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()

	if !IsSafeZone("Southern_Border", 15000, 10000) {
		t.Errorf("expected Southern_Border at (15000, 10000) to be a safe zone")
	}
	if !IsSafeZone("hub_trung_chau", 15000, 5000) {
		t.Errorf("expected hub_trung_chau to be a safe zone")
	}
	if !IsSafeZone("tan_thu_coc_trung_chau", 15000, 5000) {
		t.Errorf("expected tan_thu_coc_trung_chau to be a safe zone")
	}
	if IsSafeZone("dai_vuc_nam_cuong", 1000, 1000) {
		t.Errorf("expected wilderness dai_vuc_nam_cuong at (1000, 1000) NOT to be a safe zone")
	}
}
