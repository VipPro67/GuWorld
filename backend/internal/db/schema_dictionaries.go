package db

// Schema definitions for the master-data dictionaries. New databases get the
// complete schema inline; additive compatibility statements upgrade existing
// tables before registries and imports query the new columns.

func schemaDictionaries() error {
	if err := execSchema("item_dictionary", `
	CREATE TABLE IF NOT EXISTS item_dictionary (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		item_type VARCHAR(50) NOT NULL,
		rank INTEGER DEFAULT 1,
		effects JSONB NOT NULL DEFAULT '{"schema_version":2,"blocks":[]}',
		description TEXT,
		base_price NUMERIC(30,3) NOT NULL DEFAULT 0,
		image_url TEXT NOT NULL DEFAULT '',
		rarity INTEGER NOT NULL DEFAULT 0,
		path VARCHAR(50) NOT NULL DEFAULT ''
	);`); err != nil {
		return err
	}
	// Catalog metadata belongs to the CSV/effect envelope, not dictionary rows.
	// Remove columns left by older database schemas so imports work on upgrades.
	if err := execSchema("item_dictionary", `ALTER TABLE item_dictionary DROP COLUMN IF EXISTS schema_version, DROP COLUMN IF EXISTS catalog_revision, DROP COLUMN IF EXISTS content_hash;`); err != nil {
		return err
	}

	if err := execSchema("gu_dictionary", `
	CREATE TABLE IF NOT EXISTS gu_dictionary (
		id VARCHAR(50) NOT NULL,
		rank INTEGER NOT NULL,
		name VARCHAR(100) NOT NULL,
		path JSONB NOT NULL DEFAULT '[]'::jsonb,
		gu_type VARCHAR(50) NOT NULL,
		base_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
		effects JSONB NOT NULL DEFAULT '{"schema_version":2,"blocks":[]}',
		hunger_rate INTEGER NOT NULL,
		food_types JSONB NOT NULL DEFAULT '[]',
		max_hunger INTEGER DEFAULT 100,
		description TEXT,
		target_type VARCHAR(50) NOT NULL DEFAULT 'SINGLE',
		rarity INTEGER NOT NULL DEFAULT 0,
		primary_role VARCHAR(80),
		combat_roles JSONB DEFAULT '[]',
		combat_tags JSONB DEFAULT '[]',
		support_tags JSONB DEFAULT '[]',
		base_price NUMERIC(30,3) NOT NULL DEFAULT 0,
		image_url TEXT NOT NULL DEFAULT '',
		skill_profile JSONB,
		PRIMARY KEY (id, rank)
	);`); err != nil {
		return err
	}
	if err := execSchema("gu_dictionary_skill_profile", `ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS skill_profile JSONB;`); err != nil {
		return err
	}
	// Older databases may still have the retired authoring flag. Immortality is
	// derived from rank (rank >= 6), so remove the redundant column on startup.
	if err := execSchema("gu_dictionary", `ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS is_immortal;`); err != nil {
		return err
	}
	if err := execSchema("gu_dictionary", `ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS schema_version, DROP COLUMN IF EXISTS catalog_revision, DROP COLUMN IF EXISTS content_hash;`); err != nil {
		return err
	}
	if err := execSchema("constitution_dictionary", `
	CREATE TABLE IF NOT EXISTS constitution_dictionary (
    tier INTEGER NOT NULL DEFAULT 0,
		key VARCHAR(100) PRIMARY KEY,
		name VARCHAR(200) NOT NULL,
		effects JSONB NOT NULL DEFAULT '{"schema_version":2,"blocks":[]}'
	);`); err != nil {
		return err
	}

	if err := execSchema("race_dictionary", `
	CREATE TABLE IF NOT EXISTS race_dictionary (
		key VARCHAR(100) PRIMARY KEY,
		name VARCHAR(200) NOT NULL,
		effects JSONB NOT NULL DEFAULT '{"schema_version":2,"blocks":[]}'
	);`); err != nil {
		return err
	}

	// Persistent player statuses (trạng thái). effects uses the same JSON
	// shape as constitution/race (gamefx.Effects); default_duration_minutes 0
	// means permanent; blocked_actions gates gameplay flows (gamefx.Action*).
	if err := execSchema("status_dictionary", `
	CREATE TABLE IF NOT EXISTS status_dictionary (
		key VARCHAR(100) PRIMARY KEY,
		name VARCHAR(200) NOT NULL,
		effects JSONB NOT NULL DEFAULT '{"schema_version":2,"blocks":[]}',
		default_duration_minutes INTEGER NOT NULL DEFAULT 0,
		blocked_actions JSONB NOT NULL DEFAULT '[]',
		description TEXT NOT NULL DEFAULT ''
	);`); err != nil {
		return err
	}

	if err := execSchema("loot_tables", `
	CREATE TABLE IF NOT EXISTS loot_tables (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		entries JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("refining_recipes", `
	CREATE TABLE IF NOT EXISTS refining_recipes (
		target_gu_id VARCHAR(50) NOT NULL,
		target_rank INTEGER NOT NULL,
		variant INTEGER NOT NULL DEFAULT 1,
		required_gus JSONB NOT NULL,
		required_materials JSONB NOT NULL,
		success_rate DOUBLE PRECISION NOT NULL,
		primeval_stones_cost NUMERIC(30,3) DEFAULT 0,
		immortal_essence_stones_cost NUMERIC(30,3) DEFAULT 0,
		PRIMARY KEY (target_gu_id, target_rank, variant),
		FOREIGN KEY (target_gu_id, target_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT
	);`); err != nil {
		return err
	}

	// Shop ids are polymorphic (an item id, a gu id or a recipe key) so they
	// deliberately have no FK to item_dictionary. A shop entry is keyed by
	// (id, rank): rank > 0 pins a specific dictionary rank (used for multi-rank
	// Gus), rank = 0 means "auto-resolve" (item rank / lowest Gu rank / recipe
	// target rank) which preserves the legacy single-rank behaviour.
	if err := execSchema("caravan_shop_items", `
	CREATE TABLE IF NOT EXISTS caravan_shop_items (
		id VARCHAR(50),
		rank INTEGER NOT NULL DEFAULT 0,
		price NUMERIC(30,3) NOT NULL,
		stock INTEGER NOT NULL DEFAULT -1,
		is_default BOOLEAN NOT NULL DEFAULT TRUE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		updated_at TIMESTAMP DEFAULT NOW(),
		PRIMARY KEY (id, rank)
	);`); err != nil {
		return err
	}
	// Tracks how much of each daily-random shop item (auto pool: rank < 5 Gus +
	// their food) remains for the current day. The per-day stock resets
	// naturally because the primary key includes `day`.
	if err := execSchema("caravan_shop_daily_stock", `
	CREATE TABLE IF NOT EXISTS caravan_shop_daily_stock (
		day DATE NOT NULL,
		id VARCHAR(50) NOT NULL,
		rank INTEGER NOT NULL DEFAULT 0,
		remaining INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (day, id, rank)
	);`); err != nil {
		return err
	}

	if err := execSchema("arena_shop_items", `
	CREATE TABLE IF NOT EXISTS arena_shop_items (
		id VARCHAR(50) PRIMARY KEY,
		price BIGINT NOT NULL,
		qty INTEGER NOT NULL DEFAULT 1,
		stock INTEGER NOT NULL DEFAULT -1,
		is_default BOOLEAN NOT NULL DEFAULT TRUE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		updated_at TIMESTAMP DEFAULT NOW()
	);`); err != nil {
		return err
	}
	return nil
}

// seedDefaultRaceConstitutionRows pre-seeds the empty-key sentinel rows.
// The empty keys keep the players.race / players.constitution foreign keys
// satisfied for the ” value before CSV seed or when unassigned.
// All gameplay race and constitution records are seeded from CSV directly
// into the database during InitSchema (seedConstitutionsFromCSV / seedRacesFromCSV).
func seedDefaultRaceConstitutionRows() error {
	if _, err := schemaDB().Exec(`INSERT INTO race_dictionary (key, name, effects) VALUES ('', '(Không)', '{}'::jsonb) ON CONFLICT (key) DO NOTHING`); err != nil {
		return err
	}
	// players.race defaults to 'human', so it must exist before seedSystemPlayers
	// runs on a fresh database (Race.csv import upserts the real effects later).
	if _, err := schemaDB().Exec(`INSERT INTO race_dictionary (key, name, effects) VALUES ('human', 'Nhân Tộc', '[]'::jsonb) ON CONFLICT (key) DO NOTHING`); err != nil {
		return err
	}
	if _, err := schemaDB().Exec(`INSERT INTO constitution_dictionary (key, name, effects, tier) VALUES ('', '(Không)', '{}'::jsonb, 0) ON CONFLICT (key) DO NOTHING`); err != nil {
		return err
	}
	return nil
}

// seedDefaultStatusRows pre-seeds the built-in status dictionary so statuses
// resolve before any CSV import. The dataset mirrors gamefx.defaultStatuses;
// Status.csv imports remain authoritative.
func seedDefaultStatusRows() error {
	statuses := []struct {
		key, name, effects string
		duration           int
		blocked            string
		desc               string
	}{
		{
			"severely_wounded",
			"Trọng Thương",
			`{"stat_mods":{"attack":-0.20,"defense":-0.20,"speed":-0.20},"healing_received":-0.50,"refine":-0.5}`,
			60,
			`["explore"]`,
			"Thân trúng trọng thương, công thủ tốc đều giảm mạnh.",
		},
		{
			"poisoned",
			"Trúng Độc",
			`{"stat_mods":{"attack":-0.10,"speed":-0.10}}`,
			30,
			`[]`,
			"Độc tố xâm nhập cơ thể, suy giảm công kích và tốc độ.",
		},
		{
			"exhausted",
			"Kiệt Sức",
			`{"stat_mods":{"speed":-0.30},"cultivation_efficiency":-0.50}`,
			10,
			`["cultivate"]`,
			"Lực kiệt sức, tạm thời không thể tu luyện.",
		},
		{
			"blessed",
			"Phúc Trạch",
			`{"luck":5,"cultivation_efficiency":0.10}`,
			60,
			`[]`,
			"Nhận phúc trạch, may mắn và tu luyện hiệu quả hơn.",
		},
		{
			"death_penalty",
			"Hình Phạt Tử Vong",
			`{"stat_mods":{"attack":-0.50,"defense":-0.50,"speed":-0.50}}`,
			30,
			`[]`,
			"Sát khí chưa tan sau khi chết, toàn bộ chỉ số suy giảm.",
		},
		{
			"refine_fatigue",
			"Tâm Thần Mệt Mỏi",
			`{"refine":-0.5}`,
			10,
			`[]`,
			"Luyện cổ khiến tâm thần mệt mỏi, giảm 5% tỉ lệ luyện cổ thành công.",
		},
		{
			"refine_backfire",
			"Luyện Cổ Phản Phệ",
			`{"refine":-0.95}`,
			20,
			`[]`,
			"Bị cổ khí phản phệ do cố luyện cổ khi mệt mỏi, giảm 95% tỉ lệ luyện cổ thành công.",
		},
		{
			"sat_khi",
			"Sát Khí Đồ Sát",
			`{"death_drop_mult":1.5}`,
			120,
			`[]`,
			"Tích tụ oán khí đồ sát. Khi bị đánh bại sẽ rơi thêm từ 50% đến 250% số lượng tài sản (tùy tầng Sát Khí).",
		},
		{
			"regional_wall_suppression",
			"Áp Chế Giới Vực",
			`{"stat_mods":{"attack":-0.20,"defense":-0.20,"speed":-0.20}}`,
			15,
			`[]`,
			"Chịu áp chế của Giới Vực Bích khi di chuyển giữa các Đại Vực. Tu vi rank càng cao bị áp chế càng nặng.",
		},
		{
			"pvp_immunity",
			"Miễn Chiến Hồi Sinh",
			`{"pvp_immune":1.0}`,
			10,
			`[]`,
			"Được Thiên Đạo che chở sau khi hồi sinh, miễn nhiễm mọi đòn tấn công PvP. Chủ động khai chiến sẽ phá vỡ trạng thái này.",
		},
	}

	for _, s := range statuses {
		if _, err := schemaDB().Exec(`
			INSERT INTO status_dictionary (
				key,
				name,
				effects,
				default_duration_minutes,
				blocked_actions,
				description
			)
			VALUES ($1, $2, $3::jsonb, $4, $5::jsonb, $6)
			ON CONFLICT (key) DO NOTHING
		`, s.key, s.name, s.effects, s.duration, s.blocked, s.desc); err != nil {
			return err
		}
	}

	return nil
}
