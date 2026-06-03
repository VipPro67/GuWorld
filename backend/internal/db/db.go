package db

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// Connect establishes a connection to the PostgreSQL database
func Connect() error {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "gu_admin")
	password := getEnv("DB_PASSWORD", "gu_pass_123")
	dbname := getEnv("DB_NAME", "gu_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to the database")
	return nil
}

// InitSchema creates the necessary tables and seeds initial master data if missing
func InitSchema() error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	_, err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	if err != nil {
		log.Printf("Warning: failed to create uuid-ossp extension: %v", err)
	}

	// 1. Create item_dictionary
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS item_dictionary (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		item_type VARCHAR(50) NOT NULL,
		rarity INTEGER DEFAULT 1,
		rank INTEGER DEFAULT 1,
		path VARCHAR(50) NOT NULL DEFAULT 'None',
		effects JSONB NOT NULL DEFAULT '{}',
		tags JSONB NOT NULL DEFAULT '[]',
		description TEXT
	);`)
	if err != nil {
		return fmt.Errorf("failed to create item_dictionary table: %w", err)
	}

	// 2. Create gu_dictionary
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS gu_dictionary (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		rank INTEGER NOT NULL,
		path VARCHAR(50) NOT NULL DEFAULT 'Moon Path',
		gu_type VARCHAR(50) NOT NULL,
		base_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
		effects JSONB NOT NULL DEFAULT '{}',
		hunger_rate INTEGER NOT NULL,
		food_types JSONB NOT NULL DEFAULT '[]',
		max_hunger INTEGER DEFAULT 100,
		description TEXT,
		target_type VARCHAR(50) NOT NULL DEFAULT 'SINGLE'
	);`)
	if err != nil {
		return fmt.Errorf("failed to create gu_dictionary table: %w", err)
	}

	// 3. Create refining_recipes
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS refining_recipes (
		id VARCHAR(50) PRIMARY KEY,
		target_gu_id VARCHAR(50) NOT NULL REFERENCES gu_dictionary(id) ON DELETE CASCADE,
		required_gus JSONB NOT NULL,
		required_materials JSONB NOT NULL,
		success_rate DOUBLE PRECISION NOT NULL,
		primeval_stones_cost INTEGER DEFAULT 0,
		immortal_essence_stones_cost DOUBLE PRECISION DEFAULT 0
	);`)
	if err != nil {
		return fmt.Errorf("failed to create refining_recipes table: %w", err)
	}

	// 4. Create players table
	query := `
	CREATE TABLE IF NOT EXISTS players (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(50) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL DEFAULT '',
		rank INTEGER DEFAULT 1,
		sub_rank INTEGER DEFAULT 1,
		aptitude_type VARCHAR(10) NOT NULL,
		aptitude_percent INTEGER NOT NULL,
		current_mana DOUBLE PRECISION NOT NULL,
		max_mana DOUBLE PRECISION NOT NULL,
		primeval_stones INTEGER DEFAULT 0,
		immortal_stones DOUBLE PRECISION DEFAULT 0,
		gu_inventory JSONB NOT NULL DEFAULT '[]',
		materials_inventory JSONB NOT NULL DEFAULT '{}',
		hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		base_hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		body_mods JSONB NOT NULL DEFAULT '[]',
		x INTEGER NOT NULL DEFAULT 50,
		y INTEGER NOT NULL DEFAULT 50,
		explored_cells JSONB NOT NULL DEFAULT '["50,50"]',
		current_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border',
		dao_marks JSONB NOT NULL DEFAULT '{"strength": 0}',
		path_masteries JSONB NOT NULL DEFAULT '{}',
		active_combat JSONB DEFAULT NULL,
		last_mana_update TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		aperture_stress DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		last_cultivate_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create players table: %w", err)
	}

	query = `
	CREATE TABLE IF NOT EXISTS auction (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		seller_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		item_type VARCHAR(20) NOT NULL,
		item_data JSONB NOT NULL,
		quantity INTEGER NOT NULL,
		currency VARCHAR(20) NOT NULL,
		start_price DOUBLE PRECISION NOT NULL,
		current_price DOUBLE PRECISION NOT NULL,
		highest_bidder_id UUID REFERENCES players(id) ON DELETE SET NULL,
		end_time TIMESTAMP WITH TIME ZONE NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create auction table: %w", err)
	}

	// Create parties table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS parties (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		leader_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		members JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create parties table: %w", err)
	}

	// Create messages table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sender_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		receiver_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		is_read BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create arena_rankings table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS arena_rankings (
		player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		rank INTEGER UNIQUE NOT NULL,
		points INTEGER DEFAULT 1000,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create arena_rankings table: %w", err)
	}

	// 4.5. Create regions
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS regions (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		width INT NOT NULL DEFAULT 100,
		height INT NOT NULL DEFAULT 100,
		min_rank INT NOT NULL DEFAULT 1,
		max_rank INT NOT NULL DEFAULT 5,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create regions table: %w", err)
	}

	// 4.6. Create loot_tables
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS loot_tables (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		entries JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create loot_tables table: %w", err)
	}

	// 4.7. Create monster_templates
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS monster_templates (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		type VARCHAR(50) NOT NULL,
		rank INT NOT NULL DEFAULT 1,
		sub_rank INT NOT NULL DEFAULT 1,
		path VARCHAR(50),
		max_hp DOUBLE PRECISION NOT NULL,
		base_attack DOUBLE PRECISION NOT NULL,
		base_defense DOUBLE PRECISION NOT NULL,
		max_mana DOUBLE PRECISION NOT NULL DEFAULT 100,
		current_mana DOUBLE PRECISION NOT NULL DEFAULT 100,
		gu_inventory JSONB NOT NULL DEFAULT '[]',
		item_inventory JSONB NOT NULL DEFAULT '{}',
		dao_marks JSONB NOT NULL DEFAULT '{}',
		path_masteries JSONB NOT NULL DEFAULT '{}',
		ai_profile VARCHAR(50) NOT NULL DEFAULT 'beast_basic',
		loot_table_id VARCHAR(50),
		description TEXT,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create monster_templates table: %w", err)
	}

	// 4.8. Create monster_spawns
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS monster_spawns (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		monster_id VARCHAR(50) NOT NULL REFERENCES monster_templates(id) ON DELETE CASCADE,
		x INT,
		y INT,
		spawn_mode VARCHAR(50) NOT NULL DEFAULT 'Random',
		chance DOUBLE PRECISION NOT NULL DEFAULT 1.0,
		respawn_seconds INT NOT NULL DEFAULT 300,
		min_rank INT NOT NULL DEFAULT 1,
		max_rank INT NOT NULL DEFAULT 9,
		is_boss_spawn BOOLEAN NOT NULL DEFAULT FALSE,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create monster_spawns table: %w", err)
	}

	// 4.9. Create map_cells
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS map_cells (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		x INT NOT NULL,
		y INT NOT NULL,
		cell_type VARCHAR(50) NOT NULL,
		content_id VARCHAR(50),
		metadata JSONB NOT NULL DEFAULT '{}',
		respawn_seconds INT NOT NULL DEFAULT 300,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		UNIQUE(region_id, x, y)
	);`)
	if err != nil {
		return fmt.Errorf("failed to create map_cells table: %w", err)
	}

	// Backwards-compatible column migrations
	migrations := []struct{ sql, errMsg string }{
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';`, "password_hash"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS gu_inventory JSONB NOT NULL DEFAULT '[]';`, "gu_inventory"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS materials_inventory JSONB NOT NULL DEFAULT '{}';`, "materials_inventory"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS hp DOUBLE PRECISION NOT NULL DEFAULT 100;`, "hp"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS base_hp DOUBLE PRECISION NOT NULL DEFAULT 100;`, "base_hp"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS body_mods JSONB NOT NULL DEFAULT '[]';`, "body_mods"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS x INTEGER NOT NULL DEFAULT 50;`, "x"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS y INTEGER NOT NULL DEFAULT 50;`, "y"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS explored_cells JSONB NOT NULL DEFAULT '["50,50"]';`, "explored_cells"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS current_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border';`, "current_region"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS dao_marks JSONB NOT NULL DEFAULT '{"strength": 0}';`, "dao_marks"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS active_combat JSONB DEFAULT NULL;`, "active_combat"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS path_masteries JSONB NOT NULL DEFAULT '{}';`, "path_masteries"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS equipped_gus JSONB NOT NULL DEFAULT '[]';`, "equipped_gus"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS cleared_cells JSONB NOT NULL DEFAULT '{}';`, "cleared_cells"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS immortal_stones DOUBLE PRECISION NOT NULL DEFAULT 0;`, "immortal_stones"},
		{`ALTER TABLE refining_recipes ADD COLUMN IF NOT EXISTS immortal_essence_stones_cost DOUBLE PRECISION NOT NULL DEFAULT 0;`, "immortal_essence_stones_cost"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS origin_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border';`, "origin_region"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS path VARCHAR(50) NOT NULL DEFAULT 'Moon Path';`, "path"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS base_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0;`, "base_mana_cost"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS effects JSONB NOT NULL DEFAULT '{}';`, "effects"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS food_types JSONB NOT NULL DEFAULT '[]';`, "food_types"}, // Cultivation system columns
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS aptitude_grade INTEGER DEFAULT 4;`, "aptitude_grade"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS wall_progress DOUBLE PRECISION NOT NULL DEFAULT 0.0;`, "wall_progress"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS human_qi DOUBLE PRECISION NOT NULL DEFAULT 0.0;`, "human_qi"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS next_tribulation_time TIMESTAMPTZ NULL;`, "next_tribulation_time"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS tribulation_count INTEGER NOT NULL DEFAULT 0;`, "tribulation_count"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS rarity INTEGER DEFAULT 1;`, "item_dictionary.rarity"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS rank INTEGER DEFAULT 1;`, "item_dictionary.rank"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS path VARCHAR(50) NOT NULL DEFAULT 'None';`, "item_dictionary.path"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS effects JSONB NOT NULL DEFAULT '{}';`, "item_dictionary.effects"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]';`, "item_dictionary.tags"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS lifespan DOUBLE PRECISION NOT NULL DEFAULT 100.0;`, "lifespan"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS max_lifespan DOUBLE PRECISION NOT NULL DEFAULT 100.0;`, "max_lifespan"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS time_speed_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0;`, "time_speed_multiplier"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS party_id UUID NULL;`, "party_id"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS arena_rank INTEGER NULL;`, "arena_rank"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50) NULL;`, "phone_number"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS email VARCHAR(100) NULL;`, "email"},
		{`CREATE UNIQUE INDEX IF NOT EXISTS players_email_idx ON players (email) WHERE email IS NOT NULL AND email != '';`, "players_email_idx"},
		{`CREATE UNIQUE INDEX IF NOT EXISTS players_phone_number_idx ON players (phone_number) WHERE phone_number IS NOT NULL AND phone_number != '';`, "players_phone_number_idx"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS completed_quests JSONB NOT NULL DEFAULT '[]';`, "completed_quests"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS temp_loot JSONB NOT NULL DEFAULT '{"gu_inventory": [], "materials_inventory": {}, "primeval_stones": 0}';`, "temp_loot"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS explore_count INTEGER NOT NULL DEFAULT 0;`, "explore_count"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS last_explore_reset TIMESTAMP NOT NULL DEFAULT NOW();`, "last_explore_reset"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS aperture_stress DOUBLE PRECISION NOT NULL DEFAULT 0.0;`, "aperture_stress"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS last_cultivate_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;`, "last_cultivate_time"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS target_type VARCHAR(50) NOT NULL DEFAULT 'SINGLE';`, "target_type"},
	}
	for _, m := range migrations {
		if _, err = DB.Exec(m.sql); err != nil {
			return fmt.Errorf("failed to add column %s: %w", m.errMsg, err)
		}
	}

	// Drop obsolete columns
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_power;`)
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_damage;`)
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_defense;`)

	// 5. Seed master data
	if err = SeedDictionaries(); err != nil {
		return fmt.Errorf("database dictionaries seeding failed: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

// SeedDictionaries populates item_dictionary, gu_dictionary, and refining_recipes from CSV files.
// Falls back to hardcoded seeds if CSV files are not found.
func SeedDictionaries() error {
	if err := seedItemsFromCSV("Item.csv"); err != nil {
		log.Printf("Warning: CSV item seed failed (%v), falling back to hardcoded seeds", err)
		if err2 := seedItemsHardcoded(); err2 != nil {
			return fmt.Errorf("hardcoded item seed failed: %w", err2)
		}
	}
	if err := seedGusFromCSV("Gu.csv"); err != nil {
		log.Printf("Warning: CSV Gu seed failed (%v), falling back to hardcoded seeds", err)
		if err2 := seedGusHardcoded(); err2 != nil {
			return fmt.Errorf("hardcoded Gu seed failed: %w", err2)
		}
	}
	if err := seedRecipesFromCSV("Recipe.csv"); err != nil {
		log.Printf("Warning: CSV recipe seed failed (%v), falling back to hardcoded seeds", err)
		if err2 := seedRecipesHardcoded(); err2 != nil {
			return fmt.Errorf("hardcoded recipe seed failed: %w", err2)
		}
	}
	if err := SeedMonstersAndSpawns(); err != nil {
		return fmt.Errorf("monster and spawn seeding failed: %w", err)
	}
	return nil
}

// csvPath resolves a CSV filename by searching the working directory and parent directories.
func csvPath(filename string) string {
	candidates := []string{
		filename,
		filepath.Join("..", filename),
		filepath.Join("..", "..", filename),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filename
}

// openCSV opens a CSV file and returns the file, reader, and header row.
func openCSV(filename string) (*os.File, *csv.Reader, []string, error) {
	path := csvPath(filename)
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	header, err := r.Read()
	if err != nil {
		f.Close()
		return nil, nil, nil, fmt.Errorf("read header %s: %w", path, err)
	}
	return f, r, header, nil
}

// colIndex returns the index of a named column in a header slice, or -1.
func colIndex(header []string, name string) int {
	for i, h := range header {
		if strings.TrimSpace(h) == name {
			return i
		}
	}
	return -1
}

// seedItemsFromCSV reads Item.csv and upserts into item_dictionary.
func seedItemsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	idIdx := colIndex(header, "id")
	nameIdx := colIndex(header, "name")
	typeIdx := colIndex(header, "item_type")
	rarityIdx := colIndex(header, "rarity")
	rankIdx := colIndex(header, "rank")
	pathIdx := colIndex(header, "path")
	effectsIdx := colIndex(header, "effects")
	tagsIdx := colIndex(header, "tags")
	descIdx := colIndex(header, "description")

	if idIdx < 0 || nameIdx < 0 || typeIdx < 0 {
		return fmt.Errorf("Item.csv missing required columns (id, name, item_type)")
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}
		id := strings.TrimSpace(row[idIdx])
		if id == "" {
			continue
		}
		name := strings.TrimSpace(row[nameIdx])
		itemType := strings.TrimSpace(row[typeIdx])

		rarity := 1
		if rarityIdx >= 0 && rarityIdx < len(row) {
			fmt.Sscanf(strings.TrimSpace(row[rarityIdx]), "%d", &rarity)
		}

		rank := 1
		if rankIdx >= 0 && rankIdx < len(row) {
			fmt.Sscanf(strings.TrimSpace(row[rankIdx]), "%d", &rank)
		}

		path := "None"
		if pathIdx >= 0 && pathIdx < len(row) {
			pathVal := strings.TrimSpace(row[pathIdx])
			if pathVal != "" {
				path = pathVal
			}
		}

		effects := "{}"
		if effectsIdx >= 0 && effectsIdx < len(row) {
			effVal := strings.TrimSpace(row[effectsIdx])
			if effVal != "" {
				effects = effVal
			}
		}

		tags := "[]"
		if tagsIdx >= 0 && tagsIdx < len(row) {
			tagsVal := strings.TrimSpace(row[tagsIdx])
			if tagsVal != "" {
				tags = tagsVal
			}
		}

		desc := ""
		if descIdx >= 0 && descIdx < len(row) {
			desc = strings.TrimSpace(row[descIdx])
		}

		_, err = DB.Exec(`
			INSERT INTO item_dictionary (id, name, item_type, rarity, rank, path, effects, tags, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)
			ON CONFLICT (id) DO UPDATE
			SET name=EXCLUDED.name, item_type=EXCLUDED.item_type, rarity=EXCLUDED.rarity, rank=EXCLUDED.rank,
			    path=EXCLUDED.path, effects=EXCLUDED.effects, tags=EXCLUDED.tags, description=EXCLUDED.description
		`, id, name, itemType, rarity, rank, path, effects, tags, desc)
		if err != nil {
			return fmt.Errorf("upsert item %s: %w", id, err)
		}
	}
	log.Printf("Item dictionary seeded from %s", filename)
	return nil
}

// seedGusFromCSV reads Gu.csv and upserts into gu_dictionary.
// Duplicate IDs are resolved by keeping the last occurrence.
func seedGusFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	col := func(name string) int { return colIndex(header, name) }
	idIdx := col("id")
	nameIdx := col("name")
	rankIdx := col("rank")
	typeIdx := col("gu_type")
	hungerIdx := col("hunger_rate")
	foodIdx := col("food_type")
	maxHungerIdx := col("max_hunger")
	descIdx := col("description")
	originIdx := col("origin_region")
	pathIdx := col("path")
	manaCostIdx := col("base_mana_cost")
	effectsIdx := col("effects")

	if idIdx < 0 || nameIdx < 0 || rankIdx < 0 {
		return fmt.Errorf("Gu.csv missing required columns (id, name, rank)")
	}

	type guRow struct{ fields []string }
	seen := make(map[string]guRow)
	order := []string{}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}
		id := strings.TrimSpace(row[idIdx])
		if id == "" {
			continue
		}
		if _, exists := seen[id]; !exists {
			order = append(order, id)
		}
		seen[id] = guRow{fields: row}
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	for _, id := range order {
		row := seen[id].fields
		name := get(row, nameIdx)
		rankStr := get(row, rankIdx)
		guType := get(row, typeIdx)
		hungerStr := get(row, hungerIdx)
		foodType := get(row, foodIdx)
		maxHungerStr := get(row, maxHungerIdx)
		desc := get(row, descIdx)
		origin := get(row, originIdx)
		path := get(row, pathIdx)
		manaCostStr := get(row, manaCostIdx)
		effectsRaw := get(row, effectsIdx)

		rank := 1
		fmt.Sscanf(rankStr, "%d", &rank)
		hungerRate := 0
		fmt.Sscanf(hungerStr, "%d", &hungerRate)
		maxHunger := 100
		fmt.Sscanf(maxHungerStr, "%d", &maxHunger)
		manaCost := 0.0
		fmt.Sscanf(manaCostStr, "%f", &manaCost)

		if origin == "" {
			origin = "Southern_Border"
		}
		if path == "" {
			path = "Moon Path"
		}
		if effectsRaw == "" {
			effectsRaw = "{}"
		}

		// Build food_types JSON array from single food_type string
		foodTypesJSON := "[]"
		if foodType != "" && foodType != "none" && foodType != "unknown" {
			foodTypesJSON = fmt.Sprintf(`["%s"]`, foodType)
		}

		var effMap map[string]interface{}
		_ = json.Unmarshal([]byte(effectsRaw), &effMap)

		targetType := "SINGLE"
		if effMap != nil {
			if _, ok := effMap["aoe_row"]; ok {
				targetType = "ROW"
			} else if _, ok := effMap["aoe_col"]; ok {
				targetType = "COL"
			} else if _, ok := effMap["aoe_cross"]; ok {
				targetType = "CROSS"
			} else if _, ok := effMap["aoe_all"]; ok {
				targetType = "ALL"
			} else if _, ok := effMap["aoe"]; ok {
				targetType = "ALL"
			}
		}

		_, err := DB.Exec(`
			INSERT INTO gu_dictionary (id, name, rank, path, gu_type, origin_region, base_mana_cost, effects, hunger_rate, food_types, max_hunger, description, target_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10::jsonb, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE
			SET name=EXCLUDED.name, rank=EXCLUDED.rank, path=EXCLUDED.path, gu_type=EXCLUDED.gu_type,
			    origin_region=EXCLUDED.origin_region, base_mana_cost=EXCLUDED.base_mana_cost, effects=EXCLUDED.effects,
			    hunger_rate=EXCLUDED.hunger_rate, food_types=EXCLUDED.food_types, max_hunger=EXCLUDED.max_hunger, description=EXCLUDED.description,
			    target_type=EXCLUDED.target_type
		`, id, name, rank, path, guType, origin, manaCost, effectsRaw, hungerRate, foodTypesJSON, maxHunger, desc, targetType)
		if err != nil {
			return fmt.Errorf("upsert gu %s: %w", id, err)
		}
	}
	log.Printf("Gu dictionary seeded from %s (%d unique entries)", filename, len(order))
	return nil
}

// seedRecipesFromCSV reads Recipe.csv and upserts into refining_recipes.
// Rows whose target_gu_id is not in gu_dictionary are skipped (FK constraint).
func seedRecipesFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	col := func(name string) int { return colIndex(header, name) }
	idIdx := col("id")
	targetIdx := col("target_gu_id")
	reqGusIdx := col("required_gus")
	reqMatsIdx := col("required_materials")
	rateIdx := col("success_rate")
	stoneCostIdx := col("primeval_stones_cost")
	immortalCostIdx := col("immortal_essence_stones_cost")

	if idIdx < 0 || targetIdx < 0 || reqGusIdx < 0 {
		return fmt.Errorf("Recipe.csv missing required columns (id, target_gu_id, required_gus)")
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	seeded, skipped := 0, 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}
		id := get(row, idIdx)
		if id == "" {
			continue
		}
		targetGuID := get(row, targetIdx)
		reqGus := get(row, reqGusIdx)
		reqMats := get(row, reqMatsIdx)
		if reqMats == "" {
			reqMats = "{}"
		}
		rateStr := get(row, rateIdx)
		stoneCostStr := get(row, stoneCostIdx)
		immortalCostStr := get(row, immortalCostIdx)

		rate := 0.5
		fmt.Sscanf(rateStr, "%f", &rate)
		stoneCost := 0
		fmt.Sscanf(stoneCostStr, "%d", &stoneCost)
		immortalCost := 0.0
		fmt.Sscanf(immortalCostStr, "%f", &immortalCost)

		// Skip if target Gu doesn't exist (FK constraint)
		var exists bool
		if err = DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1)`, targetGuID).Scan(&exists); err != nil || !exists {
			log.Printf("Recipe %s skipped: target_gu_id %s not in gu_dictionary", id, targetGuID)
			skipped++
			continue
		}

		_, err = DB.Exec(`
			INSERT INTO refining_recipes (id, target_gu_id, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost)
			VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE
			SET target_gu_id=EXCLUDED.target_gu_id, required_gus=EXCLUDED.required_gus,
			    required_materials=EXCLUDED.required_materials, success_rate=EXCLUDED.success_rate,
			    primeval_stones_cost=EXCLUDED.primeval_stones_cost, immortal_essence_stones_cost=EXCLUDED.immortal_essence_stones_cost
		`, id, targetGuID, reqGus, reqMats, rate, stoneCost, immortalCost)
		if err != nil {
			return fmt.Errorf("upsert recipe %s: %w", id, err)
		}
		seeded++
	}
	log.Printf("Recipes seeded from %s (%d seeded, %d skipped)", filename, seeded, skipped)
	return nil
}

// ── Hardcoded fallback seeds ──────────────────────────────────────────────────

func seedItemsHardcoded() error {
	type itemSeed struct {
		ID, Name, Type, Path, Effects, Tags, Desc string
		Rarity, Rank                              int
	}
	items := []itemSeed{
		{"moon_orchid_petal", "Cánh Hoa Nguyệt Lan", "Consumable", "Moon Path", `{"restore_mana": 10, "mortal_only": 1}`, `["food", "moon_path_food"]`, "Cánh hoa chứa nguyệt hoa, dùng nuôi Nguyệt đạo Cổ hoặc luyện hóa Chân Nguyên.", 1, 1},
		{"green_bamboo_wine", "Rượu Thanh Trúc", "Consumable", "Wood Path", `{"restore_mana": 20, "mortal_only": 1}`, `["food", "wine", "liquor_worm_food"]`, "Rượu trúc xanh, có thể khôi phục Chân Nguyên hoặc nuôi Tửu Trùng.", 1, 1},
		{"jade_stone", "Ngọc Thạch", "Consumable", "Earth Path", `{"restore_hp": 20}`, `["material", "earth_path"]`, "Đá ngọc có linh khí, dùng dưỡng thương hoặc luyện Cổ phòng ngự.", 1, 1},
		{"wild_boar_tusk", "Nanh Lợn Rừng", "Material", "None", `{}`, `[]`, "Vật phẩm phụ dùng trong hợp luyện Bạch Ngọc Cổ.", 1, 1},
		{"boar_meat", "Thịt Lợn Rừng", "Food", "None", `{}`, `["food"]`, "Thức ăn của Bạch Thỉ cổ.", 1, 1},
		{"sour_wine", "Rượu Chua", "Food", "None", `{}`, `["food"]`, "Loại rượu chua cần cho Tứ Vị Tửu Trùng.", 1, 1},
		{"sweet_wine", "Rượu Ngọt", "Food", "None", `{}`, `["food"]`, "Loại rượu ngọt cần cho Tứ Vị Tửu Trùng.", 1, 1},
		{"bitter_wine", "Rượu Đắng", "Food", "None", `{}`, `["food"]`, "Loại rượu đắng cần cho Tứ Vị Tửu Trùng.", 1, 1},
		{"spicy_wine", "Rượu Cay", "Food", "None", `{}`, `["food"]`, "Loại rượu cay cần cho Tứ Vị Tửu Trùng.", 1, 1},
		{"grass_puppet", "Hình nộm người cỏ", "Material", "None", `{}`, `[]`, "Vật phẩm luyện tập cơ bản.", 1, 1},
	}
	for _, item := range items {
		_, err := DB.Exec(`
			INSERT INTO item_dictionary (id, name, item_type, rarity, rank, path, effects, tags, description)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9)
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, item_type=EXCLUDED.item_type, rarity=EXCLUDED.rarity,
			rank=EXCLUDED.rank, path=EXCLUDED.path, effects=EXCLUDED.effects, tags=EXCLUDED.tags, description=EXCLUDED.description
		`, item.ID, item.Name, item.Type, item.Rarity, item.Rank, item.Path, item.Effects, item.Tags, item.Desc)
		if err != nil {
			return fmt.Errorf("hardcoded item %s: %w", item.ID, err)
		}
	}
	return nil
}

func seedGusHardcoded() error {
	type guSeed struct {
		ID, Name, Path, Type, Origin, Effects string
		Rank, HungerRate, MaxHunger           int
		BaseManaCost                          float64
		FoodTypes                             []string
		Desc                                  string
	}
	gus := []guSeed{
		{"GU_MOONLIGHT", "Nguyệt Quang Cổ", "Moon Path", "Active", "Southern_Border", `{"damage": 25.0}`, 1, 5, 100, 10.0, []string{"moon_orchid_petal"}, "Bắn ra nguyệt nhận sắc lẹm màu xanh lam."},
		{"GU_JADE_SKIN", "Ngọc Bì Cổ", "Earth Path", "Passive", "Southern_Border", `{"add_base_defense": 40.0}`, 1, 2, 100, 0.0, []string{"jade_stone"}, "Cải tạo bề mặt da trở nên cứng cáp như ngọc bích."},
		{"GU_LIQUOR_WORM", "Tửu Trùng", "Refinement Path", "Support", "Southern_Border", `{"refine_mana_tier": 1.0}`, 1, 3, 100, 0.0, []string{"green_bamboo_wine"}, "Tinh luyện và nâng cao chất lượng Chân Nguyên."},
		{"GU_WHITE_BOAR", "Bạch Thỉ Cổ", "Strength Path", "Passive", "Northern_Plains", `{"add_base_attack": 15.0, "add_base_defense": 5.0}`, 1, 6, 100, 0.0, []string{"boar_meat"}, "Cải tạo thể chất vĩnh viễn, tăng sức mạnh một con lợn rừng."},
		{"GU_MOON_RAIMENT", "Nguyệt Nghê Thường", "Moon Path", "Defense", "Southern_Border", `{"add_base_defense": 75.0}`, 2, 4, 100, 15.0, []string{"moon_orchid_petal"}, "Tạo ra một bộ xiêm y bằng ánh trăng che chở cơ thể."},
		{"GU_WHITE_JADE", "Bạch Ngọc Cổ", "Earth Path", "Defense", "Southern_Border", `{"add_base_attack": 10.0, "add_base_defense": 110.0}`, 2, 3, 100, 20.0, []string{"jade_stone"}, "Biến làn da toàn thân thành ngọc trắng cứng cáp che chắn."},
		{"GU_FOUR_FLAVORS_LIQUOR", "Tứ Vị Tửu Trùng", "Refinement Path", "Support", "Southern_Border", `{"refine_mana_tier": 2.0}`, 2, 2, 100, 0.0, []string{"green_bamboo_wine"}, "Cổ trùng tinh luyện chân nguyên Nhị Chuyển tối thượng."},
		{"GU_BLACK_BOAR", "Hắc Thỉ Cổ", "Strength Path", "Passive", "Southern_Border", `{"add_base_attack": 15.0, "add_base_defense": 5.0}`, 1, 6, 100, 0.0, []string{"boar_meat"}, "Cải tạo thể chất vĩnh viễn, tăng sức mạnh một con hắc thỉ."},
		{"GU_CLEANSING_WATER", "Tịnh Thủy Cổ", "Water Path", "Support", "Southern_Border", `{"cleansing_power": 1.0}`, 1, 1, 100, 0.0, []string{"grass_puppet"}, "Sử dụng nước tinh khiết để rửa trôi các trạng thái tiêu cực."},
		{"GU_TUSITA_FLOWER", "Thố Ti Hoa Cổ", "Wood Path", "Storage", "Southern_Border", `{"storage": 20.0}`, 1, 2, 100, 0.0, []string{"moon_orchid_petal"}, "Cổ trùng lưu trữ không gian phàm cấp, tăng thêm 20 ô hành lý."},
		{"GU_RELIC", "Xá Lợi Cổ (Relic Gu)", "Refinement Path", "Consumable", "Southern_Border", `{"raise_cultivation": 1.0}`, 1, 0, 100, 0.0, []string{}, "Cổ trùng tiêu hao nâng cao tu vi ngay lập tức thêm 1 Tiểu Chuyển."},
		{"GU_STONE_APERTURE", "Thạch Khiếu Cổ (Stone Aperture Gu)", "Earth Path", "Consumable", "Southern_Border", `{"give_stones": 500.0}`, 1, 0, 100, 0.0, []string{}, "Cổ trùng tiêu hao hóa thạch Không khiếu nhận 500 Nguyên Thạch nhưng giảm 15 Chân Nguyên tối đa."},
	}
	for _, gu := range gus {
		foodTypesJSON := "[]"
		if len(gu.FoodTypes) > 0 {
			escaped := make([]string, len(gu.FoodTypes))
			for i, t := range gu.FoodTypes {
				escaped[i] = fmt.Sprintf(`"%s"`, t)
			}
			foodTypesJSON = fmt.Sprintf("[%s]", strings.Join(escaped, ","))
		}
		_, err := DB.Exec(`
			INSERT INTO gu_dictionary (id, name, rank, path, gu_type, origin_region, base_mana_cost, effects, hunger_rate, food_types, max_hunger, description)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10::jsonb,$11,$12)
			ON CONFLICT (id) DO UPDATE
			SET name=EXCLUDED.name, rank=EXCLUDED.rank, path=EXCLUDED.path, gu_type=EXCLUDED.gu_type,
			    origin_region=EXCLUDED.origin_region, base_mana_cost=EXCLUDED.base_mana_cost, effects=EXCLUDED.effects,
			    hunger_rate=EXCLUDED.hunger_rate, food_types=EXCLUDED.food_types, max_hunger=EXCLUDED.max_hunger, description=EXCLUDED.description
		`, gu.ID, gu.Name, gu.Rank, gu.Path, gu.Type, gu.Origin, gu.BaseManaCost, gu.Effects, gu.HungerRate, foodTypesJSON, gu.MaxHunger, gu.Desc)
		if err != nil {
			return fmt.Errorf("hardcoded gu %s: %w", gu.ID, err)
		}
	}
	return nil
}

func seedRecipesHardcoded() error {
	type recipeSeed struct {
		ID, Target, ReqGus, ReqMats string
		Rate                        float64
		StoneCost                   int
		ImmortalCost                float64
	}
	recipes := []recipeSeed{
		{"recipe_moon_raiment", "GU_MOON_RAIMENT", `{"GU_MOONLIGHT":1,"GU_JADE_SKIN":1}`, `{}`, 0.85, 10, 0},
		{"recipe_white_jade", "GU_WHITE_JADE", `{"GU_WHITE_BOAR":1,"GU_JADE_SKIN":1}`, `{"wild_boar_tusk":1}`, 0.80, 15, 0},
		{"recipe_four_flavors_liquor", "GU_FOUR_FLAVORS_LIQUOR", `{"GU_LIQUOR_WORM":2}`, `{"sour_wine":1,"sweet_wine":1,"bitter_wine":1,"spicy_wine":1}`, 0.60, 30, 0},
	}
	for _, recipe := range recipes {
		_, err := DB.Exec(`
			INSERT INTO refining_recipes (id, target_gu_id, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (id) DO UPDATE
			SET target_gu_id=EXCLUDED.target_gu_id, required_gus=EXCLUDED.required_gus,
			    required_materials=EXCLUDED.required_materials, success_rate=EXCLUDED.success_rate,
			    primeval_stones_cost=EXCLUDED.primeval_stones_cost, immortal_essence_stones_cost=EXCLUDED.immortal_essence_stones_cost
		`, recipe.ID, recipe.Target, recipe.ReqGus, recipe.ReqMats, recipe.Rate, recipe.StoneCost, recipe.ImmortalCost)
		if err != nil {
			return fmt.Errorf("hardcoded recipe %s: %w", recipe.ID, err)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func SeedMonstersAndSpawns() error {
	// 1. Seed regions
	regions := []struct{ ID, Name, Desc string }{
		{"qing_mao_mountain", "Thanh Mao Sơn (Qing Mao Mountain)", "Southern Border mountain containing Gu Yue, Bai, and Xiong clans."},
		{"gu_luan_slope", "Cổ Loạn Sườn (Gu Luan Slope)", "Southern Border region full of dangerous wild beast packs."},
		{"three_kings_inheritance", "Tam Vương Truyền Thừa (Three Kings Inheritance)", "An ancient inheritance ground with rich resources and relics."},
	}
	for _, r := range regions {
		_, err := DB.Exec(`
			INSERT INTO regions (id, name, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO NOTHING
		`, r.ID, r.Name, r.Desc)
		if err != nil {
			return fmt.Errorf("seed region %s: %w", r.ID, err)
		}
	}

	// 2. Seed loot tables
	lootTables := []struct {
		ID, Name, Entries string
	}{
		{"loot_boar", "Loot Dã Trĩ", `[{"item_id": "boar_meat", "min_qty": 1, "max_qty": 1, "chance": 0.5}, {"item_id": "wild_boar_tusk", "min_qty": 1, "max_qty": 1, "chance": 0.1}]`},
		{"loot_ape", "Loot Thạch Hầu", `[{"item_id": "jade_stone", "min_qty": 1, "max_qty": 1, "chance": 0.3}]`},
		{"loot_boar_king", "Loot Dã Trĩ Vương", `[{"item_id": "boar_meat", "min_qty": 2, "max_qty": 4, "chance": 0.8}, {"item_id": "wild_boar_tusk", "min_qty": 1, "max_qty": 2, "chance": 0.5}, {"gu_id": "GU_WHITE_BOAR", "chance": 0.05}]`},
		{"loot_wolf_leader", "Loot Độc Lang Vương", `[{"item_id": "jade_stone", "min_qty": 2, "max_qty": 5, "chance": 0.8}, {"gu_id": "GU_BLACK_BOAR", "chance": 0.05}]`},
		{"loot_python", "Loot Mãng Xà Dung Nham", `[{"item_id": "jade_stone", "min_qty": 3, "max_qty": 6, "chance": 0.8}]`},
		{"loot_bear", "Loot Thiết Giáp Hùng", `[{"item_id": "boar_meat", "min_qty": 3, "max_qty": 6, "chance": 0.8}]`},
		{"loot_wandering", "Loot Cổ Sư Lang Thang", `[{"item_id": "primeval_stone", "min_qty": 50, "max_qty": 100, "chance": 1.0}]`},
		{"loot_boss_wolf_king", "Loot Boss Độc Lang Vương", `[{"item_id": "jade_stone", "min_qty": 5, "max_qty": 10, "chance": 1.0}, {"gu_id": "GU_WHITE_BOAR", "chance": 0.15}]`},
	}
	for _, lt := range lootTables {
		_, err := DB.Exec(`
			INSERT INTO loot_tables (id, name, entries)
			VALUES ($1, $2, $3::jsonb)
			ON CONFLICT (id) DO UPDATE SET entries=EXCLUDED.entries
		`, lt.ID, lt.Name, lt.Entries)
		if err != nil {
			return fmt.Errorf("seed loot table %s: %w", lt.ID, err)
		}
	}

	// 3. Seed monster templates
	monsters := []struct {
		ID, Name, Type string
		Rank, SubRank  int
		Path           string
		MaxHP, Attack  float64
		Defense        float64
		MaxMana        float64
		GuInventory    string
		AIProfile      string
		LootTableID    string
		Desc           string
	}{
		{"monster_wild_boar", "Dã Trĩ (Wild Boar)", "Normal", 1, 1, "None", 50, 8, 2, 100, "[]", "beast_basic", "loot_boar", "Thịt lợn hoang dã rất bổ dưỡng."},
		{"monster_stone_ape", "Thạch Hầu (Stone Ape)", "Normal", 1, 1, "Earth Path", 60, 10, 3, 100, "[]", "beast_basic", "loot_ape", "Vượn đá sống ở vách núi."},
		{"monster_boar_king", "Dã Trĩ Vương (Boar King)", "Elite", 1, 3, "Strength Path", 120, 12, 8, 100, "[]", "beast_basic", "loot_boar_king", "Lợn lòi đầu đàn hung tợn."},
		{"monster_wolf_leader", "Độc Lang Vương (Wolf Leader)", "Elite", 1, 4, "Moon Path", 100, 15, 6, 100, "[]", "beast_basic", "loot_wolf_leader", "Sói hoang dã hung dữ."},
		{"monster_lava_python", "Mãng Xà Dung Nham (Lava Python)", "Elite", 3, 1, "Fire Path", 150, 24, 10, 100, "[]", "beast_basic", "loot_python", "Mãng xà khổng lồ sống trong khe dung nham."},
		{"monster_iron_bear", "Thiết Giáp Hùng (Iron Armored Bear)", "Elite", 3, 2, "Strength Path", 200, 18, 18, 100, "[]", "beast_basic", "loot_bear", "Gấu lớn da dày như thép."},
		{"monster_wandering", "Cổ Sư Lang Thang (Wandering Cultivator)", "HumanCultivator", 1, 1, "Moon Path", 100, 15, 5, 100, "[]", "beast_basic", "loot_wandering", "Cổ sư lang thang nguy hiểm."},
		{"boss_wolf_king", "Độc Lang Vương Boss (Boss Wolf Leader)", "Boss", 3, 4, "Moon Path", 500, 35, 18, 120, `[{"instance_id": "boss-moonlight", "gu_id": "GU_MOONLIGHT", "hunger": 100, "status": "ACTIVE", "equipped": true}]`, "boss_balanced", "loot_boss_wolf_king", "Lang vương thống lĩnh đàn sói đặc biệt nguy hiểm."},
	}
	for _, m := range monsters {
		_, err := DB.Exec(`
			INSERT INTO monster_templates (
				id, name, type, rank, sub_rank, path, max_hp, base_attack, base_defense,
				max_mana, current_mana, gu_inventory, ai_profile, loot_table_id, description
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10, $11::jsonb, $12, $13, $14)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, type=EXCLUDED.type, rank=EXCLUDED.rank, sub_rank=EXCLUDED.sub_rank,
				max_hp=EXCLUDED.max_hp, base_attack=EXCLUDED.base_attack, base_defense=EXCLUDED.base_defense,
				gu_inventory=EXCLUDED.gu_inventory, ai_profile=EXCLUDED.ai_profile, loot_table_id=EXCLUDED.loot_table_id
		`, m.ID, m.Name, m.Type, m.Rank, m.SubRank, m.Path, m.MaxHP, m.Attack, m.Defense, m.MaxMana, m.GuInventory, m.AIProfile, m.LootTableID, m.Desc)
		if err != nil {
			return fmt.Errorf("seed monster %s: %w", m.ID, err)
		}
	}

	// 4. Seed spawns and special cells (Boss spawns)
	// Seed fixed Boss spawn at Gu Luan Slope (80, 80)
	_, err := DB.Exec(`
		INSERT INTO monster_spawns (region_id, monster_id, x, y, spawn_mode, chance, respawn_seconds, is_boss_spawn)
		VALUES ('gu_luan_slope', 'boss_wolf_king', 80, 80, 'BossRoom', 1.0, 3600, TRUE)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("seed boss spawn: %w", err)
	}

	// Seed map cells for Boss Room
	_, err = DB.Exec(`
		INSERT INTO map_cells (region_id, x, y, cell_type, content_id)
		VALUES ('gu_luan_slope', 80, 80, 'Boss', 'boss_wolf_king')
		ON CONFLICT (region_id, x, y) DO UPDATE SET cell_type=EXCLUDED.cell_type, content_id=EXCLUDED.content_id
	`)
	if err != nil {
		return fmt.Errorf("seed boss room map cell: %w", err)
	}

	return nil
}
