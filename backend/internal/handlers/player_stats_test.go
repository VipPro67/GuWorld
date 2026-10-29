package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

var registerStatsTxDriver sync.Once
var statsTxDriverState = &statsTxState{}

type statsTxState struct {
	sync.Mutex
	failCommit bool
	committed  map[uuid.UUID]statsTimeDerivedFields
}

type statsTimeDerivedFields struct {
	currentMana    float64
	lastManaUpdate time.Time
	lifespan       float64
}

type statsTxDriver struct{}
type statsTxConn struct {
	pending map[uuid.UUID]statsTimeDerivedFields
}
type statsTx struct{ conn *statsTxConn }

func (statsTxDriver) Open(string) (driver.Conn, error)     { return &statsTxConn{}, nil }
func (c *statsTxConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *statsTxConn) Close() error                        { return nil }
func (c *statsTxConn) Begin() (driver.Tx, error) {
	c.pending = make(map[uuid.UUID]statsTimeDerivedFields)
	return &statsTx{conn: c}, nil
}
func (c *statsTxConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) { return c.Begin() }
func (c *statsTxConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
	idText, ok := args[3].Value.(string)
	if !ok {
		return nil, errors.New("unexpected player id")
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return nil, err
	}
	c.pending[id] = statsTimeDerivedFields{
		currentMana: args[0].Value.(float64), lastManaUpdate: args[1].Value.(time.Time), lifespan: args[2].Value.(float64),
	}
	return driver.RowsAffected(1), nil
}
func (tx *statsTx) Commit() error {
	statsTxDriverState.Lock()
	defer statsTxDriverState.Unlock()
	if statsTxDriverState.failCommit {
		return errors.New("forced commit failure")
	}
	for id, fields := range tx.conn.pending {
		statsTxDriverState.committed[id] = fields
	}
	return nil
}
func (tx *statsTx) Rollback() error { tx.conn.pending = nil; return nil }

func setupStatsHandlerTransactionTest(t *testing.T, failCommit bool) (*gin.Context, *httptest.ResponseRecorder, uuid.UUID) {
	t.Helper()
	registerStatsTxDriver.Do(func() { sql.Register("stats-handler-tx", statsTxDriver{}) })
	testDB, err := sql.Open("stats-handler-tx", "")
	if err != nil {
		t.Fatal(err)
	}
	originalDB := db.DB
	db.DB = testDB
	statsTxDriverState.Lock()
	statsTxDriverState.failCommit = failCommit
	statsTxDriverState.committed = make(map[uuid.UUID]statsTimeDerivedFields)
	statsTxDriverState.Unlock()
	t.Cleanup(func() { db.DB = originalDB; testDB.Close() })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/api/player/stats", nil)
	playerID := uuid.New()
	ctx.Set("player_id", playerID)
	return ctx, recorder, playerID
}

func TestGetPlayerStatsHandlerCommitsTimeDerivedFieldsBeforePublishing(t *testing.T) {
	ctx, recorder, playerID := setupStatsHandlerTransactionTest(t, false)
	now := time.Now().UTC().Truncate(time.Microsecond)
	originalEvaluate, originalRespond := getPlayerStatsEvaluate, getPlayerStatsRespond
	t.Cleanup(func() { getPlayerStatsEvaluate, getPlayerStatsRespond = originalEvaluate, originalRespond })
	getPlayerStatsEvaluate = func(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
		_, err := tx.Exec("UPDATE players SET current_mana=$1,last_mana_update=$2,lifespan=$3 WHERE id=$4", 42.5, now, 88.25, id)
		return models.Player{
			Account:          models.Account{ID: id},
			ResourceState:    models.ResourceState{CurrentMana: 42.5, LastManaUpdate: now},
			CultivationState: models.CultivationState{Lifespan: 88.25},
		}, err
	}
	published := 0
	getPlayerStatsRespond = func(_ *gin.Context, _ int, _ models.Player, _ ...gin.H) {
		statsTxDriverState.Lock()
		persisted, ok := statsTxDriverState.committed[playerID]
		statsTxDriverState.Unlock()
		if !ok || persisted.currentMana != 42.5 || !persisted.lastManaUpdate.Equal(now) || persisted.lifespan != 88.25 {
			t.Fatalf("response/publish ran before time-derived fields were committed: %+v", persisted)
		}
		published++
	}

	GetPlayerStatsHandler(ctx)
	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if published != 1 {
		t.Fatalf("published = %d, want 1", published)
	}
}

func TestGetPlayerStatsHandlerCommitFailureDoesNotPublishRolledBackState(t *testing.T) {
	ctx, recorder, playerID := setupStatsHandlerTransactionTest(t, true)
	originalEvaluate, originalRespond := getPlayerStatsEvaluate, getPlayerStatsRespond
	t.Cleanup(func() { getPlayerStatsEvaluate, getPlayerStatsRespond = originalEvaluate, originalRespond })
	getPlayerStatsEvaluate = func(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
		_, err := tx.Exec("UPDATE players SET current_mana=$1,last_mana_update=$2,lifespan=$3 WHERE id=$4", 75.0, time.Now(), 50.0, id)
		return models.Player{Account: models.Account{ID: id}}, err
	}
	published := 0
	getPlayerStatsRespond = func(_ *gin.Context, _ int, _ models.Player, _ ...gin.H) { published++ }

	GetPlayerStatsHandler(ctx)
	if recorder.Code != 500 {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if published != 0 {
		t.Fatalf("published rolled-back snapshot %d time(s)", published)
	}
	statsTxDriverState.Lock()
	_, persisted := statsTxDriverState.committed[playerID]
	statsTxDriverState.Unlock()
	if persisted {
		t.Fatal("time-derived fields persisted despite failed commit")
	}
}

func TestGetPassiveStats_SpeedIsolation(t *testing.T) {
	setupTestGuRegistry(t)
	defer resetTestGuRegistry(t)

	player := &models.Player{
		CultivationState: models.CultivationState{
			Rank:    1,
			SubRank: 1,
		},
		GuInventory: models.GuInventory{
			{InstanceID: "inst1", GuID: "GU_PASSIVE_SPD", Rank: 1, Status: "ACTIVE", Equipped: true},
		},
	}

	result := GetPassiveStats(player)

	// Base speed for Rank 1, SubRank 1 is 10.0 + 0 + 0 = 10.0
	// GU_PASSIVE_SPD adds 5.0 -> total passive speed = 15.0
	// GU_ACTIVE_SPD should NOT be included in passive stats (speed_boost_flat is combat-only).
	expectedPassiveSpeed := 15.0
	if actual := result.Stats["speed"]; actual != expectedPassiveSpeed {
		t.Fatalf("expected passive speed %.1f, got %.1f", expectedPassiveSpeed, actual)
	}

	// Verify details contains EXACTLY ONE entry for Key == "speed"
	speedDetailCount := 0
	for _, d := range result.Details {
		if d.Key == "speed" {
			speedDetailCount++
		}
	}
	if speedDetailCount != 1 {
		t.Fatalf("expected exactly 1 speed detail in Details, got %d", speedDetailCount)
	}

}

func TestGetPassiveStatsLoadsBlockV2PassivesFromRegistry(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	var lowEffects, highEffects models.GuEffects
	lowRaw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":3}]}]`
	highRaw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":12},{"op":"STAT_MOD","stat":"INVESTIGATION","sub_type":"ADD_FLAT","value":5},{"op":"STAT_MOD","stat":"DAMAGE_DEALT","sub_type":"MULTIPLY_PERCENT","value":0.4}]}]`
	if err := lowEffects.Scan(lowRaw); err != nil {
		t.Fatalf("scan rank 1 effects: %v", err)
	}
	if err := highEffects.Scan(highRaw); err != nil {
		t.Fatalf("scan rank 2 effects: %v", err)
	}
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_BLOCK_PASSIVE": {
			1: {ID: "GU_BLOCK_PASSIVE", Name: "Khối Bị Động", Rank: 1, Effects: lowEffects},
			2: {ID: "GU_BLOCK_PASSIVE", Name: "Khối Bị Động", Rank: 2, Effects: highEffects},
		},
	}
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 1, SubRank: 1},
		BodyState:        models.BodyState{BaseHP: 100, HP: 50},
		GuInventory: models.GuInventory{
			{InstanceID: "low", GuID: "GU_BLOCK_PASSIVE", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "high", GuID: "GU_BLOCK_PASSIVE", Rank: 2, Status: "ACTIVE", Equipped: true},
			{InstanceID: "duplicate", GuID: "GU_BLOCK_PASSIVE", Rank: 2, Status: "ACTIVE", Equipped: true},
		},
	}

	result := GetPassiveStats(player)
	if got := result.Stats["attack"]; got != 22 { // base 10 + highest-rank passive 12 once
		t.Fatalf("attack = %v, want 22", got)
	}
	// base 1.0 + highest-rank passive 5 once
	if got := result.Stats["investigation"]; got != 6 {
		t.Fatalf("investigation = %v, want 6", got)
	}
	if got := result.Stats["damage_multiplier"]; got != 0.2 {
		t.Fatalf("damage multiplier = %v, want 0.2", got)
	}
}

// setupTestGuRegistry installs a test Gu registry with block v2 speed effects.
func setupTestGuRegistry(t *testing.T) {
	t.Helper()
	passiveRaw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"SPEED","sub_type":"ADD_FLAT","value":5}]}]`
	var passiveEffects models.GuEffects
	if err := passiveEffects.Scan(passiveRaw); err != nil {
		t.Fatalf("scan passive effects: %v", err)
	}
	// Combat-only speed boost lives in ON_ACTION so it never leaks into
	// passive stats, mirroring the old speed_boost_flat semantics.
	activeRaw := `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STAT_MOD","stat":"SPEED","sub_type":"ADD_FLAT","value":20}]}]`
	var activeEffects models.GuEffects
	if err := activeEffects.Scan(activeRaw); err != nil {
		t.Fatalf("scan active effects: %v", err)
	}
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_PASSIVE_SPD": {
			1: {
				ID:      "GU_PASSIVE_SPD",
				Name:    "Bị Động Tốc Cổ",
				Rank:    1,
				Path:    []string{"Cloud Path"},
				Effects: passiveEffects,
			},
		},
		"GU_ACTIVE_SPD": {
			1: {
				ID:      "GU_ACTIVE_SPD",
				Name:    "Chủ Động Tốc Cổ",
				Rank:    1,
				Path:    []string{"Lightning Path"},
				Effects: activeEffects,
			},
		},
	}
}

// resetTestGuRegistry clears the test Gu registry.
func resetTestGuRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{}
}

// TestCombatSpeedNoDoubleCount verifies that combat speed equals passive stats speed
// with no double-counting of speed_boost_flat effects.
// TestGetPassiveStats_StatusStatMods verifies that status stat_mods scale the
// final stats (percentage on the accumulated total).
func TestGetPassiveStats_StatusStatMods(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 1, SubRank: 1},
	}
	player.Statuses = models.StatusList{{ID: "death_penalty"}}

	result := GetPassiveStats(player)
	// Base attack 10, defense 5, speed 10; death_penalty applies -50% to all.
	if got := result.Stats["attack"]; got != 5 {
		t.Fatalf("attack = %v, want 5 (10 * (1 - 0.5))", got)
	}
	if got := result.Stats["defense"]; got != 2.5 {
		t.Fatalf("defense = %v, want 2.5", got)
	}
	if got := result.Stats["speed"]; got != 5 {
		t.Fatalf("speed = %v, want 5", got)
	}
}

// TestGetPassiveStats_StatFloorAtZero verifies a suppression stronger than
// -100% floors the stat at zero instead of letting it go negative.
func TestGetPassiveStats_StatFloorAtZero(t *testing.T) {
	orig, hadOrig := gamefx.StatusRegistry["crush_test"]
	gamefx.StatusRegistry["crush_test"] = gamefx.StatusDef{
		Key:  "crush_test",
		Name: "Áp Chế Thử Nghiệm",
		Effects: gamefx.Effects{
			StatMods: map[string]float64{"attack": -1.5, "defense": -1.5, "speed": -1.5},
		},
	}
	defer func() {
		if hadOrig {
			gamefx.StatusRegistry["crush_test"] = orig
		} else {
			delete(gamefx.StatusRegistry, "crush_test")
		}
	}()

	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 1, SubRank: 1},
	}
	player.Statuses = models.StatusList{{ID: "crush_test"}}

	result := GetPassiveStats(player)
	for _, key := range []string{"attack", "defense", "speed"} {
		if got := result.Stats[key]; got != 0 {
			t.Fatalf("%s = %v, want 0 (floored)", key, got)
		}
	}
}

// TestBuildSimCombatant_StatusFactorScalesSkillDamage verifies that status
// stat_mods multiply the whole Gu skill output (not just the additive attack
// component), so heavy suppression like Áp Chế visibly reduces skill damage.
// TestGetPassiveStatsIgnoresDaoMarksAndMastery locks in that additive Gu
// passives are raw values: dao marks and path mastery must not scale passive
// ATK/DEF/HP/SPEED.
func TestGetPassiveStatsIgnoresDaoMarksAndMastery(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	raw := `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":20},{"op":"STAT_MOD","stat":"DEFENSE","sub_type":"ADD_FLAT","value":100},{"op":"STAT_MOD","stat":"HP","sub_type":"ADD_FLAT","value":200},{"op":"STAT_MOD","stat":"SPEED","sub_type":"ADD_FLAT","value":7}]}]`
	var effects models.GuEffects
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan effects: %v", err)
	}
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_RAW_PASSIVE": {
			4: {
				ID:      "GU_RAW_PASSIVE",
				Name:    "Thô Bạo Cổ",
				Rank:    4,
				Path:    []string{"Strength Path"},
				Effects: effects,
			},
		},
	}

	base := &models.Player{
		CultivationState: models.CultivationState{Rank: 4, SubRank: 1},
		BodyState:        models.BodyState{BaseHP: 1000, HP: 1000},
	}
	boosted := &models.Player{
		CultivationState: models.CultivationState{Rank: 4, SubRank: 1},
		BodyState:        models.BodyState{BaseHP: 1000, HP: 1000},
		DaoState: models.DaoState{
			PathMasteries: models.PathMasteriesMap{"strength": 1_200_000},
			DaoMarks:      models.DaoMarksMap{"strength": 2_500_000},
		},
		GuInventory: models.GuInventory{
			{InstanceID: "raw", GuID: "GU_RAW_PASSIVE", Rank: 4, Status: "ACTIVE", Equipped: true},
		},
	}

	baseStats := GetPassiveStats(base).Stats
	stats := GetPassiveStats(boosted).Stats

	if got := stats["attack"] - baseStats["attack"]; got != 20 {
		t.Errorf("attack delta = %v, want raw 20", got)
	}
	if got := stats["defense"] - baseStats["defense"]; got != 100 {
		t.Errorf("defense delta = %v, want raw 100", got)
	}
	if got := stats["hp"]; got != 200 {
		t.Errorf("hp = %v, want raw 200", got)
	}
	if got := stats["speed"] - baseStats["speed"]; got != 7 {
		t.Errorf("speed delta = %v, want raw 7", got)
	}
}

// TestBaseAttackIgnoresStrengthDaoMarks verifies base attack is derived from
// rank/sub-rank only; strength dao marks no longer add flat attack.
func TestBaseAttackIgnoresStrengthDaoMarks(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 8, SubRank: 1},
		DaoState:         models.DaoState{DaoMarks: models.DaoMarksMap{"strength": 2_560_300}},
	}

	stats := GetPassiveStats(player).Stats
	// 10 + (8-1)*10 + (1-1)*2 = 80
	if stats["attack"] != 80 {
		t.Fatalf("attack = %v, want 80 (rank-derived only)", stats["attack"])
	}
	// 5 + (8-1)*5 + (1-1)*1 = 40
	if stats["defense"] != 40 {
		t.Fatalf("defense = %v, want 40 (rank-derived only)", stats["defense"])
	}
}
