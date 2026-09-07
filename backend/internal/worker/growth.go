package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func StartGrowthWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Growth Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Growth Worker stopped")
				return
			case <-ticker.C:
				processGrowthTick(ctx, db)
			}
		}
	}()
}

// processGrowthTick advances Gu growth for every equipped ACTIVE Gu that has a
// growth profile. Instead of one transaction per player, it batch-reads all
// affected data in two queries, processes in Go, then writes back with two
// batch UPDATEs via unnest().
func processGrowthTick(ctx context.Context, db *sql.DB) {
	// ── Step 1: find affected player IDs ──────────────────────────────────
	playerIDs, err := findGrowthAffectedPlayers(db)
	if err != nil {
		logger.S.Errorf("Growth Worker Error finding affected players: %v", err)
		return
	}
	if len(playerIDs) == 0 {
		return
	}

	pqIDs := pq.Array(playerIDs)

	// ── Step 2: batch-fetch player metadata ───────────────────────────────
	type playerRow struct {
		GrowthData []byte
		BaseHP     float64
		TimeSpeed  float64
	}
	playerMap := make(map[string]*playerRow, len(playerIDs))

	pRows, err := db.Query(`
		SELECT id, growth_data, base_hp, time_speed_multiplier
		FROM players WHERE id = ANY($1::uuid[])`, pqIDs)
	if err != nil {
		logger.S.Errorf("Growth Worker Error fetching players: %v", err)
		return
	}
	defer pRows.Close()

	for pRows.Next() {
		var (
			id  string
			pr  playerRow
			gd  sql.NullString
		)
		if err := pRows.Scan(&id, &gd, &pr.BaseHP, &pr.TimeSpeed); err != nil {
			logger.S.Errorf("Growth Worker Error scanning player row: %v", err)
			continue
		}
		if gd.Valid {
			pr.GrowthData = []byte(gd.String)
		}
		playerMap[id] = &pr
	}
	if err := pRows.Err(); err != nil {
		logger.S.Errorf("Growth Worker Error iterating players: %v", err)
		return
	}

	// ── Step 3: batch-fetch all player_gu rows ───────────────────────────
	type guRow struct {
		PlayerID    string
		InstanceID  string
		GuID        string
		Rank        int
		State       string
		Equipped    bool
		IsVital     bool
		IntegrTime  float64
		EquippedAt  *time.Time
		GrowthHours float64
	}

	guByPlayer := make(map[string][]guRow)

	gRows, err := db.Query(`
		SELECT player_id::text, id, gu_id, gu_rank, state, equipped,
		       is_vital, integration_time, equipped_at, growth_hours
		FROM player_gu WHERE player_id = ANY($1::uuid[])`, pqIDs)
	if err != nil {
		logger.S.Errorf("Growth Worker Error fetching player_gu: %v", err)
		return
	}
	defer gRows.Close()

	for gRows.Next() {
		var g guRow
		if err := gRows.Scan(
			&g.PlayerID, &g.InstanceID, &g.GuID, &g.Rank, &g.State,
			&g.Equipped, &g.IsVital, &g.IntegrTime, &g.EquippedAt, &g.GrowthHours,
		); err != nil {
			continue
		}
		guByPlayer[g.PlayerID] = append(guByPlayer[g.PlayerID], g)
	}
	if err := gRows.Err(); err != nil {
		logger.S.Errorf("Growth Worker Error iterating player_gu: %v", err)
		return
	}

	// ── Step 4: process growth in Go (same logic, no DB access) ──────────
	type guUpdate struct {
		InstanceID  string
		GrowthHours float64
		EquippedAt  time.Time
	}
	type playerUpdate struct {
		GrowthData string
		BaseHP     float64
	}

	var (
		guUpdates      []guUpdate
		playerUpdates  []playerUpdate
		playerUpdateIDs []string
	)

	now := time.Now()

	type typeKey struct {
		guID string
		rank int
	}
	type growthGroup struct {
		spec     models.GuDictionaryItem
		active   []int
		maxHours float64
	}

	for pID, pRow := range playerMap {
		guList := guByPlayer[pID]
		if len(guList) == 0 {
			continue
		}

		// Build GuInventory from rows.
		guInv := make(models.GuInventory, len(guList))
		for i, g := range guList {
			guInv[i] = models.GuInstance{
				InstanceID:  g.InstanceID,
				GuID:        g.GuID,
				Rank:        g.Rank,
				Status:      g.State,
				Equipped:    g.Equipped,
				IsVital:     g.IsVital,
				IntegrationTime: g.IntegrTime,
				EquippedAt:  g.EquippedAt,
				GrowthHours: g.GrowthHours,
			}
		}

		timeSpeed := pRow.TimeSpeed
		if timeSpeed <= 0 {
			timeSpeed = 1.0
		}
		timeSpeed += models.EquippedTimeSpeedBonus(guInv)

		var growthData models.GrowthData
		if err := json.Unmarshal(pRow.GrowthData, &growthData); err != nil {
			growthData = models.GrowthData{
				Boosts:     models.GrowthBoosts{},
				FullyGrown: []models.FullyGrownInfo{},
			}
		}

		groups := make(map[typeKey]*growthGroup)

		for i := range guInv {
			gu := &guInv[i]
			spec, ok := models.GetGu(gu.GuID, gu.Rank)
			if !ok || spec.Effects.Growth == nil || spec.Effects.Growth.HoursToMax <= 0 {
				continue
			}
			key := typeKey{guID: gu.GuID, rank: gu.Rank}
			g, exists := groups[key]
			if !exists {
				g = &growthGroup{spec: spec}
				groups[key] = g
			}
			if gu.GrowthHours > g.maxHours {
				g.maxHours = gu.GrowthHours
			}
			if gu.Equipped && gu.Status == "ACTIVE" {
				g.active = append(g.active, i)
			}
		}

		hasGrowthGu := false
		baseHP := pRow.BaseHP

		for _, g := range groups {
			spec := g.spec
			hasGrowthGu = true

			typeDone := false
			for _, fg := range growthData.FullyGrown {
				if fg.GuID == spec.ID && fg.Rank == spec.Rank {
					typeDone = true
					break
				}
			}
			if typeDone || len(g.active) == 0 {
				continue
			}

			// Initialize EquippedAt for legacy Gus.
			for _, idx := range g.active {
				gu := &guInv[idx]
				if gu.EquippedAt == nil || gu.EquippedAt.IsZero() {
					gu.EquippedAt = &now
					guUpdates = append(guUpdates, guUpdate{
						InstanceID:  gu.InstanceID,
						GrowthHours: gu.GrowthHours,
						EquippedAt:  now,
					})
				}
			}

			// Carrier = the active instance holding the type's progress.
			carrier := g.active[0]
			for _, idx := range g.active[1:] {
				if guInv[idx].GrowthHours > guInv[carrier].GrowthHours {
					carrier = idx
				}
			}
			gu := &guInv[carrier]

			// Bump non-carrier active anchors.
			for _, idx := range g.active {
				if idx == carrier {
					continue
				}
				guInv[idx].EquippedAt = &now
				guUpdates = append(guUpdates, guUpdate{
					InstanceID:  guInv[idx].InstanceID,
					GrowthHours: guInv[idx].GrowthHours,
					EquippedAt:  now,
				})
			}

			elapsed := now.Sub(*gu.EquippedAt).Hours() * timeSpeed
			if elapsed <= 0 {
				continue
			}

			if gu.GrowthHours < g.maxHours {
				gu.GrowthHours = g.maxHours
			}

			atk, def, hpBoost, spd := ApplyGrowthProgressWorker(gu, spec, elapsed)
			if atk != 0 || def != 0 || hpBoost != 0 || spd != 0 {
				growthData.Boosts.Attack += atk
				growthData.Boosts.Defense += def
				growthData.Boosts.HP += hpBoost
				growthData.Boosts.Speed += spd
				baseHP += hpBoost
			}

			if gu.GrowthHours >= spec.Effects.Growth.HoursToMax {
				growthData.FullyGrown = append(growthData.FullyGrown, models.FullyGrownInfo{
					GuID:    gu.GuID,
					Rank:    gu.Rank,
					Attack:  spec.Effects.Growth.MaxAttack,
					Defense: spec.Effects.Growth.MaxDefense,
					HP:      spec.Effects.Growth.MaxHP,
					Speed:   spec.Effects.Growth.MaxSpeed,
				})
			}

			// Carrier update.
			guUpdates = append(guUpdates, guUpdate{
				InstanceID:  gu.InstanceID,
				GrowthHours: gu.GrowthHours,
				EquippedAt:  now,
			})
		}

		if !hasGrowthGu {
			continue
		}

		growthRaw, _ := json.Marshal(growthData)
		playerUpdates = append(playerUpdates, playerUpdate{
			GrowthData: string(growthRaw),
			BaseHP:     baseHP,
		})
		playerUpdateIDs = append(playerUpdateIDs, pID)
	}

	if len(guUpdates) == 0 && len(playerUpdates) == 0 {
		return
	}

	// ── Step 5: batch-write via single transaction ────────────────────────
	batchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tx, err := db.BeginTx(batchCtx, nil)
	if err != nil {
		logger.S.Errorf("Growth Worker Error starting batch transaction: %v", err)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout = '30000'`); err != nil {
		logger.S.Errorf("Growth Worker Error setting statement timeout: %v", err)
		return
	}

	// Batch update player_gu (equipped_at + growth_hours for all touched Gus).
	if len(guUpdates) > 0 {
		ids := make([]string, len(guUpdates))
		ghs := make([]float64, len(guUpdates))
		eas := make([]time.Time, len(guUpdates))
		for i, u := range guUpdates {
			ids[i] = u.InstanceID
			ghs[i] = u.GrowthHours
			eas[i] = u.EquippedAt
		}
		_, err = tx.Exec(`
			UPDATE player_gu AS pg
			SET growth_hours = v.gh, equipped_at = v.ea, updated_at = NOW()
			FROM (SELECT unnest($1::text[]) AS id, unnest($2::float8[]) AS gh, unnest($3::timestamptz[]) AS ea) AS v
			WHERE pg.id = v.id
		`, pq.Array(ids), pq.Array(ghs), pq.Array(eas))
		if err != nil {
			logger.S.Errorf("Growth Worker Error batch updating player_gu: %v", err)
			return
		}
	}

	// Batch update players (growth_data + base_hp).
	if len(playerUpdates) > 0 {
		gds := make([]string, len(playerUpdates))
		bhps := make([]float64, len(playerUpdates))
		for i, u := range playerUpdates {
			gds[i] = u.GrowthData
			bhps[i] = u.BaseHP
		}
		_, err = tx.Exec(`
			UPDATE players AS p
			SET growth_data = v.gd::jsonb, base_hp = v.bhp
			FROM (SELECT unnest($1::text[]) AS id, unnest($2::text[]) AS gd, unnest($3::float8[]) AS bhp) AS v
			WHERE p.id = v.id
		`, pq.Array(playerUpdateIDs), pq.Array(gds), pq.Array(bhps))
		if err != nil {
			logger.S.Errorf("Growth Worker Error batch updating players: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Growth Worker Error committing batch transaction: %v", err)
	}
}

// findGrowthAffectedPlayers returns players who own at least one equipped ACTIVE
// Gu with a growth profile.
func findGrowthAffectedPlayers(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT pg.player_id::text
		FROM player_gu pg
		JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
		WHERE pg.equipped AND pg.state = 'ACTIVE'
		  AND gd.effects->'growth' IS NOT NULL
		  AND COALESCE((gd.effects->'growth'->>'hours_to_max')::double precision, 0) > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			playerIDs = append(playerIDs, id)
		}
	}
	return playerIDs, rows.Err()
}

func ApplyGrowthProgressWorker(gu *models.GuInstance, spec models.GuDictionaryItem, elapsedHours float64) (attack, defense, hp, speed float64) {
	if spec.Effects.Growth == nil || elapsedHours <= 0 {
		return 0, 0, 0, 0
	}
	g := spec.Effects.Growth
	maxHours := g.HoursToMax
	if maxHours <= 0 {
		maxHours = 168
	}
	oldHours := gu.GrowthHours
	newHours := oldHours + elapsedHours
	if newHours > maxHours {
		newHours = maxHours
	}
	progress := newHours / maxHours
	if progress > 1.0 {
		progress = 1.0
	}
	oldProgress := oldHours / maxHours

	attack = g.MaxAttack * (progress - oldProgress)
	defense = g.MaxDefense * (progress - oldProgress)
	hp = g.MaxHP * (progress - oldProgress)
	speed = g.MaxSpeed * (progress - oldProgress)

	gu.GrowthHours = newHours
	now := time.Now()
	gu.EquippedAt = &now
	return
}
