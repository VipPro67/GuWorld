package db

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var DB *sql.DB

type csvImportExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// CSV imports can be pointed at a transaction while retaining the existing
// parsers. The mutex also prevents one admin import from changing the source
// file/executor used by another concurrent import.
var csvImportState struct {
	sync.Mutex
	executor csvImportExecutor
	override map[string]string
}

func activeCSVImportDB() csvImportExecutor {
	if csvImportState.executor != nil {
		return csvImportState.executor
	}
	return DB
}

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

	DB.SetMaxOpenConns(50)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(10 * time.Minute)
	DB.SetConnMaxIdleTime(1 * time.Minute)

	logger.S.Info("Successfully connected to the database")
	return nil
}

// InitSchema creates all tables, indexes and foreign keys (defined per domain
// in schema_*.go), then seeds static runtime data: the reserved system players,
// default race/constitution dictionaries, the hub region, world mine slots and
// sect mines. Gameplay dictionaries (Item, Gu, Recipe, ...) are NOT auto-seeded
// here — import them via the Admin panel "Dữ liệu" tab.
func InitSchema() error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	if _, err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`); err != nil {
		logger.S.Warnf("failed to create uuid-ossp extension: %v", err)
	}
	if _, err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS "unaccent";`); err != nil {
		logger.S.Warnf("failed to create unaccent extension: %v", err)
	}
	if _, err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS "pg_trgm";`); err != nil {
		logger.S.Warnf("failed to create pg_trgm extension: %v", err)
	}
	// Immutable wrapper around unaccent so it can be used in index expressions.
	// unaccent() itself is STABLE and cannot be indexed directly.
	if _, err := DB.Exec(`
	CREATE OR REPLACE FUNCTION public.immutable_unaccent(text)
	RETURNS text
	LANGUAGE sql
	IMMUTABLE
	PARALLEL SAFE
	STRICT
	AS $func$
		SELECT public.unaccent('public.unaccent', $1)
	$func$;`); err != nil {
		logger.S.Warnf("failed to create immutable_unaccent function: %v", err)
	}

	// Tables are declared per domain in schema_*.go, in foreign-key dependency
	// order. All CREATE statements are idempotent (IF NOT EXISTS).
	steps := []struct {
		name string
		fn   func() error
	}{
		{"dictionaries", schemaDictionaries},
		{"world", schemaWorld},
		{"parties", schemaParties},
		{"players", schemaPlayers},
		{"admin_roles", schemaAdminRoles},
		{"authorization", schemaAuthorization},
		{"player_extras", schemaPlayerExtras},
		{"sects", schemaSects},
		{"player_items", schemaPlayerItems},
		{"aperture", schemaAperture},
		{"effect_catalog_migration", schemaEffectCatalogMigration},
		{"combat", schemaCombat},
		{"trade", schemaTrade},
		{"social", schemaSocial},
		{"payment", schemaPayment},
		{"beast_betting", schemaBeastBetting},
		{"death_arena", schemaDeathArena},
		{"gu_history", schemaGuHistory},
		{"teleport", schemaTeleport},
		{"scouting", schemaScouting},
	}
	for _, s := range steps {
		if err := s.fn(); err != nil {
			return fmt.Errorf("schema step %q: %w", s.name, err)
		}
	}
	if err := installPlayerCollectionRevisionTriggers(); err != nil {
		return fmt.Errorf("install player collection revision triggers: %w", err)
	}

	// Static runtime seeds (independent of CSV imports).
	if err := seedDefaultRaceConstitutionRows(); err != nil {
		return fmt.Errorf("seed default race/constitution rows: %w", err)
	}
	if err := seedDefaultStatusRows(); err != nil {
		return fmt.Errorf("seed default status rows: %w", err)
	}
	if err := seedHubRegion(); err != nil {
		return fmt.Errorf("seed hub region: %w", err)
	}
	if err := seedSystemPlayers(); err != nil {
		return fmt.Errorf("seed system players: %w", err)
	}
	if err := seedMineSlots(); err != nil {
		return fmt.Errorf("seed mine slots: %w", err)
	}
	// if err := seedSectMines(); err != nil {
	// 	return fmt.Errorf("seed sect mines: %w", err)
	// }
	if err := backfillWallets(); err != nil {
		return fmt.Errorf("backfill wallets: %w", err)
	}
	if err := backfillLeaderboardSettings(); err != nil {
		return fmt.Errorf("backfill leaderboard settings: %w", err)
	}
	if err := ensureAdmin(); err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}
	if err := ensureBootstrapAdminRole(); err != nil {
		return fmt.Errorf("ensure bootstrap admin role: %w", err)
	}
	if err := backfillAdminRoles(); err != nil {
		return fmt.Errorf("backfill admin roles: %w", err)
	}
	if err := seedQuestsFromCSV(); err != nil {
		logger.S.Warnf("seedQuestsFromCSV: %v", err)
	}

	// Clean up any stale active_combat records for non-leader party members
	_, _ = DB.Exec(`UPDATE players SET active_combat = NULL WHERE party_id IS NOT NULL AND id NOT IN (SELECT leader_id FROM parties WHERE leader_id IS NOT NULL)`)

	logger.S.Info("Database schema initialized successfully")
	return nil
}

// execSchema runs a single DDL statement, wrapping errors with the schema name.
func execSchema(name, sql string) error {
	if _, err := DB.Exec(sql); err != nil {
		return fmt.Errorf("failed to create %s: %w", name, err)
	}
	return nil
}

// seedHubRegion seeds the default spawn region so the players.current_region FK
// (default 'Southern_Border') is satisfied on a fresh database before the admin
// imports Region.csv. The CSV import overwrites/extends this row.
func seedHubRegion() error {
	if _, err := DB.Exec(`
	INSERT INTO regions (id, name, description, width, height, min_rank, max_rank, is_active, zone_type, min_x, min_y, max_x, max_y, entry_x, entry_y, pool_weights, updated_at)
	VALUES ('hub_nam_cuong', 'Nam Cương Thành', 'Thành trì an toàn giữa thập vạn đại sơn, nơi các gia tộc Nam Cương giao thương và nghỉ chân.', 201, 201, 1, 9, true, 'HUB', 4900, 4900, 5100, 5100, 5000, 5000, '{"cell_type_weights":{"Empty":100}}', NOW())
	ON CONFLICT (id) DO NOTHING;`); err != nil {
		return err
	}
	return nil
}

// seedSystemPlayers seeds the two reserved system accounts used by the
// global/world/sect chat channels and automated notifications.
func seedSystemPlayers() error {
	if _, err := DB.Exec(`
	INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, immortal_stones, hp, base_hp, current_region, x, y, email, ref_code, is_admin)
	VALUES ('00000000-0000-0000-0000-000000000000', 'Bảo Hoàng Thiên', '', 6, 1, 'Giáp', 99, 100, 100, 0, 0, 100, 100, 'hub_nam_cuong', 50, 50, 'system@guworld.local', 'REF-SYSTEM', true)
	ON CONFLICT (id) DO NOTHING;`); err != nil {
		return err
	}
	if _, err := DB.Exec(`
	INSERT INTO players (id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, immortal_stones, hp, base_hp, current_region, x, y, email, ref_code, is_admin)
	VALUES ('11111111-1111-1111-1111-111111111111', 'Hệ thống', '', 6, 1, 'Giáp', 99, 100, 100, 0, 0, 100, 100, 'hub_nam_cuong', 50, 50, 'system_npc@guworld.local', 'REF-SYSTEM-NPC', true)
	ON CONFLICT (id) DO NOTHING;`); err != nil {
		return err
	}
	return nil
}

// seedMineSlots seeds the 31 world mine slots (1 Siêu Cấp, 3 Truyền Thuyết,
// 9 Sử Thi, 18 Hiếm) when the table is empty.
func seedMineSlots() error {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM mine_slots`).Scan(&count); err != nil {
		return err
	}
	if count >= 62 {
		return nil
	}
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
		if _, err := DB.Exec("INSERT INTO mine_slots (id, rarity, abandoned_at, mine_type) VALUES ($1, $2, NOW(), 'mortal') ON CONFLICT (id) DO NOTHING", i, slotRarities[i]); err != nil {
			logger.S.Errorf("Failed to seed mortal mine %d: %v", i, err)
		}
	}
	for i := 101; i <= 131; i++ {
		idx := i - 100
		if _, err := DB.Exec("INSERT INTO mine_slots (id, rarity, abandoned_at, mine_type) VALUES ($1, $2, NOW(), 'immortal') ON CONFLICT (id) DO NOTHING", i, slotRarities[idx]); err != nil {
			logger.S.Errorf("Failed to seed immortal mine %d: %v", i, err)
		}
	}
	logger.S.Info("Seeded world mine slots.")
	return nil
}

// seedSectMines seeds the 5 fixed sect mines.
// func seedSectMines() error {
// 	if _, err := DB.Exec(`
// 	INSERT INTO sect_mines (name, rarity) VALUES
// 	('Thái Cổ Tinh Mạch', 'Truyền Thuyết'),
// 	('Cửu U Huyết Mạch', 'Sử Thi'),
// 	('Thiên Cương Linh Khí Mạch', 'Hiếm'),
// 	('Huyền Vũ Thạch Động', 'Hiếm'),
// 	('Bách Thảo Tiên Vực', 'Thường')
// 	ON CONFLICT (name) DO NOTHING;`); err != nil {
// 		return err
// 	}
// 	return nil
// }

// backfillWallets mirrors each player's stone balances into the wallet table.
func backfillWallets() error {
	if _, err := DB.Exec(`
	INSERT INTO wallets (user_id, immortal_jade)
	SELECT id, 0
	FROM players
	ON CONFLICT (user_id) DO NOTHING;`); err != nil {
		return err
	}
	return nil
}

// backfillLeaderboardSettings ensures every player has a leaderboard row.
func backfillLeaderboardSettings() error {
	if _, err := DB.Exec(`
	INSERT INTO leaderboard_settings (player_id, show_on_leaderboard)
	SELECT id, TRUE FROM players
	ON CONFLICT (player_id) DO NOTHING;`); err != nil {
		return err
	}
	return nil
}

// ensureAdmin promotes the reserved admin accounts when no admin exists yet.
func ensureAdmin() error {
	var adminCount int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM players WHERE username = 'admin'`).Scan(&adminCount); err != nil {
		return nil
	}
	if adminCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("1231237618989@Ahihi"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		if _, err := DB.Exec(`
			INSERT INTO players (
				id, username, password_hash, email, rank, sub_rank, 
				aptitude_type, aptitude_percent, current_mana, max_mana, 
				hp, base_hp, current_region, x, y, ref_code, is_admin,
				primeval_stones, immortal_stones
			) VALUES (
				gen_random_uuid(), 'admin', $1, 'admin@guworld.local', 6, 1, 
				'Giáp', 99, 100, 100, 
				100, 100, 'hub_nam_cuong', 50, 50, 'REF-ADMIN', true,
				0, 0
			)`, string(hash)); err != nil {
			return err
		}
	} else {
		if _, err := DB.Exec(`UPDATE players SET is_admin = true WHERE username = 'admin'`); err != nil {
			return err
		}
	}
	return nil
}

// SeedDictionaries populates gameplay master data from CSV files.
func SeedDictionaries() error {
	if err := preflightEffectCSVs([]effectCSVSource{{"item", "Item.csv", "id"}, {"gu", "Gu.csv", "id"}, {"constitution", "Constitution.csv", "key"}, {"race", "Race.csv", "key"}, {"status", "Status.csv", "key"}}); err != nil {
		return err
	}
	if err := seedItemsFromCSV("Item.csv"); err != nil {
		return fmt.Errorf("CSV item seed failed: %w", err)
	}
	if err := seedGusFromCSV("Gu.csv"); err != nil {
		return fmt.Errorf("CSV Gu seed failed: %w", err)

	}
	if err := seedConstitutionsFromCSV("Constitution.csv"); err != nil {
		return fmt.Errorf("CSV constitution seed failed: %w", err)
	}
	if err := seedRacesFromCSV("Race.csv"); err != nil {
		return fmt.Errorf("CSV race seed failed: %w", err)
	}
	if err := seedStatusesFromCSV("Status.csv"); err != nil {
		return fmt.Errorf("CSV status seed failed: %w", err)
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
		filepath.Join("backend", "data", filename),
		filepath.Join("cmd", "server", "data", filename),
		filepath.Join("backend", "cmd", "server", "data", filename),
		filepath.Join("..", "data", filename),
		filepath.Join("..", "..", "data", filename),
		filepath.Join("..", "cmd", "server", "data", filename),
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
	if override := csvImportState.override[filename]; override != "" {
		filename = override
	}
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

func safeJSON(raw string, fallback string, rowLabel string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		logger.S.Warnf("invalid JSON in %s: %v", rowLabel, err)
		return fallback
	}
	return value
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

func normalizedGuEffectsJSON(raw string, rowLabel string) (string, effectcatalog.Report, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" {
		return "[]", effectcatalog.Report{}, nil
	}
	if trimmed[0] == '[' {
		issues := effectcatalog.ValidateBlockV2([]byte(trimmed), effectcatalog.Location{Source: "gu", Key: rowLabel})
		if len(issues) > 0 {
			return "", effectcatalog.Report{}, &effectcatalog.ValidationError{Issues: issues}
		}
		return trimmed, effectcatalog.Report{}, nil
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &envelope) == nil {
		if _, ok := envelope["schema_version"]; ok || envelope["blocks"] != nil {
			issues := effectcatalog.ValidateBlockV2([]byte(raw), effectcatalog.Location{Source: "gu", Key: rowLabel})
			if len(issues) > 0 {
				return "", effectcatalog.Report{}, &effectcatalog.ValidationError{Issues: issues}
			}
			return raw, effectcatalog.Report{}, nil
		}
	}
	normalized, report, err := effectcatalog.NormalizeAndValidate([]byte(raw))
	if err != nil {
		return "", report, fmt.Errorf("%s: %w", rowLabel, err)
	}
	return string(normalized), report, nil
}

type effectCSVSource struct{ source, filename, keyColumn string }

// preflightEffectCSVs completes validation for the whole requested batch
// before any seeder can issue its first INSERT/UPDATE.
func preflightEffectCSVs(sources []effectCSVSource) error {
	all := make([]effectcatalog.ValidationIssue, 0)
	for _, source := range sources {
		f, reader, header, err := openCSV(source.filename)
		if err != nil {
			return err
		}
		keyIdx, effectsIdx := colIndex(header, source.keyColumn), colIndex(header, "effects")
		if keyIdx < 0 || effectsIdx < 0 {
			f.Close()
			return fmt.Errorf("%s missing required columns (%s, effects)", source.filename, source.keyColumn)
		}
		rowNumber := 1
		for {
			row, err := reader.Read()
			rowNumber++
			if err == io.EOF {
				break
			}
			if err != nil {
				f.Close()
				return fmt.Errorf("%s row %d: %w", source.filename, rowNumber, err)
			}
			if len(row) == 0 {
				continue
			}
			cell := func(index int) string {
				if index < 0 || index >= len(row) {
					return ""
				}
				return row[index]
			}
			key := strings.TrimSpace(cell(keyIdx))
			raw := strings.TrimSpace(cell(effectsIdx))
			if raw == "" {
				raw = `[]`
			}
			all = append(all, effectcatalog.ValidateBlockV2([]byte(raw), effectcatalog.Location{Source: source.source, Row: rowNumber, Key: key})...)
		}
		f.Close()
	}
	if len(all) > 0 {
		return &effectcatalog.ValidationError{Issues: all}
	}
	return nil
}

// parseRarity parses the rarity column (1-9 tier, empty/unset = 0). Non-numeric
// legacy values (e.g. stray booleans or word tiers) are coerced to 0.
func parseRarity(raw string) int {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0
	}
	n := 0
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	if n < 0 {
		return 0
	}
	if n > 9 {
		return 9
	}
	return n
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
	rarityIdx := colIndex(header, "rarity")
	pathIdx := colIndex(header, "path")

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

		effects := "[]"
		if effectsIdx >= 0 && effectsIdx < len(row) {
			effVal := strings.TrimSpace(row[effectsIdx])
			if effVal != "" {
				effects = safeJSON(effVal, "[]", fmt.Sprintf("item %s effects", id))
			}
		}

		desc := ""
		if descIdx >= 0 && descIdx < len(row) {
			desc = strings.TrimSpace(row[descIdx])
		}

		basePrice := 0.0
		if basePriceIdx >= 0 && basePriceIdx < len(row) {
			fmt.Sscanf(strings.TrimSpace(row[basePriceIdx]), "%f", &basePrice)
		}

		imageURL := ""
		if imageURLIdx >= 0 && imageURLIdx < len(row) {
			imageURL = strings.TrimSpace(row[imageURLIdx])
		}

		rarity := 0
		if rarityIdx >= 0 && rarityIdx < len(row) {
			rarity = parseRarity(strings.TrimSpace(row[rarityIdx]))
		}

		path := ""
		if pathIdx >= 0 && pathIdx < len(row) {
			path = strings.TrimSpace(row[pathIdx])
		}

		_, err = activeCSVImportDB().Exec(`
			INSERT INTO item_dictionary (id, name, item_type, rank, effects, description, base_price, image_url, rarity, path)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO UPDATE
			SET name=EXCLUDED.name, item_type=EXCLUDED.item_type, rank=EXCLUDED.rank,
			    effects=EXCLUDED.effects, description=EXCLUDED.description,
			    base_price=CASE WHEN EXCLUDED.base_price > 0 THEN EXCLUDED.base_price ELSE item_dictionary.base_price END,
			    image_url=CASE WHEN EXCLUDED.image_url <> '' THEN EXCLUDED.image_url ELSE item_dictionary.image_url END,
			    rarity=EXCLUDED.rarity, path=EXCLUDED.path
		`, id, name, itemType, rank, effects, desc, basePrice, imageURL, rarity, path)
		if err != nil {
			return fmt.Errorf("upsert item %s: %w", id, err)
		}
	}
	logger.S.Infof("Item dictionary seeded from %s", filename)
	return nil
}

// countRows helper - we need to count rows read

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
	targetTypeIdx := col("target_type")

	if idIdx < 0 || nameIdx < 0 || rankIdx < 0 {
		return fmt.Errorf("Gu.csv missing required columns (id, name, rank)")
	}

	type guRow struct {
		fields            []string
		normalizedEffects string
		line              int
	}
	type guKey struct{ id, rank string }
	seen := make(map[guKey]guRow)
	order := []guKey{}

	line := 1
	for {
		line++
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Gu.csv line %d: %w", line, err)
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
		seen[key] = guRow{fields: row, line: line}
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	// Validate the complete effect catalog before the first database write. A
	// malformed row must not turn a bulk import into a partially-updated
	// dictionary. Warnings retain legacy/unwired fields for later migration.
	warningCount := 0
	for _, key := range order {
		entry := seen[key]
		effectsRaw := get(entry.fields, effectsIdx)
		normalized, report, err := normalizedGuEffectsJSON(effectsRaw, fmt.Sprintf("Gu.csv line %d, %s rank %s effects", entry.line, key.id, key.rank))
		if err != nil {
			return err
		}
		entry.normalizedEffects = normalized
		seen[key] = entry
		warningCount += len(report.Warnings())
	}
	if warningCount > 0 {
		logger.S.Warnf("Gu effect catalog contains %d legacy/unwired fields; values were preserved", warningCount)
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
		effectsRaw := seen[key].normalizedEffects
		rarity := parseRarity(get(row, rarityIdx))
		isImmortalRaw := strings.ToLower(get(row, isImmortalIdx))
		primaryRole := get(row, primaryRoleIdx)
		combatRolesRaw := get(row, combatRolesIdx)
		combatTagsRaw := get(row, combatTagsIdx)
		supportTagsRaw := get(row, supportTagsIdx)
		basePriceStr := get(row, basePriceIdx)
		basePrice := 0.0
		fmt.Sscanf(basePriceStr, "%f", &basePrice)

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
			pathJSON = `["Star Path"]`
		}
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

		targetType := strings.ToUpper(get(row, targetTypeIdx))
		if targetType == "" && effMap != nil {
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
		if targetType == "" {
			targetType = "SINGLE"
		}

		_, err := activeCSVImportDB().Exec(`
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

// seedConstitutionsFromCSV reads Constitution.csv and upserts into
// constitution_dictionary.
func seedConstitutionsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	keyIdx := colIndex(header, "key")
	nameIdx := colIndex(header, "name")
	effectsIdx := colIndex(header, "effects")
	tierIdx := colIndex(header, "tier")
	if keyIdx < 0 || nameIdx < 0 || effectsIdx < 0 {
		return fmt.Errorf("%s missing required columns (key, name, effects)", filename)
	}

	seen := make(map[string][]string)
	order := []string{}
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
		key := strings.TrimSpace(row[keyIdx])
		if key == "" {
			continue
		}
		if _, exists := seen[key]; !exists {
			order = append(order, key)
		}
		seen[key] = row
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	for _, key := range order {
		row := seen[key]
		name := get(row, nameIdx)
		effects := safeJSON(get(row, effectsIdx), "[]", fmt.Sprintf("%s effects", key))
		tierStr := get(row, tierIdx)
		tierVal := 0
		if tierStr != "" {
			if v, err := strconv.Atoi(tierStr); err == nil {
				tierVal = v
			}
		}
		_, err := activeCSVImportDB().Exec(`
		INSERT INTO constitution_dictionary (key, name, effects, tier)
		VALUES ($1, $2, $3::jsonb, $4)
		ON CONFLICT (key) DO UPDATE
		SET name=EXCLUDED.name, effects=EXCLUDED.effects, tier=EXCLUDED.tier
		`, key, name, effects, tierVal)
		if err != nil {
			return fmt.Errorf("upsert constitution %s: %w", key, err)
		}
	}
	logger.S.Infof("Constitution dictionary seeded from %s (%d entries)", filename, len(order))
	return nil
}

// seedRacesFromCSV reads Race.csv and upserts into race_dictionary.
func seedRacesFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	keyIdx := colIndex(header, "key")
	nameIdx := colIndex(header, "name")
	effectsIdx := colIndex(header, "effects")
	if keyIdx < 0 || nameIdx < 0 || effectsIdx < 0 {
		return fmt.Errorf("%s missing required columns (key, name, effects)", filename)
	}

	seen := make(map[string][]string)
	order := []string{}
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
		key := strings.TrimSpace(row[keyIdx])
		if key == "" {
			continue
		}
		if _, exists := seen[key]; !exists {
			order = append(order, key)
		}
		seen[key] = row
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	for _, key := range order {
		row := seen[key]
		name := get(row, nameIdx)
		effects := safeJSON(get(row, effectsIdx), "[]", fmt.Sprintf("%s effects", key))
		_, err := activeCSVImportDB().Exec(`
			INSERT INTO race_dictionary (key, name, effects)
			VALUES ($1, $2, $3::jsonb)
			ON CONFLICT (key) DO UPDATE
			SET name=EXCLUDED.name, effects=EXCLUDED.effects
		`, key, name, effects)
		if err != nil {
			return fmt.Errorf("upsert race %s: %w", key, err)
		}
	}
	logger.S.Infof("Race dictionary seeded from %s (%d entries)", filename, len(order))
	return nil
}

// seedStatusesFromCSV reads Status.csv and upserts into status_dictionary.
func seedStatusesFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	keyIdx := colIndex(header, "key")
	nameIdx := colIndex(header, "name")
	effectsIdx := colIndex(header, "effects")
	durationIdx := colIndex(header, "default_duration_minutes")
	blockedIdx := colIndex(header, "blocked_actions")
	descIdx := colIndex(header, "description")
	if keyIdx < 0 || nameIdx < 0 {
		return fmt.Errorf("%s missing required columns (key, name)", filename)
	}

	seen := make(map[string][]string)
	order := []string{}
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
		key := strings.TrimSpace(row[keyIdx])
		if key == "" {
			continue
		}
		if _, exists := seen[key]; !exists {
			order = append(order, key)
		}
		seen[key] = row
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	for _, key := range order {
		row := seen[key]
		name := get(row, nameIdx)
		effects := safeJSON(get(row, effectsIdx), "[]", fmt.Sprintf("%s effects", key))
		blocked := safeJSONArray(get(row, blockedIdx), "[]", fmt.Sprintf("%s blocked_actions", key))
		duration := 0
		if d := get(row, durationIdx); d != "" {
			fmt.Sscanf(d, "%d", &duration)
		}
		description := get(row, descIdx)
		_, err := activeCSVImportDB().Exec(`
			INSERT INTO status_dictionary (key, name, effects, default_duration_minutes, blocked_actions, description)
			VALUES ($1, $2, $3::jsonb, $4, $5::jsonb, $6)
			ON CONFLICT (key) DO UPDATE
			SET name=EXCLUDED.name, effects=EXCLUDED.effects,
			    default_duration_minutes=EXCLUDED.default_duration_minutes,
			    blocked_actions=EXCLUDED.blocked_actions, description=EXCLUDED.description
		`, key, name, effects, duration, blocked, description)
		if err != nil {
			return fmt.Errorf("upsert status %s: %w", key, err)
		}
	}
	logger.S.Infof("Status dictionary seeded from %s (%d entries)", filename, len(order))
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
		if err = activeCSVImportDB().QueryRow(`SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, targetGuID, targetRank).Scan(&exists); err != nil || !exists {
			logger.S.Warnf("Recipe %s skipped: target %s rank %d not in gu_dictionary", id, targetGuID, targetRank)
			skipped++
			continue
		}

		_, err = activeCSVImportDB().Exec(`
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
	if err := activeCSVImportDB().QueryRow(`SELECT MIN(rank) FROM gu_dictionary WHERE id = $1`, guID).Scan(&r); err != nil || r <= 0 {
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
	rankIdx, stockIdx, sortIdx, defaultIdx := col("rank"), col("stock"), col("sort_order"), col("is_default")
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
		rank, price, stock, sortOrder := 0, 0, -1, seeded
		if rankRaw := get(row, rankIdx); rankRaw != "" {
			fmt.Sscanf(rankRaw, "%d", &rank)
		}
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
		_, err = activeCSVImportDB().Exec(`
			INSERT INTO caravan_shop_items (id, rank, price, stock, is_default, sort_order, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (id, rank) DO UPDATE
			SET price=EXCLUDED.price, stock=EXCLUDED.stock, is_default=EXCLUDED.is_default,
			    sort_order=EXCLUDED.sort_order, updated_at=NOW()
		`, id, rank, price, stock, isDefault, sortOrder)
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
		_, err = activeCSVImportDB().Exec(`
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

		_, err = activeCSVImportDB().Exec(`
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
	parentIdx, typeIdx := col("parent_id"), col("zone_type")
	minXIdx, minYIdx, maxXIdx, maxYIdx, entryXIdx, entryYIdx, poolIdx, safeXIdx, safeYIdx := col("min_x"), col("min_y"), col("max_x"), col("max_y"), col("entry_x"), col("entry_y"), col("pool_weights"), col("safe_zone_x"), col("safe_zone_y")
	if idIdx < 0 || nameIdx < 0 {
		return fmt.Errorf("Region.csv missing required columns (id, name)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	parseInt := func(row []string, idx int, def int) int {
		if v := get(row, idx); v != "" {
			var n int
			fmt.Sscanf(v, "%d", &n)
			return n
		}
		return def
	}
	type regionRow struct {
		id, name, desc, zoneType, poolWeights string
		width, height, minRank, maxRank       int
		active                                bool
		parentID                              interface{}
		minX, minY, maxX, maxY                int
		entryX, entryY, safeX, safeY          int
	}

	var rowsToInsert []regionRow

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
		width, height := parseInt(row, widthIdx, 100), parseInt(row, heightIdx, 100)
		minRank, maxRank := parseInt(row, minRankIdx, 1), parseInt(row, maxRankIdx, 5)
		active := true
		av := strings.ToLower(get(row, activeIdx))
		if av == "false" || av == "0" || av == "no" {
			active = false
		}
		zoneType := get(row, typeIdx)
		if zoneType == "" {
			zoneType = "KHU_VUC"
		}
		minX, minY := parseInt(row, minXIdx, 0), parseInt(row, minYIdx, 0)
		maxX, maxY := parseInt(row, maxXIdx, width-1), parseInt(row, maxYIdx, height-1)
		entryX, entryY := parseInt(row, entryXIdx, (minX+maxX)/2), parseInt(row, entryYIdx, (minY+maxY)/2)
		safeX := parseInt(row, safeXIdx, entryX)
		safeY := parseInt(row, safeYIdx, entryY)
		poolWeights := "{}"
		if v := get(row, poolIdx); v != "" {
			poolWeights = filterImmortalWildGus(v)
		}
		var parentID interface{}
		if p := get(row, parentIdx); p != "" {
			parentID = p
		}
		rowsToInsert = append(rowsToInsert, regionRow{
			id: id, name: get(row, nameIdx), desc: get(row, descIdx),
			width: width, height: height, minRank: minRank, maxRank: maxRank,
			active: active, parentID: parentID, zoneType: zoneType,
			minX: minX, minY: minY, maxX: maxX, maxY: maxY,
			entryX: entryX, entryY: entryY, safeX: safeX, safeY: safeY,
			poolWeights: poolWeights,
		})
	}

	// Pass 1: Upsert all rows with parent_id = NULL to ensure all region IDs exist in table
	seeded := 0
	for _, item := range rowsToInsert {
		_, err = activeCSVImportDB().Exec(`
			INSERT INTO regions (id, name, description, width, height, min_rank, max_rank, is_active, parent_id, zone_type, min_x, min_y, max_x, max_y, entry_x, entry_y, safe_zone_x, safe_zone_y, pool_weights, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULL,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::jsonb,NOW())
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, description=EXCLUDED.description,
			width=EXCLUDED.width, height=EXCLUDED.height, min_rank=EXCLUDED.min_rank, max_rank=EXCLUDED.max_rank,
			is_active=EXCLUDED.is_active, zone_type=EXCLUDED.zone_type,
			min_x=EXCLUDED.min_x, min_y=EXCLUDED.min_y, max_x=EXCLUDED.max_x, max_y=EXCLUDED.max_y,
			entry_x=EXCLUDED.entry_x, entry_y=EXCLUDED.entry_y, safe_zone_x=EXCLUDED.safe_zone_x, safe_zone_y=EXCLUDED.safe_zone_y, pool_weights=EXCLUDED.pool_weights, updated_at=NOW()
		`, item.id, item.name, item.desc, item.width, item.height, item.minRank, item.maxRank, item.active,
			item.zoneType, item.minX, item.minY, item.maxX, item.maxY, item.entryX, item.entryY, item.safeX, item.safeY, item.poolWeights)
		if err != nil {
			return fmt.Errorf("upsert region %s (pass 1): %w", item.id, err)
		}
		seeded++
	}

	// Pass 2: Update parent_id for rows that specify a parent
	for _, item := range rowsToInsert {
		if item.parentID != nil {
			_, err = activeCSVImportDB().Exec(`UPDATE regions SET parent_id = $1 WHERE id = $2`, item.parentID, item.id)
			if err != nil {
				return fmt.Errorf("update region parent_id %s -> %v: %w", item.id, item.parentID, err)
			}
		}
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
		entries = FilterImmortalLootEntries(entries)
		_, err = activeCSVImportDB().Exec(`INSERT INTO loot_tables (id, name, entries, updated_at) VALUES ($1,$2,$3::jsonb,NOW()) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, entries=EXCLUDED.entries, updated_at=NOW()`, id, get(row, nameIdx), entries)
		if err != nil {
			return fmt.Errorf("upsert loot table %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Loot tables seeded from %s (%d seeded)", filename, seeded)
	return nil
}

// FilterImmortalLootEntries removes loot entries that would grant a rank >= 6
// (Tiên Cổ) gu from an imported loot table entries JSON, since immortal Gus
// must only be obtained from gacha/refinement/auction, never from loot.
func FilterImmortalLootEntries(entries string) string {
	var list []models.LootEntry
	if err := json.Unmarshal([]byte(entries), &list); err != nil {
		return entries
	}
	hasImmortal := false
	for _, e := range list {
		if e.GuID != "" && models.HasImmortalRank(e.GuID) {
			hasImmortal = true
			break
		}
	}
	if !hasImmortal {
		return entries
	}
	kept := list[:0]
	for _, e := range list {
		if e.GuID != "" && models.HasImmortalRank(e.GuID) {
			continue
		}
		kept = append(kept, e)
	}
	out, err := json.Marshal(kept)
	if err != nil {
		return entries
	}
	return string(out)
}

// filterImmortalWildGus removes rank >= 6 (Tiên Cổ) gu ids from an imported
// region pool_weights JSON, since immortal Gus must never spawn as wild Gus.
func filterImmortalWildGus(poolWeights string) string {
	var pools models.ZonePools
	if err := json.Unmarshal([]byte(poolWeights), &pools); err != nil {
		return poolWeights
	}
	hasImmortal := false
	for _, id := range pools.WildGus {
		if models.HasImmortalRank(id) {
			hasImmortal = true
			break
		}
	}
	if !hasImmortal {
		return poolWeights
	}
	kept := pools.WildGus[:0]
	for _, id := range pools.WildGus {
		if models.HasImmortalRank(id) {
			continue
		}
		kept = append(kept, id)
	}
	pools.WildGus = kept
	out, err := json.Marshal(pools)
	if err != nil {
		return poolWeights
	}
	return string(out)
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
		// loot_table_id FKs loot_tables(id); normalize '' to NULL so monsters
		// exported with an empty loot table (COALESCE('')) can be re-imported.
		var lootTableID interface{}
		if v := strings.TrimSpace(get(row, lootIdx)); v != "" {
			lootTableID = v
		}
		_, err = activeCSVImportDB().Exec(`
			INSERT INTO monster_templates (id, name, type, max_hp, base_attack, base_defense, gu_inventory, item_inventory, dao_marks, path_masteries, ai_profile, loot_table_id, description, is_active, is_capturable, capture_chance, pet_hunger_per_hour, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9::jsonb,$10::jsonb,$11,$12,$13,$14,$15,$16,$17,NOW())
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, type=EXCLUDED.type,
			max_hp=EXCLUDED.max_hp, base_attack=EXCLUDED.base_attack, base_defense=EXCLUDED.base_defense,
			gu_inventory=EXCLUDED.gu_inventory, item_inventory=EXCLUDED.item_inventory, dao_marks=EXCLUDED.dao_marks, path_masteries=EXCLUDED.path_masteries,
			ai_profile=EXCLUDED.ai_profile, loot_table_id=EXCLUDED.loot_table_id, description=EXCLUDED.description, is_active=EXCLUDED.is_active,
			is_capturable=EXCLUDED.is_capturable, capture_chance=EXCLUDED.capture_chance, pet_hunger_per_hour=EXCLUDED.pet_hunger_per_hour, updated_at=NOW()
		`, id, get(row, nameIdx), get(row, typeIdx), maxHP, attack, defense, guInv, itemInv, daoMarks, masteries, ai, lootTableID, get(row, descIdx), active, capturable, captureChance, petHungerRate)
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
		_, err = activeCSVImportDB().Exec(`
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

// seedMapCellsFromCSV reads MapCell.csv and upserts persistent shared cells
// (bosses, high-tier resources, safe points) into map_cells.
func seedMapCellsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	regionIdx, xIdx, yIdx := col("region_id"), col("x"), col("y")
	typeIdx, contentIdx, metaIdx, respawnIdx, activeIdx := col("cell_type"), col("content_id"), col("metadata"), col("respawn_seconds"), col("is_active")
	if regionIdx < 0 || xIdx < 0 || yIdx < 0 || typeIdx < 0 {
		return fmt.Errorf("MapCell.csv missing required columns (region_id, x, y, cell_type)")
	}
	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	parseInt := func(row []string, idx int, def int) int {
		if v := get(row, idx); v != "" {
			var n int
			fmt.Sscanf(v, "%d", &n)
			return n
		}
		return def
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
		x, y := parseInt(row, xIdx, -1), parseInt(row, yIdx, -1)
		cellType := get(row, typeIdx)
		if cellType == "" || x < 0 || y < 0 {
			continue
		}
		meta := get(row, metaIdx)
		if meta == "" {
			meta = "{}"
		}
		respawn := parseInt(row, respawnIdx, 300)
		active := true
		av := strings.ToLower(get(row, activeIdx))
		if av == "false" || av == "0" || av == "no" {
			active = false
		}
		_, err = activeCSVImportDB().Exec(`
			INSERT INTO map_cells (region_id, x, y, cell_type, content_id, metadata, respawn_seconds, is_active)
			VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8)
			ON CONFLICT (x, y) DO UPDATE SET region_id=EXCLUDED.region_id, cell_type=EXCLUDED.cell_type,
			content_id=EXCLUDED.content_id, metadata=EXCLUDED.metadata, respawn_seconds=EXCLUDED.respawn_seconds,
			is_active=EXCLUDED.is_active
		`, get(row, regionIdx), x, y, cellType, get(row, contentIdx), meta, respawn, active)
		if err != nil {
			return fmt.Errorf("upsert map cell %d,%d: %w", x, y, err)
		}
		seeded++
	}
	logger.S.Infof("Map cells seeded from %s (%d seeded)", filename, seeded)
	return nil
}

// dreamRealmPaths lists the 49 lưu phái (path) canonical keys with their
// Vietnamese display names. Keys must match the lowercase path keys used in
// players.path_masteries and Gu.csv (see docs/CANH_GIOI_DAO_NGAN.md).
var dreamRealmPaths = []struct {
	Key string // lowercase path key, e.g. "fire"
	VN  string // Vietnamese display name, e.g. "Hỏa Đạo"
}{
	// Ngũ hành & Tự nhiên
	{"metal", "Kim Đạo"},
	{"wood", "Mộc Đạo"},
	{"water", "Thủy Đạo"},
	{"fire", "Hỏa Đạo"},
	{"earth", "Thổ Đạo"},
	{"wind", "Phong Đạo"},
	{"lightning", "Lôi Đạo"},
	{"ice", "Băng Tuyết Đạo"},
	{"cloud", "Vân Đạo"},
	{"light", "Quang Đạo"},
	{"dark", "Ám Đạo"},
	{"star", "Tinh Đạo"},
	{"sound", "Âm Đạo"},

	// Không - Thời gian & Vũ trụ
	{"space", "Vũ Đạo"},
	{"time", "Trụ Đạo"},
	{"heaven", "Thiên Đạo"},

	// Quy tắc & Kỹ nghệ
	{"rule", "Luật Đạo"},
	{"refinement", "Luyện Đạo"},
	{"formation", "Trận Đạo"},
	{"qi", "Khí Đạo"},

	// Cơ thể, Tinh thần & Xã hội
	{"strength", "Lực Đạo"},
	{"soul", "Hồn Đạo"},
	{"wisdom", "Trí Đạo"},
	{"emotion", "Tình Đạo"},
	{"transformation", "Biến Hóa Đạo"},
	{"enslavement", "Nô Đạo"},
	{"information", "Tín Đạo"},
	{"blood", "Huyết Đạo"},
	{"poison", "Độc Đạo"},
	{"bone", "Cốt Đạo"},
	{"human", "Nhân Đạo"},
	{"dream", "Mộng Đạo"},
	{"food", "Thực Đạo"},
	{"luck", "Vận Đạo"},

	// Vũ khí, Sát phạt & Đặc thù
	{"sword", "Kiếm Đạo"},
	{"blade", "Đao Đạo"},
	{"theft", "Thâu Đạo"},
	{"phantom", "Huyễn Đạo"},
	{"painting", "Họa Đạo"},
	{"weapon", "Binh Đạo"},
	{"killing", "Sát Đạo"},
	{"restriction", "Cấm Đạo"},
	{"pill", "Đan Đạo"},
}

var pathMaterialIDs = map[string]bool{
	// Ngũ hành & Tự nhiên
	"metal": true, "wood": true, "water": true, "fire": true, "earth": true,
	"wind": true, "lightning": true, "ice": true, "cloud": true,
	"light": true, "dark": true, "star": true, "sound": true,

	// Vũ trụ, Quy tắc & Kỹ nghệ
	"space": true, "time": true, "heaven": true,
	"rule": true, "refinement": true, "formation": true, "qi": true,

	// Cơ thể, Tinh thần & Xã hội
	"strength": true, "soul": true, "wisdom": true, "emotion": true,
	"transformation": true, "enslavement": true, "information": true,
	"blood": true, "poison": true, "bone": true, "human": true,
	"dream": true, "food": true, "luck": true,

	// Vũ khí, Sát phạt & Đặc thù
	"sword": true, "blade": true, "theft": true, "phantom": true,
	"painting": true, "weapon": true, "killing": true, "restriction": true, "pill": true,
}

// pathMaterialID returns the reward material id for a path key, falling back
// to guts_gu when the path has no dedicated *_path_material item.
func pathMaterialID(pathKey string) string {
	if pathKey == "dream" {
		return "dream_path_material"
	}
	if pathMaterialIDs[pathKey] {
		return pathKey + "_path_material"
	}
	return "guts_gu"
}

// pathVN returns the Vietnamese display name for a path key, falling back to
// the key itself for unknown paths.
func pathVN(pathKey string) string {
	for _, p := range dreamRealmPaths {
		if p.Key == pathKey {
			return p.VN
		}
	}
	return pathKey
}

// buildDreamRealmMapData builds a generic DREAM_REALM stage tree themed to a
// given lưu phái. Content is template-generated for now; dynamic per-path
// storytelling can replace it later.
func buildDreamRealmMapData(pathKey, pathVN string) string {
	mat := pathMaterialID(pathKey)
	return fmt.Sprintf(`[
		{
			"stage_index": 0,
			"description": "Đạo hữu chìm vào giấc mộng huyền ảo của %s. Hồn phách lơ lửng giữa vô vàn tinh quang đại đạo, mỗi bước lĩnh ngộ đều tiêu hao Hồn Lực. Trước mặt có hai con đường:",
			"options": [
				{"text": "Tĩnh tâm nhập định, cảm nhận đại đạo (Tiêu hao 15 Soul HP)", "next_stage": 1, "requirements": {}},
				{"text": "Mạnh dạn lao sâu vào ảo cảnh thử thách (Tiêu hao 25 Soul HP)", "next_stage": 2, "requirements": {}}
			]
		},
		{
			"stage_index": 1,
			"description": "Một khối linh khí %s ngưng tụ thành cự thú chặn đường. Cần hồn phách kiên cường hoặc tài liệu tương ứng để hóa giải.",
			"options": [
				{"text": "Dùng lực hồn phách trấn áp (Yêu cầu Soul HP >= 40)", "next_stage": 3, "requirements": {"min_soul_hp": 40}},
				{"text": "Tế lễ tài liệu để cự thú rút lui (Mất 1 %s)", "next_stage": 3, "requirements": {"material_id": "%s", "amount": 1}}
			]
		},
		{
			"stage_index": 2,
			"description": "Ảo cảnh đảo lộn, hồn phách bị phản phệ dữ dội. Đạo hữu phải quyết định nhanh:",
			"options": [
				{"text": "Ổn định tâm thần, chịu đựng phản phệ (Mất 20 Soul HP)", "next_stage": 3, "requirements": {}},
				{"text": "Rút lui khỏi mộng cảnh ngay lập tức (Thất bại)", "next_stage": -2, "requirements": {}}
			]
		},
		{
			"stage_index": 3,
			"description": "Cuối cùng đạo hữu chạm tới hạch tâm đại đạo của %s. Tinh quang Chân Ý dâng trào, chỉ cần hấp thụ là hoàn thành truyền thừa:",
			"options": [
				{"text": "Nhập định hấp thụ Chân Ý và rời Mộng Cảnh", "next_stage": -1, "requirements": {}}
			]
		}
	]`, pathVN, pathVN, mat, mat, pathVN)
}

// buildSecretRealmMapData builds a generic SECRET_REALM stage tree themed to a
// given lưu phái. Unlike dream realms it drains normal HP instead of Soul HP.
func buildSecretRealmMapData(pathKey, pathVN string) string {
	mat := pathMaterialID(pathKey)
	return fmt.Sprintf(`[
		{
			"stage_index": 0,
			"description": "Đạo hữu lần theo cổ địa đồ tìm vào bí cảnh của %s. Không khí âm u nặng nề, vô số cấm chế cổ xưa đang ẩn nấp trong bóng tối:",
			"options": [
				{"text": "Cẩn thận men theo vách đá tìm lối đi an toàn (Mất 20 HP)", "next_stage": 1, "requirements": {}},
				{"text": "Liều mình xông thẳng qua vùng cấm chế (Mất 40 HP)", "next_stage": 2, "requirements": {}}
			]
		},
		{
			"stage_index": 1,
			"description": "Một trận đồ cổ của %s phát sáng chặn ngang đường đi. Cần sức mạnh cơ thể hoặc tài liệu quý để phá trận:",
			"options": [
				{"text": "Dùng sức mạnh đập vỡ trận nhãn (Mất 25 HP)", "next_stage": 3, "requirements": {}},
				{"text": "Hiến tế tài liệu để trận đồ tự tan (Mất 1 %s)", "next_stage": 3, "requirements": {"material_id": "%s", "amount": 1}}
			]
		},
		{
			"stage_index": 2,
			"description": "Thủ hộ linh chấn nộ phá ải mà lao tới! Chiến đấu ác liệt đến kiệt sức:",
			"options": [
				{"text": "Liều mình chống trả đến cùng (Mất 35 HP)", "next_stage": 3, "requirements": {}},
				{"text": "Rút lui khỏi bí cảnh ngay lập tức (Thất bại)", "next_stage": -2, "requirements": {}}
			]
		},
		{
			"stage_index": 3,
			"description": "Cổ điện cuối cùng hiện ra, cơ duyên của %s nằm trong bảo rương cổ xưa giữa đại điện:",
			"options": [
				{"text": "Mở bảo rương nhận cơ duyên và rời bí cảnh", "next_stage": -1, "requirements": {}}
			]
		}
	]`, pathVN, pathVN, mat, mat, pathVN)
}

// seedDreamRealmsFromCSV seeds DREAM_REALM instance templates from
// DreamRealm.csv. Each row maps 1:1 to an instance_templates record. When
// map_data is empty it is generated from the path key; rewards fall back to
// {} when empty. Path-specific rows are id "dream_realm_<path>".
func seedDreamRealmsFromCSV(filename string) error {
	f, r, header, err := openCSV(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	col := func(name string) int { return colIndex(header, name) }
	idIdx, nameIdx := col("id"), col("name")
	pathIdx, typeIdx := col("path"), col("type")
	rankIdx, entryIdx := col("required_rank"), col("entry_item")
	descIdx, rewardsIdx, mapIdx, cdIdx := col("description"), col("rewards"), col("map_data"), col("cooldown_hours")
	if idIdx < 0 || nameIdx < 0 {
		return fmt.Errorf("%s missing required columns (id, name)", filename)
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
		pathKey := get(row, pathIdx)
		mapData := get(row, mapIdx)
		if mapData == "" {
			mapData = buildDreamRealmMapData(pathKey, pathVN(pathKey))
		}
		rewards := safeJSONObject(get(row, rewardsIdx), "{}", filename+":"+id+":rewards")
		rank := 3
		if v, err := strconv.Atoi(get(row, rankIdx)); err == nil && v > 0 {
			rank = v
		}
		cd := 24
		if v, err := strconv.Atoi(get(row, cdIdx)); err == nil && v > 0 {
			cd = v
		}
		rtype := get(row, typeIdx)
		if rtype == "" {
			rtype = "DREAM_REALM"
		}
		_, err = activeCSVImportDB().Exec(`
			INSERT INTO instance_templates (id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, type=EXCLUDED.type, required_rank=EXCLUDED.required_rank,
				entry_item=EXCLUDED.entry_item, description=EXCLUDED.description,
				rewards=EXCLUDED.rewards, map_data=EXCLUDED.map_data, cooldown_hours=EXCLUDED.cooldown_hours
		`, id, get(row, nameIdx), rtype, rank, get(row, entryIdx), get(row, descIdx), rewards, mapData, cd)
		if err != nil {
			return fmt.Errorf("failed to seed dream realm %s: %w", id, err)
		}
		seeded++
	}
	logger.S.Infof("Dream realms seeded from %s (%d seeded)", filename, seeded)
	return nil
}

func SeedInstanceTemplates() error {
	templates := []struct {
		ID            string
		Name          string
		Type          string
		ReqRank       int
		EntryItem     string
		Description   string
		Rewards       string
		MapData       string
		CooldownHours int
	}{
		{
			ID:            "dungeon_flower_wine",
			Name:          "Truyền Thừa Hoa Tửu Hành Giả",
			Type:          "INHERITANCE_DEMONIC",
			ReqRank:       1,
			EntryItem:     "NONE",
			Description:   "Di tích động đá ẩn mật của Hoa Tửu Hành Giả lưu lại từ trăm năm trước, ẩn chứa cơ duyên Moonlight Gu và nguy cơ tử vong.",
			Rewards:       `{"materials": {"guts_gu": 2, "green_bamboo_wine": 1}, "primeval_stones": 100, "attainment": {"path": "Star Path", "amount": 10}}`,
			CooldownHours: 24,
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
	}

	// Dream realms (Mộng Cảnh) are data-driven from DreamRealm.csv. Each row
	// maps 1:1 to an instance_templates record; empty map_data is generated
	// from the path key. See docs/GAME_DATA.md.
	if err := seedDreamRealmsFromCSV("DreamRealm.csv"); err != nil {
		return err
	}

	// Bí cảnh (SECRET_REALM): combat-style instances rewarding Đạo Ngân.
	secretRealms := []struct {
		Key     string
		VN      string
		ID      string
		Name    string
		Desc    string
		Rewards string
	}{
		{"sword", "Kiếm Đạo", "secret_realm_sword_mountain", "Bí Cảnh Kiếm Sơn",
			"Vết tích tàn dư của đại trận Kiếm Tông cổ đại, nơi vô số kiếm ấn ngủ vùi chờ người hữu duyên.",
			`{"dao_marks": {"sword": 500}, "primeval_stones": 200, "materials": {"sword_path_material": 10}}`},
		{"fire", "Hỏa Đạo", "secret_realm_fire_cave", "Bí Cảnh Hỏa Huyệt",
			"Động dung nham nóng bỏng từng là lò luyện của Hỏa Tổ, linh hỏa muôn đời không tắt.",
			`{"dao_marks": {"fire": 500}, "primeval_stones": 200, "materials": {"fire_path_material": 10}}`},
		{"water", "Thủy Đạo", "secret_realm_water_palace", "Bí Cảnh Thủy Cung",
			"Long cung chìm sâu dưới đáy biển, chứa đựng thủy tinh đại đạo của Thủy Thần cổ xưa.",
			`{"dao_marks": {"water": 500}, "primeval_stones": 200, "materials": {"water_path_material": 10}}`},
	}

	for _, s := range secretRealms {
		templates = append(templates, struct {
			ID            string
			Name          string
			Type          string
			ReqRank       int
			EntryItem     string
			Description   string
			Rewards       string
			MapData       string
			CooldownHours int
		}{
			ID:            s.ID,
			Name:          s.Name,
			Type:          "SECRET_REALM",
			ReqRank:       4,
			EntryItem:     pathMaterialID(s.Key),
			Description:   s.Desc,
			Rewards:       s.Rewards,
			MapData:       buildSecretRealmMapData(s.Key, s.VN),
			CooldownHours: 48,
		})
	}

	for _, t := range templates {
		_, err := activeCSVImportDB().Exec(`
			INSERT INTO instance_templates (id, name, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, type=EXCLUDED.type, required_rank=EXCLUDED.required_rank,
				entry_item=EXCLUDED.entry_item, description=EXCLUDED.description,
				rewards=EXCLUDED.rewards, map_data=EXCLUDED.map_data, cooldown_hours=EXCLUDED.cooldown_hours
		`, t.ID, t.Name, t.Type, t.ReqRank, t.EntryItem, t.Description, t.Rewards, t.MapData, t.CooldownHours)
		if err != nil {
			return fmt.Errorf("failed to seed instance %s: %w", t.ID, err)
		}
	}
	return nil
}

func SeedSects() error {
	// Sect data is now managed entirely through the Admin panel.
	// No CSV seeding is performed on startup.
	if err := seedSectMissionsFromCSV("SectMission.csv"); err != nil {
		return err
	}
	return nil
}

// Function removed: Sect data is now managed through Admin panel only.
// CSV seeding for sects has been discontinued.

func syncSectVaultFromJSON(sectID, vaultGu, vaultMat, vaultRec string) error {
	// 1. Materials
	if strings.TrimSpace(vaultMat) != "" && vaultMat != "{}" && vaultMat != "null" {
		var mats map[string]int64
		if err := json.Unmarshal([]byte(vaultMat), &mats); err == nil {
			for itemID, qty := range mats {
				if qty > 0 {
					_, _ = activeCSVImportDB().Exec(`
						INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
						VALUES ($1, $2, $3)
						ON CONFLICT (sect_id, item_id) DO UPDATE SET
							quantity = EXCLUDED.quantity, updated_at = NOW()
					`, sectID, itemID, qty)
				}
			}
		}
	}

	// 2. Gu
	if strings.TrimSpace(vaultGu) != "" && vaultGu != "[]" && vaultGu != "null" {
		type guItem struct {
			ID         string  `json:"id"`
			GuID       string  `json:"gu_id,omitempty"`
			Rank       int     `json:"rank"`
			Cost       int64   `json:"cost"`
			Quantity   int64   `json:"quantity"`
			Hunger     float64 `json:"hunger"`
			InstanceID string  `json:"instance_id,omitempty"`
		}
		var gus []guItem
		if err := json.Unmarshal([]byte(vaultGu), &gus); err == nil {
			for _, g := range gus {
				guID := g.ID
				if guID == "" {
					guID = g.GuID
				}
				if guID == "" {
					continue
				}
				rank := g.Rank
				if rank <= 0 {
					rank = 1
				}
				qty := g.Quantity
				if qty <= 0 {
					qty = 1
				}
				hunger := g.Hunger
				if hunger <= 0 {
					hunger = 100
				}
				instID := g.InstanceID
				if instID == "" && qty == 1 {
					instID = fmt.Sprintf("gu_%s_%s_%d", sectID, guID, rank)
				}
				rowID := fmt.Sprintf("%s:%s:%d", sectID, guID, rank)
				if instID != "" {
					rowID = fmt.Sprintf("%s:%s", sectID, instID)
				}
				_, _ = activeCSVImportDB().Exec(`
					INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, cost, hunger, instance_id)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
					ON CONFLICT (id) DO UPDATE SET
						quantity = EXCLUDED.quantity, hunger = GREATEST(sect_vault_gu.hunger, EXCLUDED.hunger),
						cost = EXCLUDED.cost, updated_at = NOW()
				`, rowID, sectID, guID, rank, qty, g.Cost, hunger, instID)
			}
		}
	}

	// 3. Recipes
	if strings.TrimSpace(vaultRec) != "" && vaultRec != "[]" && vaultRec != "null" {
		type recItem struct {
			RecipeID       string  `json:"recipe_id"`
			CompletionRate float64 `json:"completion_rate"`
		}
		var recs []recItem
		if err := json.Unmarshal([]byte(vaultRec), &recs); err == nil {
			for _, r := range recs {
				if r.RecipeID == "" {
					continue
				}
				rate := r.CompletionRate
				if rate <= 0 {
					rate = 100.0
				}
				_, _ = activeCSVImportDB().Exec(`
					INSERT INTO sect_vault_recipes (sect_id, recipe_id, completion_rate)
					VALUES ($1, $2, $3)
					ON CONFLICT (sect_id, recipe_id) DO UPDATE SET
						completion_rate = GREATEST(sect_vault_recipes.completion_rate, EXCLUDED.completion_rate)
				`, sectID, r.RecipeID, rate)
			}
		}
	}
	return nil
}

func backfillSectVaultTables() error {
	rows, err := activeCSVImportDB().Query(`SELECT id, COALESCE(vault_gu::text, '[]'), COALESCE(vault_materials::text, '{}'), COALESCE(vault_recipes::text, '[]') FROM sects`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type sectRow struct {
		id       string
		vaultGu  string
		vaultMat string
		vaultRec string
	}
	var sects []sectRow
	for rows.Next() {
		var s sectRow
		if err := rows.Scan(&s.id, &s.vaultGu, &s.vaultMat, &s.vaultRec); err == nil {
			sects = append(sects, s)
		}
	}

	for _, s := range sects {
		// Chỉ backfill nếu tông môn này chưa có bất kỳ dữ liệu kho nào trong các bảng mới.
		// Tránh ghi đè dữ liệu gameplay mới bằng dữ liệu JSONB cũ khi server restart lại nhiều lần.
		var matCount, guCount, recCount int
		_ = activeCSVImportDB().QueryRow(`SELECT COUNT(*) FROM sect_vault_materials WHERE sect_id = $1`, s.id).Scan(&matCount)
		_ = activeCSVImportDB().QueryRow(`SELECT COUNT(*) FROM sect_vault_gu WHERE sect_id = $1`, s.id).Scan(&guCount)
		_ = activeCSVImportDB().QueryRow(`SELECT COUNT(*) FROM sect_vault_recipes WHERE sect_id = $1`, s.id).Scan(&recCount)

		if matCount == 0 && guCount == 0 && recCount == 0 {
			_ = syncSectVaultFromJSON(s.id, s.vaultGu, s.vaultMat, s.vaultRec)
		}
	}
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
		_, err = activeCSVImportDB().Exec(`
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

// CheckImmortalGuExists verifies if a specific Immortal Gu (id) exists anywhere in the world,
// by matching the same gu_id at any rank >= 6 (a tiên cổ is unique by id: different rank of the
// same id still counts as a duplicate). Excludes instances whose InstanceID is in
// excludeInstanceIDs (being consumed in a refine). Sources scanned:
//  1. player_gu — every player's inventory
//  2. auction — ACTIVE Gu listings
//  3. sects.vault_gu — sect vault (kho tông môn)
func CheckImmortalGuExists(tx *sql.Tx, guID string, rank int, excludeInstanceIDs []string) (bool, error) {
	var runner interface {
		QueryRow(query string, args ...any) *sql.Row
	}

	if tx != nil {
		runner = tx
		var lockOk bool
		var err error
		err = tx.QueryRow(`SELECT pg_try_advisory_xact_lock(hashtext('refine_immortal_gu_' || $1))`, guID).Scan(&lockOk)
		if err != nil {
			return false, err
		}
		if !lockOk {
			return false, nil
		}
	} else if DB != nil {
		runner = DB
	} else {
		return false, nil
	}

	var exists bool

	// 1. Kiểm tra trong bảng phụ player_gu (toàn bộ cổ của mọi người chơi)
	err := runner.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM player_gu
			WHERE gu_id = $1
			  AND gu_rank >= 6
			  AND ($2::text[] IS NULL OR cardinality($2::text[]) = 0 OR NOT (id = ANY($2::text[])))
		)
	`, guID, pq.Array(excludeInstanceIDs)).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// 2. Kiểm tra trên sàn giao dịch/đấu giá (auction) và chợ public buyback_orders dạng sell
	err = runner.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM auction,
			jsonb_array_elements(
				CASE 
					WHEN jsonb_typeof(item_data) = 'array' THEN item_data 
					ELSE '[]'::jsonb 
				END
			) AS elem
			WHERE item_type = 'Gu'
			  AND status = 'ACTIVE'
			  AND elem->>'gu_id' = $1::text
			  AND COALESCE((elem->>'rank')::int, 0) >= 6
			UNION
			SELECT 1 FROM buyback_orders
			WHERE item_type = 'Gu'
			  AND status = 'ACTIVE'
			  AND order_type = 'SELL'
			  AND item_target->>'gu_id' = $1::text
			  AND COALESCE((item_target->>'rank')::int, 0) >= 6
		)
	`, guID).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// 3. Kiểm tra trong kho tông môn (sect_vault_gu)
	err = runner.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM sect_vault_gu
			WHERE gu_id = $1 AND rank >= 6 AND quantity > 0
		)
	`, guID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetExistingImmortalGuIDs returns the set of gu_ids that currently have at least one
// rank >= 6 (Immortal Gu) instance anywhere in the world: any player's player_gu,
// ACTIVE auction listings, or the sect vault (sect_vault_gu / sects.vault_gu).
func GetExistingImmortalGuIDs(tx *sql.Tx) (map[string]bool, error) {
	rows, err := tx.Query(`
		SELECT DISTINCT gu_id FROM player_gu WHERE gu_rank >= 6
		UNION
		SELECT DISTINCT elem->>'gu_id' FROM auction,
			jsonb_array_elements(
				CASE 
					WHEN jsonb_typeof(item_data) = 'array' THEN item_data 
					ELSE '[]'::jsonb 
				END
			) AS elem
			WHERE item_type = 'Gu' AND status = 'ACTIVE' AND COALESCE((elem->>'rank')::int, 0) >= 6
		UNION
		SELECT DISTINCT gu_id FROM sect_vault_gu WHERE rank >= 6 AND quantity > 0
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
	err := activeCSVImportDB().QueryRow("SELECT COUNT(*) FROM npcs").Scan(&count)
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
			_, err = activeCSVImportDB().Exec(`
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

// ExportToCSV exports every dataset from the same registry used by individual exports.
func ExportToCSV() error {
	for _, dataset := range Datasets() {
		if dataset.Export == nil {
			continue
		}
		if err := dataset.Export(); err != nil {
			return fmt.Errorf("export dataset %s: %w", dataset.Key, err)
		}
	}
	logger.S.Info("All CSV data exported successfully")
	return nil
}

// AvailableTables returns UI-supported dataset keys from the canonical registry.
func AvailableTables() []string {
	var tables []string
	for _, dataset := range Datasets() {
		if dataset.UISupported {
			tables = append(tables, dataset.Key)
		}
	}
	return tables
}

// TableImport imports a single table from its CSV file.
func TableImport(table string) error {
	dataset, ok := DatasetByKey(table)
	if !ok || dataset.Import == nil {
		return fmt.Errorf("unknown table %q: dataset does not support import", table)
	}
	if dataset.Preflight == nil {
		return fmt.Errorf("dataset %q does not declare preflight behavior", table)
	}
	if err := dataset.Preflight(); err != nil {
		return err
	}
	return dataset.Import()
}

// TableExport exports a single table to its CSV file.
type exportDef struct {
	filename string
	columns  []string
	query    string
}

var individualExportDefs = map[string]exportDef{
	"item": {
		filename: "Item.csv",
		columns:  []string{"id", "name", "item_type", "rank", "effects", "description", "base_price", "image_url", "rarity", "path"},
		query:    `SELECT id, name, item_type, rank, COALESCE(effects::text,'[]'), COALESCE(description,''), COALESCE(base_price,0), COALESCE(image_url,''), CASE WHEN COALESCE(rarity,0) = 0 THEN '' ELSE rarity::text END, COALESCE(path,'') FROM item_dictionary ORDER BY id`,
	},
	"gu": {
		filename: "Gu.csv",
		columns:  []string{"id", "name", "rank", "gu_type", "hunger_rate", "food_types", "max_hunger", "description", "path", "base_mana_cost", "effects", "target_type", "rarity", "is_immortal", "primary_role", "combat_roles", "combat_tags", "support_tags", "base_price", "image_url"},
		query:    `SELECT id, name, rank, gu_type, COALESCE(hunger_rate,0), COALESCE(food_types::text,'[]'), COALESCE(max_hunger,100), COALESCE(description,''), COALESCE(path::text,'[]'), COALESCE(base_mana_cost,0), COALESCE(effects::text,'[]'), COALESCE(target_type,'SINGLE'), CASE WHEN COALESCE(rarity,0) = 0 THEN '' ELSE rarity::text END, COALESCE(is_immortal,false), COALESCE(primary_role,''), COALESCE(combat_roles::text,'[]'), COALESCE(combat_tags::text,'[]'), COALESCE(support_tags::text,'[]'), COALESCE(base_price,0), COALESCE(image_url,'') FROM gu_dictionary ORDER BY id`,
	},
	"constitution": {
		filename: "Constitution.csv",
		columns:  []string{"key", "name", "effects"},
		query:    `SELECT key, name, COALESCE(effects::text,'[]') FROM constitution_dictionary ORDER BY key`,
	},
	"race": {
		filename: "Race.csv",
		columns:  []string{"key", "name", "effects"},
		query:    `SELECT key, name, COALESCE(effects::text,'[]') FROM race_dictionary ORDER BY key`,
	},
	"status": {
		filename: "Status.csv",
		columns:  []string{"key", "name", "effects", "default_duration_minutes", "blocked_actions", "description"},
		query:    `SELECT key, name, COALESCE(effects::text,'[]'), COALESCE(default_duration_minutes,0), COALESCE(blocked_actions::text,'[]'), COALESCE(description,'') FROM status_dictionary ORDER BY key`,
	},
	"recipe": {
		filename: "Recipe.csv",
		columns:  []string{"id", "target_gu_id", "target_rank", "variant", "required_gus", "required_materials", "success_rate", "primeval_stones_cost", "immortal_essence_stones_cost"},
		query:    `SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) AS id, target_gu_id, target_rank, variant, COALESCE(required_gus::text,'[]'), COALESCE(required_materials::text,'{}'), COALESCE(success_rate,0), COALESCE(primeval_stones_cost,0), COALESCE(immortal_essence_stones_cost,0) FROM refining_recipes ORDER BY target_gu_id, target_rank, variant`,
	},
	"shop_item": {
		filename: "ShopItem.csv",
		columns:  []string{"id", "rank", "price", "stock", "is_default", "sort_order"},
		query:    `SELECT id, rank, price, COALESCE(stock,-1), is_default, COALESCE(sort_order,0) FROM caravan_shop_items ORDER BY id, rank`,
	},
	"arena_shop_item": {
		filename: "ArenaShopItem.csv",
		columns:  []string{"id", "price", "qty", "stock", "is_default", "sort_order"},
		query:    `SELECT id, price, COALESCE(qty,1), COALESCE(stock,-1), is_default, COALESCE(sort_order,0) FROM arena_shop_items ORDER BY id`,
	},
	"region": {
		filename: "Region.csv",
		columns:  []string{"id", "name", "description", "width", "height", "min_rank", "max_rank", "is_active", "parent_id", "zone_type", "min_x", "min_y", "max_x", "max_y", "entry_x", "entry_y", "pool_weights"},
		query:    `SELECT id, name, COALESCE(description,''), COALESCE(width,100), COALESCE(height,100), COALESCE(min_rank,1), COALESCE(max_rank,5), COALESCE(is_active,true), COALESCE(parent_id,''), COALESCE(zone_type,'KHU_VUC'), COALESCE(min_x,0), COALESCE(min_y,0), COALESCE(max_x,0), COALESCE(max_y,0), COALESCE(entry_x,0), COALESCE(entry_y,0), COALESCE(pool_weights::text,'{}') FROM regions ORDER BY id`,
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
	"map_cell": {
		filename: "MapCell.csv",
		columns:  []string{"region_id", "x", "y", "cell_type", "content_id", "metadata", "respawn_seconds", "is_active"},
		query:    `SELECT region_id, x, y, cell_type, COALESCE(content_id,''), COALESCE(metadata::text,'{}'), COALESCE(respawn_seconds,300), COALESCE(is_active,true) FROM map_cells ORDER BY x, y`,
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
	"dream_realm": {
		filename: "DreamRealm.csv",
		columns:  []string{"id", "name", "path", "type", "required_rank", "entry_item", "description", "rewards", "map_data", "cooldown_hours"},
		query:    `SELECT id, name, CASE WHEN id LIKE 'dream_realm_%' THEN substring(id from 13) ELSE '' END, type, required_rank, entry_item, COALESCE(description,''), COALESCE(rewards::text,'{}'), COALESCE(map_data::text,'[]'), cooldown_hours FROM instance_templates WHERE type='DREAM_REALM' ORDER BY id`,
	},
}

func TableExport(table string) error {
	dataset, ok := DatasetByKey(table)
	if !ok || dataset.Export == nil {
		return fmt.Errorf("unknown table %q: dataset does not support export", table)
	}
	return dataset.Export()
}
