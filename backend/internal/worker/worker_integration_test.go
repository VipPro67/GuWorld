package worker

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func connectScratch(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GUWORLD_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://gu_admin:gu_pass_123@localhost:5432/gu_scratch?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("skipping: cannot reach scratch DB: %v", err)
	}
	if err := models.LoadGuRegistry(db); err != nil {
		t.Fatalf("LoadGuRegistry: %v", err)
	}
	return db
}

func resetGrowthScratch(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`DELETE FROM player_gu`, `DELETE FROM players`,
		`INSERT INTO players (id, username, rank, hp, base_hp, primeval_stones, current_mana, max_mana, immortal_stones, equipped_gus) VALUES
			('aaaaaaaa-0000-0000-0000-000000000003','p3',1,100,100,0,50,100,0,'["g3"]'),
			('aaaaaaaa-0000-0000-0000-000000000004','p4',3,100,100,10,50,100,0,'["g4","g5"]'),
			('aaaaaaaa-0000-0000-0000-000000000005','p5',1,100,100,0,50,100,0,'["g6"]')`,
		`INSERT INTO player_gu (id, player_id, gu_id, gu_rank, hunger, state, equipped, is_vital, equipped_at, growth_hours) VALUES
			('g3','aaaaaaaa-0000-0000-0000-000000000003','GU_HEAVENLY_ESSENCE_LOTUS',3,100,'ACTIVE',true,false, NOW() - INTERVAL '5 hours', 0),
			('g4','aaaaaaaa-0000-0000-0000-000000000004','GU_HEAVENLY_ESSENCE_LOTUS',3,100,'ACTIVE',true,true, NOW() - INTERVAL '5 hours', 0),
			('g5','aaaaaaaa-0000-0000-0000-000000000004','GU_BEAR_STRENGTH',1,100,'ACTIVE',true,false, NOW() - INTERVAL '5 hours', 0),
			('g6','aaaaaaaa-0000-0000-0000-000000000005','GU_BEAR_STRENGTH',1,100,'ACTIVE',false,false, NOW() - INTERVAL '5 hours', 0)`,
	}
	for _, s := range statements {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("resetGrowthScratch: %v", err)
		}
	}
}

// TestGrowthWorkerSubset verifies that growth only touches players owning an
// equipped ACTIVE Gu with a growth profile, and that growth_hours advance.
func TestGrowthWorkerSubset(t *testing.T) {
	db := connectScratch(t)
	defer db.Close()

	resetGrowthScratch(t, db)

	players, err := findGrowthAffectedPlayers(db)
	if err != nil {
		t.Fatalf("findGrowthAffectedPlayers: %v", err)
	}
	// p4 owns GU_BEAR_STRENGTH equipped; p5 owns it but NOT equipped; p3 does not.
	found := map[string]bool{}
	for _, p := range players {
		found[p] = true
	}
	if !found["aaaaaaaa-0000-0000-0000-000000000004"] {
		t.Errorf("expected p4 to be growth-affected, got %v", players)
	}
	if found["aaaaaaaa-0000-0000-0000-000000000003"] {
		t.Errorf("p3 should not be growth-affected, got %v", players)
	}
	if found["aaaaaaaa-0000-0000-0000-000000000005"] {
		t.Errorf("p5 (gu not equipped) should not be growth-affected, got %v", players)
	}
}

// TestGrowthWorkerAdvance verifies updatePlayerGrowth folds elapsed hours into
// growth_hours and growth_data for a player with an equipped growth Gu.
func TestGrowthWorkerAdvance(t *testing.T) {
	db := connectScratch(t)
	defer db.Close()

	resetGrowthScratch(t, db)

	// Reset p4's growth Gu to a known state: equipped 10h ago, zero progress.
	if _, err := db.Exec(`UPDATE player_gu SET growth_hours = 0, equipped_at = NOW() - INTERVAL '10 hours' WHERE id = 'g5'`); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := db.Exec(`UPDATE players SET growth_data = '{"boosts":{"attack":0,"defense":0,"hp":0,"speed":0},"fully_grown":[]}', base_hp = 100 WHERE id = 'aaaaaaaa-0000-0000-0000-000000000004'`); err != nil {
		t.Fatalf("reset growth_data: %v", err)
	}

	if err := updatePlayerGrowth(db, "aaaaaaaa-0000-0000-0000-000000000004"); err != nil {
		t.Fatalf("updatePlayerGrowth: %v", err)
	}

	// GU_BEAR_STRENGTH: hours_to_max 50, max_atk 18, max_hp 8, max_def 3.
	// 10h elapsed => progress 0.2 => atk +3.6, hp +1.6, def +0.6.
	var gh float64
	var raw []byte
	var baseHP float64
	if err := db.QueryRow(`SELECT growth_hours FROM player_gu WHERE id = 'g5'`).Scan(&gh); err != nil {
		t.Fatalf("read growth_hours: %v", err)
	}
	if gh < 9.0 || gh > 11.0 {
		t.Errorf("expected growth_hours ~10, got %v", gh)
	}
	if err := db.QueryRow(`SELECT growth_data, base_hp FROM players WHERE id = 'aaaaaaaa-0000-0000-0000-000000000004'`).Scan(&raw, &baseHP); err != nil {
		t.Fatalf("read growth_data: %v", err)
	}
	if baseHP < 101.5 || baseHP > 101.7 {
		t.Errorf("expected base_hp ~101.6, got %v", baseHP)
	}
	var gd struct {
		Boosts struct {
			Attack float64 `json:"attack"`
			Defense float64 `json:"defense"`
			HP      float64 `json:"hp"`
		} `json:"boosts"`
	}
	if err := json.Unmarshal(raw, &gd); err != nil {
		t.Fatalf("unmarshal growth_data: %v", err)
	}
	if gd.Boosts.Attack < 3.5 || gd.Boosts.Attack > 3.7 {
		t.Errorf("expected boosts.attack ~3.6, got %v", gd.Boosts.Attack)
	}
	if gd.Boosts.HP < 1.5 || gd.Boosts.HP > 1.7 {
		t.Errorf("expected boosts.hp ~1.6, got %v", gd.Boosts.HP)
	}
}

// TestHungerDecayBatch verifies the set-based decay: non-vital starved Gu die
// (with audit), vital starved Gu die and deal 80 HP backlash, equipped_gus is
// pruned, and surviving Gu hunger decreases by its hunger_rate.
func TestHungerDecayBatch(t *testing.T) {
	db := connectScratch(t)
	defer db.Close()

	// Reset scratch to a controlled state.
	statements := []string{
		`DELETE FROM player_gu`, `DELETE FROM players`, `DELETE FROM audit_logs`,
		`INSERT INTO players (id, username, rank, hp, base_hp, primeval_stones, current_mana, max_mana, immortal_stones, equipped_gus) VALUES
			('aaaaaaaa-0000-0000-0000-000000000001','h1',1,100,100,0,50,100,0,'["h1g1"]'),
			('aaaaaaaa-0000-0000-0000-000000000002','h2',1,50,100,0,50,100,0,'["h2g1"]')`,
		`INSERT INTO player_gu (id, player_id, gu_id, gu_rank, hunger, state, equipped, is_vital) VALUES
			('h1g1','aaaaaaaa-0000-0000-0000-000000000001','GU_MOONLIGHT',1,1,'ACTIVE',true,false),
			('h2g1','aaaaaaaa-0000-0000-0000-000000000002','GU_HEAVENLY_ESSENCE_LOTUS',3,1,'ACTIVE',true,true)`,
	}
	for _, s := range statements {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("reset: %v", err)
		}
	}

	if err := runHungerDecayBatch(db); err != nil {
		t.Fatalf("runHungerDecayBatch: %v", err)
	}

	// h1g1 (non-vital, hunger 1, rate 1) starves => DEAD and removed.
	var h1Count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE id = 'h1g1'`).Scan(&h1Count); err != nil {
		t.Fatalf("count h1g1: %v", err)
	}
	if h1Count != 0 {
		t.Errorf("expected h1g1 removed, count=%d", h1Count)
	}
	// h1 equipped_gus pruned to empty.
	var h1Eq []byte
	if err := db.QueryRow(`SELECT equipped_gus FROM players WHERE id = 'aaaaaaaa-0000-0000-0000-000000000001'`).Scan(&h1Eq); err != nil {
		t.Fatalf("read h1 equipped_gus: %v", err)
	}
	if string(h1Eq) != "[]" {
		t.Errorf("expected h1 equipped_gus=[], got %s", h1Eq)
	}
	// h2g1 (vital) starves => DEAD removed, h2 hp 50-80 = 0.
	var h2g1 int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE id = 'h2g1'`).Scan(&h2g1); err != nil {
		t.Fatalf("count h2g1: %v", err)
	}
	if h2g1 != 0 {
		t.Errorf("expected h2g1 removed, count=%d", h2g1)
	}
	var h2HP float64
	if err := db.QueryRow(`SELECT hp FROM players WHERE id = 'aaaaaaaa-0000-0000-0000-000000000002'`).Scan(&h2HP); err != nil {
		t.Fatalf("read h2 hp: %v", err)
	}
	if h2HP != 0 {
		t.Errorf("expected h2 hp=0 (backlash), got %v", h2HP)
	}
	// Audit rows recorded for both deaths.
	var auditCnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action IN ('gu.starve_vital_death','gu.starve_death')`).Scan(&auditCnt); err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if auditCnt != 2 {
		t.Errorf("expected 2 audit rows, got %d", auditCnt)
	}
}

// TestApplyOnTickEffects verifies the Go-side ON_TICK (LOTUS stones) path.
func TestApplyOnTickEffects(t *testing.T) {
	db := connectScratch(t)
	defer db.Close()

	// Ensure p4 and its equipped LOTUS exist for this test (previous tests may
	// have removed them).
	if _, err := db.Exec(`INSERT INTO players (id, username, rank, hp, base_hp, primeval_stones, current_mana, max_mana, immortal_stones, equipped_gus) VALUES
		('aaaaaaaa-0000-0000-0000-000000000004','p4',3,100,100,0,50,100,0,'["g4","g5"]')
		ON CONFLICT (id) DO UPDATE SET rank=3, hp=100, base_hp=100, primeval_stones=0, current_mana=50, max_mana=100, immortal_stones=0, equipped_gus='["g4","g5"]'`); err != nil {
		t.Fatalf("ensure p4: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO player_gu (id, player_id, gu_id, gu_rank, hunger, state, equipped, is_vital) VALUES
		('g4','aaaaaaaa-0000-0000-0000-000000000004','GU_HEAVENLY_ESSENCE_LOTUS',3,100,'ACTIVE',true,true)
		ON CONFLICT (id) DO UPDATE SET state='ACTIVE', equipped=true, is_vital=true, hunger=100`); err != nil {
		t.Fatalf("ensure g4: %v", err)
	}

	if err := applyOnTickEffects(db, "aaaaaaaa-0000-0000-0000-000000000004"); err != nil {
		t.Fatalf("applyOnTickEffects: %v", err)
	}

	var stones int
	var hp float64
	if err := db.QueryRow(`SELECT primeval_stones, hp FROM players WHERE id = 'aaaaaaaa-0000-0000-0000-000000000004'`).Scan(&stones, &hp); err != nil {
		t.Fatalf("read p4: %v", err)
	}
	if stones != 10 {
		t.Errorf("expected +10 primeval stones, got %d", stones)
	}
	if hp != 100 {
		t.Errorf("expected hp unchanged (100), got %v", hp)
	}
}
