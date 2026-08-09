package db

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var DB *sql.DB

// baseRankCache memoizes the minimum dictionary rank of each Gu id, used when
// upgrading legacy recipe required_gus values to the ranked format.
var baseRankCache = map[string]int{}

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

	// Abort queries stuck on locks/deadlocks instead of hanging forever.
	// Defaults: statement 15s, lock wait 10s. Override via env (ms).
	statementTimeout := getEnv("DB_STATEMENT_TIMEOUT_MS", "15000")
	lockTimeout := getEnv("DB_LOCK_TIMEOUT_MS", "10000")
	connStr += fmt.Sprintf(" options='-c statement_timeout=%s -c lock_timeout=%s'", statementTimeout, lockTimeout)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(5 * time.Minute)

	logger.S.Info("Successfully connected to the database")
	return nil
}

// InitSchema creates the necessary tables and seeds initial master data if missing
func InitSchema() error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	_, err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	if err != nil {
		logger.S.Warnf("failed to create uuid-ossp extension: %v", err)
	}

	_, err = DB.Exec(`CREATE EXTENSION IF NOT EXISTS "unaccent";`)
	if err != nil {
		logger.S.Warnf("failed to create unaccent extension: %v", err)
	}

	// 1. Create item_dictionary
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS item_dictionary (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		item_type VARCHAR(50) NOT NULL,
		rank INTEGER DEFAULT 1,
		effects JSONB NOT NULL DEFAULT '{}',
		description TEXT
	);`)
	if err != nil {
		return fmt.Errorf("failed to create item_dictionary table: %w", err)
	}

	// 1.1. Create caravan_shop_items
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS caravan_shop_items (
		id VARCHAR(50) PRIMARY KEY,
		price INTEGER NOT NULL,
		stock INTEGER NOT NULL DEFAULT -1,
		is_default BOOLEAN NOT NULL DEFAULT TRUE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		updated_at TIMESTAMP DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create caravan_shop_items table: %w", err)
	}

	// 1.2. Create arena_shop_items
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS arena_shop_items (
		id VARCHAR(50) PRIMARY KEY,
		price INTEGER NOT NULL,
		qty INTEGER NOT NULL DEFAULT 1,
		stock INTEGER NOT NULL DEFAULT -1,
		is_default BOOLEAN NOT NULL DEFAULT TRUE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		updated_at TIMESTAMP DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create arena_shop_items table: %w", err)
	}

	// 2. Create gu_dictionary
	_, err = DB.Exec(`
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
		rarity VARCHAR(50),
		is_immortal BOOLEAN DEFAULT false,
		primary_role VARCHAR(80),
		combat_roles JSONB DEFAULT '[]',
		combat_tags JSONB DEFAULT '[]',
		support_tags JSONB DEFAULT '[]',
		PRIMARY KEY (id, rank)
	);`)
	if err != nil {
		return fmt.Errorf("failed to create gu_dictionary table: %w", err)
	}

	// Drop old refining_recipes table if it contains the legacy id column
	_, err = DB.Exec(`
	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'refining_recipes' AND column_name = 'id'
		) THEN
			DROP TABLE IF EXISTS refining_recipes CASCADE;
		END IF;
	END $$;`)
	if err != nil {
		return fmt.Errorf("failed to check/drop legacy refining_recipes table: %w", err)
	}

	// 3. Create refining_recipes
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS refining_recipes (
		target_gu_id VARCHAR(50) NOT NULL,
		target_rank INTEGER NOT NULL,
		variant INTEGER NOT NULL DEFAULT 1,
		required_gus JSONB NOT NULL,
		required_materials JSONB NOT NULL,
		success_rate DOUBLE PRECISION NOT NULL,
		primeval_stones_cost DOUBLE PRECISION DEFAULT 0,
		immortal_essence_stones_cost DOUBLE PRECISION DEFAULT 0,
		PRIMARY KEY (target_gu_id, target_rank, variant),
		FOREIGN KEY (target_gu_id, target_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT
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
		primeval_stones DOUBLE PRECISION DEFAULT 0,
		immortal_stones DOUBLE PRECISION DEFAULT 0,
		hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		base_hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		body_mods JSONB NOT NULL DEFAULT '[]',
		x INTEGER NOT NULL DEFAULT 50,
		y INTEGER NOT NULL DEFAULT 50,
		explored_cells JSONB NOT NULL DEFAULT '["50,50"]',
		current_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border',
		path_masteries JSONB NOT NULL DEFAULT '{}',
		active_combat JSONB DEFAULT NULL,
		last_mana_update TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		aperture_stress DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		last_cultivate_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		email VARCHAR(255) UNIQUE NOT NULL DEFAULT '',
		google_id VARCHAR(255) UNIQUE NULL,
		ref_code VARCHAR(50) UNIQUE NOT NULL DEFAULT '',
		earned_achievements JSONB NOT NULL DEFAULT '[]',
		selected_title VARCHAR(100) NOT NULL DEFAULT '',
		selected_frame VARCHAR(50) NOT NULL DEFAULT '',
		growth_data JSONB NOT NULL DEFAULT '{"boosts":{"attack":0,"defense":0,"hp":0,"speed":0},"fully_grown":[]}',
		race VARCHAR(50) NOT NULL DEFAULT '',
		gender VARCHAR(20) NOT NULL DEFAULT '',
		constitution VARCHAR(100) NOT NULL DEFAULT '',
		statuses JSONB NOT NULL DEFAULT '[]'
	);
	`
	_, err = DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create players table: %w", err)
	}

	// Seed system player for global/world/sect chat channels
	_, _ = DB.Exec(`
		INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, immortal_stones, hp, base_hp, current_region, x, y, email, ref_code, is_admin)
		VALUES ('00000000-0000-0000-0000-000000000000', 'Bảo Hoàng Thiên', '', 6, 1, 'Giáp', 99, 100, 100, 0, 0, 100, 100, 'Southern_Border', 50, 50, 'system@guworld.local', 'REF-SYSTEM', true)
		ON CONFLICT (id) DO NOTHING;
	`)

	// Seed system NPC player "Hệ thống" for automated notifications
	_, _ = DB.Exec(`
		INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, immortal_stones, hp, base_hp, current_region, x, y, email, ref_code, is_admin)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Hệ thống', '', 6, 1, 'Giáp', 99, 100, 100, 0, 0, 100, 100, 'Southern_Border', 50, 50, 'system_npc@guworld.local', 'REF-SYSTEM-NPC', true)
		ON CONFLICT (id) DO NOTHING;
	`)

	// Create referrals table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS referrals (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		referrer_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		referee_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE UNIQUE,
		referee_ip VARCHAR(50) NOT NULL DEFAULT '',
		referee_device TEXT NOT NULL DEFAULT '',
		is_suspicious BOOLEAN NOT NULL DEFAULT false,
		has_deposited BOOLEAN NOT NULL DEFAULT false,
		deposit_amount_stones INTEGER NOT NULL DEFAULT 0,
		milestone1_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone2_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone3_rewarded BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create referrals table: %w", err)
	}

	// 5. Create immortal_apertures table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS immortal_apertures (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL UNIQUE REFERENCES players(id) ON DELETE CASCADE,
		grade VARCHAR(20) NOT NULL,
		time_rate DOUBLE PRECISION NOT NULL DEFAULT 10,
		aperture_hp BIGINT NOT NULL DEFAULT 10000,
		max_aperture_hp BIGINT NOT NULL DEFAULT 10000,
		area_size BIGINT NOT NULL DEFAULT 3000000,
		immortal_essence BIGINT NOT NULL DEFAULT 0,
		essence_production_per_year INTEGER NOT NULL DEFAULT 10,
		internal_years_passed DOUBLE PRECISION NOT NULL DEFAULT 0,
		next_tribulation_year DOUBLE PRECISION NOT NULL DEFAULT 10,
		tribulation_count INTEGER NOT NULL DEFAULT 0,
		rank_start_year DOUBLE PRECISION NOT NULL DEFAULT 0,
		earth_calamity_count INTEGER NOT NULL DEFAULT 0,
		heavenly_tribulation_count INTEGER NOT NULL DEFAULT 0,
		grand_tribulation_count INTEGER NOT NULL DEFAULT 0,
		myriad_tribulation_count INTEGER NOT NULL DEFAULT 0,
		chaos_calamity_count INTEGER NOT NULL DEFAULT 0,
		state VARCHAR(30) NOT NULL DEFAULT 'STABLE',
		dao_marks JSONB NOT NULL DEFAULT '{"strength": 0}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create immortal_apertures table: %w", err)
	}

	// Migration: add new columns if missing (safe for existing DBs)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE;`)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ALTER COLUMN updated_at TYPE TIMESTAMP WITH TIME ZONE;`)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS rank_start_year DOUBLE PRECISION NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS earth_calamity_count INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS heavenly_tribulation_count INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS grand_tribulation_count INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS myriad_tribulation_count INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE immortal_apertures ADD COLUMN IF NOT EXISTS chaos_calamity_count INTEGER NOT NULL DEFAULT 0;`)

	// 7. Create aperture_resource_nodes table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS aperture_resource_nodes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		node_type VARCHAR(50) NOT NULL,
		name VARCHAR(100) NOT NULL,
		path VARCHAR(50),
		level INTEGER NOT NULL DEFAULT 1,
		hp INTEGER NOT NULL DEFAULT 1000,
		max_hp INTEGER NOT NULL DEFAULT 1000,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		production_type VARCHAR(50),
		production_amount INTEGER NOT NULL DEFAULT 0,
		production_interval_hours INTEGER NOT NULL DEFAULT 24,
		last_collected_at TIMESTAMP WITH TIME ZONE,
		is_destroyed BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create aperture_resource_nodes table: %w", err)
	}

	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ALTER COLUMN last_collected_at TYPE TIMESTAMP WITH TIME ZONE;`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE;`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ALTER COLUMN updated_at TYPE TIMESTAMP WITH TIME ZONE;`)

	// 7b. Defense columns for aperture_resource_nodes
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS node_category VARCHAR(20) NOT NULL DEFAULT 'RESOURCE';`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS defense_power INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS attack_range INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS hp_bonus INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS damage_reduction INTEGER NOT NULL DEFAULT 0;`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS blocks_movement BOOLEAN NOT NULL DEFAULT false;`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS required_gu_id VARCHAR(50);`)
	_, _ = DB.Exec(`ALTER TABLE aperture_resource_nodes ADD COLUMN IF NOT EXISTS required_gu_rank INTEGER DEFAULT 0;`)
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_aperture_nodes_category ON aperture_resource_nodes(aperture_id, node_category);`)

	// 8. Create tribulation_events table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS tribulation_events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		tribulation_type VARCHAR(30) NOT NULL,
		path VARCHAR(50) NOT NULL,
		difficulty INTEGER NOT NULL DEFAULT 1,
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		aperture_hp_before BIGINT NOT NULL,
		aperture_hp_after BIGINT,
		reward_dao_marks BIGINT NOT NULL DEFAULT 0,
		started_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		ended_at TIMESTAMP WITH TIME ZONE
	);`)
	if err != nil {
		return fmt.Errorf("failed to create tribulation_events table: %w", err)
	}

	_, _ = DB.Exec(`ALTER TABLE tribulation_events ALTER COLUMN started_at TYPE TIMESTAMP WITH TIME ZONE;`)
	_, _ = DB.Exec(`ALTER TABLE tribulation_events ALTER COLUMN ended_at TYPE TIMESTAMP WITH TIME ZONE;`)

	// 9. Create tribulation_enemies table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS tribulation_enemies (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		event_id UUID NOT NULL REFERENCES tribulation_events(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		enemy_type VARCHAR(50) NOT NULL,
		path VARCHAR(50) NOT NULL,
		hp BIGINT NOT NULL,
		max_hp BIGINT NOT NULL,
		attack BIGINT NOT NULL,
		defense BIGINT NOT NULL,
		target_node_id UUID REFERENCES aperture_resource_nodes(id) ON DELETE SET NULL,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		is_boss BOOLEAN NOT NULL DEFAULT false,
		is_dead BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP DEFAULT now(),
		updated_at TIMESTAMP DEFAULT now()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create tribulation_enemies table: %w", err)
	}

	// dao_marks (relational mirror) removed: dao marks live in immortal_apertures.dao_marks JSONB
	_, err = DB.Exec(`DROP TABLE IF EXISTS dao_marks;`)
	if err != nil {
		return fmt.Errorf("failed to drop legacy dao_marks table: %w", err)
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
		listing_type VARCHAR(20) NOT NULL DEFAULT 'AUCTION',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		duration_hours INTEGER NOT NULL DEFAULT 24
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
	// Create mine_slots table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS mine_slots (
		id INTEGER PRIMARY KEY,
		rarity VARCHAR(50) NOT NULL,
		occupier_id UUID REFERENCES players(id) ON DELETE SET NULL,
		monster_id VARCHAR(50) NULL,
		occupied_at TIMESTAMPTZ NULL,
		last_claimed_at TIMESTAMPTZ NULL,
		abandoned_at TIMESTAMPTZ DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create mine_slots table: %w", err)
	}

	// Seed mine slots if empty
	var mineCount int
	err = DB.QueryRow("SELECT COUNT(*) FROM mine_slots").Scan(&mineCount)
	if err == nil && mineCount == 0 {
		// 31 slots: 1 Siêu Cấp, 3 Truyền Thuyết, 9 Sử Thi, 18 Hiếm
		slotRarities := map[int]string{}
		for i := 1; i <= 31; i++ {
			switch {
			case i == 1:
				slotRarities[i] = "Siêu Cấp"
			case i <= 4:
				slotRarities[i] = "Truyền Thuyết"
			case i <= 13:
				slotRarities[i] = "Sử Thi"
			default:
				slotRarities[i] = "Hiếm"
			}
		}
		for i := 1; i <= 31; i++ {
			_, err = DB.Exec("INSERT INTO mine_slots (id, rarity, abandoned_at) VALUES ($1, $2, NOW()) ON CONFLICT (id) DO NOTHING", i, slotRarities[i])
			if err != nil {
				logger.S.Errorf("Failed to seed mine %d: %v", i, err)
			}
		}
		logger.S.Info("Seeded 31 initial world mine slots.")
	}

	// Migration: upgrade existing 20-slot DB to 31-slot layout
	if mineCount > 0 && mineCount <= 20 {
		logger.S.Info("Migrating mine slots from 20 to 31 layout...")
		// Update rarities for existing slots
		updates := map[int]string{
			1: "Siêu Cấp",
			2: "Truyền Thuyết", 3: "Truyền Thuyết", 4: "Truyền Thuyết",
			5: "Sử Thi", 6: "Sử Thi", 7: "Sử Thi", 8: "Sử Thi", 9: "Sử Thi",
			10: "Sử Thi", 11: "Sử Thi", 12: "Sử Thi", 13: "Sử Thi",
		}
		for id, rarity := range updates {
			_, _ = DB.Exec("UPDATE mine_slots SET rarity = $1 WHERE id = $2", rarity, id)
		}
		// Remove old CommonModule slots (id 14-20 if they were Thường)
		_, _ = DB.Exec("DELETE FROM mine_slots WHERE id > 20")
		// Insert new slots 21-31 (all Hiếm)
		for i := 21; i <= 31; i++ {
			_, _ = DB.Exec("INSERT INTO mine_slots (id, rarity, abandoned_at) VALUES ($1, 'Hiếm', NOW()) ON CONFLICT (id) DO NOTHING", i)
		}
		logger.S.Info("Mine slots migrated to 31 layout.")
	}

	// Fix rarity typo: "Truyện Thuyết" → "Truyền Thuyết"
	_, _ = DB.Exec(`UPDATE mine_slots SET rarity = 'Truyền Thuyết' WHERE rarity = 'Truyện Thuyết'`)

	// Migrate existing mine_slots to have monster_id and abandoned_at if not exist
	_, _ = DB.Exec(`ALTER TABLE mine_slots ADD COLUMN IF NOT EXISTS monster_id VARCHAR(50) NULL;`)
	_, _ = DB.Exec(`ALTER TABLE mine_slots ADD COLUMN IF NOT EXISTS abandoned_at TIMESTAMPTZ DEFAULT NOW();`)
	_, _ = DB.Exec(`ALTER TABLE mine_slots ADD COLUMN IF NOT EXISTS monster_rank INT NOT NULL DEFAULT 1;`)
	_, _ = DB.Exec(`ALTER TABLE mine_slots ADD COLUMN IF NOT EXISTS monster_sub_rank INT NOT NULL DEFAULT 1;`)

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
		daily_challenges INTEGER NOT NULL DEFAULT 0,
		last_challenge_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create arena_rankings table: %w", err)
	}

	// Create leaderboard_settings table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS leaderboard_settings (
		player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		show_on_leaderboard BOOLEAN NOT NULL DEFAULT TRUE
	);`)
	if err != nil {
		return fmt.Errorf("failed to create leaderboard_settings table: %w", err)
	}

	// Ensure all existing players have a leaderboard_settings row
	_, _ = DB.Exec(`
		INSERT INTO leaderboard_settings (player_id, show_on_leaderboard)
		SELECT id, TRUE FROM players
		ON CONFLICT (player_id) DO NOTHING
	`)

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
		max_hp DOUBLE PRECISION NOT NULL,
		base_attack DOUBLE PRECISION NOT NULL,
		base_defense DOUBLE PRECISION NOT NULL,
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

	// Drop old columns if they exist
	_, _ = DB.Exec(`
		ALTER TABLE monster_templates 
		DROP COLUMN IF EXISTS rank, 
		DROP COLUMN IF EXISTS sub_rank, 
		DROP COLUMN IF EXISTS path, 
		DROP COLUMN IF EXISTS max_mana, 
		DROP COLUMN IF EXISTS current_mana;
	`)

	// Monster capture columns
	_, err = DB.Exec(`
		ALTER TABLE monster_templates ADD COLUMN IF NOT EXISTS is_capturable BOOLEAN NOT NULL DEFAULT false;
		ALTER TABLE monster_templates ADD COLUMN IF NOT EXISTS capture_chance DOUBLE PRECISION NOT NULL DEFAULT 0.20;
		ALTER TABLE monster_templates ADD COLUMN IF NOT EXISTS pet_hunger_per_hour DOUBLE PRECISION NOT NULL DEFAULT 2.0;
	`)
	if err != nil {
		logger.S.Warnf("failed to add monster_templates capture columns: %v", err)
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

	// 4.9. Create quests table (seeded from CSV)
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS quests (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		req_desc TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		rewards JSONB NOT NULL DEFAULT '[]'
	);`)
	if err != nil {
		return fmt.Errorf("failed to create quests table: %w", err)
	}

	// 4.10. Create map_cells (fka 4.9)
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

	// 4.11. Create instance_templates
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS instance_templates (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		type VARCHAR(50) NOT NULL, -- "INHERITANCE_RIGHTEOUS", "INHERITANCE_DEMONIC", "DREAM_REALM"
		required_rank INTEGER NOT NULL,
		entry_item VARCHAR(50) NOT NULL, -- Item ID needed to enter or "NONE"
		description TEXT NOT NULL,
		rewards JSONB NOT NULL DEFAULT '{}', -- e.g., {"attainment": {"path": "Sword", "amount": 1}}
		map_data JSONB NOT NULL DEFAULT '[]' -- Sequence of stages / rooms
	);`)
	if err != nil {
		return fmt.Errorf("failed to create instance_templates table: %w", err)
	}

	// 4.12. Create player_instance_sessions
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS player_instance_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		instance_id VARCHAR(50) NOT NULL REFERENCES instance_templates(id) ON DELETE CASCADE,
		current_stage INTEGER NOT NULL DEFAULT 0,
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- "ACTIVE", "COMPLETED", "FAILED"
		temp_inventory JSONB NOT NULL DEFAULT '[]', -- Lock player's own items for Dream Realms
		created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create player_instance_sessions table: %w", err)
	}

	_, _ = DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS player_active_session_idx ON player_instance_sessions(player_id) WHERE status = 'ACTIVE';`)

	// 4.13. Create sects
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS sects (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL,
		region VARCHAR(50) NOT NULL,
		vault_gu JSONB NOT NULL DEFAULT '[]',
		vault_materials JSONB NOT NULL DEFAULT '{}',
		vault_recipes JSONB NOT NULL DEFAULT '[]',
		funds INTEGER NOT NULL DEFAULT 0,
		level INTEGER NOT NULL DEFAULT 1
	);`)
	if err != nil {
		return fmt.Errorf("failed to create sects table: %w", err)
	}

	// 4.14. Create sect_members
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS sect_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE UNIQUE,
		member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
		contribution_points INTEGER NOT NULL DEFAULT 0,
		master_id VARCHAR(50) DEFAULT NULL,
		joined_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create sect_members table: %w", err)
	}

	// 4.15. Create sect_missions
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS sect_missions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
		title VARCHAR(100) NOT NULL,
		description TEXT NOT NULL,
		mission_type VARCHAR(30) NOT NULL,
		target_id VARCHAR(50) NOT NULL,
		target_amount INTEGER NOT NULL DEFAULT 1,
		reward_contribution INTEGER NOT NULL DEFAULT 0,
		required_member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
		expires_at TIMESTAMP WITHOUT TIME ZONE
	);`)
	if err != nil {
		return fmt.Errorf("failed to create sect_missions table: %w", err)
	}
	_, _ = DB.Exec(`ALTER TABLE sect_missions ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITHOUT TIME ZONE;`)
	// Deduplicate and prevent duplicate mission rows
	_, _ = DB.Exec(`DELETE FROM sect_missions WHERE id NOT IN (SELECT MIN(id) FROM sect_missions GROUP BY sect_id, title);`)
	_, _ = DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_sect_mission_title ON sect_missions (sect_id, title);`)

	// 4.16. Create player_sect_missions
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS player_sect_missions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		mission_id UUID NOT NULL REFERENCES sect_missions(id) ON DELETE CASCADE,
		current_progress INTEGER NOT NULL DEFAULT 0,
		status VARCHAR(30) NOT NULL DEFAULT 'IN_PROGRESS',
		assigned_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
		expires_at TIMESTAMP WITHOUT TIME ZONE,
		UNIQUE(player_id, mission_id)
	);`)
	if err != nil {
		return fmt.Errorf("failed to create player_sect_missions table: %w", err)
	}
	_, _ = DB.Exec(`ALTER TABLE player_sect_missions ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITHOUT TIME ZONE;`)

	// 4.17. Create npcs
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS npcs (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE SET NULL,
		role VARCHAR(50) DEFAULT 'DISCIPLE',
		aptitude_percent INTEGER DEFAULT 50,
		created_at TIMESTAMP DEFAULT now()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create npcs table: %w", err)
	}

	// Premium wallet, top-up, and shop accounting tables
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS wallets (
		user_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		primeval_stone BIGINT NOT NULL DEFAULT 0,
		immortal_essence_stone BIGINT NOT NULL DEFAULT 0,
		immortal_jade BIGINT NOT NULL DEFAULT 0,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create wallets table: %w", err)
	}

	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS topup_orders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		order_code BIGINT NOT NULL UNIQUE,
		amount_vnd BIGINT NOT NULL CHECK (amount_vnd > 0),
		jade_amount BIGINT NOT NULL CHECK (jade_amount > 0),
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PAID', 'EXPIRED', 'FAILED', 'CANCELLED')),
		provider VARCHAR(20) NOT NULL DEFAULT 'PAYOS',
		checkout_url TEXT,
		qr_code TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		paid_at TIMESTAMP WITH TIME ZONE NULL
	);`)
	if err != nil {
		return fmt.Errorf("failed to create topup_orders table: %w", err)
	}

	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS wallet_transactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		type VARCHAR(30) NOT NULL CHECK (type IN ('TOPUP', 'SPEND', 'REFUND', 'ADMIN_ADJUST', 'PACKAGE_REWARD')),
		currency VARCHAR(40) NOT NULL CHECK (currency IN ('PRIMEVAL_STONE', 'IMMORTAL_ESSENCE_STONE', 'IMMORTAL_JADE')),
		amount BIGINT NOT NULL,
		balance_before BIGINT NOT NULL,
		balance_after BIGINT NOT NULL,
		ref_order_code BIGINT NULL,
		ref_package_id VARCHAR(80) NULL,
		note TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create wallet_transactions table: %w", err)
	}

	_, err = DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS wallet_transactions_topup_order_unique ON wallet_transactions (ref_order_code) WHERE type = 'TOPUP' AND ref_order_code IS NOT NULL;`)
	if err != nil {
		return fmt.Errorf("failed to create wallet topup transaction unique index: %w", err)
	}

	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS shop_package_purchases (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		package_id VARCHAR(80) NOT NULL,
		price_jade BIGINT NOT NULL,
		rewards_snapshot JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create shop_package_purchases table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS shop_package_purchases_user_package_created_idx ON shop_package_purchases (user_id, package_id, created_at);`)
	if err != nil {
		return fmt.Errorf("failed to create shop package purchases index: %w", err)
	}

	_, err = DB.Exec(`
		INSERT INTO wallets (user_id, primeval_stone, immortal_essence_stone, immortal_jade)
		SELECT id, COALESCE(primeval_stones, 0)::BIGINT, FLOOR(COALESCE(immortal_stones, 0))::BIGINT, 0
		FROM players
		ON CONFLICT (user_id) DO NOTHING;
	`)
	if err != nil {
		return fmt.Errorf("failed to backfill wallets: %w", err)
	}

	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS killer_moves (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		name VARCHAR(80) NOT NULL,
		path VARCHAR(50) NOT NULL DEFAULT '',
		move_type VARCHAR(30) NOT NULL DEFAULT 'UTILITY',
		rank_required INTEGER NOT NULL DEFAULT 1,
		core_player_gu_id VARCHAR(80) NOT NULL,
		total_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
		success_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
		damage_value DOUBLE PRECISION NOT NULL DEFAULT 0,
		defense_value DOUBLE PRECISION NOT NULL DEFAULT 0,
		cooldown INTEGER NOT NULL DEFAULT 0,
		channel_time INTEGER NOT NULL DEFAULT 1,
		effects JSONB NOT NULL DEFAULT '{}',
		is_mastered BOOLEAN NOT NULL DEFAULT FALSE,
		is_reliable BOOLEAN NOT NULL DEFAULT FALSE,
		is_instant_cast BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create killer_moves table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS killer_moves_player_path_idx ON killer_moves (player_id, path);`)
	if err != nil {
		return fmt.Errorf("failed to create killer_moves index: %w", err)
	}

	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS killer_move_slots (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		killer_move_id UUID NOT NULL REFERENCES killer_moves(id) ON DELETE CASCADE,
		player_gu_id VARCHAR(80) NOT NULL,
		slot_type VARCHAR(20) NOT NULL CHECK (slot_type IN ('CORE','SUPPORT')),
		slot_index INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (killer_move_id, slot_index)
	);`)
	if err != nil {
		return fmt.Errorf("failed to create killer_move_slots table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS killer_move_slots_move_idx ON killer_move_slots (killer_move_id);`)
	if err != nil {
		return fmt.Errorf("failed to create killer_move_slots index: %w", err)
	}

	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS player_gu (
		id VARCHAR(80) PRIMARY KEY,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		gu_id VARCHAR(50) NOT NULL,
		gu_rank INTEGER NOT NULL DEFAULT 1,
		rank INTEGER NOT NULL DEFAULT 1,
		FOREIGN KEY (gu_id, gu_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT,
		level INTEGER NOT NULL DEFAULT 1,
		hunger DOUBLE PRECISION NOT NULL DEFAULT 100,
		state VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		location VARCHAR(30) NOT NULL DEFAULT 'INVENTORY',
		is_locked BOOLEAN NOT NULL DEFAULT false,
		equipped BOOLEAN NOT NULL DEFAULT false,
		is_vital BOOLEAN NOT NULL DEFAULT false,
		integration_time DOUBLE PRECISION NOT NULL DEFAULT 0,
		equipped_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
		growth_hours DOUBLE PRECISION NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("failed to create player_gu table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS player_gu_player_idx ON player_gu (player_id);`)
	if err != nil {
		return fmt.Errorf("failed to create player_gu index: %w", err)
	}

	// player_gu_player_equipped_idx / player_gu_player_gu_idx are created after
	// the migration loop below, because the equipped column is added by a
	// migration on databases that predate this schema.

	// Backwards-compatible column migrations
	migrations := []struct{ sql, errMsg string }{
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';`, "password_hash"},
		{`ALTER TABLE players ALTER COLUMN primeval_stones TYPE DOUBLE PRECISION;`, "primeval_stones_type"},
		{`ALTER TABLE refining_recipes ALTER COLUMN primeval_stones_cost TYPE DOUBLE PRECISION;`, "primeval_stones_cost_type"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS materials_inventory JSONB NOT NULL DEFAULT '{}';`, "materials_inventory"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS hp DOUBLE PRECISION NOT NULL DEFAULT 100;`, "hp"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS base_hp DOUBLE PRECISION NOT NULL DEFAULT 100;`, "base_hp"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS body_mods JSONB NOT NULL DEFAULT '[]';`, "body_mods"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS x INTEGER NOT NULL DEFAULT 50;`, "x"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS y INTEGER NOT NULL DEFAULT 50;`, "y"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS explored_cells JSONB NOT NULL DEFAULT '["50,50"]';`, "explored_cells"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS current_region VARCHAR(50) NOT NULL DEFAULT 'Southern_Border';`, "current_region"},

		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS active_combat JSONB DEFAULT NULL;`, "active_combat"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS path_masteries JSONB NOT NULL DEFAULT '{}';`, "path_masteries"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS equipped_gus JSONB NOT NULL DEFAULT '[]';`, "equipped_gus"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS cleared_cells JSONB NOT NULL DEFAULT '{}';`, "cleared_cells"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS immortal_stones DOUBLE PRECISION NOT NULL DEFAULT 0;`, "immortal_stones"},
		{`ALTER TABLE refining_recipes ADD COLUMN IF NOT EXISTS immortal_essence_stones_cost DOUBLE PRECISION NOT NULL DEFAULT 0;`, "immortal_essence_stones_cost"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS path JSONB NOT NULL DEFAULT '[]'::jsonb;`, "path"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS base_mana_cost DOUBLE PRECISION NOT NULL DEFAULT 0;`, "base_mana_cost"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS effects JSONB NOT NULL DEFAULT '{"active":{},"support":{},"passive":{},"house":{}}';`, "effects"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS food_types JSONB NOT NULL DEFAULT '[]';`, "food_types"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS rarity VARCHAR(50);`, "gu_dictionary.rarity"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS is_immortal BOOLEAN DEFAULT false;`, "gu_dictionary.is_immortal"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS primary_role VARCHAR(80);`, "gu_dictionary.primary_role"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS combat_roles JSONB DEFAULT '[]';`, "gu_dictionary.combat_roles"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS combat_tags JSONB DEFAULT '[]';`, "gu_dictionary.combat_tags"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS support_tags JSONB DEFAULT '[]';`, "gu_dictionary.support_tags"}, // Cultivation system columns
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS aptitude_grade INTEGER DEFAULT 4;`, "aptitude_grade"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS wall_progress DOUBLE PRECISION NOT NULL DEFAULT 0.0;`, "wall_progress"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS human_qi DOUBLE PRECISION NOT NULL DEFAULT 0.0;`, "human_qi"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS next_tribulation_time TIMESTAMPTZ NULL;`, "next_tribulation_time"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS tribulation_count INTEGER NOT NULL DEFAULT 0;`, "tribulation_count"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS rank INTEGER DEFAULT 1;`, "item_dictionary.rank"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS effects JSONB NOT NULL DEFAULT '{}';`, "item_dictionary.effects"},

		// Composite PK (id, rank) migration for gu_dictionary
		// Must drop FKs first before dropping the PK they reference
		{`ALTER TABLE refining_recipes ADD COLUMN IF NOT EXISTS target_rank INTEGER NOT NULL DEFAULT 1;`, "add_target_rank"},
		// Deterministic backfill: use the gu's minimum dictionary rank instead of
		// a FROM-join (which is nondeterministic for multi-rank gus and could
		// collide with an existing (gu_id, target_rank, variant) row). Rows whose
		// target (gu_id, rank) already exists are left untouched.
		{`UPDATE refining_recipes rr
			SET target_rank = gd.min_rank
			FROM (SELECT id, MIN(rank) AS min_rank FROM gu_dictionary GROUP BY id) gd
			WHERE gd.id = rr.target_gu_id
			  AND rr.target_rank = 1
			  AND NOT EXISTS (
				  SELECT 1 FROM refining_recipes r2
				  WHERE r2.target_gu_id = rr.target_gu_id
				    AND r2.target_rank = gd.min_rank
				    AND r2.variant = rr.variant
			  );`, "backfill_target_rank"},
		// Drop stale legacy rows left at rank 1 that are exact duplicates of a
		// canonical (gu_id, min_rank, variant) row, otherwise the FK re-added
		// below would reject them.
		{`DELETE FROM refining_recipes rr
			WHERE rr.target_rank = 1
			  AND (SELECT MIN(gd.rank) FROM gu_dictionary gd WHERE gd.id = rr.target_gu_id) > 1
			  AND EXISTS (
				  SELECT 1 FROM refining_recipes r2
				  WHERE r2.target_gu_id = rr.target_gu_id
				    AND r2.target_rank = (SELECT MIN(gd2.rank) FROM gu_dictionary gd2 WHERE gd2.id = rr.target_gu_id)
				    AND r2.variant = rr.variant
			  );`, "dedupe_stale_target_rank"},
		{`ALTER TABLE refining_recipes DROP CONSTRAINT IF EXISTS refining_recipes_target_gu_id_fkey;`, "drop_recipe_fk"},
		{`ALTER TABLE refining_recipes DROP CONSTRAINT IF EXISTS refining_recipes_target_gu_id_target_rank_fkey;`, "drop_recipe_fk2"},
		{`ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS gu_rank INTEGER NOT NULL DEFAULT 1;`, "add_player_gu_rank"},
		{`UPDATE player_gu SET gu_rank = rank WHERE gu_rank = 1;`, "backfill_player_gu_rank"},
		{`ALTER TABLE player_gu DROP CONSTRAINT IF EXISTS player_gu_gu_id_fkey;`, "drop_player_gu_fk"},
		{`ALTER TABLE player_gu DROP CONSTRAINT IF EXISTS player_gu_gu_id_gu_rank_fkey;`, "drop_player_gu_fk2"},
		{`ALTER TABLE gu_dictionary DROP CONSTRAINT IF EXISTS gu_dictionary_pkey;`, "drop_gu_dict_pk"},
		{`ALTER TABLE gu_dictionary ADD PRIMARY KEY (id, rank);`, "add_gu_dict_pk"},
		{`ALTER TABLE refining_recipes ADD FOREIGN KEY (target_gu_id, target_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT;`, "add_recipe_fk"},
		{`ALTER TABLE player_gu ADD FOREIGN KEY (gu_id, gu_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT;`, "add_player_gu_fk"},
		{`ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS equipped BOOLEAN NOT NULL DEFAULT false;`, "player_gu.equipped"},
		{`ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS is_vital BOOLEAN NOT NULL DEFAULT false;`, "player_gu.is_vital"},
		{`ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS integration_time DOUBLE PRECISION NOT NULL DEFAULT 0;`, "player_gu.integration_time"},
		{`ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS equipped_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;`, "player_gu.equipped_at"},
		{`ALTER TABLE player_gu ADD COLUMN IF NOT EXISTS growth_hours DOUBLE PRECISION NOT NULL DEFAULT 0;`, "player_gu.growth_hours"},
		{`DO $$
			BEGIN
				IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'players' AND column_name = 'gu_inventory') THEN
					-- Normalize each inventory entry to a dictionary-valid (gu_id, rank):
					--   A) rank 0 -> the dictionary's minimum rank for that gu_id (starter Gus were stored with rank 0)
					--   B) if the (gu_id, rank) pair still has no match, fall back to the dictionary's minimum rank
					--      so the FK constraint holds (rows are preserved instead of dropped).
					INSERT INTO player_gu (id, player_id, gu_id, gu_rank, rank, level, hunger, state, location, is_locked, equipped, is_vital, integration_time, equipped_at, growth_hours)
					WITH normalized AS (
						SELECT
							elem->>'instance_id' AS instance_id,
							p.id AS player_id,
							elem->>'gu_id' AS gu_id,
							CASE
								WHEN (elem->>'rank')::int = 0 THEN COALESCE((SELECT MIN(gd.rank) FROM gu_dictionary gd WHERE gd.id = elem->>'gu_id'), 1)
								ELSE COALESCE((elem->>'rank')::int, 1)
							END AS gu_rank,
							COALESCE((elem->>'hunger')::int, 100) AS hunger,
							COALESCE(NULLIF(elem->>'status', ''), 'ACTIVE') AS state,
							COALESCE((elem->>'equipped')::boolean, false) AS equipped,
							COALESCE((elem->>'is_vital')::boolean, false) AS is_vital,
							COALESCE((elem->>'integration_time')::double precision, 0) AS integration_time,
							NULLIF(elem->>'equipped_at', '')::timestamptz AS equipped_at,
							COALESCE((elem->>'growth_hours')::double precision, 0) AS growth_hours
						FROM players p
						CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory, '[]'::jsonb)) AS elem
						WHERE elem->>'instance_id' IS NOT NULL
						  AND elem->>'gu_id' IS NOT NULL
					)
					SELECT n.instance_id, n.player_id, n.gu_id, n.gu_rank, n.gu_rank, 1, n.hunger, n.state, 'INVENTORY', false, n.equipped, n.is_vital, n.integration_time, n.equipped_at, n.growth_hours
					FROM normalized n
					WHERE NOT EXISTS (SELECT 1 FROM player_gu pg WHERE pg.id = n.instance_id)
					  AND EXISTS (SELECT 1 FROM gu_dictionary gd WHERE gd.id = n.gu_id AND gd.rank = n.gu_rank);
				END IF;
			END $$;`, "migrate_player_gu_from_inventory"},
		{`ALTER TABLE players DROP COLUMN IF EXISTS gu_inventory;`, "drop_player_gu_inventory"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS temp_loot JSONB NOT NULL DEFAULT '{"gu_inventory": [], "materials_inventory": {}, "primeval_stones": 0}';`, "temp_loot"},
		{`UPDATE players SET temp_loot = COALESCE(temp_loot, '{}'::jsonb) WHERE temp_loot IS NULL;`, "fix_null_temp_loot"},
		{`UPDATE players SET temp_loot = jsonb_set(temp_loot, '{gu_inventory}', COALESCE((SELECT jsonb_agg(CASE WHEN elem->>'rank' IS NULL OR elem->>'rank' = '' THEN elem || jsonb_build_object('rank', COALESCE((SELECT gd.rank FROM gu_dictionary gd WHERE gd.id = elem->>'gu_id' LIMIT 1), 1)) ELSE elem END) FROM jsonb_array_elements(COALESCE(temp_loot->'gu_inventory', '[]'::jsonb)) AS elem), '[]'::jsonb), false) WHERE temp_loot IS NOT NULL AND temp_loot::text != '{}' AND temp_loot::text != 'null';`, "backfill_temp_loot_rank"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS lifespan DOUBLE PRECISION NOT NULL DEFAULT 100.0;`, "lifespan"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS time_speed_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0;`, "time_speed_multiplier"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS party_id UUID NULL;`, "party_id"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS arena_rank INTEGER NULL;`, "arena_rank"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50) NULL;`, "phone_number"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS email VARCHAR(100) NULL;`, "email"},
		{`CREATE UNIQUE INDEX IF NOT EXISTS players_email_idx ON players (email) WHERE email IS NOT NULL AND email != '';`, "players_email_idx"},
		{`CREATE UNIQUE INDEX IF NOT EXISTS players_phone_number_idx ON players (phone_number) WHERE phone_number IS NOT NULL AND phone_number != '';`, "players_phone_number_idx"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS explore_count INTEGER NOT NULL DEFAULT 0;`, "explore_count"},
		{`ALTER TABLE arena_rankings ADD COLUMN IF NOT EXISTS daily_challenges INTEGER NOT NULL DEFAULT 0;`, "arena_rankings.daily_challenges"},
		{`ALTER TABLE arena_rankings ADD COLUMN IF NOT EXISTS last_challenge_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;`, "arena_rankings.last_challenge_at"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS last_explore_reset TIMESTAMP NOT NULL DEFAULT NOW();`, "last_explore_reset"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS aperture_stress DOUBLE PRECISION NOT NULL DEFAULT 0.0;`, "aperture_stress"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS last_cultivate_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;`, "last_cultivate_time"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS target_type VARCHAR(50) NOT NULL DEFAULT 'SINGLE';`, "target_type"},
		{`ALTER TABLE messages ADD COLUMN IF NOT EXISTS channel VARCHAR(50) NOT NULL DEFAULT 'private';`, "messages.channel"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS soul_hp INTEGER NOT NULL DEFAULT 10;`, "soul_hp"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS max_soul INTEGER NOT NULL DEFAULT 10;`, "max_soul"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS soul_tier VARCHAR(30) NOT NULL DEFAULT 'NHAN_HON';`, "soul_tier"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS needs_soul_refine BOOLEAN NOT NULL DEFAULT FALSE;`, "needs_soul_refine"},
		{`ALTER TABLE messages ADD COLUMN IF NOT EXISTS sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE CASCADE;`, "messages.sect_id"},
		{`ALTER TABLE sects ADD COLUMN IF NOT EXISTS funds INTEGER NOT NULL DEFAULT 0;`, "sects.funds"},
		{`ALTER TABLE sects ADD COLUMN IF NOT EXISTS level INTEGER NOT NULL DEFAULT 1;`, "sects.level"},
		{`ALTER TABLE sects ADD COLUMN IF NOT EXISTS require_approval BOOLEAN NOT NULL DEFAULT FALSE;`, "sects.require_approval"},
		{`ALTER TABLE sects ADD COLUMN IF NOT EXISTS require_promotion_approval BOOLEAN NOT NULL DEFAULT FALSE;`, "sects.require_promotion_approval"},
		{`CREATE TABLE IF NOT EXISTS sect_join_requests (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
			UNIQUE(sect_id, player_id)
		);`, "sect_join_requests"},
		{`CREATE TABLE IF NOT EXISTS sect_promotion_requests (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			current_rank VARCHAR(30) NOT NULL,
			target_rank VARCHAR(30) NOT NULL,
			status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
			UNIQUE(sect_id, player_id)
		);`, "sect_promotion_requests"},
		{`CREATE TABLE IF NOT EXISTS aperture_variant_humans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
			race VARCHAR(50) NOT NULL,
			population INTEGER NOT NULL DEFAULT 0,
			happiness DOUBLE PRECISION NOT NULL DEFAULT 70.0,
			oppression DOUBLE PRECISION NOT NULL DEFAULT 0.0,
			rebellion_rate DOUBLE PRECISION NOT NULL DEFAULT 0.0,
			labor_intensity VARCHAR(20) NOT NULL DEFAULT 'NORMAL',
			created_at TIMESTAMP DEFAULT now(),
			updated_at TIMESTAMP DEFAULT now(),
			UNIQUE(aperture_id, race)
		);`, "aperture_variant_humans"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS bio TEXT DEFAULT 'Hãy viết gì đó về bản thân...';`, "bio"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS fake_rank INTEGER DEFAULT 1;`, "fake_rank"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS fake_sub_rank INTEGER DEFAULT 1;`, "fake_sub_rank"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS fake_vital_gu VARCHAR(50) DEFAULT '';`, "fake_vital_gu"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS is_disguised BOOLEAN DEFAULT true;`, "is_disguised"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false;`, "is_admin"},
		{`CREATE TABLE IF NOT EXISTS feedback (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			description TEXT NOT NULL,
			category VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			total_votes INTEGER NOT NULL DEFAULT 0,
			developer_response TEXT DEFAULT NULL,
			rewarded BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`, "feedback_table"},
		{`CREATE TABLE IF NOT EXISTS feedback_votes (
			feedback_id UUID NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			vote_type INTEGER NOT NULL,
			PRIMARY KEY (feedback_id, user_id)
		);`, "feedback_votes_table"},
		{`CREATE TABLE IF NOT EXISTS feedback_comments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			feedback_id UUID NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`, "feedback_comments_table"},
		{`UPDATE players SET email = 'system@guworld.local' WHERE id = '00000000-0000-0000-0000-000000000000' AND (email IS NULL OR email = '');`, "set_system_player_email"},
		{`DROP INDEX IF EXISTS players_phone_number_idx;`, "drop_phone_number_idx"},
		{`DROP INDEX IF EXISTS players_email_idx;`, "drop_email_idx"},
		{`ALTER TABLE players ALTER COLUMN email SET DATA TYPE VARCHAR(255);`, "alter_email_type"},
		{`UPDATE players SET email = 'placeholder_' || id || '@guworld.local' WHERE email IS NULL OR email = '';`, "fill_empty_emails"},
		{`ALTER TABLE players ALTER COLUMN email SET NOT NULL;`, "make_email_not_null"},
		{`ALTER TABLE players DROP CONSTRAINT IF EXISTS players_email_unique;`, "drop_email_unique_constraint"},
		{`ALTER TABLE players ADD CONSTRAINT players_email_unique UNIQUE (email);`, "make_email_unique"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS google_id VARCHAR(255) NULL;`, "add_google_id"},
		{`CREATE UNIQUE INDEX IF NOT EXISTS players_google_id_idx ON players (google_id) WHERE google_id IS NOT NULL AND google_id != '';`, "create_google_id_idx"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS ref_code VARCHAR(50) NULL;`, "add_ref_code_column"},
		{`UPDATE players SET ref_code = 'REF-SYSTEM' WHERE id = '00000000-0000-0000-0000-000000000000' AND (ref_code IS NULL OR ref_code = '');`, "set_system_ref_code"},
		{`UPDATE players SET ref_code = 'REF-' || UPPER(SUBSTRING(id::text, 1, 8)) WHERE ref_code IS NULL OR ref_code = '';`, "fill_empty_ref_codes"},
		{`ALTER TABLE players ALTER COLUMN ref_code SET NOT NULL;`, "make_ref_code_not_null"},
		{`ALTER TABLE players DROP CONSTRAINT IF EXISTS players_ref_code_unique;`, "drop_ref_code_unique_constraint"},
		{`ALTER TABLE players ADD CONSTRAINT players_ref_code_unique UNIQUE (ref_code);`, "make_ref_code_unique"},
		{`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name='referrals' AND column_name='referred_id'
			) THEN
				DROP TABLE referrals;
			END IF;
		END $$;`, "conditionally_drop_old_referrals_table"},
		{`CREATE TABLE IF NOT EXISTS referrals (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			referrer_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			referee_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE UNIQUE,
			referee_ip VARCHAR(50) NOT NULL DEFAULT '',
			referee_device TEXT NOT NULL DEFAULT '',
			is_suspicious BOOLEAN NOT NULL DEFAULT false,
			has_deposited BOOLEAN NOT NULL DEFAULT false,
			deposit_amount_stones INTEGER NOT NULL DEFAULT 0,
			milestone1_rewarded BOOLEAN NOT NULL DEFAULT false,
			milestone2_rewarded BOOLEAN NOT NULL DEFAULT false,
			milestone3_rewarded BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`, "create_referrals_table"},
		{`ALTER TABLE auction ADD COLUMN IF NOT EXISTS listing_type VARCHAR(20) NOT NULL DEFAULT 'AUCTION';`, "add_listing_type_to_auction"},
		{`ALTER TABLE auction ADD COLUMN IF NOT EXISTS target_buyer_id UUID DEFAULT NULL;`, "add_target_buyer_id_to_auction"},
		{`CREATE TABLE IF NOT EXISTS trades (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			sender_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			receiver_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			sender_offer JSONB NOT NULL DEFAULT '{"primeval_stones":0,"immortal_stones":0,"gus":[],"materials":{}}',
			receiver_offer JSONB NOT NULL DEFAULT '{"primeval_stones":0,"immortal_stones":0,"gus":[],"materials":{}}',
			sender_locked BOOLEAN NOT NULL DEFAULT false,
			receiver_locked BOOLEAN NOT NULL DEFAULT false,
			sender_confirmed BOOLEAN NOT NULL DEFAULT false,
			receiver_confirmed BOOLEAN NOT NULL DEFAULT false,
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`, "create_trades_table"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS avatar_url TEXT DEFAULT '';`, "avatar_url"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS combat_settings JSONB NOT NULL DEFAULT '{}';`, "combat_settings"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS growth_data JSONB NOT NULL DEFAULT '{"boosts":{"attack":0,"defense":0,"hp":0,"speed":0},"fully_grown":[]}';`, "growth_data"},
		{`CREATE TABLE IF NOT EXISTS sect_mines (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL UNIQUE,
			rarity VARCHAR(50) NOT NULL,
			occupier_sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE SET NULL,
			occupied_at TIMESTAMP WITH TIME ZONE,
			shield_until TIMESTAMP WITH TIME ZONE,
			last_claimed_at TIMESTAMP WITH TIME ZONE
		);`, "create_sect_mines_table"},
		{`INSERT INTO sect_mines (name, rarity) VALUES 
			('Thái Cổ Tinh Mạch', 'Truyền Thuyết'),
			('Cửu U Huyết Mạch', 'Sử Thi'),
			('Thiên Cương Linh Khí Mạch', 'Hiếm'),
			('Huyền Vũ Thạch Động', 'Hiếm'),
			('Bách Thảo Tiên Vực', 'Thường')
			ON CONFLICT (name) DO NOTHING;`, "seed_sect_mines"},
		{`ALTER TABLE killer_moves ADD COLUMN IF NOT EXISTS usage_count INTEGER NOT NULL DEFAULT 0;`, "usage_count"},

		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS needs_onboarding BOOLEAN NOT NULL DEFAULT false;`, "needs_onboarding"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS race VARCHAR(50) NOT NULL DEFAULT '';`, "race"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS gender VARCHAR(20) NOT NULL DEFAULT '';`, "gender"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS constitution VARCHAR(100) NOT NULL DEFAULT '';`, "constitution"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS statuses JSONB NOT NULL DEFAULT '[]';`, "statuses"},
		{`ALTER TABLE killer_moves ADD COLUMN IF NOT EXISTS success_rate DOUBLE PRECISION NOT NULL DEFAULT 0;`, "add_success_rate"},
		{`ALTER TABLE killer_moves ADD COLUMN IF NOT EXISTS channel_time INTEGER NOT NULL DEFAULT 1;`, "add_channel_time"},
		{`ALTER TABLE killer_moves ADD COLUMN IF NOT EXISTS is_reliable BOOLEAN NOT NULL DEFAULT FALSE;`, "add_is_reliable"},
		{`CREATE TABLE IF NOT EXISTS combat_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			winner VARCHAR(20) NOT NULL DEFAULT 'Phe Địch',
			logs JSONB NOT NULL DEFAULT '[]',
			summary JSONB NOT NULL DEFAULT '{}',
			enemy_name VARCHAR(100) NOT NULL DEFAULT '',
			enemy_type VARCHAR(50) NOT NULL DEFAULT '',
			region VARCHAR(50) NOT NULL DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`, "create_combat_history_table"},
		{`ALTER TABLE combat_history ADD COLUMN IF NOT EXISTS combatants JSONB NOT NULL DEFAULT '[]';`, "add_combatants_to_history"},
		{`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			player_id UUID REFERENCES players(id) ON DELETE SET NULL,
			action TEXT NOT NULL,
			entity_type TEXT,
			entity_id TEXT,
			request_summary TEXT,
			metadata JSONB DEFAULT '{}',
			ip_address TEXT,
			success BOOLEAN NOT NULL DEFAULT false,
			error_message TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`, "create_audit_logs_table"},
		// Unified pricing: base_price column as single source of truth for all pricing calculations
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS base_price INTEGER NOT NULL DEFAULT 0;`, "gu_dictionary.base_price"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS base_price INTEGER NOT NULL DEFAULT 0;`, "item_dictionary.base_price"},
		{`ALTER TABLE gu_dictionary ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';`, "gu_dictionary.image_url"},
		{`ALTER TABLE item_dictionary ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';`, "item_dictionary.image_url"},
		{`CREATE TABLE IF NOT EXISTS email_otps (
			email VARCHAR(255) PRIMARY KEY,
			otp VARCHAR(10) NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`, "create_email_otps_table"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS last_activity TIMESTAMP WITH TIME ZONE DEFAULT NULL;`, "last_activity"},
		{`CREATE TABLE IF NOT EXISTS gift_codes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code VARCHAR(50) NOT NULL UNIQUE,
			rewards JSONB NOT NULL DEFAULT '[]',
			max_uses INTEGER NOT NULL DEFAULT 1,
			used_count INTEGER NOT NULL DEFAULT 0,
			max_per_player INTEGER NOT NULL DEFAULT 1,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_by UUID REFERENCES players(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`, "create_gift_codes_table"},
		{`CREATE TABLE IF NOT EXISTS gift_code_redemptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			gift_code_id UUID NOT NULL REFERENCES gift_codes(id) ON DELETE CASCADE,
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			redeemed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`, "create_gift_code_redemptions_table"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS current_thought DOUBLE PRECISION NOT NULL DEFAULT 100.0;`, "current_thought"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS max_thought DOUBLE PRECISION NOT NULL DEFAULT 100.0;`, "max_thought"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS thought_regen_rate DOUBLE PRECISION NOT NULL DEFAULT 1.0;`, "thought_regen_rate"},
		{`ALTER TABLE players ADD COLUMN IF NOT EXISTS immortal_state JSONB NOT NULL DEFAULT '{}';`, "immortal_state"},
		{`ALTER TABLE auction ADD COLUMN IF NOT EXISTS duration_hours INTEGER NOT NULL DEFAULT 24;`, "duration_hours"},
		{`ALTER TABLE immortal_apertures ALTER COLUMN immortal_essence TYPE BIGINT;`, "alter_immortal_essence_type"},
		{`ALTER TABLE gift_code_redemptions DROP CONSTRAINT IF EXISTS gift_code_redemptions_gift_code_id_player_id_key;`, "drop_gift_code_redemptions_unique_constraint"},
		{`CREATE INDEX IF NOT EXISTS gift_code_redemptions_gift_code_id_player_id_idx ON gift_code_redemptions(gift_code_id, player_id);`, "create_gift_code_redemptions_index"},
		{`ALTER TABLE tribulation_enemies ALTER COLUMN hp TYPE BIGINT;`, "alter_tribulation_enemies_hp_type"},
		{`ALTER TABLE tribulation_enemies ALTER COLUMN max_hp TYPE BIGINT;`, "alter_tribulation_enemies_max_hp_type"},
		{`ALTER TABLE tribulation_enemies ALTER COLUMN attack TYPE BIGINT;`, "alter_tribulation_enemies_attack_type"},
		{`ALTER TABLE tribulation_enemies ALTER COLUMN defense TYPE BIGINT;`, "alter_tribulation_enemies_defense_type"},
		{`ALTER TABLE tribulation_events ALTER COLUMN aperture_hp_before TYPE BIGINT;`, "alter_tribulation_events_aperture_hp_before_type"},
		{`ALTER TABLE tribulation_events ALTER COLUMN aperture_hp_after TYPE BIGINT;`, "alter_tribulation_events_aperture_hp_after_type"},
		{`ALTER TABLE tribulation_events ALTER COLUMN reward_dao_marks TYPE BIGINT;`, "alter_tribulation_events_reward_dao_marks_type"},
		{`ALTER TABLE immortal_apertures ALTER COLUMN aperture_hp TYPE BIGINT;`, "alter_immortal_apertures_aperture_hp_type"},
		{`ALTER TABLE immortal_apertures ALTER COLUMN max_aperture_hp TYPE BIGINT;`, "alter_immortal_apertures_max_aperture_hp_type"},
		{`CREATE TABLE IF NOT EXISTS party_invitations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			party_id UUID NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
			inviter_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			invitee_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(party_id, invitee_id)
		);`, "create_party_invitations_table"},
		{`ALTER TABLE players DROP COLUMN IF EXISTS phone_number;`, "drop_phone_number_column"},
		{`ALTER TABLE players DROP COLUMN IF EXISTS max_lifespan;`, "drop_max_lifespan_column"},
		{`ALTER TABLE players DROP COLUMN IF EXISTS wisdom;`, "drop_wisdom_column"},
		{`UPDATE players SET time_speed_multiplier = 1.0;`, "reset_time_speed_multiplier"},
		{`ALTER TABLE players DROP COLUMN IF EXISTS tribulation_count;`, "drop_tribulation_count_column"},
	}
	for _, m := range migrations {
		if _, err = DB.Exec(m.sql); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", m.errMsg, err)
		}
	}

	// player_gu secondary indexes — created here so they work on databases whose
	// player_gu table predates the equipped/gu_rank columns (added by migration).
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS player_gu_player_equipped_idx ON player_gu (player_id, equipped);`)
	if err != nil {
		return fmt.Errorf("failed to create player_gu equipped index: %w", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS player_gu_player_gu_idx ON player_gu (player_id, gu_id);`)
	if err != nil {
		return fmt.Errorf("failed to create player_gu gu_id index: %w", err)
	}

	// Phase 1 tables: player_recipe_progress
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_recipe_progress (
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			recipe_id VARCHAR(50) NOT NULL,
			completion_rate FLOAT NOT NULL DEFAULT 0,
			is_unlocked BOOLEAN NOT NULL DEFAULT false,
			deduce_count INT NOT NULL DEFAULT 0,
			last_deduce_at TIMESTAMPTZ,
			favorite BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (player_id, recipe_id)
		);`)
	if err != nil {
		return fmt.Errorf("failed to create player_recipe_progress table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_recipe_progress_player ON player_recipe_progress(player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create idx_recipe_progress_player: %v", err)
	}

	// Phase 1 tables: player_map_progress
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_map_progress (
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			region_id VARCHAR(50) NOT NULL DEFAULT '',
			cell_x INT NOT NULL,
			cell_y INT NOT NULL,
			explored BOOLEAN NOT NULL DEFAULT false,
			cleared BOOLEAN NOT NULL DEFAULT false,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (player_id, region_id, cell_x, cell_y)
		);`)
	if err != nil {
		return fmt.Errorf("failed to create player_map_progress table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_map_progress_player ON player_map_progress(player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create idx_map_progress_player: %v", err)
	}

	// Phase 1 tables: combat_sessions
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS combat_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			state JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(player_id)
		);`)
	if err != nil {
		return fmt.Errorf("failed to create combat_sessions table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_combat_sessions_player ON combat_sessions(player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create idx_combat_sessions_player: %v", err)
	}

	// Phase 3 tables: player_dao_path
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_dao_path (
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			path VARCHAR(50) NOT NULL,
			mastery VARCHAR(50) NOT NULL DEFAULT 'Ordinary',
			dao_marks INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (player_id, path)
		);`)
	if err != nil {
		return fmt.Errorf("failed to create player_dao_path table: %w", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_player_dao_path_player ON player_dao_path(player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create idx_player_dao_path_player: %v", err)
	}

	// Phase 6 tables: player_inventory_items
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_inventory_items (
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			item_id VARCHAR(50) NOT NULL,
			category VARCHAR(30) NOT NULL DEFAULT 'materials',
			quantity INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (player_id, item_id, category)
		);`)
	if err != nil {
		return fmt.Errorf("failed to create player_inventory_items table: %w", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_inventory_player ON player_inventory_items(player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create idx_inventory_player: %v", err)
	}

	// Player pets table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_pets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			monster_template_id VARCHAR(50) NOT NULL REFERENCES monster_templates(id),
			nickname VARCHAR(60) NOT NULL DEFAULT '',
			rank INT NOT NULL DEFAULT 1,
			sub_rank INT NOT NULL DEFAULT 1,
			hp DOUBLE PRECISION NOT NULL,
			max_hp DOUBLE PRECISION NOT NULL,
			base_attack DOUBLE PRECISION NOT NULL,
			base_defense DOUBLE PRECISION NOT NULL,
			speed DOUBLE PRECISION NOT NULL DEFAULT 10,
			count INT NOT NULL DEFAULT 1,
			hunger DOUBLE PRECISION NOT NULL DEFAULT 100,
			max_hunger DOUBLE PRECISION NOT NULL DEFAULT 100,
			pet_hunger_per_hour DOUBLE PRECISION NOT NULL DEFAULT 2.0,
			state VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
			formation_slot INT NOT NULL DEFAULT 0,
			captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`)
	if err != nil {
		logger.S.Warnf("failed to create player_pets table: %v", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS player_pets_player_idx ON player_pets (player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create player_pets index: %v", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS player_pets_formation_idx ON player_pets (player_id, formation_slot);`)
	if err != nil {
		logger.S.Warnf("failed to create player_pets formation index: %v", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS player_pets_monster_idx ON player_pets (player_id, monster_template_id);`)
	if err != nil {
		logger.S.Warnf("failed to create player_pets monster index: %v", err)
	}

	// Phase 6 tables: player_quest_progress
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_quest_progress (
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			quest_id VARCHAR(50) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
			progress_data JSONB NOT NULL DEFAULT '{}',
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (player_id, quest_id)
		);`)
	if err != nil {
		return fmt.Errorf("failed to create player_quest_progress table: %w", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_quest_progress_player ON player_quest_progress(player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create idx_quest_progress_player: %v", err)
	}

	// Phase 6 tables: player_achievements
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_achievements (
			player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			achievement_id VARCHAR(50) NOT NULL,
			selected_title VARCHAR(100) NOT NULL DEFAULT '',
			selected_frame VARCHAR(50) NOT NULL DEFAULT '',
			unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (player_id, achievement_id)
		);`)
	if err != nil {
		return fmt.Errorf("failed to create player_achievements table: %w", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_achievements_player ON player_achievements(player_id);`)
	if err != nil {
		logger.S.Warnf("failed to create idx_achievements_player: %v", err)
	}

	// Set is_admin = true only if no admin exists yet (first deploy)
	var adminCount int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM players WHERE is_admin = true`).Scan(&adminCount)
	if adminCount == 0 {
		_, err = DB.Exec(`UPDATE players SET is_admin = true WHERE username IN ('admin', 'Bảo Hoàng Thiên')`)
		if err != nil {
			logger.S.Warnf("failed to update is_admin status: %v", err)
		}
	}

	// Drop obsolete columns
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_power;`)
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_damage;`)
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS base_defense;`)
	_, _ = DB.Exec(`ALTER TABLE gu_dictionary DROP COLUMN IF EXISTS origin_region;`)

	// Migrations for path column in gu_dictionary (varchar to jsonb)
	var dataType string
	_ = DB.QueryRow(`
		SELECT data_type FROM information_schema.columns 
		WHERE table_name = 'gu_dictionary' AND column_name = 'path'
	`).Scan(&dataType)
	if dataType == "character varying" {
		logger.S.Info("Migrating gu_dictionary.path from VARCHAR to JSONB...")
		_, _ = DB.Exec(`ALTER TABLE gu_dictionary ALTER COLUMN path TYPE JSONB USING jsonb_build_array(path);`)
		_, _ = DB.Exec(`ALTER TABLE gu_dictionary ALTER COLUMN path SET DEFAULT '[]'::jsonb;`)
	}

	// 4b. Create indexes for frequently queried foreign key columns
	indexes := []struct {
		name   string
		table  string
		column string
	}{
		{"idx_players_username", "players", "username"},
		{"idx_combat_history_player", "combat_history", "player_id"},
		{"idx_messages_sender", "messages", "sender_id"},
		{"idx_messages_receiver", "messages", "receiver_id"},
		{"idx_messages_sect", "messages", "sect_id"},
		{"idx_topup_orders_user", "topup_orders", "user_id"},
		{"idx_topup_orders_status", "topup_orders", "status"},
		{"idx_wallet_transactions_user", "wallet_transactions", "user_id"},
		{"idx_player_gu_gu_id", "player_gu", "gu_id"},
		{"idx_auction_seller", "auction", "seller_id"},
		{"idx_auction_bidder", "auction", "highest_bidder_id"},
		{"idx_auction_status", "auction", "status"},
		{"idx_trades_sender", "trades", "sender_id"},
		{"idx_trades_receiver", "trades", "receiver_id"},
		{"idx_trades_status", "trades", "status"},
		{"idx_monster_spawns_region", "monster_spawns", "region_id"},
		{"idx_monster_spawns_monster", "monster_spawns", "monster_id"},
		{"idx_sect_members_sect", "sect_members", "sect_id"},
		{"idx_sect_missions_sect", "sect_missions", "sect_id"},
		{"idx_referrals_referrer", "referrals", "referrer_id"},
		{"idx_parties_leader", "parties", "leader_id"},
		{"idx_mine_slots_occupier", "mine_slots", "occupier_id"},
		{"idx_feedback_user", "feedback", "user_id"},
		{"idx_feedback_status", "feedback", "status"},
		{"idx_feedback_comments_feedback", "feedback_comments", "feedback_id"},
		{"idx_audit_logs_player", "audit_logs", "player_id"},
		{"idx_audit_logs_action", "audit_logs", "action"},
		{"idx_audit_logs_created", "audit_logs", "created_at"},
		{"idx_messages_receiver_channel_created", "messages", "receiver_id, channel, created_at DESC"},
		{"idx_auction_status_endtime_buyer", "auction", "status, end_time, target_buyer_id"},
	}
	for _, idx := range indexes {
		_, err := DB.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", idx.name, idx.table, idx.column))
		if err != nil {
			logger.S.Warnf("failed to create index %s: %v", idx.name, err)
		}
	}

	logger.S.Info("Database schema initialized successfully (seed skipped, use admin panel to import data)")
	return nil
}

// SeedDictionaries populates gameplay master data from CSV files.
func SeedDictionaries() error {
	if err := seedItemsFromCSV("Item.csv"); err != nil {
		return fmt.Errorf("CSV item seed failed: %w", err)
	}
	if err := seedGusFromCSV("Gu.csv"); err != nil {
		return fmt.Errorf("CSV Gu seed failed: %w", err)

	}
	if err := seedRecipesFromCSV("Recipe.csv"); err != nil {
		return fmt.Errorf("CSV recipe seed failed: %w", err)
	}
	if err := seedShopItemsFromCSV("ShopItem.csv"); err != nil {
		return fmt.Errorf("CSV shop item seed failed: %w", err)
	}
	if err := seedArenaShopItemsFromCSV("ArenaShopItem.csv"); err != nil {
		return fmt.Errorf("CSV arena shop item seed failed: %w", err)
	}
	if err := SeedMonstersAndSpawns(); err != nil {
		return fmt.Errorf("monster and spawn seeding failed: %w", err)
	}
	return nil
}

// SeedFromCSV runs all CSV-based data seeding in dependency order.
// This is called manually by admin (AdminImportData) instead of at startup.
func SeedFromCSV() error {
	if err := SeedDictionaries(); err != nil {
		return fmt.Errorf("dictionaries seeding failed: %w", err)
	}
	if err := SeedInstanceTemplates(); err != nil {
		return fmt.Errorf("instances seeding failed: %w", err)
	}
	if err := SeedSects(); err != nil {
		return fmt.Errorf("sects seeding failed: %w", err)
	}
	if err := seedQuestsFromCSV(); err != nil {
		return fmt.Errorf("quests seeding failed: %w", err)
	}
	if err := seedNPCs(); err != nil {
		return fmt.Errorf("npcs seeding failed: %w", err)
	}
	logger.S.Info("All CSV data seeded successfully")
	return nil
}

// csvPath resolves a CSV filename by searching the working directory and parent directories.
func csvPath(filename string) string {
	candidates := []string{
		filepath.Join("data", filename),
		filepath.Join("cmd/server/data", filename),
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
	r.FieldsPerRecord = -1
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
		if strings.TrimPrefix(strings.TrimSpace(h), "\uFEFF") == name {
			return i
		}
	}
	return -1
}

func safeJSONObject(raw string, fallback string, rowLabel string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		logger.S.Warnf("invalid JSON object in %s: %v", rowLabel, err)
		return fallback
	}
	if _, ok := parsed.(map[string]interface{}); !ok {
		logger.S.Warnf("JSON object expected in %s", rowLabel)
		return fallback
	}
	return value
}

func safeJSONArray(raw string, fallback string, rowLabel string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		logger.S.Warnf("invalid JSON array in %s: %v", rowLabel, err)
		return fallback
	}
	if _, ok := parsed.([]interface{}); !ok {
		logger.S.Warnf("JSON array expected in %s", rowLabel)
		return fallback
	}
	return value
}

func normalizedGuEffectsJSON(raw string, rowLabel string) string {
	value := safeJSONObject(raw, `{}`, rowLabel)
	var parsed map[string]interface{}
	_ = json.Unmarshal([]byte(value), &parsed)
	sections := map[string]interface{}{
		"active":     map[string]interface{}{},
		"support":    map[string]interface{}{},
		"passive":    map[string]interface{}{},
		"house":      map[string]interface{}{},
		"explore":    map[string]interface{}{},
		"refinement": map[string]interface{}{},
		"growth":     map[string]interface{}{},
		"other":      []interface{}{},
	}
	knownSections := []string{"active", "support", "passive", "house", "explore", "refinement", "growth", "other"}
	sectionFound := false
	for _, section := range knownSections {
		if rawSection, ok := parsed[section]; ok {
			sectionFound = true
			sections[section] = rawSection
		}
	}
	if !sectionFound {
		sections["active"] = parsed
	}
	bytes, _ := json.Marshal(sections)
	return string(bytes)
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
	rankIdx := colIndex(header, "rank")
	effectsIdx := colIndex(header, "effects")
	descIdx := colIndex(header, "description")
	basePriceIdx := colIndex(header, "base_price")
	imageURLIdx := colIndex(header, "image_url")

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

		rank := 1
		if rankIdx >= 0 && rankIdx < len(row) {
			fmt.Sscanf(strings.TrimSpace(row[rankIdx]), "%d", &rank)
		}

		effects := "{}"
		if effectsIdx >= 0 && effectsIdx < len(row) {
			effVal := strings.TrimSpace(row[effectsIdx])
			if effVal != "" {
				effects = effVal
			}
		}

		desc := ""
		if descIdx >= 0 && descIdx < len(row) {
			desc = strings.TrimSpace(row[descIdx])
		}

		basePrice := 0
		if basePriceIdx >= 0 && basePriceIdx < len(row) {
			fmt.Sscanf(strings.TrimSpace(row[basePriceIdx]), "%d", &basePrice)
		}

		imageURL := ""
		if imageURLIdx >= 0 && imageURLIdx < len(row) {
			imageURL = strings.TrimSpace(row[imageURLIdx])
		}

		_, err = DB.Exec(`
			INSERT INTO item_dictionary (id, name, item_type, rank, effects, description, base_price, image_url)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE
			SET name=EXCLUDED.name, item_type=EXCLUDED.item_type, rank=EXCLUDED.rank,
			    effects=EXCLUDED.effects, description=EXCLUDED.description,
			    base_price=CASE WHEN EXCLUDED.base_price > 0 THEN EXCLUDED.base_price ELSE item_dictionary.base_price END,
			    image_url=CASE WHEN EXCLUDED.image_url <> '' THEN EXCLUDED.image_url ELSE item_dictionary.image_url END
		`, id, name, itemType, rank, effects, desc, basePrice, imageURL)
		if err != nil {
			return fmt.Errorf("upsert item %s: %w", id, err)
		}
	}
	logger.S.Infof("Item dictionary seeded from %s", filename)
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
	foodIdx := col("food_types")
	maxHungerIdx := col("max_hunger")
	descIdx := col("description")
	pathIdx := col("path")
	manaCostIdx := col("base_mana_cost")
	effectsIdx := col("effects")
	rarityIdx := col("rarity")
	isImmortalIdx := col("is_immortal")
	primaryRoleIdx := col("primary_role")
	combatRolesIdx := col("combat_roles")
	combatTagsIdx := col("combat_tags")
	supportTagsIdx := col("support_tags")
	basePriceIdx := col("base_price")
	imageURLIdx := col("image_url")

	if idIdx < 0 || nameIdx < 0 || rankIdx < 0 {
		return fmt.Errorf("Gu.csv missing required columns (id, name, rank)")
	}

	type guRow struct{ fields []string }
	type guKey struct{ id, rank string }
	seen := make(map[guKey]guRow)
	order := []guKey{}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(row) == 0 {
			continue
		}
		id := strings.TrimSpace(row[idIdx])
		if id == "" {
			continue
		}
		rank := strings.TrimSpace(row[rankIdx])
		key := guKey{id, rank}
		if _, exists := seen[key]; !exists {
			order = append(order, key)
		}
		seen[key] = guRow{fields: row}
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	for _, key := range order {
		row := seen[key].fields
		id := key.id
		rankStr := key.rank
		name := get(row, nameIdx)
		guType := get(row, typeIdx)
		hungerStr := get(row, hungerIdx)
		foodType := get(row, foodIdx)
		maxHungerStr := get(row, maxHungerIdx)
		desc := get(row, descIdx)
		path := get(row, pathIdx)
		manaCostStr := get(row, manaCostIdx)
		effectsRaw := get(row, effectsIdx)
		rarity := get(row, rarityIdx)
		isImmortalRaw := strings.ToLower(get(row, isImmortalIdx))
		primaryRole := get(row, primaryRoleIdx)
		combatRolesRaw := get(row, combatRolesIdx)
		combatTagsRaw := get(row, combatTagsIdx)
		supportTagsRaw := get(row, supportTagsIdx)
		basePriceStr := get(row, basePriceIdx)
		basePrice := 0
		fmt.Sscanf(basePriceStr, "%d", &basePrice)

		imageURL := get(row, imageURLIdx)

		rank := 1
		fmt.Sscanf(rankStr, "%d", &rank)
		hungerRate := 0
		fmt.Sscanf(hungerStr, "%d", &hungerRate)
		maxHunger := 100
		fmt.Sscanf(maxHungerStr, "%d", &maxHunger)
		manaCost := 0.0
		fmt.Sscanf(manaCostStr, "%f", &manaCost)

		pathJSON := "[]"
		if path != "" {
			if strings.HasPrefix(path, "[") && strings.HasSuffix(path, "]") {
				pathJSON = safeJSONArray(path, "[]", fmt.Sprintf("%s path", id))
			} else {
				pathJSON = fmt.Sprintf(`["%s"]`, path)
			}
		} else {
			pathJSON = `["Moon Path"]`
		}
		effectsRaw = normalizedGuEffectsJSON(effectsRaw, fmt.Sprintf("%s effects", id))
		foodTypesJSON := "[]"
		if foodType != "" && foodType != "none" && foodType != "unknown" {
			if strings.HasPrefix(foodType, "[") && strings.HasSuffix(foodType, "]") {
				foodTypesJSON = safeJSONArray(foodType, "[]", fmt.Sprintf("%s food_types", id))
			} else {
				foodTypesJSON = fmt.Sprintf(`["%s"]`, foodType)
			}
		}
		combatRolesJSON := safeJSONArray(combatRolesRaw, "[]", fmt.Sprintf("%s combat_roles", id))
		combatTagsJSON := safeJSONArray(combatTagsRaw, "[]", fmt.Sprintf("%s combat_tags", id))
		supportTagsJSON := safeJSONArray(supportTagsRaw, "[]", fmt.Sprintf("%s support_tags", id))
		isImmortal := isImmortalRaw == "true" || isImmortalRaw == "1" || isImmortalRaw == "yes"

		var effEnvelope map[string]map[string]interface{}
		_ = json.Unmarshal([]byte(effectsRaw), &effEnvelope)
		effMap := map[string]interface{}{}
		for _, values := range effEnvelope {
			for key, value := range values {
				effMap[key] = value
			}
		}

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
			INSERT INTO gu_dictionary (id, name, rank, path, gu_type, base_mana_cost, effects, hunger_rate, food_types, max_hunger, description, target_type, rarity, is_immortal, primary_role, combat_roles, combat_tags, support_tags, base_price, image_url)
			VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7::jsonb, $8, $9::jsonb, $10, $11, $12, $13, $14, $15, $16::jsonb, $17::jsonb, $18::jsonb, $19, $20)
			ON CONFLICT (id, rank) DO UPDATE
			SET name=EXCLUDED.name, rank=EXCLUDED.rank, path=EXCLUDED.path, gu_type=EXCLUDED.gu_type,
			    base_mana_cost=EXCLUDED.base_mana_cost, effects=EXCLUDED.effects,
			    hunger_rate=EXCLUDED.hunger_rate, food_types=EXCLUDED.food_types, max_hunger=EXCLUDED.max_hunger, description=EXCLUDED.description,
			    target_type=EXCLUDED.target_type, rarity=EXCLUDED.rarity, is_immortal=EXCLUDED.is_immortal, primary_role=EXCLUDED.primary_role,
			    combat_roles=EXCLUDED.combat_roles, combat_tags=EXCLUDED.combat_tags, support_tags=EXCLUDED.support_tags,
			    base_price=CASE WHEN EXCLUDED.base_price > 0 THEN EXCLUDED.base_price ELSE gu_dictionary.base_price END,
			    image_url=CASE WHEN EXCLUDED.image_url <> '' THEN EXCLUDED.image_url ELSE gu_dictionary.image_url END
		`, id, name, rank, pathJSON, guType, manaCost, effectsRaw, hungerRate, foodTypesJSON, maxHunger, desc, targetType, rarity, isImmortal, primaryRole, combatRolesJSON, combatTagsJSON, supportTagsJSON, basePrice, imageURL)
		if err != nil {
			logger.S.Warnf("upsert gu %s failed: %v", id, err)
			continue
		}
	}
	logger.S.Infof("Gu dictionary seeded from %s (%d unique entries)", filename, len(order))
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
	targetRankIdx := col("target_rank")
	variantIdx := col("variant")
	reqGusIdx := col("required_gus")
	reqMatsIdx := col("required_materials")
	rateIdx := col("success_rate")
	stoneCostIdx := col("primeval_stones_cost")
	immortalCostIdx := col("immortal_essence_stones_cost")

	if targetIdx < 0 || reqGusIdx < 0 {
		return fmt.Errorf("Recipe.csv missing required columns (target_gu_id, required_gus)")
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	variantMap := make(map[string]int)
	seeded, skipped := 0, 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}
		targetGuID := get(row, targetIdx)
		if targetGuID == "" {
			continue
		}
		targetRankStr := get(row, targetRankIdx)
		variantStr := get(row, variantIdx)

		id := ""
		if idIdx >= 0 {
			id = get(row, idIdx)
		}
		if id == "" {
			id = fmt.Sprintf("%s_R%s_V%s", targetGuID, targetRankStr, variantStr)
		}
		reqGus := normalizeRecipeRequiredGus(get(row, reqGusIdx))
		reqMats := get(row, reqMatsIdx)
		if reqMats == "" {
			reqMats = "{}"
		}
		rateStr := get(row, rateIdx)
		stoneCostStr := get(row, stoneCostIdx)
		immortalCostStr := get(row, immortalCostIdx)

		targetRank := 1
		if targetRankStr != "" {
			fmt.Sscanf(targetRankStr, "%d", &targetRank)
		}

		variant := 0
		if variantStr != "" {
			fmt.Sscanf(variantStr, "%d", &variant)
		}

		if variant <= 0 {
			key := fmt.Sprintf("%s_%d", targetGuID, targetRank)
			variantMap[key]++
			variant = variantMap[key]
		}

		rate := 0.5
		fmt.Sscanf(rateStr, "%f", &rate)
		stoneCost := 0
		fmt.Sscanf(stoneCostStr, "%d", &stoneCost)
		immortalCost := 0.0
		fmt.Sscanf(immortalCostStr, "%f", &immortalCost)

		// Skip if target Gu doesn't exist (FK constraint)
		var exists bool
		if err = DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, targetGuID, targetRank).Scan(&exists); err != nil || !exists {
			logger.S.Warnf("Recipe %s skipped: target %s rank %d not in gu_dictionary", id, targetGuID, targetRank)
			skipped++
			continue
		}

		_, err = DB.Exec(`
			INSERT INTO refining_recipes (target_gu_id, target_rank, variant, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost)
			VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7, $8)
			ON CONFLICT (target_gu_id, target_rank, variant) DO UPDATE
			SET required_gus=EXCLUDED.required_gus,
			    required_materials=EXCLUDED.required_materials, success_rate=EXCLUDED.success_rate,
			    primeval_stones_cost=EXCLUDED.primeval_stones_cost, immortal_essence_stones_cost=EXCLUDED.immortal_essence_stones_cost
		`, targetGuID, targetRank, variant, reqGus, reqMats, rate, stoneCost, immortalCost)
		if err != nil {
			return fmt.Errorf("upsert recipe %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Recipes seeded from %s (%d seeded, %d skipped)", filename, seeded, skipped)
	return nil
}

// normalizeRecipeRequiredGus converts a required_gus JSON document into the
// ranked form {"GU_ID": {"rank": N, "qty": Q}}. Legacy numeric quantities
// ({"GU_ID": Q}) are upgraded to the material Gu's base (minimum) dictionary
// rank. Already-ranked object values are preserved. Unparseable input yields
// an empty object.
func normalizeRecipeRequiredGus(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rawMap); err != nil {
		return "{}"
	}
	out := make(map[string]map[string]int, len(rawMap))
	for k, v := range rawMap {
		trimmed := strings.TrimSpace(string(v))
		if len(trimmed) > 0 && trimmed[0] == '{' {
			var item struct {
				Rank int `json:"rank"`
				Qty  int `json:"qty"`
			}
			if err := json.Unmarshal([]byte(trimmed), &item); err != nil {
				continue
			}
			if item.Qty <= 0 {
				item.Qty = 1
			}
			out[k] = map[string]int{"rank": item.Rank, "qty": item.Qty}
			continue
		}
		qty := 0
		fmt.Sscanf(trimmed, "%d", &qty)
		if qty <= 0 {
			qty = 1
		}
		out[k] = map[string]int{"rank": baseGuRank(k), "qty": qty}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// baseGuRank returns the minimum dictionary rank for a Gu id, defaulting to 1
// when the Gu is absent from gu_dictionary.
func baseGuRank(guID string) int {
	if r, ok := baseRankCache[guID]; ok {
		return r
	}
	var r int
	if err := DB.QueryRow(`SELECT MIN(rank) FROM gu_dictionary WHERE id = $1`, guID).Scan(&r); err != nil || r <= 0 {
		r = 1
	}
	baseRankCache[guID] = r
	return r
}

// seedShopItemsFromCSV reads ShopItem.csv and upserts caravan shop inventory.
func seedShopItemsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	col := func(name string) int { return colIndex(header, name) }
	idIdx, priceIdx := col("id"), col("price")
	stockIdx, sortIdx, defaultIdx := col("stock"), col("sort_order"), col("is_default")
	if idIdx < 0 || priceIdx < 0 {
		return fmt.Errorf("ShopItem.csv missing required columns (id, price)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	seeded := 0
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
		price, stock, sortOrder := 0, -1, seeded
		if priceRaw := get(row, priceIdx); priceRaw != "" {
			fmt.Sscanf(get(row, priceIdx), "%d", &price)
		}
		if stockRaw := get(row, stockIdx); stockRaw != "" {
			fmt.Sscanf(stockRaw, "%d", &stock)
		}
		if sortRaw := get(row, sortIdx); sortRaw != "" {
			fmt.Sscanf(sortRaw, "%d", &sortOrder)
		}
		isDefault := true
		if defaultIdx >= 0 {
			defaultRaw := strings.ToLower(get(row, defaultIdx))
			if defaultRaw == "false" || defaultRaw == "0" || defaultRaw == "no" {
				isDefault = false
			}
		}
		_, err = DB.Exec(`
			INSERT INTO caravan_shop_items (id, price, stock, is_default, sort_order, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (id) DO UPDATE
			SET price=EXCLUDED.price, stock=EXCLUDED.stock, is_default=EXCLUDED.is_default,
			    sort_order=EXCLUDED.sort_order, updated_at=NOW()
		`, id, price, stock, isDefault, sortOrder)
		if err != nil {
			return fmt.Errorf("upsert shop item %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Shop items seeded from %s (%d seeded)", filename, seeded)
	return nil
}

// seedArenaShopItemsFromCSV reads ArenaShopItem.csv and upserts arena shop inventory.
func seedArenaShopItemsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	col := func(name string) int { return colIndex(header, name) }
	idIdx, priceIdx := col("id"), col("price")
	qtyIdx, stockIdx, sortIdx, defaultIdx := col("qty"), col("stock"), col("sort_order"), col("is_default")
	if idIdx < 0 || priceIdx < 0 {
		return fmt.Errorf("ArenaShopItem.csv missing required columns (id, price)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	seeded := 0
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
		price, qty, stock, sortOrder := 0, 1, -1, seeded
		if priceRaw := get(row, priceIdx); priceRaw != "" {
			fmt.Sscanf(get(row, priceIdx), "%d", &price)
		}
		if qtyRaw := get(row, qtyIdx); qtyRaw != "" {
			fmt.Sscanf(qtyRaw, "%d", &qty)
		}
		if stockRaw := get(row, stockIdx); stockRaw != "" {
			fmt.Sscanf(stockRaw, "%d", &stock)
		}
		if sortRaw := get(row, sortIdx); sortRaw != "" {
			fmt.Sscanf(sortRaw, "%d", &sortOrder)
		}
		isDefault := true
		if defaultIdx >= 0 {
			defaultRaw := strings.ToLower(get(row, defaultIdx))
			if defaultRaw == "false" || defaultRaw == "0" || defaultRaw == "no" {
				isDefault = false
			}
		}
		_, err = DB.Exec(`
			INSERT INTO arena_shop_items (id, price, qty, stock, is_default, sort_order, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (id) DO UPDATE
			SET price=EXCLUDED.price, qty=EXCLUDED.qty, stock=EXCLUDED.stock, is_default=EXCLUDED.is_default,
			    sort_order=EXCLUDED.sort_order, updated_at=NOW()
		`, id, price, qty, stock, isDefault, sortOrder)
		if err != nil {
			return fmt.Errorf("upsert arena shop item %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Arena shop items seeded from %s (%d seeded)", filename, seeded)
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func SeedMonstersAndSpawns() error {
	if err := seedRegionsFromCSV("Region.csv"); err != nil {
		return err
	}
	if err := seedLootTablesFromCSV("LootTable.csv"); err != nil {
		return err
	}
	if err := seedMonsterTemplatesFromCSV("Monster.csv"); err != nil {
		return err
	}
	if err := seedMonsterSpawnsFromCSV("MonsterSpawn.csv"); err != nil {
		return err
	}
	return nil
}

// seedQuestsFromCSV reads Quest.csv and upserts into quests table.
func seedQuestsFromCSV() error {
	f, r, header, err := openCSV("Quest.csv")
	if err != nil {
		return err
	}
	defer f.Close()

	idIdx := colIndex(header, "id")
	titleIdx := colIndex(header, "title")
	descIdx := colIndex(header, "desc")
	reqDescIdx := colIndex(header, "req_desc")
	msgIdx := colIndex(header, "message")
	rewardsIdx := colIndex(header, "rewards")

	if idIdx < 0 || titleIdx < 0 || descIdx < 0 {
		return fmt.Errorf("Quest.csv missing required columns (id, title, desc)")
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
		title := ""
		if titleIdx >= 0 && titleIdx < len(row) {
			title = strings.TrimSpace(row[titleIdx])
		}
		desc := ""
		if descIdx >= 0 && descIdx < len(row) {
			desc = strings.TrimSpace(row[descIdx])
		}
		reqDesc := ""
		if reqDescIdx >= 0 && reqDescIdx < len(row) {
			reqDesc = strings.TrimSpace(row[reqDescIdx])
		}
		message := ""
		if msgIdx >= 0 && msgIdx < len(row) {
			message = strings.TrimSpace(row[msgIdx])
		}
		rewards := "[]"
		if rewardsIdx >= 0 && rewardsIdx < len(row) {
			rVal := strings.TrimSpace(row[rewardsIdx])
			if rVal != "" {
				rewards = rVal
			}
		}

		_, err = DB.Exec(`
			INSERT INTO quests (id, title, description, req_desc, message, rewards)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb)
			ON CONFLICT (id) DO UPDATE
			SET title=EXCLUDED.title, description=EXCLUDED.description,
			    req_desc=EXCLUDED.req_desc, message=EXCLUDED.message,
			    rewards=EXCLUDED.rewards
		`, id, title, desc, reqDesc, message, rewards)
		if err != nil {
			return fmt.Errorf("upsert quest %s: %w", id, err)
		}
	}
	logger.S.Info("Quests seeded from Quest.csv")
	return nil
}

func seedRegionsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	idIdx, nameIdx := col("id"), col("name")
	descIdx, widthIdx, heightIdx, minRankIdx, maxRankIdx, activeIdx := col("description"), col("width"), col("height"), col("min_rank"), col("max_rank"), col("is_active")
	if idIdx < 0 || nameIdx < 0 {
		return fmt.Errorf("Region.csv missing required columns (id, name)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	seeded := 0
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
		width, height, minRank, maxRank := 100, 100, 1, 5
		if v := get(row, widthIdx); v != "" {
			fmt.Sscanf(v, "%d", &width)
		}
		if v := get(row, heightIdx); v != "" {
			fmt.Sscanf(v, "%d", &height)
		}
		if v := get(row, minRankIdx); v != "" {
			fmt.Sscanf(v, "%d", &minRank)
		}
		if v := get(row, maxRankIdx); v != "" {
			fmt.Sscanf(v, "%d", &maxRank)
		}
		active := true
		av := strings.ToLower(get(row, activeIdx))
		if av == "false" || av == "0" || av == "no" {
			active = false
		}
		_, err = DB.Exec(`
			INSERT INTO regions (id, name, description, width, height, min_rank, max_rank, is_active, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, description=EXCLUDED.description,
			width=EXCLUDED.width, height=EXCLUDED.height, min_rank=EXCLUDED.min_rank, max_rank=EXCLUDED.max_rank,
			is_active=EXCLUDED.is_active, updated_at=NOW()
		`, id, get(row, nameIdx), get(row, descIdx), width, height, minRank, maxRank, active)
		if err != nil {
			return fmt.Errorf("upsert region %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Regions seeded from %s (%d seeded)", filename, seeded)
	return nil
}

func seedLootTablesFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	idIdx, nameIdx, entriesIdx := col("id"), col("name"), col("entries")
	if idIdx < 0 || nameIdx < 0 || entriesIdx < 0 {
		return fmt.Errorf("LootTable.csv missing required columns (id, name, entries)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	seeded := 0
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
		entries := safeJSONArray(get(row, entriesIdx), "[]", fmt.Sprintf("loot table %s entries", id))
		_, err = DB.Exec(`INSERT INTO loot_tables (id, name, entries, updated_at) VALUES ($1,$2,$3::jsonb,NOW()) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, entries=EXCLUDED.entries, updated_at=NOW()`, id, get(row, nameIdx), entries)
		if err != nil {
			return fmt.Errorf("upsert loot table %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Loot tables seeded from %s (%d seeded)", filename, seeded)
	return nil
}

func seedMonsterTemplatesFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	idIdx, nameIdx, typeIdx := col("id"), col("name"), col("type")
	hpIdx, attackIdx, defenseIdx := col("max_hp"), col("base_attack"), col("base_defense")
	guIdx, itemIdx, daoIdx, masteryIdx := col("gu_inventory"), col("item_inventory"), col("dao_marks"), col("path_masteries")
	aiIdx, lootIdx, descIdx, activeIdx := col("ai_profile"), col("loot_table_id"), col("description"), col("is_active")
	capturableIdx, captureChanceIdx, hungerRateIdx := col("is_capturable"), col("capture_chance"), col("pet_hunger_per_hour")
	if idIdx < 0 || nameIdx < 0 || typeIdx < 0 || hpIdx < 0 || attackIdx < 0 || defenseIdx < 0 {
		return fmt.Errorf("Monster.csv missing required columns")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	seeded := 0
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
		maxHP, attack, defense := 0.0, 0.0, 0.0
		fmt.Sscanf(get(row, hpIdx), "%f", &maxHP)
		fmt.Sscanf(get(row, attackIdx), "%f", &attack)
		fmt.Sscanf(get(row, defenseIdx), "%f", &defense)
		ai := get(row, aiIdx)
		if ai == "" {
			ai = "beast_basic"
		}
		guInv := safeJSONArray(get(row, guIdx), "[]", fmt.Sprintf("monster %s gu_inventory", id))
		itemInv := safeJSONObject(get(row, itemIdx), "{}", fmt.Sprintf("monster %s item_inventory", id))
		daoMarks := safeJSONObject(get(row, daoIdx), "{}", fmt.Sprintf("monster %s dao_marks", id))
		masteries := safeJSONObject(get(row, masteryIdx), "{}", fmt.Sprintf("monster %s path_masteries", id))
		active := true
		av := strings.ToLower(get(row, activeIdx))
		if av == "false" || av == "0" || av == "no" {
			active = false
		}
		capturable := false
		cv := strings.ToLower(get(row, capturableIdx))
		if cv == "true" || cv == "1" || cv == "yes" {
			capturable = true
		}
		captureChance := 0.20
		if v := get(row, captureChanceIdx); v != "" {
			fmt.Sscanf(v, "%f", &captureChance)
		}
		petHungerRate := 2.0
		if v := get(row, hungerRateIdx); v != "" {
			fmt.Sscanf(v, "%f", &petHungerRate)
		}
		_, err = DB.Exec(`
			INSERT INTO monster_templates (id, name, type, max_hp, base_attack, base_defense, gu_inventory, item_inventory, dao_marks, path_masteries, ai_profile, loot_table_id, description, is_active, is_capturable, capture_chance, pet_hunger_per_hour, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9::jsonb,$10::jsonb,$11,$12,$13,$14,$15,$16,$17,NOW())
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, type=EXCLUDED.type,
			max_hp=EXCLUDED.max_hp, base_attack=EXCLUDED.base_attack, base_defense=EXCLUDED.base_defense,
			gu_inventory=EXCLUDED.gu_inventory, item_inventory=EXCLUDED.item_inventory, dao_marks=EXCLUDED.dao_marks, path_masteries=EXCLUDED.path_masteries,
			ai_profile=EXCLUDED.ai_profile, loot_table_id=EXCLUDED.loot_table_id, description=EXCLUDED.description, is_active=EXCLUDED.is_active,
			is_capturable=EXCLUDED.is_capturable, capture_chance=EXCLUDED.capture_chance, pet_hunger_per_hour=EXCLUDED.pet_hunger_per_hour, updated_at=NOW()
		`, id, get(row, nameIdx), get(row, typeIdx), maxHP, attack, defense, guInv, itemInv, daoMarks, masteries, ai, get(row, lootIdx), get(row, descIdx), active, capturable, captureChance, petHungerRate)
		if err != nil {
			return fmt.Errorf("upsert monster %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Monster templates seeded from %s (%d seeded)", filename, seeded)
	return nil
}

func seedMonsterSpawnsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	regionIdx, monsterIdx := col("region_id"), col("monster_id")
	xIdx, yIdx, modeIdx, chanceIdx := col("x"), col("y"), col("spawn_mode"), col("chance")
	respawnIdx, minRankIdx, maxRankIdx, bossIdx, activeIdx := col("respawn_seconds"), col("min_rank"), col("max_rank"), col("is_boss_spawn"), col("is_active")
	if regionIdx < 0 || monsterIdx < 0 {
		return fmt.Errorf("MonsterSpawn.csv missing required columns (region_id, monster_id)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	seeded := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}
		regionID, monsterID := get(row, regionIdx), get(row, monsterIdx)
		if regionID == "" || monsterID == "" {
			continue
		}
		var x, y interface{}
		if v := get(row, xIdx); v != "" {
			var n int
			fmt.Sscanf(v, "%d", &n)
			x = n
		}
		if v := get(row, yIdx); v != "" {
			var n int
			fmt.Sscanf(v, "%d", &n)
			y = n
		}
		mode, chance, respawn, minRank, maxRank := get(row, modeIdx), 1.0, 300, 1, 9
		if mode == "" {
			mode = "Random"
		}
		if v := get(row, chanceIdx); v != "" {
			fmt.Sscanf(v, "%f", &chance)
		}
		if v := get(row, respawnIdx); v != "" {
			fmt.Sscanf(v, "%d", &respawn)
		}
		if v := get(row, minRankIdx); v != "" {
			fmt.Sscanf(v, "%d", &minRank)
		}
		if v := get(row, maxRankIdx); v != "" {
			fmt.Sscanf(v, "%d", &maxRank)
		}
		boss := false
		bv := strings.ToLower(get(row, bossIdx))
		if bv == "true" || bv == "1" || bv == "yes" {
			boss = true
		}
		active := true
		av := strings.ToLower(get(row, activeIdx))
		if av == "false" || av == "0" || av == "no" {
			active = false
		}
		_, err = DB.Exec(`
			INSERT INTO monster_spawns (region_id, monster_id, x, y, spawn_mode, chance, respawn_seconds, min_rank, max_rank, is_boss_spawn, is_active)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT DO NOTHING
		`, regionID, monsterID, x, y, mode, chance, respawn, minRank, maxRank, boss, active)
		if err != nil {
			return fmt.Errorf("insert monster spawn %s/%s: %w", regionID, monsterID, err)
		}
		seeded++
	}
	logger.S.Infof("Monster spawns seeded from %s (%d read)", filename, seeded)
	return nil
}

func SeedInstanceTemplates() error {
	templates := []struct {
		ID          string
		Name        string
		Type        string
		ReqRank     int
		EntryItem   string
		Description string
		Rewards     string
		MapData     string
	}{
		{
			ID:          "dungeon_flower_wine",
			Name:        "Truyền Thừa Hoa Tửu Hành Giả",
			Type:        "INHERITANCE_DEMONIC",
			ReqRank:     1,
			EntryItem:   "NONE",
			Description: "Di tích động đá ẩn mật của Hoa Tửu Hành Giả lưu lại từ trăm năm trước, ẩn chứa cơ duyên Moonlight Gu và nguy cơ tử vong.",
			Rewards:     `{"materials": {"guts_gu": 2, "green_bamboo_wine": 1}, "primeval_stones": 100, "attainment": {"path": "Moon Path", "amount": 10}}`,
			MapData: `[
				{
					"stage_index": 0,
					"description": "Đạo hữu tiến vào mật thất tối tăm bên vách đá. Phía trước có hai cửa đá dẫn sâu vào lòng đất: cửa bên trái bốc lên mùi thơm trà nhẹ, cửa bên phải dán đầy tàn cốt.",
					"options": [
						{"text": "Đi sang ngả rẽ bên trái (Cửa Trà)", "next_stage": 1, "requirements": {}},
						{"text": "Đi sang ngả rẽ bên phải (Cửa Tàn Cốt)", "next_stage": 2, "requirements": {}}
					]
				},
				{
					"stage_index": 1,
					"description": "Một vách đá xám lớn đổ sụp chắn ngang lối đi của đạo hữu. Để phá vỡ tảng đá này, đạo hữu cần sức mạnh cơ thể lớn hoặc một lượng rượu xanh để làm mềm khe nứt.",
					"options": [
						{"text": "Dùng sức mạnh đập đá (Lệch hệ / Sát thương 20 HP)", "next_stage": 3, "requirements": {}},
						{"text": "Tiêu hao 1 bình Rượu Xanh (Green Bamboo Wine) để hòa tan vách đá", "next_stage": 3, "requirements": {"material_id": "green_bamboo_wine", "amount": 1}}
					]
				},
				{
					"stage_index": 2,
					"description": "Một đàn dã thú lao ra từ trong bóng tối tấn công đạo hữu! Đàn dã thú cắn xé cơ thể làm đạo hữu tổn thất nặng nề.",
					"options": [
						{"text": "Quay lại ngả rẽ bên trái", "next_stage": 1, "requirements": {}},
						{"text": "Liều mình chống trả (Mất 40 HP)", "next_stage": 3, "requirements": {}}
					]
				},
				{
					"stage_index": 3,
					"description": "Đạo hữu bước vào chính điện, trên bục đá là bộ xương khô của Hoa Tửu Hành Giả. Dưới đất là một hàng chữ khắc: 'Người kế nghiệp ta phải dập đầu lạy ta ba lạy!'",
					"options": [
						{"text": "Kính cẩn lạy bộ xương 3 lạy", "next_stage": 4, "requirements": {}},
						{"text": "Khám xét trực tiếp bộ xương khô để tìm bảo vật", "next_stage": 5, "requirements": {}}
					]
				},
				{
					"stage_index": 4,
					"description": "Bộ xương khô đột ngột sụp xuống, vách ngăn phía dưới mở ra hé lộ Moonlight Gu và rương Nguyên Thạch. Đạo hữu đã thành công kế thừa y bát Hoa Tửu Hành Giả!",
					"options": [
						{"text": "Thu nhận phần thưởng và kết thúc phó bản", "next_stage": -1, "requirements": {}}
					]
				},
				{
					"stage_index": 5,
					"description": "Hoa Tửu Hành Giả đã giấu độc phấn trên xương! Đạo hữu trúng độc tố chấn động cơ thể thất bại thảm hại.",
					"options": [
						{"text": "Chấp nhận thất bại và rời khỏi truyền thừa", "next_stage": -2, "requirements": {}}
					]
				}
			]`,
		},
		{
			ID:          "dream_realm_star_constellation",
			Name:        "Mộng Cảnh Tinh Túc Tiên Tôn",
			Type:        "DREAM_REALM",
			ReqRank:     5,
			EntryItem:   "NONE",
			Description: "Thám hiểm thời niên thiếu đầy thử thách của Tinh Túc Tiên Tôn khi nghiên cứu trận đồ giữa bầu trời tinh thần đại đạo.",
			Rewards:     `{"attainment": {"path": "Dream Path", "amount": 50}, "materials": {"guts_gu": 5}}`,
			MapData: `[
				{
					"stage_index": 0,
					"description": "Đạo hữu nhập vai vào tiểu đồng tử nghiên cứu tinh thần đại đạo. Mỗi hành động ở đây sẽ trừ 20 Soul HP. Phía trước tinh đồ bùng nổ, hãy chọn hướng nghiên cứu:",
					"options": [
						{"text": "Tìm hiểu Bắc Cực Tinh (Tiêu hao 20 Soul HP)", "next_stage": 1, "requirements": {}},
						{"text": "Bày bố trận nhãn Tinh đồ cơ bản (Tiêu hao 20 Soul HP)", "next_stage": 2, "requirements": {}}
					]
				},
				{
					"stage_index": 1,
					"description": "Nghiên cứu quá tải khiến Tinh linh phản phệ mạnh mẽ. Đạo hữu cần năng lượng linh hồn kiên cường hoặc Cổ Giải Mộng để hóa giải hiểm cảnh.",
					"options": [
						{"text": "Sử dụng lực lượng hồn phách trấn áp (Yêu cầu Soul HP >= 80)", "next_stage": 3, "requirements": {"min_soul_hp": 80}},
						{"text": "Sử dụng Cổ Giải Mộng (Dream Gu) skip qua ải hiểm nguy", "next_stage": 4, "requirements": {"gu_id": "GU_DREAM", "amount": 1}}
					]
				},
				{
					"stage_index": 2,
					"description": "Một đầu Hoang Thú ngưng tụ từ Tinh Thần Lực lao đến! Hãy chọn cách ứng biến nhanh:",
					"options": [
						{"text": "Sử dụng Trận Pháp Cổ của mộng cảnh phòng ngự (Mất 20 Soul HP)", "next_stage": 3, "requirements": {}},
						{"text": "Tấn công vật lý trực diện (Thất bại, tổn thất 50 Soul HP)", "next_stage": 1, "requirements": {}}
					]
				},
				{
					"stage_index": 3,
					"description": "Trải qua gian nan tinh đồ mở rộng, hình bóng Tinh Túc Tiên Tôn xuất hiện chỉ tay về vòm trời tinh quang truyền thừa Chân Ý.",
					"options": [
						{"text": "Nhập định hấp thu Tinh Thần Chân Ý (Tiêu hao 20 Soul HP)", "next_stage": 4, "requirements": {}}
					]
				},
				{
					"stage_index": 4,
					"description": "Tinh quang hòa tan linh hồn đạo hữu, nâng tầm cảnh giới thấu hiểu đại đạo tinh thần. Đạo hữu hoàn thành thám hiểm mộng cảnh!",
					"options": [
						{"text": "Hấp thụ Chân Ý và rời Mộng Cảnh", "next_stage": -1, "requirements": {}}
					]
				}
			]`,
		},
	}

	for _, t := range templates {
		_, err := DB.Exec(`
			INSERT INTO instance_templates (id, name, type, required_rank, entry_item, description, rewards, map_data)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, type=EXCLUDED.type, required_rank=EXCLUDED.required_rank,
				entry_item=EXCLUDED.entry_item, description=EXCLUDED.description,
				rewards=EXCLUDED.rewards, map_data=EXCLUDED.map_data
		`, t.ID, t.Name, t.Type, t.ReqRank, t.EntryItem, t.Description, t.Rewards, t.MapData)
		if err != nil {
			return fmt.Errorf("failed to seed instance %s: %w", t.ID, err)
		}
	}
	return nil
}

func SeedSects() error {
	if err := seedSectsFromCSV("Sect.csv"); err != nil {
		return err
	}
	if err := seedSectMissionsFromCSV("SectMission.csv"); err != nil {
		return err
	}
	return nil
}

func seedSectsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	idIdx, nameIdx := col("id"), col("name")
	descIdx, regionIdx := col("description"), col("region")
	vaultGuIdx, vaultMatIdx, vaultRecIdx := col("vault_gu"), col("vault_materials"), col("vault_recipes")
	if idIdx < 0 || nameIdx < 0 {
		return fmt.Errorf("Sect.csv missing required columns (id, name)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	seeded := 0
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
		vaultGu := safeJSONArray(get(row, vaultGuIdx), "[]", "Sect.csv:"+id+":vault_gu")
		vaultMat := safeJSONObject(get(row, vaultMatIdx), "{}", "Sect.csv:"+id+":vault_materials")
		vaultRec := safeJSONArray(get(row, vaultRecIdx), "[]", "Sect.csv:"+id+":vault_recipes")
		_, err = DB.Exec(`
			INSERT INTO sects (id, name, description, region, vault_gu, vault_materials, vault_recipes)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, description=EXCLUDED.description, region=EXCLUDED.region,
				vault_gu=EXCLUDED.vault_gu, vault_materials=EXCLUDED.vault_materials, vault_recipes=EXCLUDED.vault_recipes
		`, id, get(row, nameIdx), get(row, descIdx), get(row, regionIdx), vaultGu, vaultMat, vaultRec)
		if err != nil {
			return fmt.Errorf("failed to seed sect %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Seeded %d sects from %s", seeded, filename)
	return nil
}

func seedSectMissionsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	sectIdIdx, titleIdx := col("sect_id"), col("title")
	descIdx, typeIdx := col("description"), col("mission_type")
	targetIdIdx, targetAmtIdx := col("target_id"), col("target_amount")
	rewardIdx, rankIdx := col("reward_contribution"), col("required_member_rank")
	if sectIdIdx < 0 || titleIdx < 0 {
		return fmt.Errorf("SectMission.csv missing required columns (sect_id, title)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	getInt := func(row []string, idx int, def int) int {
		v := get(row, idx)
		if v == "" {
			return def
		}
		n := def
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	seeded := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}
		sectId := get(row, sectIdIdx)
		if sectId == "" {
			continue
		}
		_, err = DB.Exec(`
			INSERT INTO sect_missions (sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT DO NOTHING
		`, sectId, get(row, titleIdx), get(row, descIdx), get(row, typeIdx),
			get(row, targetIdIdx), getInt(row, targetAmtIdx, 1),
			getInt(row, rewardIdx, 0), get(row, rankIdx))
		if err != nil {
			return fmt.Errorf("failed to seed mission %s: %w", get(row, titleIdx), err)
		}
		seeded++
	}
	logger.S.Infof("Seeded %d sect missions from %s", seeded, filename)
	return nil
}

// CheckImmortalGuExists verifies if a specific Immortal Gu (id, rank) exists anywhere in the world,
// excluding instances whose InstanceID is in excludeInstanceIDs (being consumed in a refine).
func CheckImmortalGuExists(tx *sql.Tx, guID string, rank int, excludeInstanceIDs []string) (bool, error) {
	var exists bool

	// 1. Kiểm tra trong bảng phụ player_gu (toàn bộ cổ của mọi người chơi)
	err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM player_gu
			WHERE gu_id = $1
			  AND gu_rank = $2
			  AND NOT (id = ANY($3::text[]))
		)
	`, guID, rank, pq.Array(excludeInstanceIDs)).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// 2. Kiểm tra trên sàn giao dịch/đấu giá (auction)
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM auction,
			jsonb_array_elements(item_data) AS elem
			WHERE item_type = 'Gu'
			  AND status = 'ACTIVE'
			  AND elem->>'gu_id' = $1::text
			  AND COALESCE((elem->>'rank')::int, 0) = $2
		)
	`, guID, rank).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// 3. Kiểm tra trong rương tạm temp_loot
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM players,
			jsonb_array_elements(COALESCE(temp_loot->'gu_inventory', '[]'::jsonb)) AS elem
			WHERE elem->>'gu_id' = $1::text
			  AND COALESCE((elem->>'rank')::int, 0) = $2
			  AND NOT (elem->>'instance_id' = ANY($3::text[]))
		)
	`, guID, rank, pq.Array(excludeInstanceIDs)).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetExistingImmortalGuIDs returns the set of gu_ids that currently have at least one
// rank >= 6 (Immortal Gu) instance anywhere in the world: any player's player_gu,
// ACTIVE auction listings, and temp_loot.
func GetExistingImmortalGuIDs(tx *sql.Tx) (map[string]bool, error) {
	rows, err := tx.Query(`
		SELECT DISTINCT gu_id FROM player_gu WHERE gu_rank >= 6
		UNION
		SELECT DISTINCT elem->>'gu_id' FROM auction,
			jsonb_array_elements(item_data) AS elem
			WHERE item_type = 'Gu' AND status = 'ACTIVE' AND COALESCE((elem->>'rank')::int, 0) >= 6
		UNION
		SELECT DISTINCT elem->>'gu_id' FROM players,
			jsonb_array_elements(COALESCE(temp_loot->'gu_inventory', '[]'::jsonb)) AS elem
			WHERE COALESCE((elem->>'rank')::int, 0) >= 6
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var guID string
		if err := rows.Scan(&guID); err == nil {
			existing[guID] = true
		}
	}
	return existing, rows.Err()
}

func seedNPCs() error {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM npcs").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		npcs := []struct {
			id              string
			name            string
			sectID          string
			role            string
			aptitudePercent int
		}{
			{"npc_disciple_1", "Thạch Hạo (Đệ tử)", "gu_yue_clan", "DISCIPLE", 65},
			{"npc_disciple_2", "Lâm Động (Đệ tử)", "gu_yue_clan", "DISCIPLE", 55},
			{"npc_disciple_3", "Tiêu Viêm (Đệ tử)", "gu_yue_clan", "DISCIPLE", 70},
			{"npc_clansman_1", "Cổ Nguyệt Đạt (Tộc nhân)", "gu_yue_clan", "CLANSMAN", 45},
			{"npc_clansman_2", "Cổ Nguyệt Đạo (Tộc nhân)", "gu_yue_clan", "CLANSMAN", 50},
		}
		for _, npc := range npcs {
			_, err = DB.Exec(`
				INSERT INTO npcs (id, name, sect_id, role, aptitude_percent)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (id) DO NOTHING
			`, npc.id, npc.name, npc.sectID, npc.role, npc.aptitudePercent)
			if err != nil {
				return err
			}
		}
		logger.S.Info("Seeded initial NPCs for Huyết Sọ Cổ hiến tế.")
	}
	return nil
}

// writeCSV is a helper that runs a SQL query and writes the result as a CSV file.
func writeCSV(filename string, columns []string, query string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create %s: %w", filename, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(columns); err != nil {
		return fmt.Errorf("write header %s: %w", filename, err)
	}

	rows, err := DB.Query(query)
	if err != nil {
		return fmt.Errorf("query %s: %w", filename, err)
	}
	defer rows.Close()

	for rows.Next() {
		vals := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan %s: %w", filename, err)
		}
		row := make([]string, len(columns))
		for i, v := range vals {
			if v == nil {
				row[i] = ""
			} else {
				switch val := v.(type) {
				case []byte:
					row[i] = string(val)
				case string:
					row[i] = val
				default:
					row[i] = fmt.Sprint(val)
				}
			}
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("write row %s: %w", filename, err)
		}
	}
	return w.Error()
}

// ensureDataDir creates the data directory if it doesn't exist.
func ensureDataDir() {
	if err := os.MkdirAll("data", 0755); err != nil {
		logger.S.Warnf("failed to create data directory: %v", err)
	}
	for _, d := range []string{"storage/images/gu", "storage/images/item"} {
		if err := os.MkdirAll(d, 0755); err != nil {
			logger.S.Warnf("failed to create %s: %v", d, err)
		}
	}
}

// ExportToCSV exports all dictionary tables to CSV files in the data/ directory.
func ExportToCSV() error {
	type exportDef struct {
		filename string
		columns  []string
		query    string
	}
	ensureDataDir()
	exports := []exportDef{
		{
			filename: "Item.csv",
			columns:  []string{"id", "name", "item_type", "rank", "effects", "description", "base_price", "image_url"},
			query:    `SELECT id, name, item_type, rank, COALESCE(effects::text,'{}'), COALESCE(description,''), COALESCE(base_price,0), COALESCE(image_url,'') FROM item_dictionary ORDER BY id`,
		},
		{
			filename: "Gu.csv",
			columns:  []string{"id", "name", "rank", "gu_type", "hunger_rate", "food_types", "max_hunger", "description", "path", "base_mana_cost", "effects", "rarity", "is_immortal", "primary_role", "combat_roles", "combat_tags", "support_tags", "base_price", "image_url"},
			query:    `SELECT id, name, rank, gu_type, COALESCE(hunger_rate,0), COALESCE(food_types::text,'[]'), COALESCE(max_hunger,100), COALESCE(description,''), COALESCE(path::text,'[]'), COALESCE(base_mana_cost,0), COALESCE(effects::text,'{"active":{},"support":{},"passive":{},"house":{}}'), COALESCE(rarity,''), COALESCE(is_immortal,false), COALESCE(primary_role,''), COALESCE(combat_roles::text,'[]'), COALESCE(combat_tags::text,'[]'), COALESCE(support_tags::text,'[]'), COALESCE(base_price,0), COALESCE(image_url,'') FROM gu_dictionary ORDER BY id`,
		},
		{
			filename: "Recipe.csv",
			columns:  []string{"id", "target_gu_id", "target_rank", "variant", "required_gus", "required_materials", "success_rate", "primeval_stones_cost", "immortal_essence_stones_cost"},
			query:    `SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) AS id, target_gu_id, target_rank, variant, COALESCE(required_gus::text,'[]'), COALESCE(required_materials::text,'{}'), COALESCE(success_rate,0), COALESCE(primeval_stones_cost,0), COALESCE(immortal_essence_stones_cost,0) FROM refining_recipes ORDER BY target_gu_id, target_rank, variant`,
		},
		{
			filename: "ShopItem.csv",
			columns:  []string{"id", "price", "stock", "is_default", "sort_order"},
			query:    `SELECT id, price, COALESCE(stock,-1), is_default, COALESCE(sort_order,0) FROM caravan_shop_items ORDER BY id`,
		},
		{
			filename: "ArenaShopItem.csv",
			columns:  []string{"id", "price", "qty", "stock", "is_default", "sort_order"},
			query:    `SELECT id, price, COALESCE(qty,1), COALESCE(stock,-1), is_default, COALESCE(sort_order,0) FROM arena_shop_items ORDER BY id`,
		},
		{
			filename: "Region.csv",
			columns:  []string{"id", "name", "description", "width", "height", "min_rank", "max_rank", "is_active"},
			query:    `SELECT id, name, COALESCE(description,''), COALESCE(width,100), COALESCE(height,100), COALESCE(min_rank,1), COALESCE(max_rank,5), COALESCE(is_active,true) FROM regions ORDER BY id`,
		},
		{
			filename: "LootTable.csv",
			columns:  []string{"id", "name", "entries"},
			query:    `SELECT id, name, COALESCE(entries::text,'[]') FROM loot_tables ORDER BY id`,
		},
		{
			filename: "Monster.csv",
			columns:  []string{"id", "name", "type", "max_hp", "base_attack", "base_defense", "gu_inventory", "item_inventory", "dao_marks", "path_masteries", "ai_profile", "loot_table_id", "description", "is_active"},
			query:    `SELECT id, name, type, max_hp, base_attack, base_defense, COALESCE(gu_inventory::text,'[]'), COALESCE(item_inventory::text,'{}'), COALESCE(dao_marks::text,'{}'), COALESCE(path_masteries::text,'{}'), COALESCE(ai_profile,'beast_basic'), COALESCE(loot_table_id,''), COALESCE(description,''), COALESCE(is_active,true) FROM monster_templates ORDER BY id`,
		},
		{
			filename: "MonsterSpawn.csv",
			columns:  []string{"region_id", "monster_id", "x", "y", "spawn_mode", "chance", "respawn_seconds", "min_rank", "max_rank", "is_boss_spawn", "is_active"},
			query:    `SELECT region_id, monster_id, x, y, COALESCE(spawn_mode,'Random'), COALESCE(chance,1.0), COALESCE(respawn_seconds,300), COALESCE(min_rank,1), COALESCE(max_rank,9), COALESCE(is_boss_spawn,false), COALESCE(is_active,true) FROM monster_spawns ORDER BY region_id, monster_id`,
		},
		{
			filename: "Sect.csv",
			columns:  []string{"id", "name", "description", "region", "vault_gu", "vault_materials", "vault_recipes"},
			query:    `SELECT id, name, COALESCE(description,''), COALESCE(region,''), COALESCE(vault_gu::text,'[]'), COALESCE(vault_materials::text,'{}'), COALESCE(vault_recipes::text,'[]') FROM sects ORDER BY id`,
		},
		{
			filename: "SectMission.csv",
			columns:  []string{"sect_id", "title", "description", "mission_type", "target_id", "target_amount", "reward_contribution", "required_member_rank"},
			query:    `SELECT sect_id, title, COALESCE(description,''), COALESCE(mission_type,'GATHER'), COALESCE(target_id,''), COALESCE(target_amount,1), COALESCE(reward_contribution,0), COALESCE(required_member_rank,'NGOAI_MON') FROM sect_missions ORDER BY sect_id, title`,
		},
		{
			filename: "Quest.csv",
			columns:  []string{"id", "title", "desc", "req_desc", "message", "rewards"},
			query:    `SELECT id, title, description, req_desc, message, COALESCE(rewards::text,'[]') FROM quests ORDER BY id`,
		},
	}

	for _, e := range exports {
		p := filepath.Join("data", e.filename)
		if err := writeCSV(p, e.columns, e.query); err != nil {
			return fmt.Errorf("export %s: %w", p, err)
		}
		logger.S.Infof("Exported %s", p)
	}
	logger.S.Info("All CSV data exported successfully")
	return nil
}

// AvailableTables returns the list of table keys that can be individually imported/exported.
func AvailableTables() []string {
	return []string{
		"item", "gu", "recipe", "shop_item", "arena_shop_item",
		"region", "loot_table", "monster", "monster_spawn",
		"sect", "sect_mission", "quest",
	}
}

// TableImport imports a single table from its CSV file.
func TableImport(table string) error {
	switch table {
	case "item":
		return seedItemsFromCSV("Item.csv")
	case "gu":
		return seedGusFromCSV("Gu.csv")
	case "recipe":
		return seedRecipesFromCSV("Recipe.csv")
	case "shop_item":
		return seedShopItemsFromCSV("ShopItem.csv")
	case "arena_shop_item":
		return seedArenaShopItemsFromCSV("ArenaShopItem.csv")
	case "region":
		return seedRegionsFromCSV("Region.csv")
	case "loot_table":
		return seedLootTablesFromCSV("LootTable.csv")
	case "monster":
		return seedMonsterTemplatesFromCSV("Monster.csv")
	case "monster_spawn":
		return seedMonsterSpawnsFromCSV("MonsterSpawn.csv")
	case "sect":
		return seedSectsFromCSV("Sect.csv")
	case "sect_mission":
		return seedSectMissionsFromCSV("SectMission.csv")
	case "quest":
		return seedQuestsFromCSV()
	default:
		return fmt.Errorf("unknown table: %s", table)
	}
}

// TableExport exports a single table to its CSV file.
func TableExport(table string) error {
	ensureDataDir()
	type exportDef struct {
		filename string
		columns  []string
		query    string
	}
	defs := map[string]exportDef{
		"item": {
			filename: "Item.csv",
			columns:  []string{"id", "name", "item_type", "rank", "effects", "description", "base_price", "image_url"},
			query:    `SELECT id, name, item_type, rank, COALESCE(effects::text,'{}'), COALESCE(description,''), COALESCE(base_price,0), COALESCE(image_url,'') FROM item_dictionary ORDER BY id`,
		},
		"gu": {
			filename: "Gu.csv",
			columns:  []string{"id", "name", "rank", "gu_type", "hunger_rate", "food_types", "max_hunger", "description", "path", "base_mana_cost", "effects", "rarity", "is_immortal", "primary_role", "combat_roles", "combat_tags", "support_tags", "base_price", "image_url"},
			query:    `SELECT id, name, rank, gu_type, COALESCE(hunger_rate,0), COALESCE(food_types::text,'[]'), COALESCE(max_hunger,100), COALESCE(description,''), COALESCE(path::text,'[]'), COALESCE(base_mana_cost,0), COALESCE(effects::text,'{"active":{},"support":{},"passive":{},"house":{}}'), COALESCE(rarity,''), COALESCE(is_immortal,false), COALESCE(primary_role,''), COALESCE(combat_roles::text,'[]'), COALESCE(combat_tags::text,'[]'), COALESCE(support_tags::text,'[]'), COALESCE(base_price,0), COALESCE(image_url,'') FROM gu_dictionary ORDER BY id`,
		},
		"recipe": {
			filename: "Recipe.csv",
			columns:  []string{"id", "target_gu_id", "target_rank", "variant", "required_gus", "required_materials", "success_rate", "primeval_stones_cost", "immortal_essence_stones_cost"},
			query:    `SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) AS id, target_gu_id, target_rank, variant, COALESCE(required_gus::text,'[]'), COALESCE(required_materials::text,'{}'), COALESCE(success_rate,0), COALESCE(primeval_stones_cost,0), COALESCE(immortal_essence_stones_cost,0) FROM refining_recipes ORDER BY target_gu_id, target_rank, variant`,
		},
		"shop_item": {
			filename: "ShopItem.csv",
			columns:  []string{"id", "price", "stock", "is_default", "sort_order"},
			query:    `SELECT id, price, COALESCE(stock,-1), is_default, COALESCE(sort_order,0) FROM caravan_shop_items ORDER BY id`,
		},
		"arena_shop_item": {
			filename: "ArenaShopItem.csv",
			columns:  []string{"id", "price", "qty", "stock", "is_default", "sort_order"},
			query:    `SELECT id, price, COALESCE(qty,1), COALESCE(stock,-1), is_default, COALESCE(sort_order,0) FROM arena_shop_items ORDER BY id`,
		},
		"region": {
			filename: "Region.csv",
			columns:  []string{"id", "name", "description", "width", "height", "min_rank", "max_rank", "is_active"},
			query:    `SELECT id, name, COALESCE(description,''), COALESCE(width,100), COALESCE(height,100), COALESCE(min_rank,1), COALESCE(max_rank,5), COALESCE(is_active,true) FROM regions ORDER BY id`,
		},
		"loot_table": {
			filename: "LootTable.csv",
			columns:  []string{"id", "name", "entries"},
			query:    `SELECT id, name, COALESCE(entries::text,'[]') FROM loot_tables ORDER BY id`,
		},
		"monster": {
			filename: "Monster.csv",
			columns:  []string{"id", "name", "type", "max_hp", "base_attack", "base_defense", "gu_inventory", "item_inventory", "dao_marks", "path_masteries", "ai_profile", "loot_table_id", "description", "is_active"},
			query:    `SELECT id, name, type, max_hp, base_attack, base_defense, COALESCE(gu_inventory::text,'[]'), COALESCE(item_inventory::text,'{}'), COALESCE(dao_marks::text,'{}'), COALESCE(path_masteries::text,'{}'), COALESCE(ai_profile,'beast_basic'), COALESCE(loot_table_id,''), COALESCE(description,''), COALESCE(is_active,true) FROM monster_templates ORDER BY id`,
		},
		"monster_spawn": {
			filename: "MonsterSpawn.csv",
			columns:  []string{"region_id", "monster_id", "x", "y", "spawn_mode", "chance", "respawn_seconds", "min_rank", "max_rank", "is_boss_spawn", "is_active"},
			query:    `SELECT region_id, monster_id, x, y, COALESCE(spawn_mode,'Random'), COALESCE(chance,1.0), COALESCE(respawn_seconds,300), COALESCE(min_rank,1), COALESCE(max_rank,9), COALESCE(is_boss_spawn,false), COALESCE(is_active,true) FROM monster_spawns ORDER BY region_id, monster_id`,
		},
		"sect": {
			filename: "Sect.csv",
			columns:  []string{"id", "name", "description", "region", "vault_gu", "vault_materials", "vault_recipes"},
			query:    `SELECT id, name, COALESCE(description,''), COALESCE(region,''), COALESCE(vault_gu::text,'[]'), COALESCE(vault_materials::text,'{}'), COALESCE(vault_recipes::text,'[]') FROM sects ORDER BY id`,
		},
		"sect_mission": {
			filename: "SectMission.csv",
			columns:  []string{"sect_id", "title", "description", "mission_type", "target_id", "target_amount", "reward_contribution", "required_member_rank"},
			query:    `SELECT sect_id, title, COALESCE(description,''), COALESCE(mission_type,'GATHER'), COALESCE(target_id,''), COALESCE(target_amount,1), COALESCE(reward_contribution,0), COALESCE(required_member_rank,'NGOAI_MON') FROM sect_missions ORDER BY sect_id, title`,
		},
		"quest": {
			filename: "Quest.csv",
			columns:  []string{"id", "title", "desc", "req_desc", "message", "rewards"},
			query:    `SELECT id, title, description, req_desc, message, COALESCE(rewards::text,'[]') FROM quests ORDER BY id`,
		},
	}

	d, ok := defs[table]
	if !ok {
		return fmt.Errorf("unknown table: %s", table)
	}
	p := filepath.Join("data", d.filename)
	if err := writeCSV(p, d.columns, d.query); err != nil {
		return fmt.Errorf("export %s: %w", p, err)
	}
	logger.S.Infof("Exported %s", p)
	return nil
}
