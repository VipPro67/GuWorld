package nav

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

const (
	canonicalMapPath = "../../../data/mmo/maps/realtime_preview.json"
	clientMapPath    = "../../../../frontend/public/game/maps/realtime_preview.json"
)

func TestPreviewMapParsesAndBlocksBorders(t *testing.T) {
	loaded, err := LoadFile(canonicalMapPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "realtime_preview" || loaded.Version != "preview-1" || loaded.Width != 64 || loaded.Height != 48 {
		t.Fatalf("unexpected map identity: %+v", loaded)
	}
	if loaded.Ruleset != "pve" {
		t.Fatalf("ruleset=%q, want pve", loaded.Ruleset)
	}
	if loaded.Hash == "" {
		t.Fatal("map hash is empty")
	}
	if loaded.Hash != "2a23f2f6ba350e15c5707309d7581536cd038b0c65f2edfa390f9b128c9f631b" {
		t.Fatalf("preview map hash drifted from the shared client expectation: %s", loaded.Hash)
	}
	if !loaded.IsSolid(0.5, 0.5) || !loaded.IsSolid(63.5, 20.5) || !loaded.IsSolid(0.5, 47.5) {
		t.Fatal("border tiles must be solid")
	}
	if loaded.IsSolid(8.5, 8.5) || loaded.IsSolid(25.5, 18.5) {
		t.Fatal("authored spawn tiles must be walkable")
	}
	if len(loaded.Spawns) < 3 {
		t.Fatalf("expected authored spawns, got %d", len(loaded.Spawns))
	}
	if len(loaded.Portals) != 1 || loaded.Portals[0].TargetZoneID != "realtime_preview" || loaded.Portals[0].TargetX != 8.5 {
		t.Fatalf("portal metadata was not parsed: %+v", loaded.Portals)
	}
	for _, spawn := range loaded.Spawns {
		if loaded.IsSolid(spawn.X, spawn.Y) {
			t.Fatalf("spawn %q is inside a solid tile", spawn.Name)
		}
		if spawn.Kind == "monster" && spawn.MonsterID == "" {
			t.Fatalf("monster spawn %q is missing monster_id", spawn.Name)
		}
	}
}

func TestConvertedLegacyWorldMapHasExpectedRealtimeTransformAndMonsters(t *testing.T) {
	loaded, err := LoadFile("../../../data/mmo/maps/world_map.json")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "world_map" || loaded.Version != "legacy-converted-v2" || loaded.ContentVersion != "legacy-regions-v2" {
		t.Fatalf("unexpected converted map identity: id=%q map=%q content=%q", loaded.ID, loaded.Version, loaded.ContentVersion)
	}
	if loaded.Width != 1200 || loaded.Height != 800 || loaded.LegacyScale != 25 {
		t.Fatalf("unexpected converted map transform: %dx%d scale=%v", loaded.Width, loaded.Height, loaded.LegacyScale)
	}
	players, monsters := 0, 0
	for _, spawn := range loaded.Spawns {
		switch spawn.Kind {
		case "player":
			players++
		case "monster":
			monsters++
			// 0.8 is the largest current monster archetype radius. Checking the
			// full collision circle catches spawns that are walkable by center
			// but still intersect an edge wall.
			if loaded.Collision().Blocked(sim.Vec2{X: spawn.X, Y: spawn.Y}, 0.8) {
				t.Fatalf("monster spawn %q intersects collision at its max archetype radius: (%v, %v)", spawn.Name, spawn.X, spawn.Y)
			}
		}
	}
	if players == 0 || monsters != 500 {
		t.Fatalf("converted map spawn counts: players=%d monsters=%d, want player entries and 500 monsters", players, monsters)
	}
}

func TestSafeSpawnFallsBackForLegacyCoordinates(t *testing.T) {
	loaded, err := LoadFile(canonicalMapPath)
	if err != nil {
		t.Fatal(err)
	}
	spawn, ok := loaded.SafeSpawn(sim.Vec2{X: 100_000, Y: 100_000}, 0.4)
	if !ok {
		t.Fatal("no safe spawn found for out-of-map legacy position")
	}
	if loaded.IsSolid(spawn.X, spawn.Y) {
		t.Fatalf("safe spawn is solid: %+v", spawn)
	}
	inside, ok := loaded.SafeSpawn(sim.Vec2{X: 8.5, Y: 8.5}, 0.4)
	if !ok || inside != (sim.Vec2{X: 8.5, Y: 8.5}) {
		t.Fatalf("walkable preferred position should be preserved: %+v ok=%v", inside, ok)
	}
}

func TestServerAndClientMapCopiesStayIdentical(t *testing.T) {
	serverBytes, err := os.ReadFile(canonicalMapPath)
	if err != nil {
		t.Fatal(err)
	}
	clientBytes, err := os.ReadFile(clientMapPath)
	if err != nil {
		t.Fatalf("client map copy missing: %v", err)
	}
	if !bytes.Equal(serverBytes, clientBytes) {
		t.Fatal("server and client map copies drifted; copy backend/data/mmo/maps/*.json into frontend/public/game/maps/")
	}
}

func TestAlphaMapsParseAndLink(t *testing.T) {
	serverDir := "../../../data/mmo/maps"
	clientDir := "../../../../frontend/public/game/maps"
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		t.Fatal(err)
	}
	loaded := map[string]*Map{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(serverDir, entry.Name())
		artifact, parseErr := LoadFile(path)
		if parseErr != nil {
			t.Fatalf("%s: %v", entry.Name(), parseErr)
		}
		if artifact.ID+".json" != entry.Name() {
			t.Fatalf("%s declares map_id %q", entry.Name(), artifact.ID)
		}
		if _, duplicate := loaded[artifact.ID]; duplicate {
			t.Fatalf("duplicate map %q", artifact.ID)
		}
		loaded[artifact.ID] = artifact

		serverBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		clientBytes, readErr := os.ReadFile(filepath.Join(clientDir, entry.Name()))
		if readErr != nil {
			t.Fatalf("client copy for %s missing: %v", entry.Name(), readErr)
		}
		if !bytes.Equal(serverBytes, clientBytes) {
			t.Fatalf("%s drifted; copy backend/data/mmo/maps/*.json into frontend/public/game/maps/", entry.Name())
		}
	}
	if len(loaded) < 4 {
		t.Fatalf("expected at least 4 alpha maps, got %d", len(loaded))
	}
	for id, artifact := range loaded {
		if len(artifact.Spawns) == 0 {
			t.Fatalf("map %q has no spawn points", id)
		}
		for _, spawn := range artifact.Spawns {
			if artifact.IsSolid(spawn.X, spawn.Y) {
				t.Fatalf("map %q spawn %q is solid", id, spawn.Name)
			}
			if spawn.Kind == "monster" && spawn.MonsterID == "" {
				t.Fatalf("map %q monster spawn %q lacks monster_id", id, spawn.Name)
			}
		}
		for _, portal := range artifact.Portals {
			target, ok := loaded[portal.TargetZoneID]
			if !ok {
				t.Fatalf("map %q portal %q targets unknown map %q", id, portal.Name, portal.TargetZoneID)
			}
			if portal.TargetX < 0 || portal.TargetY < 0 || portal.TargetX >= float64(target.Width) || portal.TargetY >= float64(target.Height) {
				t.Fatalf("map %q portal %q target is outside %q", id, portal.Name, target.ID)
			}
			if target.IsSolid(portal.TargetX, portal.TargetY) {
				t.Fatalf("map %q portal %q target is solid in %q", id, portal.Name, target.ID)
			}
		}
	}
	if len(loaded["crimson_ravine"].Portals) == 0 || loaded["crimson_ravine"].Ruleset != "pvp" {
		t.Fatal("crimson_ravine must be a linked PvP map")
	}
	if loaded["emerald_grotto"].Ruleset != "safe" {
		t.Fatal("emerald_grotto must be a safe hub")
	}
}

func TestAlphaMapMonsterIDsExistInCSV(t *testing.T) {
	file, err := os.Open("../../../data/Monster.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatal("Monster.csv is empty")
	}
	idColumn := -1
	for index, header := range rows[0] {
		if strings.TrimPrefix(header, "\ufeff") == "id" {
			idColumn = index
			break
		}
	}
	if idColumn < 0 {
		t.Fatal("Monster.csv has no id column")
	}
	known := make(map[string]struct{}, len(rows)-1)
	for _, row := range rows[1:] {
		if idColumn < len(row) {
			known[row[idColumn]] = struct{}{}
		}
	}
	entries, err := os.ReadDir("../../../data/mmo/maps")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		artifact, parseErr := LoadFile(filepath.Join("../../../data/mmo/maps", entry.Name()))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		for _, spawn := range artifact.Spawns {
			if spawn.Kind != "monster" {
				continue
			}
			if _, ok := known[spawn.MonsterID]; !ok {
				t.Fatalf("map %q spawn %q references unknown Monster.csv id %q", artifact.ID, spawn.Name, spawn.MonsterID)
			}
		}
	}
}

func TestBuildRejectsUnsafeMaps(t *testing.T) {
	valid := func() TiledMapJSON {
		return TiledMapJSON{
			Type: "map", Orientation: "orthogonal", Width: 4, Height: 4, TileWidth: 32, TileHeight: 32,
			MapID: "test", MapVersion: "v1",
			Layers: []TiledLayer{
				{Type: "tilelayer", Name: "collision", Data: []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
			},
		}
	}
	cases := map[string]func(*TiledMapJSON){
		"orientation": func(m *TiledMapJSON) { m.Orientation = "isometric" },
		"dimensions":  func(m *TiledMapJSON) { m.Width = 0 },
		"no collision": func(m *TiledMapJSON) {
			m.Layers = nil
		},
		"collision length": func(m *TiledMapJSON) { m.Layers[0].Data = m.Layers[0].Data[:4] },
		"spawn in wall": func(m *TiledMapJSON) {
			m.Layers[0].Data[5] = 1
			m.Layers = append(m.Layers, TiledLayer{Type: "objectgroup", Name: "spawns", Objects: []TiledObject{{Type: "player", X: 1.5, Y: 1.5}}})
		},
		"spawn out of bounds": func(m *TiledMapJSON) {
			m.Layers = append(m.Layers, TiledLayer{Type: "objectgroup", Name: "spawns", Objects: []TiledObject{{Type: "player", X: 90, Y: 1.5}}})
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			raw := valid()
			mutate(&raw)
			if _, err := Build(raw); err == nil {
				t.Fatal("invalid map was accepted")
			}
		})
	}
}

func TestProceduralBorderMap(t *testing.T) {
	raw := TiledMapJSON{
		Type: "map", Orientation: "orthogonal", Width: 2000, Height: 2000, TileWidth: 24, TileHeight: 24,
		MapID: "test_sparse", MapVersion: "v1",
		Layers: []TiledLayer{
			{Type: "tilelayer", Name: "ground", Data: []int{}},
			{Type: "tilelayer", Name: "collision", Data: []int{}},
			{Type: "objectgroup", Name: "spawns", Objects: []TiledObject{{Type: "player", Name: "spawn", X: 1000, Y: 1000}}},
		},
	}
	loaded, err := Build(raw)
	if err != nil {
		t.Fatalf("procedural map build failed: %v", err)
	}
	if loaded.Width != 2000 || loaded.Height != 2000 {
		t.Fatalf("unexpected dimensions: %dx%d", loaded.Width, loaded.Height)
	}
	// Borders must be solid
	if !loaded.IsSolid(0.5, 0.5) || !loaded.IsSolid(1999.5, 1000) || !loaded.IsSolid(500, 0.5) || !loaded.IsSolid(500, 1999.5) {
		t.Fatal("border tiles must be solid")
	}
	// Center must be walkable
	if loaded.IsSolid(1000, 1000) || loaded.IsSolid(2, 2) || loaded.IsSolid(1998, 1998) {
		t.Fatal("interior tiles must be walkable")
	}
}
