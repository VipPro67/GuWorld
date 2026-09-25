package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/util"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// QueryRower is satisfied by *sql.DB and *sql.Tx for read-only queries.
type QueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// PlayerService handles player persistence and state management.
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

// Load reads a full player row from the database.
func (s *PlayerService) Load(q QueryRower, id uuid.UUID, forUpdate bool) (models.Player, error) {
	player, err := s.loadRow(q, id, forUpdate)
	if err != nil {
		return player, err
	}
	if s.cleanExpiredStatuses(q, &player) {
		_ = q.QueryRow(`SELECT p.state_revision + COALESCE((SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id), 0) FROM players p WHERE p.id = $1`, id).Scan(&player.StateRevision)
	}

	if s.db != nil {
		var qy db.Queryer
		if tx, ok := q.(db.Queryer); ok {
			qy = tx
		} else {
			qy = s.db
		}
		inv, err := db.LoadPlayerGus(qy, id.String(), false)
		if err == nil {
			player.GuInventory = inv
		}
	}
	return player, nil
}

// LoadForUpdate loads a player with a row-level lock for mutation flows.
func (s *PlayerService) LoadForUpdate(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return s.Load(tx, id, true)
}

// LoadForUpdateEquipped is like LoadForUpdate but only loads equipped/vital Gus
// (used by light mana evaluation paths where full inventory is not needed).
func (s *PlayerService) LoadForUpdateEquipped(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	player, err := s.loadRow(tx, id, true)
	if err != nil {
		return player, err
	}
	if s.cleanExpiredStatuses(tx, &player) {
		_ = tx.QueryRow(`SELECT p.state_revision + COALESCE((SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id), 0) FROM players p WHERE p.id = $1`, id).Scan(&player.StateRevision)
	}
	if s.db != nil {
		if inv, errR := db.LoadPlayerGus(tx, id.String(), true); errR == nil {
			player.GuInventory = inv
		}
	}
	return player, nil
}

// LoadEquipped loads a player with only equipped/vital Gus populated in
// GuInventory. Used for lightweight read-only status snapshots where the full
// inventory is not needed (the dedicated inventory endpoint serves full data).
func (s *PlayerService) LoadEquipped(q QueryRower, id uuid.UUID) (models.Player, error) {
	player, err := s.loadRow(q, id, false)
	if err != nil {
		return player, err
	}
	if s.cleanExpiredStatuses(q, &player) {
		_ = q.QueryRow(`SELECT p.state_revision + COALESCE((SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id), 0) FROM players p WHERE p.id = $1`, id).Scan(&player.StateRevision)
	}
	if s.db != nil {
		var qy db.Queryer
		if tx, ok := q.(db.Queryer); ok {
			qy = tx
		} else {
			qy = s.db
		}
		if inv, err := db.LoadPlayerGus(qy, id.String(), true); err == nil {
			player.GuInventory = inv
		}
	}
	return player, nil
}

// cleanExpiredStatuses lazily persists status cleanup for players that are
// actually loaded by a request. The original JSON condition prevents a stale
// read from overwriting a status granted concurrently.
func (s *PlayerService) cleanExpiredStatuses(q QueryRower, player *models.Player) bool {
	exec, ok := q.(execer)
	if !ok {
		return false
	}
	original, err := player.Statuses.Value()
	if err != nil || !player.Statuses.Expire(time.Now()) {
		return false
	}

	cleaned, err := player.Statuses.Value()
	if err != nil {
		logger.S.Warnf("Failed to serialize expired statuses for player %s: %v", player.ID, err)
		return false
	}
	_, err = exec.Exec(`
		UPDATE players
		SET statuses = $1
		WHERE id = $2 AND statuses = $3
	`, cleaned, player.ID, original)
	if err != nil {
		logger.S.Warnf("Failed to lazily clean expired statuses for player %s: %v", player.ID, err)
		return false
	}
	return true
}

// loadRow loads the player row, active combat state and materials inventory,
// but not the Gu inventory (callers decide equipped-only vs full).
func (s *PlayerService) loadRow(q QueryRower, id uuid.UUID, forUpdate bool) (models.Player, error) {
	var player models.Player

	query := `
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade,
		       current_mana, max_mana, primeval_stones, immortal_stones,
		       wall_progress, human_qi, next_tribulation_time,
		       hp, base_hp, body_mods, x, y,
		       equipped_gus, cleared_cells, visited_subzones, current_region, path_masteries,
		       active_combat, last_mana_update, COALESCE(last_hunger_tick_at, created_at, NOW()), created_at, lifespan, time_speed_multiplier,
		       party_id, arena_rank, COALESCE(email, ''), google_id,
		       explore_count, last_explore_reset, aperture_stress,
		       last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine,
		       COALESCE(bio, ''), COALESCE(fake_rank, 1),
		       COALESCE(fake_sub_rank, 1), COALESCE(fake_vital_gu, ''),
		       COALESCE(is_disguised, false), COALESCE(is_admin, false),
		       COALESCE(ref_code, ''), COALESCE(avatar_url, ''), combat_settings,
		       COALESCE(needs_onboarding, false),
		       COALESCE(current_thought, 100.0), COALESCE(max_thought, 100.0), COALESCE(thought_regen_rate, 1.0),
   COALESCE(immortal_state, '{}'), growth_data,
		       COALESCE(race, ''), COALESCE(gender, ''), COALESCE(constitution, ''), COALESCE(is_dead, false), statuses,
		       COALESCE(p.version, 0), COALESCE(p.state_revision, 0) + COALESCE((
		           SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id
		       ), 0)
FROM players p
		WHERE id = $1
	`
	if forUpdate {
		query += " FOR UPDATE"
	}

	var activeCombatBytes []byte
	err := q.QueryRow(query, id).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y,
		&player.EquippedGus, &player.ClearedCells, &player.VisitedSubZones,
		&player.CurrentRegion, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.LastHungerTickAt, &player.CreatedAt, &player.Lifespan, &player.TimeSpeedMultiplier,
		&player.PartyID, &player.ArenaRank, &player.Email, &player.GoogleID,
		&player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine,
		&player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.AvatarURL,
		&player.CombatSettings,
		&player.NeedsOnboarding,
		&player.CurrentThought, &player.MaxThought, &player.RegenRate,
		&player.ImmortalState,
		&player.GrowthData,
		&player.Race, &player.Gender, &player.Constitution, &player.IsDead, &player.Statuses,
		&player.Version, &player.StateRevision,
	)
	if err != nil {
		return player, err
	}

	if player.Rank >= 6 {
		var daoMarks models.DaoMarksMap
		errDao := q.QueryRow(`SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, player.ID).Scan(&daoMarks)
		if errDao == nil {
			player.DaoMarks = daoMarks
		} else {
			player.DaoMarks = models.DaoMarksMap{"strength": 0}
		}
	} else {
		player.DaoMarks = models.DaoMarksMap{"strength": 0}
	}

	if len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if errU := json.Unmarshal(activeCombatBytes, &cs); errU == nil {
			if !cs.IsEmpty() {
				if player.PartyID != nil {
					var leaderID string
					errL := q.QueryRow(`SELECT leader_id FROM parties WHERE id = $1`, player.PartyID).Scan(&leaderID)
					if errL == nil && leaderID != player.ID.String() {
						player.ActiveCombat = nil
						if execer, ok := q.(interface {
							Exec(query string, args ...any) (sql.Result, error)
						}); ok {
							_, _ = execer.Exec(`UPDATE players SET active_combat = NULL WHERE id = $1`, player.ID)
						}
					} else {
						player.ActiveCombat = &cs
					}
				} else {
					player.ActiveCombat = &cs
				}
			}
		} else {
			logger.S.Warnf("LoadForUpdate: failed to unmarshal active_combat for player=%s: %v", player.ID, errU)
		}
	}

	var inventoryQuery db.Queryer
	if qy, ok := q.(db.Queryer); ok {
		inventoryQuery = qy
	} else {
		inventoryQuery = s.db
	}
	if inventoryQuery != nil {
		invRows, err := inventoryQuery.Query(`
			SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1
		`, id.String())
		if err == nil {
			defer invRows.Close()
			mi := make(models.MaterialsInventory)
			for invRows.Next() {
				var itemID string
				var qty int64
				if invRows.Scan(&itemID, &qty) == nil {
					mi[itemID] += qty
				}
			}
			player.MaterialsInventory = mi
		}
	}
	return player, nil
}

// setDefaults initializes default values for empty fields.
func (s *PlayerService) setDefaults(player *models.Player) {
	dummyP := models.Player{}
	dummyP.Race = player.Race
	if gamefx.Get(&dummyP).Immortal {
		player.IsDead = false
	} else if player.Lifespan <= 0 {
		player.Lifespan = 0
		player.IsDead = true
	}
	if player.TimeSpeedMultiplier <= 0 {
		player.TimeSpeedMultiplier = 1.0
	}
}

// applyApertureProjection returns immortal aperture time rate and essence production for Rank >= 6.
func (s *PlayerService) applyApertureProjection(q QueryRower, player *models.Player) (timeRate float64, essenceProd int) {
	timeRate = 1.0
	essenceProd = 10
	if player.Rank < 6 {
		return timeRate, essenceProd
	}
	_ = q.QueryRow(`SELECT time_rate, essence_production_per_year FROM immortal_apertures WHERE player_id = $1`, player.ID).Scan(&timeRate, &essenceProd)
	if timeRate <= 0 {
		timeRate = 1.0
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

// applyProjectedStatus computes mana regen, HP regen, lifespan decay, and stress decay,
func applyProjectedStatus(player *models.Player, essenceProd int, timeRate float64, now time.Time) (poolChanged, stressChanged bool) {
	poolChanged = normalizeEnergyPool(player)
	stressChanged = false

	var regenRatePerHour float64
	if player.Rank >= 6 {
		subRank := player.SubRank
		if subRank < 1 {
			subRank = 1
		}
		if subRank > 4 {
			subRank = 4
		}
		subRankFactor := 1.0 + float64(subRank-1)*0.15
		regenRatePerHour = (float64(essenceProd) / 120.0) * subRankFactor
	} else {
		regenRatePerHour = energyRegenRatePerHour(player)
	}

	if player.Lifespan <= 0 {
		regenRatePerHour = regenRatePerHour * 0.5
	}

	regenRatePerHour += models.ManaRegenTickPerHour(player.GuInventory, player.Rank)

	player.RegenRate += models.ThoughtRegenTickPerHour(player.GuInventory, player.Rank)

	player.ManaRegenRate = regenRatePerHour * (1.0 + gamefx.Get(player).ManaRegenBonus)

	effMaxHP := gamefx.EffectiveMaxHP(player)
	player.HPRegenRate = (effMaxHP*0.05 + gamefx.Get(player).HPRegenBonus) * (1.0 + gamefx.Get(player).HealingReceivedBonus)

	player.SoulRegenRate = (float64(player.MaxSoul)*0.10)*(1.0+gamefx.Get(player).SoulRegenBonus) + models.SoulRegenTickPerHour(player.GuInventory, player.Rank)

	secondsPassed := now.Sub(player.LastManaUpdate).Seconds()
	if secondsPassed > 0 {
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
		earnedMana := (subjectiveSeconds / 3600.0) * regenRatePerHour * (1.0 + gamefx.Get(player).ManaRegenBonus)
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
			earnedThought := (secondsPassed * effectiveBase / 3600.0) * player.RegenRate
			newThought := player.CurrentThought + earnedThought
			if newThought > player.MaxThought {
				newThought = player.MaxThought
			}
			if newThought != player.CurrentThought {
				player.CurrentThought = newThought
				poolChanged = true
			}
		}

		dummyP := models.Player{}
		dummyP.Race = player.Race
		if !gamefx.Get(&dummyP).Immortal {
			yearsPassed := (secondsPassed * effectiveBase) / 36000.0
			player.Lifespan -= yearsPassed
			if player.Lifespan <= 0 {
				player.Lifespan = 0
				player.IsDead = true
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
	return rate * (1.0 + gamefx.Get(player).ManaRegenBonus)
}

// ProjectStatus computes the read-only projected status for UI refreshes.
func (s *PlayerService) ProjectStatus(q QueryRower, id uuid.UUID) (models.Player, error) {
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
			    is_dead = $11, version = version + 1
			WHERE id = $12
		`, player.CurrentMana, player.MaxMana, player.LastManaUpdate,
		player.Lifespan, player.TimeSpeedMultiplier, player.ApertureStress,
		player.LastCultivateTime, player.HP, player.SoulHP, player.CurrentThought, player.IsDead, player.ID)
	if err != nil {
		return player, err
	}
	player.Version++

	if player.Rank >= 6 {
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)
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
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)
	}

	return player, nil
}
