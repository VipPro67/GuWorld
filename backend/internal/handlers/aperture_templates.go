package handlers

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type ResourceNodeTemplate struct {
	NodeType          string
	Name              string
	Path              string
	CostStones        int
	CostImmortal      int
	ProdType          string // "Material" | "Stones"
	ProdKey           string // e.g. "moon_orchid_petal"
	ProdAmount        int64
	ProdIntervalHours int
	NodeCategory      string   // "RESOURCE" | "DEFENSE"
	HPMax             int      // max HP for node (default 1000)
	DefensePower      int      // damage per tick (WATCHTOWER)
	AttackRange       int      // radius in cells (Manhattan distance)
	HPBonus           int      // bonus HP to nearby nodes (GUARDIAN_STATUE)
	DamageReduction   int      // % damage reduction (BARRACKS, FORCE_FIELD)
	BlocksMovement    bool     // wall blocks enemy movement
	CostGu            []string // required Gu IDs
	CostGuRank        int      // minimum rank for each required Gu
	FoodPath          string   // path of the food item produced (if any)
	ItemTier          string   // tier of the item (T1-T9)
	ZoneType          string   // zone type (khu_trong, khu_nuoi, khu_khai_thac, khu_che_bien, khu_phong_thu, khu_thuc_an)
	SeedAmount        int      // quantity of the produced item consumed as seed when placing the node (0 = no seed cost)
}

// ResourceNodeTemplates holds aperture node definitions loaded from
// data/ApertureNodeTemplate.csv at startup. Edit the CSV and restart to
// rebalance aperture nodes without a code change.
var ResourceNodeTemplates = loadResourceNodeTemplates()

func GetResourceNodeTemplate(nodeType string) (ResourceNodeTemplate, bool) {
	nodeType = strings.TrimSpace(nodeType)
	if len(ResourceNodeTemplates) == 0 {
		ResourceNodeTemplates = loadResourceNodeTemplates()
	}
	t, ok := ResourceNodeTemplates[nodeType]
	return t, ok
}

func loadResourceNodeTemplates() map[string]ResourceNodeTemplate {
	f, err := os.Open(csvPath("ApertureNodeTemplate.csv"))
	if err != nil {
		if logger.S != nil {
			logger.S.Warnf("aperture node templates not found: %v", err)
		}
		return map[string]ResourceNodeTemplate{}
	}
	defer f.Close()
	return parseResourceNodeTemplates(f)
}

func parseResourceNodeTemplates(r io.Reader) map[string]ResourceNodeTemplate {
	templates := make(map[string]ResourceNodeTemplate)

	rows, err := csv.NewReader(r).ReadAll()
	if err != nil || len(rows) < 2 {
		if logger.S != nil {
			logger.S.Errorf("failed to parse aperture node templates: %v", err)
		}
		return templates
	}
	header := make(map[string]int, len(rows[0]))
	for i, h := range rows[0] {
		header[strings.TrimSpace(h)] = i
	}
	atoi := func(s string) int {
		n, _ := strconv.Atoi(strings.TrimSpace(s))
		return n
	}
	for _, row := range rows[1:] {
		at := func(key string) string {
			if i, ok := header[key]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
		t := ResourceNodeTemplate{
			NodeType:          at("node_type"),
			Name:              at("name"),
			Path:              at("path"),
			CostStones:        atoi(at("cost_stones")),
			CostImmortal:      atoi(at("cost_immortal")),
			ProdType:          at("prod_type"),
			ProdKey:           at("prod_key"),
			ProdAmount:        int64(atoi(at("prod_amount"))),
			ProdIntervalHours: atoi(at("prod_interval_hours")),
			NodeCategory:      at("node_category"),
			HPMax:             atoi(at("hp_max")),
			DefensePower:      atoi(at("defense_power")),
			AttackRange:       atoi(at("attack_range")),
			HPBonus:           atoi(at("hp_bonus")),
			DamageReduction:   atoi(at("damage_reduction")),
			BlocksMovement:    at("blocks_movement") == "true",
			CostGuRank:        atoi(at("cost_gu_rank")),
			FoodPath:          at("food_path"),
			ItemTier:          at("item_tier"),
			ZoneType:          at("zone_type"),
			SeedAmount:        atoi(at("seed_amount")),
		}
		if cg := at("cost_gu"); cg != "" {
			t.CostGu = strings.Split(cg, ";")
		}
		if t.NodeType != "" {
			templates[t.NodeType] = t
		}
	}
	return templates
}
