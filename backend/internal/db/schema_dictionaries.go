package db

// Schema definitions for the master-data dictionaries. All columns are declared
// inline in CREATE TABLE (no ALTER migrations), so a fresh database gets the
// full schema. Statements stay idempotent (IF NOT EXISTS) so InitSchema can be
// re-run safely.

func schemaDictionaries() error {
	if err := execSchema("item_dictionary", `
	CREATE TABLE IF NOT EXISTS item_dictionary (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		item_type VARCHAR(50) NOT NULL,
		rank INTEGER DEFAULT 1,
		effects JSONB NOT NULL DEFAULT '{}',
		description TEXT,
		base_price INTEGER NOT NULL DEFAULT 0,
		image_url TEXT NOT NULL DEFAULT '',
		rarity INTEGER NOT NULL DEFAULT 0,
		path VARCHAR(50) NOT NULL DEFAULT ''
	);`); err != nil {
		return err
	}

	if err := execSchema("item_dictionary_path_upgrade", `
	ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS path VARCHAR(50) NOT NULL DEFAULT '';
	`); err != nil {
		return err
	}

	if err := execSchema("migrate_node_outputs_to_items", `
	DO $$
	BEGIN
		IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'node_outputs') THEN
			INSERT INTO item_dictionary (id, name, item_type, rank, rarity, description, path)
			SELECT item_id, item_name, item_type, rank, 
			       CASE WHEN item_tier = 'T1' THEN 1 WHEN item_tier = 'T2' THEN 2 WHEN item_tier = 'T3' THEN 3 WHEN item_tier = 'T4' THEN 4 WHEN item_tier = 'T5' THEN 5 WHEN item_tier = 'T6' THEN 6 WHEN item_tier = 'T7' THEN 7 WHEN item_tier = 'T8' THEN 8 WHEN item_tier = 'T9' THEN 9 ELSE 1 END,
			       description, COALESCE(food_path, '')
			FROM node_outputs
			ON CONFLICT (id) DO UPDATE SET 
			    path = EXCLUDED.path, 
			    rank = EXCLUDED.rank, 
			    rarity = EXCLUDED.rarity;
			
			DROP TABLE node_outputs CASCADE;
		END IF;
	END $$;
	`); err != nil {
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
		effects JSONB NOT NULL DEFAULT '{"active":{},"support":{},"passive":{},"house":{}}',
		hunger_rate INTEGER NOT NULL,
		food_types JSONB NOT NULL DEFAULT '[]',
		max_hunger INTEGER DEFAULT 100,
		description TEXT,
		target_type VARCHAR(50) NOT NULL DEFAULT 'SINGLE',
		rarity INTEGER NOT NULL DEFAULT 0,
		is_immortal BOOLEAN DEFAULT false,
		primary_role VARCHAR(80),
		combat_roles JSONB DEFAULT '[]',
		combat_tags JSONB DEFAULT '[]',
		support_tags JSONB DEFAULT '[]',
		base_price INTEGER NOT NULL DEFAULT 0,
		image_url TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (id, rank)
	);`); err != nil {
		return err
	}

	if err := execSchema("constitution_dictionary", `
	CREATE TABLE IF NOT EXISTS constitution_dictionary (
    tier INTEGER NOT NULL DEFAULT 0,
		key VARCHAR(100) PRIMARY KEY,
		name VARCHAR(200) NOT NULL,
		effects JSONB NOT NULL DEFAULT '{}'
	);`); err != nil {
		return err
	}

	if err := execSchema("race_dictionary", `
	CREATE TABLE IF NOT EXISTS race_dictionary (
		key VARCHAR(100) PRIMARY KEY,
		name VARCHAR(200) NOT NULL,
		effects JSONB NOT NULL DEFAULT '{}'
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
		effects JSONB NOT NULL DEFAULT '{}',
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
		primeval_stones_cost BIGINT DEFAULT 0,
		immortal_essence_stones_cost BIGINT DEFAULT 0,
		PRIMARY KEY (target_gu_id, target_rank, variant),
		FOREIGN KEY (target_gu_id, target_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT
	);`); err != nil {
		return err
	}

	if err := execSchema("refining_recipes_stones_type_upgrade", `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='refining_recipes' AND column_name='primeval_stones_cost' AND data_type='double precision') THEN
			ALTER TABLE refining_recipes ALTER COLUMN primeval_stones_cost TYPE BIGINT USING COALESCE(primeval_stones_cost, 0)::BIGINT;
		END IF;
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='refining_recipes' AND column_name='immortal_essence_stones_cost' AND data_type='double precision') THEN
			ALTER TABLE refining_recipes ALTER COLUMN immortal_essence_stones_cost TYPE BIGINT USING COALESCE(immortal_essence_stones_cost, 0)::BIGINT;
		END IF;
	END $$;`); err != nil {
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
		price INTEGER NOT NULL,
		stock INTEGER NOT NULL DEFAULT -1,
		is_default BOOLEAN NOT NULL DEFAULT TRUE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		updated_at TIMESTAMP DEFAULT NOW(),
		PRIMARY KEY (id, rank)
	);`); err != nil {
		return err
	}
	// Upgrade legacy databases that still carry the single-column (id) primary
	// key: add the rank column and promote the key to the composite (id, rank).
	if err := execSchema("caravan_shop_items_rank_upgrade", `
	ALTER TABLE caravan_shop_items ADD COLUMN IF NOT EXISTS rank INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE caravan_shop_items DROP CONSTRAINT IF EXISTS caravan_shop_items_pkey;
	ALTER TABLE caravan_shop_items ADD CONSTRAINT caravan_shop_items_pkey PRIMARY KEY (id, rank);`); err != nil {
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
		price INTEGER NOT NULL,
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

// seedDefaultRaceConstitutionRows pre-seeds the built-in race/constitution
// dictionaries plus an empty-key sentinel row. The empty keys keep the
// players.race / players.constitution foreign keys satisfied for the ” value
// written by GrantRace/admin paths; the default datasets mirror
// gamefx.defaultRaces/defaultConstitutions so the pre-import registry behaves
// exactly like the gamefx fallback (CSV imports remain authoritative).
func seedDefaultRaceConstitutionRows() error {
	races := []struct{ key, name, effects string }{
		{"", "(Không)", "{}"},
		{"human", "Nhân Tộc", "{}"},
		{"fur_people", "Người Lông", `{"refine":0.10}`},
		{"merman", "Nhân Ngư", `{"dao_mark":{"water":0.10},"path_damage":{"water":0.05}}`},
		{"feathered", "Vũ Dân", `{"stat_mods":{"speed":0.10},"dao_mark":{"space":0.05}}`},
		{"inkman", "Mặc Nhân", `{"dao_mark":{"wisdom":0.08,"soul":0.05}}`},
		{"rockman", "Thạch Nhân", `{"stat_mods":{"defense":0.15,"speed":-0.10}}`},
		{"snowman", "Tuyết Dân", `{"dao_mark":{"ice":0.15},"path_resistance":{"fire":0.10}}`},
		{"variant_beastman", "Thú Nhân", `{"stat_mods":{"attack":0.08,"defense":0.05},"refine":-0.05}`},
		{"dragonman", "Long Nhân", `{"stat_mods":{"attack":0.10,"defense":0.10},"refine":-0.05}`},
		{"giant_sun_descendant", "Cự Dương Huyết Mạch", `{"luck":5,"cultivation_efficiency":0.05}`},
	}
	for _, r := range races {
		if _, err := DB.Exec(`INSERT INTO race_dictionary (key, name, effects) VALUES ($1,$2,$3::jsonb) ON CONFLICT (key) DO NOTHING`, r.key, r.name, r.effects); err != nil {
			return err
		}
	}

	constitutions := []struct{ key, name, effects string; tier int }{
    {"", "(Không)", "{}", 0},
    {"solar_sun_moon_body", "Thái Nhật Dương Mãng Thể", `{"dao_mark":{"light":0.50},"path_damage":{"light":0.20},"max_mana":0.50,"mana_regen":1}`, 3},
    {"ancient_moon_yin_desolate_body", "Cổ Nguyệt Âm Hoang Thể", `{"dao_mark":{"dark":0.50},"max_mana":0.50,"mana_regen":1,"mental_resistance":0.20}`, 3},
    {"northern_dark_ice_soul_body", "Bắc Minh Băng Phách Thể", `{"dao_mark":{"ice":0.50},"path_resistance":{"ice":0.10},"max_mana":0.50,"mana_regen":1}`, 3},
    {"forest_sea_reincarnation_body", "Sâm Hải Luân Hồi Thể", `{"dao_mark":{"water":0.50},"healing_received":0.50,"max_mana":0.50,"mana_regen":1}`, 3},
    {"myriad_gold_brilliant_flower_body", "Vạn Kim Diệu Hoa Thể", `{"dao_mark":{"metal":0.50},"stat_mods":{"attack":0.30,"defense":0.30},"max_mana":0.50,"mana_regen":1}`, 3},
    {"blazing_glory_lightning_brilliance_body", "Viêm Hoàng Lôi Trạch Thể", `{"dao_mark":{"lightning":0.50},"stat_mods":{"speed":0.20},"max_mana":0.50,"mana_regen":1}`, 3},
    {"great_strength_true_martial_body", "Đại Lực Chân Võ Thể", `{"dao_mark":{"strength":0.50},"stat_mods":{"attack":0.30,"defense":0.30},"max_mana":0.50,"mana_regen":1}`, 3},
    {"carefree_wisdom_heart_body", "Tiêu Dao Trí Tâm Thể", `{"dao_mark":{"wisdom":0.50},"mental_resistance":0.30,"max_mana":0.50,"mana_regen":1}`, 3},
    {"great_earth_origin_body", "Hậu Thổ Nguyên Ương Thể", `{"dao_mark":{"earth":0.50},"stat_mods":{"defense":0.45},"mental_resistance":0.45,"max_mana":0.50,"mana_regen":1}`, 3},
    {"universe_great_evolution_body", "Vũ Trụ Đại Diễn Thể", `{"dao_mark":{"space":0.50},"stat_mods":{"attack":0.15,"defense":0.15,"speed":0.15},"max_mana":0.50,"mana_regen":1}`, 3},
    {"pure_dream_reality_seeker_body", "Thuần Mộng Cầu Chân Thể", `{"dao_mark":{"dream":0.40},"path_damage":{"dream":0.10},"max_mana":0.40,"mana_regen":1}`, 3},
}
	for _, c := range constitutions {
		if _, err := DB.Exec(`INSERT INTO constitution_dictionary (key, name, effects, tier) VALUES ($1,$2,$3::jsonb,$4) ON CONFLICT (key) DO NOTHING`, c.key, c.name, c.effects, c.tier); err != nil {
			return err
		}
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
		{"severely_wounded", "Trọng Thương", `{"stat_mods":{"attack":-0.20,"defense":-0.20,"speed":-0.20},"healing_received":-0.50}`, 0, `["explore"]`, "Thân trúng trọng thương, công thủ tốc đều giảm mạnh."},
		{"poisoned", "Trúng Độc", `{"stat_mods":{"attack":-0.10,"speed":-0.10}}`, 30, `[]`, "Độc tố xâm nhập cơ thể, suy giảm công kích và tốc độ."},
		{"exhausted", "Kiệt Sức", `{"stat_mods":{"speed":-0.30},"cultivation_efficiency":-0.50}`, 10, `["cultivate"]`, "Lực kiệt sức, tạm thời không thể tu luyện."},
		{"blessed", "Phúc Trạch", `{"luck":5,"cultivation_efficiency":0.10}`, 60, `[]`, "Nhận phúc trạch, may mắn và tu luyện hiệu quả hơn."},
		{"death_penalty", "Hình Phạt Tử Vong", `{"stat_mods":{"attack":-0.50,"defense":-0.50,"speed":-0.50}}`, 30, `[]`, "Sát khí chưa tan sau khi chết, toàn bộ chỉ số suy giảm."},
	}
	for _, s := range statuses {
		if _, err := DB.Exec(`
			INSERT INTO status_dictionary (key, name, effects, default_duration_minutes, blocked_actions, description)
			VALUES ($1,$2,$3::jsonb,$4,$5::jsonb,$6)
			ON CONFLICT (key) DO NOTHING`, s.key, s.name, s.effects, s.duration, s.blocked, s.desc); err != nil {
			return err
		}
	}
	return nil
}
