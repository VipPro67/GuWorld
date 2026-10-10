package db

import (
	"fmt"
	"path/filepath"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Dataset describes every capability and runtime side effect of an
// administrator-managed CSV dataset. Function fields stay server-side while
// the remaining fields form the contract consumed by the Admin UI.
type Dataset struct {
	Key               string             `json:"key"`
	Label             string             `json:"label"`
	CSVFile           string             `json:"csv_file"`
	Import            func() error       `json:"-"`
	Export            func() error       `json:"-"`
	Preflight         func() error       `json:"-"`
	CacheInvalidation []string           `json:"cache_invalidation"`
	RegistryReload    []string           `json:"registry_reload"`
	Permissions       DatasetPermissions `json:"permissions"`
	UISupported       bool               `json:"ui_supported"`
}

type DatasetPermissions struct {
	Read   string `json:"read"`
	Import string `json:"import,omitempty"`
	Export string `json:"export,omitempty"`
}

// DatasetCapabilities is serialized explicitly so implementation presence,
// rather than duplicated booleans, is the source of truth.
type DatasetCapabilities struct {
	Import    bool `json:"import"`
	Export    bool `json:"export"`
	Preflight bool `json:"preflight"`
}

type DatasetInfo struct {
	Key               string              `json:"key"`
	Label             string              `json:"label"`
	CSVFile           string              `json:"csv_file"`
	Capabilities      DatasetCapabilities `json:"capabilities"`
	CacheInvalidation []string            `json:"cache_invalidation"`
	RegistryReload    []string            `json:"registry_reload"`
	Permissions       DatasetPermissions  `json:"permissions"`
	UISupported       bool                `json:"ui_supported"`
}

var knownDatasetCacheBehaviors = map[string]bool{
	"none": true, "dictionary": true, "recipe_catalog": true,
	"shop_catalog": true, "arena_shop": true, "quests": true,
	"world_monsters": true,
}

var knownDatasetRegistryReloads = map[string]bool{
	"none": true, "item": true, "gu": true, "constitution": true,
	"race": true, "status": true, "monster": true, "zone": true,
	"quest": true, "instance": true,
}

func IsKnownDatasetCacheBehavior(behavior string) bool  { return knownDatasetCacheBehaviors[behavior] }
func IsKnownDatasetRegistryReload(registry string) bool { return knownDatasetRegistryReloads[registry] }

var standardDatasetPermissions = DatasetPermissions{
	Read: "game_data.read", Import: "game_data.import", Export: "game_data.export",
}

func noDatasetPreflight() error { return nil }

func effectPreflight(table, filename, key string) func() error {
	return func() error {
		return preflightEffectCSVs([]effectCSVSource{{source: table, filename: filename, keyColumn: key}})
	}
}

func datasetExport(filename string, columns []string, query string) func() error {
	return func() error {
		ensureDataDir()
		path := filepath.Join("data", filename)
		if err := writeCSV(path, columns, query); err != nil {
			return fmt.Errorf("export %s: %w", path, err)
		}
		logger.S.Infof("Exported %s", path)
		return nil
	}
}

func dataset(key, label, filename string, importFn func() error, exportDef exportDef, preflight func() error, caches, reloads []string) Dataset {
	var exportFn func() error
	if exportDef.filename != "" {
		exportFn = datasetExport(exportDef.filename, exportDef.columns, exportDef.query)
	}
	return Dataset{Key: key, Label: label, CSVFile: filename, Import: importFn, Export: exportFn,
		Preflight: preflight, CacheInvalidation: caches, RegistryReload: reloads,
		Permissions: standardDatasetPermissions, UISupported: true}
}

// Datasets is the single ordered registry used by DB dispatch, handlers and UI.
func Datasets() []Dataset {
	dictionary := []string{"dictionary"}
	none := []string{"none"}
	return []Dataset{
		dataset("item", "Vật phẩm", "Item.csv", func() error { return seedItemsFromCSV("Item.csv") }, individualExportDefs["item"], effectPreflight("item", "Item.csv", "id"), dictionary, []string{"item"}),
		dataset("gu", "Cổ trùng", "Gu.csv", func() error { return seedGusFromCSV("Gu.csv") }, individualExportDefs["gu"], effectPreflight("gu", "Gu.csv", "id"), dictionary, []string{"gu"}),
		dataset("constitution", "Thể chất", "Constitution.csv", func() error { return seedConstitutionsFromCSV("Constitution.csv") }, individualExportDefs["constitution"], effectPreflight("constitution", "Constitution.csv", "key"), dictionary, []string{"constitution"}),
		dataset("race", "Chủng tộc", "Race.csv", func() error { return seedRacesFromCSV("Race.csv") }, individualExportDefs["race"], effectPreflight("race", "Race.csv", "key"), dictionary, []string{"race"}),
		dataset("status", "Trạng thái", "Status.csv", func() error { return seedStatusesFromCSV("Status.csv") }, individualExportDefs["status"], effectPreflight("status", "Status.csv", "key"), dictionary, []string{"status"}),
		dataset("recipe", "Cổ phương", "Recipe.csv", func() error { return seedRecipesFromCSV("Recipe.csv") }, individualExportDefs["recipe"], noDatasetPreflight, []string{"recipe_catalog"}, []string{"none"}),
		dataset("shop_item", "Cửa hàng", "ShopItem.csv", func() error { return seedShopItemsFromCSV("ShopItem.csv") }, individualExportDefs["shop_item"], noDatasetPreflight, []string{"shop_catalog"}, none),
		dataset("arena_shop_item", "Cửa hàng Đấu trường", "ArenaShopItem.csv", func() error { return seedArenaShopItemsFromCSV("ArenaShopItem.csv") }, individualExportDefs["arena_shop_item"], noDatasetPreflight, []string{"arena_shop"}, none),
		dataset("region", "Khu vực", "Region.csv", func() error { return seedRegionsFromCSV("Region.csv") }, individualExportDefs["region"], noDatasetPreflight, []string{"dictionary", "world_monsters"}, []string{"zone", "monster"}),
		dataset("loot_table", "Bảng loot", "LootTable.csv", func() error { return seedLootTablesFromCSV("LootTable.csv") }, individualExportDefs["loot_table"], noDatasetPreflight, dictionary, []string{"monster"}),
		dataset("monster", "Quái vật", "Monster.csv", func() error { return seedMonsterTemplatesFromCSV("Monster.csv") }, individualExportDefs["monster"], noDatasetPreflight, dictionary, []string{"monster"}),
		dataset("monster_spawn", "Spawn quái", "MonsterSpawn.csv", func() error { return seedMonsterSpawnsFromCSV("MonsterSpawn.csv") }, individualExportDefs["monster_spawn"], noDatasetPreflight, dictionary, []string{"zone", "monster"}),
		dataset("map_cell", "Bản đồ", "MapCell.csv", func() error { return seedMapCellsFromCSV("MapCell.csv") }, individualExportDefs["map_cell"], noDatasetPreflight, dictionary, []string{"zone", "monster"}),
		dataset("sect_mission", "Nhiệm vụ tông môn", "SectMission.csv", func() error { return seedSectMissionsFromCSV("SectMission.csv") }, individualExportDefs["sect_mission"], noDatasetPreflight, none, none),
		dataset("quest", "Nhiệm vụ", "Quest.csv", seedQuestsFromCSV, individualExportDefs["quest"], noDatasetPreflight, []string{"quests"}, []string{"quest"}),
		dataset("dream_realm", "Mộng cảnh", "DreamRealm.csv", func() error { return seedDreamRealmsFromCSV("DreamRealm.csv") }, individualExportDefs["dream_realm"], noDatasetPreflight, none, []string{"instance"}),
	}
}

func DatasetByKey(key string) (Dataset, bool) {
	for _, d := range Datasets() {
		if d.Key == key {
			return d, true
		}
	}
	return Dataset{}, false
}

func DatasetInfos() []DatasetInfo {
	infos := make([]DatasetInfo, 0, len(Datasets()))
	for _, d := range Datasets() {
		infos = append(infos, DatasetInfo{Key: d.Key, Label: d.Label, CSVFile: d.CSVFile,
			Capabilities:      DatasetCapabilities{Import: d.Import != nil, Export: d.Export != nil, Preflight: d.Preflight != nil},
			CacheInvalidation: d.CacheInvalidation, RegistryReload: d.RegistryReload,
			Permissions: d.Permissions, UISupported: d.UISupported})
	}
	return infos
}
