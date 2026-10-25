package services

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
	"github.com/vippro67/GuWorld/backend/internal/util"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

// QueryRower is satisfied by *sql.DB and *sql.Tx for read-only queries.
type QueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

// PlayerService owns player persistence and time projection. The durable row
// shape lives in playerstate; this type adds the gameplay projection on top.
type PlayerService struct {
	db *sql.DB
}

func NewPlayerService(db *sql.DB) *PlayerService {
	return &PlayerService{db: db}
}

// DB returns the underlying database handle.
func (s *PlayerService) DB() *sql.DB {
	return s.db
}

// Load reads the full player snapshot: canonical row, all Gu, materials and
// the lazy repair.
func (s *PlayerService) Load(q db.Queryer, id uuid.UUID, forUpdate bool) (models.Player, error) {
	return playerstate.Load(q, id, playerstate.Options{ForUpdate: forUpdate, Gu: playerstate.GuFull})
}

// LoadForUpdate loads a player with a row-level lock for mutation flows.
func (s *PlayerService) LoadForUpdate(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return s.Load(tx, id, true)
}

// LoadForUpdateEquipped is like LoadForUpdate but only loads equipped/vital Gus
// (used by light mana evaluation paths where full inventory is not needed).
func (s *PlayerService) LoadForUpdateEquipped(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return s.loadEquipped(tx, id, true)
}

// LoadEquipped loads a player with only equipped/vital Gus populated in
// GuInventory. Used for lightweight read-only status snapshots where the full
// inventory is not needed (the dedicated inventory endpoint serves full data).
func (s *PlayerService) LoadEquipped(q db.Queryer, id uuid.UUID) (models.Player, error) {
	return s.loadEquipped(q, id, false)
}

func (s *PlayerService) loadEquipped(q db.Queryer, id uuid.UUID, forUpdate bool) (models.Player, error) {
	return playerstate.Load(q, id, playerstate.Options{ForUpdate: forUpdate, Gu: playerstate.GuEquipped})
}

// setDefaults initializes default values for empty fields.
func (s *PlayerService) setDefaults(player *models.Player) {
	if player.Lifespan < 0 {
		player.Lifespan = 0
	}
	if player.TimeSpeedMultiplier <= 0 {
		player.TimeSpeedMultiplier = 1.0
	}
}

// applyApertureProjection returns immortal aperture time rate and essence production for Rank >= 6.
// Equipped Gu reductions attenuate the Dao purity penalty, while a permanent
// sovereign body mod removes that penalty entirely.
func (s *PlayerService) applyApertureProjection(q db.Queryer, player *models.Player) (timeRate float64, essenceProd int) {
	timeRate = 1.0
	essenceProd = 10
	if player.Rank < 6 {
		return timeRate, essenceProd
	}
	_ = q.QueryRow(`
		SELECT ia.time_rate, ia.essence_production_per_year
		FROM immortal_apertures ia WHERE ia.player_id = $1
	`, player.ID).Scan(&timeRate, &essenceProd)
	if timeRate <= 0 {
		timeRate = 1.0
	}
	if _, total, purity := gamefx.DaoPurity(player.DaoMarks); total > 0 {
		reductionPct := gamefx.Get(player).DaoConflictReductionPct
		purityMultiplier := gamefx.EcologyPurityFactor(purity)
		effectivePenalty := (1.0 - purityMultiplier) * (1.0 - reductionPct/100.0)
		purityMultiplier = 1.0 - effectivePenalty
		if purityMultiplier < 1.0 {
			essenceProd = int(float64(essenceProd) * purityMultiplier)
			if essenceProd < 1 {
				essenceProd = 1
			}
		}
	}
	return timeRate, essenceProd
}

// isImmortalCultivator checks if the player is an immortal (Rank >= 6).
func isImmortalCultivator(player *models.Player) bool {
	return util.IsImmortalPlayer(player)
}

// normalizeEnergyPool adjusts the mana pool for immortal cultivators.
func normalizeEnergyPool(player *models.Player) bool {
	if !isImmortalCultivator(player) {
		effMax := gamefx.EffectiveMaxMana(player)
		if player.CurrentMana > effMax {
			player.CurrentMana = effMax
			return true
		}
		return false
	}
	unlimitedMaxMana := 99999999999.0
	if player.MaxMana != unlimitedMaxMana {
		player.MaxMana = unlimitedMaxMana
		return true
	}
	return false
}

type automaticRegenRates struct {
	mana    float64
	hp      float64
	thought float64
	soul    float64
}

// worldYearHours is the number of real-world hours in one outside-world year
// when time_speed_multiplier is 1. An immortal aperture advances time_rate
// internal years during that period.
const worldYearHours = 10.0

// calculateAutomaticRegenRates resolves every passive recovery source once,
// then applies the final 0/1 policy supplied by Block-v2 character effects.
// Consumable and other explicit recovery actions do not use these rates, so a
// race that cannot recover naturally can still spend an item to recover.
func calculateAutomaticRegenRates(player *models.Player, essenceProd int) automaticRegenRates {
	var mana float64
	if player.Rank >= 6 {
		// essenceProd is the amount generated per internal aperture year. The
		// aperture time rate is applied later when wall-clock time is projected.
		mana = float64(essenceProd) / worldYearHours
	} else {
		mana = energyRegenRatePerHour(player)
	}
	if player.Lifespan <= 0 {
		mana *= 0.5
	}

	fx := gamefx.Get(player)
	rates := automaticRegenRates{
		mana:    (mana + models.ManaRegenTickPerHour(player.GuInventory, player.Rank)) * (1.0 + fx.ManaRegenBonus),
		hp:      (gamefx.EffectiveMaxHP(player)*0.05 + fx.HPRegenBonus) * (1.0 + fx.HealingReceivedBonus),
		thought: player.RegenRate + models.ThoughtRegenTickPerHour(player.GuInventory, player.Rank),
		soul:    (float64(player.MaxSoul)*0.10)*(1.0+fx.SoulRegenBonus) + models.SoulRegenTickPerHour(player.GuInventory, player.Rank),
	}
	if fx.DisableManaRegen || fx.DisableResourceRegen {
		rates.mana = 0
	}
	if fx.DisableHPRegen {
		rates.hp = 0
	}
	if fx.DisableResourceRegen {
		rates.thought = 0
	}
	if fx.DisableSoulRegen {
		rates.soul = 0
	}
	return rates
}

// applyProjectedStatus computes mana regen, HP regen, lifespan decay, and stress decay,
func applyProjectedStatus(player *models.Player, essenceProd int, timeRate float64, now time.Time) (poolChanged, stressChanged bool) {
	poolChanged = normalizeEnergyPool(player)
	stressChanged = false
	rates := calculateAutomaticRegenRates(player, essenceProd)
	player.ManaRegenRate = rates.mana
	player.HPRegenRate = rates.hp
	player.SoulRegenRate = rates.soul
	player.ThoughtRegenRate = rates.thought
	effMaxHP := gamefx.EffectiveMaxHP(player)

	secondsPassed := now.Sub(player.LastManaUpdate).Seconds()
	if secondsPassed > 0 {
		// Consume the interval even when every automatic rate is zero, so
		// disabled time cannot be paid retroactively after an effect changes.
		poolChanged = true
		equippedBonus := models.EquippedTimeSpeedBonus(player.GuInventory)
		effectiveBase := player.TimeSpeedMultiplier + equippedBonus
		if effectiveBase <= 0 {
			effectiveBase = 1.0
		}
		effectiveMultiplier := effectiveBase
		if player.Rank >= 6 {
			effectiveMultiplier = effectiveBase * timeRate
		}
		subjectiveSeconds := secondsPassed * effectiveMultiplier
		earnedMana := (subjectiveSeconds / 3600.0) * rates.mana
		effMaxMana := gamefx.EffectiveMaxMana(player)
		newMana := player.CurrentMana + earnedMana
		if newMana > effMaxMana {
			newMana = effMaxMana
		}
		if newMana != player.CurrentMana {
			player.CurrentMana = newMana
			poolChanged = true
		}

		if player.HP > 0 && player.HP < effMaxHP {
			earnedHP := (secondsPassed * effectiveBase / 3600.0) * player.HPRegenRate
			newHP := player.HP + earnedHP
			if newHP > effMaxHP {
				newHP = effMaxHP
			}
			if newHP != player.HP {
				player.HP = newHP
				poolChanged = true
			}
		}

		if player.SoulHP < player.MaxSoul {
			soulRegenPerHour := player.SoulRegenRate
			earnedSoulFloat := (secondsPassed * effectiveBase / 3600.0) * soulRegenPerHour
			earnedSoulInt := int(earnedSoulFloat)
			fraction := earnedSoulFloat - float64(earnedSoulInt)
			if fraction > 0 && rand.Float64() < fraction {
				earnedSoulInt += 1
			}
			if earnedSoulInt > 0 {
				player.SoulHP += earnedSoulInt
				if player.SoulHP > player.MaxSoul {
					player.SoulHP = player.MaxSoul
				}
				poolChanged = true
			}
		}

		if player.CurrentThought < player.MaxThought {
			earnedThought := (secondsPassed * effectiveBase / 3600.0) * rates.thought
			newThought := player.CurrentThought + earnedThought
			if newThought > player.MaxThought {
				newThought = player.MaxThought
			}
			if newThought != player.CurrentThought {
				player.CurrentThought = newThought
				poolChanged = true
			}
		}
		if !gamefx.IsAgingImmune(player) {
			yearsPassed := (secondsPassed * effectiveBase) / 36000.0
			player.Lifespan -= yearsPassed
			if player.Lifespan <= 0 {
				player.Lifespan = 0
			}
			poolChanged = true
		}
	}

	if player.ApertureStress > 0 && !player.LastCultivateTime.IsZero() {
		elapsedHours := now.Sub(player.LastCultivateTime).Hours()
		if elapsedHours > 0 {
			cycles := elapsedHours / 2.0
			decayFactor := math.Pow(0.5, cycles)
			newStress := player.ApertureStress * decayFactor
			if newStress < 1 {
				newStress = 0
			}
			if newStress != player.ApertureStress {
				player.ApertureStress = newStress
				poolChanged = true
				stressChanged = true
			}
		}
	}

	return poolChanged, stressChanged
}

// energyRegenRatePerHour computes mana regen rate based on aptitude.
func energyRegenRatePerHour(player *models.Player) float64 {
	var rate float64
	if player.Rank >= 6 {
		rate = 0.5
	} else {
		aptitudeMap := map[string]float64{
			"Giáp": 10.0, "Ất": 8.0, "Bính": 6.0, "Đinh": 4.0,
		}
		rate = aptitudeMap[player.AptitudeType]
		if rate == 0 {
			rate = 4.0
		}
	}
	return rate
}

// ProjectStatus computes the read-only projected status for UI refreshes.
func (s *PlayerService) ProjectStatus(q db.Queryer, id uuid.UUID) (models.Player, error) {
	player, err := s.LoadEquipped(q, id)
	if err != nil {
		return player, err
	}

	s.setDefaults(&player)
	timeRate, essenceProd := s.applyApertureProjection(q, &player)
	applyProjectedStatus(&player, essenceProd, timeRate, time.Now())

	return player, nil
}

// isLockContention reports Postgres lock/statement timeouts and deadlocks.
// Callers use it to skip best-effort persists instead of failing reads.
func isLockContention(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, []string{"55P03", "57014", "40P01", "could not obtain lock", "canceling statement due to"})
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if len(sub) == 0 {
			continue
		}
		if searchSubstr(s, sub) {
			return true
		}
	}
	return false
}

func searchSubstr(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// setFastFailTimeouts bounds lock waits inside an open transaction so one
// contended row fails fast instead of queueing behind statement_timeout (15s).
func setFastFailTimeouts(tx *sql.Tx) error {
	return tryFastFail(tx)
}

// tryFastFail applies per-Tx timeouts, ignoring errors and panics from
// minimal test doubles that only implement the queries they stub, but returning
// real PostgreSQL connection/statement timeout errors so callers don't continue
// on an aborted transaction block (25P02).
func tryFastFail(tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	defer func() {
		_ = recover()
	}()
	_, err := tx.Exec(`SELECT set_config('lock_timeout', '2000', true), set_config('statement_timeout', '8000', true)`)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && (pqErr.Code == "57014" || pqErr.Code == "55P03" || pqErr.Code == "08006" || pqErr.Code == "08001" || pqErr.Code == "25P02") {
			return err
		}
	}
	return nil
}

// ProjectAndPersistBestEffort serves polling reads without holding a write
// lock across the whole request. It projects status lock-free, then attempts a
// short persist; on lock contention the projected values are returned anyway.
func (s *PlayerService) ProjectAndPersistBestEffort(id uuid.UUID, equippedOnly bool) (models.Player, error) {
	if s == nil {
		return models.Player{}, sql.ErrNoRows
	}
	handle := s.db
	if handle == nil {
		handle = db.DB
	}
	if handle == nil {
		return models.Player{}, sql.ErrNoRows
	}
	var player models.Player
	var err error
	if equippedOnly {
		player, err = s.ProjectStatusEquipped(handle, id)
	} else {
		player, err = s.ProjectStatus(handle, id)
	}
	if err != nil {
		return player, err
	}
	// Throttle persists: frontend polls /auth/me and /explore/grid at high
	// frequency. Polling reads (equippedOnly = true) should remain completely
	// lock-free because actual gameplay actions (combat, cultivate, inventory,
	// travel) already persist authoritative mana & status on mutation.
	// This eliminates row-lock contention and DB connection pool starvation.
	if equippedOnly || time.Since(player.LastManaUpdate) < 45*time.Second {
		return player, nil
	}
	tx, err := handle.Begin()
	if err != nil {
		return player, nil
	}
	defer tx.Rollback()
	if err := setFastFailTimeouts(tx); err != nil {
		return player, nil
	}
	if equippedOnly {
		_, err = s.EvaluateAndPersistManaEquipped(tx, id)
	} else {
		_, err = s.EvaluateAndPersistMana(tx, id)
	}
	if err != nil {
		if isLockContention(err) {
			return player, nil
		}
		return player, nil
	}
	if err := tx.Commit(); err != nil {
		return player, nil
	}
	// Re-project so response reflects persisted timestamps without another lock.
	if equippedOnly {
		if p2, err2 := s.ProjectStatusEquipped(handle, id); err2 == nil {
			return p2, nil
		}
	} else {
		if p2, err2 := s.ProjectStatus(handle, id); err2 == nil {
			return p2, nil
		}
	}
	return player, nil
}

// ProjectStatusEquipped is the equipped-only counterpart of ProjectStatus.
func (s *PlayerService) ProjectStatusEquipped(q db.Queryer, id uuid.UUID) (models.Player, error) {
	player, err := s.LoadEquipped(q, id)
	if err != nil {
		return player, err
	}
	s.setDefaults(&player)
	timeRate, essenceProd := s.applyApertureProjection(q, &player)
	applyProjectedStatus(&player, essenceProd, timeRate, time.Now())
	return player, nil
}

// EvaluateAndPersistMana is the authoritative status evaluator for mutation flows.
// It loads with FOR UPDATE, projects time-variant fields, and persists changes.
func (s *PlayerService) EvaluateAndPersistMana(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	if err := setFastFailTimeouts(tx); err != nil {
		return models.Player{}, err
	}
	if err := worker.EvaluatePlayerHungerTx(tx, id); err != nil {
		return models.Player{}, fmt.Errorf("evaluate player hunger: %w", err)
	}

	player, err := s.LoadForUpdate(tx, id)
	if err != nil {
		return player, err
	}

	s.setDefaults(&player)

	timeRate, essenceProd := s.applyApertureProjection(tx, &player)
	now := time.Now()
	projectedChanged, stressChanged := applyProjectedStatus(&player, essenceProd, timeRate, now)
	if projectedChanged {
		player.LastManaUpdate = now
	}
	if stressChanged {
		player.LastCultivateTime = now
	}

	_, err = tx.Exec(`
			UPDATE players
			SET current_mana = $1, max_mana = $2, last_mana_update = $3,
			    lifespan = $4, time_speed_multiplier = $5, aperture_stress = $6,
			    last_cultivate_time = $7, hp = $8, soul_hp = $9, current_thought = $10,
			    version = version + 1
			WHERE id = $11
		`, player.CurrentMana, player.MaxMana, player.LastManaUpdate,
		player.Lifespan, player.TimeSpeedMultiplier, player.ApertureStress,
		player.LastCultivateTime, player.HP, player.SoulHP, player.CurrentThought, player.ID)
	if err != nil {
		return player, err
	}
	player.Version++

	if player.Rank >= 6 {
		// Only write when the projected essence actually changed; this runs on
		// every mana persist and was rewriting the same value millions of times.
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2 AND immortal_essence <> $1`, int(player.CurrentMana), player.ID)
	}

	return player, nil
}

// EvaluateManaEquipped loads a player with only equipped/vital Gus, takes a
// FOR UPDATE lock, and projects time-variant fields (mana/HP regen, lifespan,
// stress, thought) in memory WITHOUT persisting. Intended for callers that
// build a combatant from the projected values but manage their own writes.
func (s *PlayerService) EvaluateManaEquipped(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return s.evaluateManaEquipped(tx, id)
}

// evaluateManaEquipped is the shared projection core for the equipped-only
// evaluators: load with FOR UPDATE, apply defaults, and project status in
// memory. It does not write to the database.
func (s *PlayerService) evaluateManaEquipped(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	if err := worker.EvaluatePlayerHungerTx(tx, id); err != nil {
		return models.Player{}, fmt.Errorf("evaluate player hunger: %w", err)
	}

	player, err := s.LoadForUpdateEquipped(tx, id)
	if err != nil {
		return player, err
	}
	player.IsSparseGuInventory = true

	s.setDefaults(&player)

	timeRate, essenceProd := s.applyApertureProjection(tx, &player)
	now := time.Now()
	projectedChanged, stressChanged := applyProjectedStatus(&player, essenceProd, timeRate, now)
	if projectedChanged {
		player.LastManaUpdate = now
	}
	if stressChanged {
		player.LastCultivateTime = now
	}

	return player, nil
}

// EvaluateAndPersistManaEquipped is like EvaluateAndPersistMana but loads only
// equipped/vital Gus. Used for read-heavy endpoints (auth/me, challenge) where
// shipping the full 12k+ inventory is unnecessary — the frontend fetches it
// lazily via the dedicated inventory endpoint.
func (s *PlayerService) EvaluateAndPersistManaEquipped(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	if err := setFastFailTimeouts(tx); err != nil {
		return models.Player{}, err
	}
	player, err := s.evaluateManaEquipped(tx, id)
	if err != nil {
		return player, err
	}

	_, err = tx.Exec(`
			UPDATE players
			SET current_mana = $1, max_mana = $2, last_mana_update = $3,
			    lifespan = $4, time_speed_multiplier = $5, aperture_stress = $6,
			    last_cultivate_time = $7, hp = $8, soul_hp = $9, current_thought = $10,
			    version = version + 1
			WHERE id = $11
		`, player.CurrentMana, player.MaxMana, player.LastManaUpdate,
		player.Lifespan, player.TimeSpeedMultiplier, player.ApertureStress,
		player.LastCultivateTime, player.HP, player.SoulHP, player.CurrentThought, player.ID)
	if err != nil {
		return player, err
	}
	player.Version++

	if player.Rank >= 6 {
		// Only write when the projected essence actually changed; this runs on
		// every mana persist and was rewriting the same value millions of times.
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2 AND immortal_essence <> $1`, int(player.CurrentMana), player.ID)
	}

	return player, nil
}
