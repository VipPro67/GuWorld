// Package playerstate owns the shape of a durable player snapshot.
//
// Every read of the players row that needs more than one or two columns goes
// through this package: the column list, the scan order, the collection
// revision expression, the Gu/material attachments and the lazy repairs live
// here. Callers choose a view (which collections to attach) and get a typed
// not-found error; they never re-derive the row shape.
package playerstate

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// RevisionExpr is the single definition of the client-visible collection
// revision. It assumes the players table is aliased as p.
const RevisionExpr = `COALESCE(p.state_revision, 0) + COALESCE((
	SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id
), 0)`

// Columns is the one full-width player row projection. Scan below is its only
// consumer; when a column is added or reordered, change both together.
const Columns = `p.id, p.username, p.rank, p.sub_rank, p.aptitude_type, p.aptitude_percent, p.aptitude_grade,
	p.current_mana, p.max_mana, p.primeval_stones, p.immortal_stones,
	p.wall_progress, p.human_qi, p.next_tribulation_time,
	p.hp, p.base_hp, p.body_mods, p.x, p.y,
	p.equipped_gus, p.cleared_cells, p.visited_subzones, p.current_region, COALESCE(p.world_zone_id, ''), p.path_masteries,
	p.active_combat, p.last_mana_update, COALESCE(p.last_hunger_tick_at, p.created_at, NOW()), p.created_at, p.lifespan, p.time_speed_multiplier,
	p.party_id, p.arena_rank, COALESCE(p.email, ''), p.google_id,
	p.explore_count, p.last_explore_reset, p.aperture_stress,
	p.last_cultivate_time, p.soul_hp, p.max_soul, p.soul_tier, p.needs_soul_refine,
	COALESCE(p.bio, ''), COALESCE(p.fake_rank, 1),
	COALESCE(p.fake_sub_rank, 1), COALESCE(p.fake_vital_gu, ''),
	COALESCE(p.is_disguised, false), COALESCE(p.is_admin, false),
	COALESCE(p.ref_code, ''), COALESCE(p.avatar_url, ''), p.combat_settings,
	COALESCE(p.needs_onboarding, false),
	COALESCE(p.current_thought, 100.0), COALESCE(p.max_thought, 100.0), COALESCE(p.thought_regen_rate, 1.0),
	COALESCE(p.immortal_state, '{}'), p.growth_data,
	COALESCE(p.race, ''), COALESCE(p.gender, ''), COALESCE(p.constitution, ''), p.statuses,
	COALESCE(p.version, 0), ` + RevisionExpr

// ColumnCount is the number of destinations Scan expects. Keeping it next to
// Columns lets a DB-free test catch column/scan drift at build time.
const ColumnCount = 64

// GuScope selects which Gu instances the snapshot attaches.
type GuScope int

const (
	// GuNone attaches no Gu inventory. Use for callers that only need the row.
	GuNone GuScope = iota
	// GuEquipped attaches equipped and vital Gu instances only.
	GuEquipped
	// GuFull attaches the whole inventory.
	GuFull
)

// Options selects the snapshot view. Materials and the lazy repairs are always
// included: they are part of what "load this player" has always meant, and a
// read that skipped them would silently answer a different question.
type Options struct {
	// ForUpdate takes a row lock for the duration of the caller's transaction.
	ForUpdate bool
	// Gu selects the Gu inventory scope.
	Gu GuScope
}

// IsNotFound reports whether a snapshot load missed the player row. It is the
// one not-found check callers should use; the concrete error stays
// sql.ErrNoRows so existing handlers keep their 404 branches.
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// RowScanner is satisfied by *sql.Row and *sql.Rows.
type RowScanner interface {
	Scan(dest ...any) error
}

// Scan maps the canonical Columns projection. activeCombat is returned raw so
// callers can decide whether to decode it (the repair path may drop it).
func Scan(row RowScanner) (models.Player, []byte, error) {
	var player models.Player
	var activeCombat []byte
	err := row.Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y,
		&player.EquippedGus, &player.ClearedCells, &player.VisitedSubZones,
		&player.CurrentRegion, &player.WorldZoneID, &player.PathMasteries, &activeCombat,
		&player.LastManaUpdate, &player.LastHungerTickAt, &player.CreatedAt, &player.Lifespan, &player.TimeSpeedMultiplier,
		&player.PartyID, &player.ArenaRank, &player.Email, &player.GoogleID,
		&player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine,
		&player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.AvatarURL,
		&player.CombatSettings, &player.NeedsOnboarding,
		&player.CurrentThought, &player.MaxThought, &player.RegenRate,
		&player.ImmortalState, &player.GrowthData,
		&player.Race, &player.Gender, &player.Constitution, &player.Statuses,
		&player.Version, &player.StateRevision,
	)
	return player, activeCombat, err
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// Load returns the durable player snapshot for id. Collections are selected by
// opts; the row is always read from the canonical projection.
func Load(q db.Queryer, id uuid.UUID, opts Options) (models.Player, error) {
	var player models.Player
	if q == nil {
		return player, sql.ErrNoRows
	}

	query := "SELECT " + Columns + " FROM players p WHERE p.id = $1"
	if opts.ForUpdate {
		query += " FOR UPDATE"
	}
	var activeCombatBytes []byte
	player, activeCombatBytes, err := Scan(q.QueryRow(query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return player, sql.ErrNoRows
		}
		return player, fmt.Errorf("load player snapshot: %w", err)
	}

	hydrateDaoMarks(q, &player)
	hydrateActiveCombat(q, &player, activeCombatBytes)
	loadMaterials(q, &player)
	if RepairExpiredStatuses(q, &player) {
		if rev, revErr := Revision(q, id); revErr == nil {
			player.StateRevision = rev
		}
	}
	if opts.Gu != GuNone {
		loadGu(q, &player, opts.Gu)
	}
	return player, nil
}

func hydrateDaoMarks(q db.Queryer, player *models.Player) {
	player.DaoMarks = models.DaoMarksMap{"strength": 0}
	if player.Rank < 6 {
		return
	}
	var marks models.DaoMarksMap
	if err := q.QueryRow(`SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, player.ID).Scan(&marks); err == nil {
		player.DaoMarks = marks
	}
}

// ShouldClearActiveCombat reports whether a party member's persisted combat is
// stale: only the party leader owns a shared encounter. Kept pure so the rule
// is testable without a transaction.
func ShouldClearActiveCombat(playerID uuid.UUID, partyID *uuid.UUID, leaderID string) bool {
	return partyID != nil && leaderID != "" && leaderID != playerID.String()
}

func hydrateActiveCombat(q db.Queryer, player *models.Player, raw []byte) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	var cs models.CombatState
	if err := json.Unmarshal(raw, &cs); err != nil {
		logger.S.Warnf("load player snapshot: failed to unmarshal active_combat for player=%s: %v", player.ID, err)
		return
	}
	if cs.IsEmpty() {
		return
	}
	if player.PartyID != nil {
		var leaderID string
		err := q.QueryRow(`SELECT leader_id FROM parties WHERE id = $1`, player.PartyID).Scan(&leaderID)
		if ShouldClearActiveCombat(player.ID, player.PartyID, leaderID) {
			player.ActiveCombat = nil
			if exec, ok := q.(execer); ok {
				_, _ = exec.Exec(`UPDATE players SET active_combat = NULL WHERE id = $1`, player.ID)
			}
			return
		}
		if err != nil {
			// A missing party row means the combat cannot belong to a shared
			// encounter any more; keep the current "no leader, keep combat"
			// behavior for transient errors.
			logger.S.Warnf("load player snapshot: party lookup failed for player=%s: %v", player.ID, err)
		}
	}
	player.ActiveCombat = &cs
}

func loadMaterials(q db.Queryer, player *models.Player) {
	rows, err := q.Query(`SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1`, player.ID.String())
	if err != nil {
		return
	}
	defer rows.Close()
	mi := make(models.MaterialsInventory)
	for rows.Next() {
		var itemID string
		var qty int64
		if rows.Scan(&itemID, &qty) == nil {
			mi[itemID] += qty
		}
	}
	player.MaterialsInventory = mi
}

func loadGu(q db.Queryer, player *models.Player, scope GuScope) {
	inventory, err := db.LoadPlayerGus(q, player.ID.String(), scope == GuEquipped)
	if err != nil {
		return
	}
	player.GuInventory = inventory
}

// ExpiredStatusUpdate decides whether a snapshot has expired statuses and
// returns the before/after JSON used for the compare-and-set write. Pure, so
// the expiry rule is testable without a database.
func ExpiredStatusUpdate(player *models.Player, now time.Time) (original, cleaned any, changed bool) {
	original, err := player.Statuses.Value()
	if err != nil || !player.Statuses.Expire(now) {
		return nil, nil, false
	}
	cleaned, err = player.Statuses.Value()
	if err != nil {
		return nil, nil, false
	}
	return original, cleaned, true
}

// RepairExpiredStatuses lazily persists expired status cleanup for a player
// that is actually loaded by a request. The original JSON condition prevents a
// stale read from overwriting a status granted concurrently. It reports
// whether the row was written (callers then re-read the revision).
func RepairExpiredStatuses(q db.Queryer, player *models.Player) bool {
	exec, ok := q.(execer)
	if !ok {
		return false
	}
	original, cleaned, changed := ExpiredStatusUpdate(player, time.Now())
	if !changed {
		return false
	}
	_, err := exec.Exec(`
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

// Revision reads the client-visible collection revision through one query.
func Revision(q db.Queryer, id uuid.UUID) (int64, error) {
	var revision int64
	err := q.QueryRow("SELECT "+RevisionExpr+" FROM players p WHERE p.id = $1", id).Scan(&revision)
	return revision, err
}

// RefreshResponse rewrites the response-facing fields of player from one MVCC
// snapshot, including the collection revision. It deliberately loads a narrow
// view instead of the full row: action responses need fresh scalars, not the
// collections.
func RefreshResponse(q db.Queryer, player *models.Player) error {
	return q.QueryRow(`
		SELECT `+RevisionExpr+`, p.primeval_stones, p.immortal_stones, p.human_qi,
		p.hp, p.base_hp, p.current_mana, p.max_mana, p.current_thought, p.soul_hp, p.lifespan,
		p.aptitude_percent, p.sub_rank, p.rank, p.wall_progress
		FROM players p WHERE p.id = $1
	`, player.ID).Scan(
		&player.StateRevision, &player.PrimevalStones,
		&player.ImmortalStones, &player.HumanQi,
		&player.HP, &player.BaseHP, &player.CurrentMana, &player.MaxMana,
		&player.CurrentThought, &player.SoulHP, &player.Lifespan,
		&player.AptitudePercent, &player.SubRank, &player.Rank, &player.WallProgress,
	)
}
