package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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
		origin_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border',
		base_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
		effects JSONB NOT NULL DEFAULT '{}',
		hunger_rate INTEGER NOT NULL,
		food_type VARCHAR(50) REFERENCES item_dictionary(id) ON DELETE SET NULL,
		food_types JSONB NOT NULL DEFAULT '[]',
		max_hunger INTEGER DEFAULT 100,
		description TEXT
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
		primeval_stones_cost INTEGER DEFAULT 0
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
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create players table: %w", err)
	}

	// Ensure columns exist (for backwards compatibility/migrations)
	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';`)
	if err != nil {
		return fmt.Errorf("failed to add password_hash: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS gu_inventory JSONB NOT NULL DEFAULT '[]';`)
	if err != nil {
		return fmt.Errorf("failed to add gu_inventory: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS materials_inventory JSONB NOT NULL DEFAULT '{}';`)
	if err != nil {
		return fmt.Errorf("failed to add materials_inventory: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS hp DOUBLE PRECISION NOT NULL DEFAULT 100;`)
	if err != nil {
		return fmt.Errorf("failed to add hp column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS base_hp DOUBLE PRECISION NOT NULL DEFAULT 100;`)
	if err != nil {
		return fmt.Errorf("failed to add base_hp column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS body_mods JSONB NOT NULL DEFAULT '[]';`)
	if err != nil {
		return fmt.Errorf("failed to add body_mods column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS x INTEGER NOT NULL DEFAULT 50;`)
	if err != nil {
		return fmt.Errorf("failed to add x column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS y INTEGER NOT NULL DEFAULT 50;`)
	if err != nil {
		return fmt.Errorf("failed to add y column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS explored_cells JSONB NOT NULL DEFAULT '["50,50"]';`)
	if err != nil {
		return fmt.Errorf("failed to add explored_cells column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS current_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border';`)
	if err != nil {
		return fmt.Errorf("failed to add current_region column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS dao_marks JSONB NOT NULL DEFAULT '{"strength": 0}';`)
	if err != nil {
		return fmt.Errorf("failed to add dao_marks column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS active_combat JSONB DEFAULT NULL;`)
	if err != nil {
		return fmt.Errorf("failed to add active_combat column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS path_masteries JSONB NOT NULL DEFAULT '{}';`)
	if err != nil {
		return fmt.Errorf("failed to add path_masteries: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS equipped_gus JSONB NOT NULL DEFAULT '[]';`)
	if err != nil {
		return fmt.Errorf("failed to add equipped_gus: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS cleared_cells JSONB NOT NULL DEFAULT '{}';`)
	if err != nil {
		return fmt.Errorf("failed to add cleared_cells: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE players ADD COLUMN IF NOT EXISTS immortal_stones DOUBLE PRECISION NOT NULL DEFAULT 0;`)
	if err != nil {
		return fmt.Errorf("failed to add cleared_cells: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS origin_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border';`)
	if err != nil {
		return fmt.Errorf("failed to add origin_region column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS path VARCHAR(50) NOT NULL DEFAULT 'Moon Path';`)
	if err != nil {
		return fmt.Errorf("failed to add path column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS base_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0;`)
	if err != nil {
		return fmt.Errorf("failed to add base_mana_cost column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS effects JSONB NOT NULL DEFAULT '{}';`)
	if err != nil {
		return fmt.Errorf("failed to add effects column: %w", err)
	}

	_, err = DB.Exec(`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS food_types JSONB NOT NULL DEFAULT '[]';`)
	if err != nil {
		return fmt.Errorf("failed to add effects column: %w", err)
	}

	// Drop old columns if they exist
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_power;`)
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_damage;`)
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_defense;`)

	// 5. Seed master data
	err = SeedDictionaries()
	if err != nil {
		return fmt.Errorf("database dictionaries seeding failed: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

// SeedDictionaries populates item_dictionary, gu_dictionary, and refining_recipes
func SeedDictionaries() error {
	// Seed items
	itemSeeds := []struct {
		ID   string
		Name string
		Type string
		Desc string
	}{
		{"moon_orchid_petal", "Cánh hoa Nguyệt Lan", "Food", "Thức ăn chuyên dụng của Nguyệt Quang cổ."},
		{"green_bamboo_wine", "Rượu Thanh Trúc", "Food", "Thức ăn của Tửu Trùng."},
		{"jade_stone", "Ngọc Thạch", "Food", "Thức ăn của Ngọc Bì cổ."},
		{"wild_boar_tusk", "Nanh Lợn Rừng", "Material", "Vật phẩm phụ dùng trong hợp luyện Bạch Ngọc Cổ."},
		{"boar_meat", "Thịt Lợn Rừng", "Food", "Thức ăn của Bạch Thỉ cổ."},
		{"sour_wine", "Rượu Chua", "Food", "Loại rượu chua cần cho Tứ Vị Tửu Trùng."},
		{"sweet_wine", "Rượu Ngọt", "Food", "Loại rượu ngọt cần cho Tứ Vị Tửu Trùng."},
		{"bitter_wine", "Rượu Đắng", "Food", "Loại rượu đắng cần cho Tứ Vị Tửu Trùng."},
		{"spicy_wine", "Rượu Cay", "Food", "Loại rượu cay cần cho Tứ Vị Tửu Trùng."},
		{"grass_puppet", "Hình nộm người cỏ", "Material", "Vật phẩm luyện tập cơ bản."},
	}

	for _, item := range itemSeeds {
		_, err := DB.Exec(`
			INSERT INTO item_dictionary (id, name, item_type, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE 
			SET name = EXCLUDED.name, item_type = EXCLUDED.item_type, description = EXCLUDED.description
		`, item.ID, item.Name, item.Type, item.Desc)
		if err != nil {
			return fmt.Errorf("failed to seed item %s: %w", item.ID, err)
		}
	}

	// Seed Gu
	guSeeds := []struct {
		ID           string
		Name         string
		Rank         int
		Path         string
		Type         string
		OriginRegion string
		BaseManaCost float64
		Effects      string
		FoodType     string
		HungerRate   int
		MaxHunger    int
		Desc         string
	}{
		{"GU_MOONLIGHT", "Nguyệt Quang Cổ", 1, "Moon Path", "Active", "Southern_Border", 10.0, `{"damage": 25.0}`, "moon_orchid_petal", 5, 100, "Bắn ra nguyệt nhận sắc lẹm màu xanh lam."},
		{"GU_JADE_SKIN", "Ngọc Bì Cổ", 1, "Earth Path", "Passive", "Southern_Border", 0.0, `{"add_base_defense": 40.0}`, "jade_stone", 2, 100, "Cải tạo bề mặt da trở nên cứng cáp như ngọc bích."},
		{"GU_LIQUOR_WORM", "Tửu Trùng", 1, "Refinement Path", "Support", "Southern_Border", 0.0, `{"refine_mana_tier": 1.0}`, "green_bamboo_wine", 3, 100, "Tinh luyện và nâng cao chất lượng Chân Nguyên."},
		{"GU_WHITE_BOAR", "Bạch Thỉ Cổ", 1, "Strength Path", "Passive", "Northern_Plains", 0.0, `{"add_base_attack": 15.0, "add_base_defense": 5.0}`, "boar_meat", 6, 100, "Cải tạo thể chất vĩnh viễn, tăng sức mạnh một con lợn rừng."},
		{"GU_MOON_RAIMENT", "Nguyệt Nghê Thường", 2, "Moon Path", "Defense", "Southern_Border", 15.0, `{"add_base_defense": 75.0}`, "moon_orchid_petal", 4, 100, "Tạo ra một bộ xiêm y bằng ánh trăng che chở cơ thể."},
		{"GU_WHITE_JADE", "Bạch Ngọc Cổ", 2, "Earth Path", "Defense", "Southern_Border", 20.0, `{"add_base_attack": 10.0, "add_base_defense": 110.0}`, "jade_stone", 3, 100, "Biến làn da toàn thân thành ngọc trắng cứng cáp che chắn."},
		{"GU_FOUR_FLAVORS_LIQUOR", "Tứ Vị Tửu Trùng", 2, "Refinement Path", "Support", "Southern_Border", 0.0, `{"refine_mana_tier": 2.0}`, "green_bamboo_wine", 2, 100, "Cổ trùng tinh luyện chân nguyên Nhị Chuyển tối thượng."},
		{"GU_BLACK_BOAR", "Hắc Thỉ Cổ", 1, "Strength Path", "Passive", "Southern_Border", 0.0, `{"add_base_attack": 15.0, "add_base_defense": 5.0}`, "boar_meat", 6, 100, "Cải tạo thể chất vĩnh viễn, tăng sức mạnh một con hắc thỉ (heo rừng đen)."},
		{"GU_CLEANSING_WATER", "Tịnh Thủy Cổ", 1, "Rule Path", "Support", "Southern_Border", 15.0, `{"cleansing_power": 1.0}`, "green_bamboo_wine", 4, 100, "Tẩy sạch cặn bã trong Không khiếu và tẩy trừ ý chí của cổ sư khác."},
		{"GU_SHADOW_STEPS", "Ảnh Bộ Cổ", 1, "Space Path", "Movement", "Southern_Border", 5.0, `{"movement": 2.0}`, "green_bamboo_wine", 3, 100, "Cổ trùng di chuyển hỗ trợ thân thủ nhanh thoăn thoắt, nhảy vọt 2 bước."},
		{"GU_EARTH_COMMUNICATION", "Địa Thính Nhục Chi Nha", 1, "Earth Path", "Investigation", "Southern_Border", 5.0, `{"investigation": 3.0}`, "jade_stone", 3, 100, "Cổ trùng trinh thám thông qua chấn động lòng đất, khuếch đại tầm nhìn lên 3 ô."},
		{"GU_TUSITA_FLOWER", "Thố Ti Hoa Cổ", 1, "Wood Path", "Storage", "Southern_Border", 0.0, `{"storage": 20.0}`, "moon_orchid_petal", 2, 100, "Cổ trùng lưu trữ không gian phàm cấp, tăng thêm 20 ô hành lý."},
		{"GU_RELIC", "Xá Lợi Cổ (Relic Gu)", 1, "Refinement Path", "Consumable", "Southern_Border", 0.0, `{"raise_cultivation": 1.0}`, "", 0, 100, "Cổ trùng tiêu hao nâng cao tu vi ngay lập tức thêm 1 Tiểu Chuyển."},
		{"GU_STONE_APERTURE", "Thạch Khiếu Cổ (Stone Aperture Gu)", 1, "Earth Path", "Consumable", "Southern_Border", 0.0, `{"give_stones": 500.0}`, "", 0, 100, "Cổ trùng tiêu hao hóa thạch Không khiếu nhận 500 Nguyên Thạch nhưng giảm 15 Chân Nguyên tối đa."},
	}

	for _, gu := range guSeeds {
		var foodType interface{}
		if gu.FoodType != "" {
			foodType = gu.FoodType
		}
		_, err := DB.Exec(`
			INSERT INTO gu_dictionary (id, name, rank, path, gu_type, origin_region, base_mana_cost, effects, hunger_rate, food_type, food_types, max_hunger, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11::jsonb, $12, $13)
			ON CONFLICT (id) DO UPDATE 
			SET name = EXCLUDED.name, rank = EXCLUDED.rank, path = EXCLUDED.path, gu_type = EXCLUDED.gu_type, 
			    origin_region = EXCLUDED.origin_region, base_mana_cost = EXCLUDED.base_mana_cost, effects = EXCLUDED.effects,
			    hunger_rate = EXCLUDED.hunger_rate, food_type = EXCLUDED.food_type, food_types = EXCLUDED.food_types, max_hunger = EXCLUDED.max_hunger, description = EXCLUDED.description
		`, gu.ID, gu.Name, gu.Rank, gu.Path, gu.Type, gu.OriginRegion, gu.BaseManaCost, gu.Effects, gu.HungerRate, foodType, fmt.Sprintf("[\"%s\"]", gu.FoodType), gu.MaxHunger, gu.Desc)
		if err != nil {
			return fmt.Errorf("failed to seed gu %s: %w", gu.ID, err)
		}
	}

	// Seed Refining Recipes
	recipeSeeds := []struct {
		ID        string
		Target    string
		ReqGus    string
		ReqMats   string
		Rate      float64
		StoneCost int
	}{
		{
			"recipe_moon_raiment",
			"GU_MOON_RAIMENT",
			`{"GU_MOONLIGHT":1,"GU_JADE_SKIN":1}`,
			`{}`,
			0.85,
			10,
		},
		{
			"recipe_white_jade",
			"GU_WHITE_JADE",
			`{"GU_WHITE_BOAR":1,"GU_JADE_SKIN":1}`,
			`{"wild_boar_tusk":1}`,
			0.80,
			15,
		},
		{
			"recipe_four_flavors_liquor",
			"GU_FOUR_FLAVORS_LIQUOR",
			`{"GU_LIQUOR_WORM":2}`,
			`{"sour_wine":1,"sweet_wine":1,"bitter_wine":1,"spicy_wine":1}`,
			0.60,
			30,
		},
	}

	for _, recipe := range recipeSeeds {
		_, err := DB.Exec(`
			INSERT INTO refining_recipes (id, target_gu_id, required_gus, required_materials, success_rate, primeval_stones_cost)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE 
			SET target_gu_id = EXCLUDED.target_gu_id, 
			    required_gus = EXCLUDED.required_gus, 
			    required_materials = EXCLUDED.required_materials, 
			    success_rate = EXCLUDED.success_rate, 
			    primeval_stones_cost = EXCLUDED.primeval_stones_cost
		`, recipe.ID, recipe.Target, recipe.ReqGus, recipe.ReqMats, recipe.Rate, recipe.StoneCost)
		if err != nil {
			return fmt.Errorf("failed to seed recipe %s: %w", recipe.ID, err)
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
