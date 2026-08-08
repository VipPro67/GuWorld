package services

import (
	"database/sql"
	"encoding/json"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// QueryRower is satisfied by *sql.DB and *sql.Tx for read-only queries.
type QueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
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
// The Gu inventory load runs in parallel with the row load for better performance.
func (s *PlayerService) Load(q QueryRower, id uuid.UUID, forUpdate bool) (models.Player, error) {
	// Start Gu inventory load in background
	var (
		inv     models.GuInventory
		invErr  error
		invOnce sync.Once
		invDone = make(chan struct{})
	)
	if s.db != nil {
		go func() {
			defer close(invDone)
			g, err := db.LoadPlayerGus(s.db, id.String(), false)
			invOnce.Do(func() {
				inv = g
				invErr = err
			})
		}()
	}

	player, err := s.loadRow(q, id, forUpdate)
	if err != nil {
		if s.db != nil {
			<-invDone
		}
		return player, err
	}

	if s.db != nil {
		<-invDone
		if invErr == nil {
			player.GuInventory = inv
		}
	}
	return player, nil
}

// LoadForUpdate loads a player with a row-level lock for mutation flows.
func (s *PlayerService) LoadForUpdate(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	return s.Load(tx, id, true)
}

// LoadEquipped loads a player with only equipped/vital Gus populated in
// GuInventory. Used for lightweight read-only status snapshots where the full
// inventory is not needed (the dedicated inventory endpoint serves full data).
func (s *PlayerService) LoadEquipped(q QueryRower, id uuid.UUID) (models.Player, error) {
	player, err := s.loadRow(q, id, false)
	if err != nil {
		return player, err
	}
	if s.db != nil {
		if inv, err := db.LoadPlayerGus(s.db, id.String(), true); err == nil {
			player.GuInventory = inv
		}
	}
	return player, nil
}

// loadRow loads the player row, active combat state and materials inventory,
// but not the Gu inventory (callers decide equipped-only vs full).
func (s *PlayerService) loadRow(q QueryRower, id uuid.UUID, forUpdate bool) (models.Player, error) {
	var player models.Player

	query := `
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade,
		       current_mana, max_mana, primeval_stones, immortal_stones,
		       wall_progress, human_qi, next_tribulation_time,
		       hp, base_hp, body_mods, x, y, explored_cells,
		       equipped_gus, cleared_cells, current_region, path_masteries,
		       active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier,
		       party_id, arena_rank, COALESCE(email, ''), google_id,
		       temp_loot, explore_count, last_explore_reset, aperture_stress,
		       last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine,
		       COALESCE(bio, ''), COALESCE(fake_rank, 1),
		       COALESCE(fake_sub_rank, 1), COALESCE(fake_vital_gu, ''),
		       COALESCE(is_disguised, false), COALESCE(is_admin, false),
		       COALESCE(ref_code, ''), COALESCE(avatar_url, ''), combat_settings,
		       COALESCE(needs_onboarding, false),
		       COALESCE(current_thought, 100.0), COALESCE(max_thought, 100.0), COALESCE(thought_regen_rate, 1.0),
   COALESCE(immortal_state, '{}'), growth_data,
		       COALESCE(race, ''), COALESCE(gender, ''), COALESCE(constitution, ''), statuses
FROM players
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
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt, &player.Lifespan, &player.TimeSpeedMultiplier,
		&player.PartyID, &player.ArenaRank, &player.Email, &player.GoogleID,
		&player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine,
		&player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.AvatarURL,
		&player.CombatSettings,
		&player.NeedsOnboarding,
		&player.CurrentThought, &player.MaxThought, &player.RegenRate,
		&player.ImmortalState,
		&player.GrowthData,
		&player.Race, &player.Gender, &player.Constitution, &player.Statuses,
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
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	if s.db != nil {
		invRows, err := s.db.Query(`
			SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1
		`, id.String())
		if err == nil {
			defer invRows.Close()
			mi := make(map[string]int)
			for invRows.Next() {
				var itemID string
				var qty int
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
	if player.Lifespan <= 0 {
		player.Lifespan = 100.0
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
	return player.Rank >= 6
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

	player.ManaRegenRate = regenRatePerHour * (1.0 + gamefx.Get(player).ManaRegenBonus)

	secondsPassed := now.Sub(player.LastManaUpdate).Seconds()
	if secondsPassed > 0 {
		effectiveMultiplier := player.TimeSpeedMultiplier
		if player.Rank >= 6 {
			effectiveMultiplier = player.TimeSpeedMultiplier * timeRate
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

		if player.HP > 0 && player.HP < player.BaseHP {
			hpRegenPerHour := player.BaseHP * 0.05 * (1.0 + gamefx.Get(player).HealingReceivedBonus)
			earnedHP := (secondsPassed * player.TimeSpeedMultiplier / 3600.0) * hpRegenPerHour
			newHP := player.HP + earnedHP
			if newHP > player.BaseHP {
				newHP = player.BaseHP
			}
			if newHP != player.HP {
				player.HP = newHP
				poolChanged = true
			}
		}

		if player.SoulHP > 0 && player.SoulHP < player.MaxSoul {
			soulRegenPerHour := float64(player.MaxSoul) * 0.10
			earnedSoulFloat := (secondsPassed * player.TimeSpeedMultiplier / 3600.0) * soulRegenPerHour
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
			earnedThought := (secondsPassed * player.TimeSpeedMultiplier / 3600.0) * player.RegenRate
			newThought := player.CurrentThought + earnedThought
			if newThought > player.MaxThought {
				newThought = player.MaxThought
			}
			if newThought != player.CurrentThought {
				player.CurrentThought = newThought
				poolChanged = true
			}
		}

		yearsPassed := (secondsPassed * player.TimeSpeedMultiplier) / 36000.0
		player.Lifespan -= yearsPassed
		if player.Lifespan < 0 {
			player.Lifespan = 0
		}
		poolChanged = true
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
			"Giáp": 8.0, "Ất": 6.0, "Bính": 4.0, "Đinh": 2.0,
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
		    last_cultivate_time = $7, hp = $8, soul_hp = $9, current_thought = $10
		WHERE id = $11
	`, player.CurrentMana, player.MaxMana, player.LastManaUpdate,
		player.Lifespan, player.TimeSpeedMultiplier, player.ApertureStress,
		player.LastCultivateTime, player.HP, player.SoulHP, player.CurrentThought, player.ID)
	if err != nil {
		return player, err
	}

	if player.Rank >= 6 {
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)
	}

	return player, nil
}
