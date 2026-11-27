package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type ApertureCheck struct {
	ID                       string
	PlayerID                 string
	TimeRate                 float64
	AreaSize                 float64
	ApertureHP               float64
	MaxApertureHP            float64
	InternalYearsPassed      float64
	NextTribulationYear      float64
	EssenceProductionPerYear int
	RankStartYear            float64
	EarthCalamityCount       int
	HeavenlyTribulationCount int
	GrandTribulationCount    int
	MyriadTribulationCount   int
	ChaosCalamityCount       int
	TribulationPendingAt     sql.NullTime
	State                    string
	DaoMarks                 models.DaoMarksMap
	IsPlaced                 bool
	PlacedRegion             string
}

func checkAndTriggerSingleAperture(apertureID string) error {
	var ap struct {
		ID                       string
		PlayerID                 string
		TimeRate                 float64
		ApertureHP               float64
		MaxApertureHP            float64
		InternalYearsPassed      float64
		NextTribulationYear      float64
		EssenceProductionPerYear int
		RankStartYear            float64
		EarthCalamityCount       int
		HeavenlyTribulationCount int
		GrandTribulationCount    int
		MyriadTribulationCount   int
		ChaosCalamityCount       int
		State                    string
	}

	err := db.DB.QueryRow(`
		SELECT id, player_id, time_rate, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, essence_production_per_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, state
		FROM immortal_apertures
		WHERE id = $1
	`, apertureID).Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.EssenceProductionPerYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.State)
	if err != nil {
		return err
	}

	if ap.State != "STABLE" && ap.State != "DAMAGED" {
		return nil
	}

	var rank int
	var race string
	err = db.DB.QueryRow(`SELECT rank, COALESCE(race, '') FROM players WHERE id = $1`, ap.PlayerID).Scan(&rank, &race)
	if err != nil {
		return err
	}

	if models.IsVinhSinhRank(rank) {
		// Vĩnh Sinh: bất diệt, không còn tai kiếp giáng xuống.
		return nil
	}

	dummyP := models.Player{}
	dummyP.Race = race
	if gamefx.Get(&dummyP).Immortal {
		// Tiên Cương: Tử Khiếu ngưng trệ, tai kiếp không giáng xuống
		return nil
	}

	calcNextYear, calcNextType := models.GetNextTribulationInfo(rank, ap.RankStartYear, ap.InternalYearsPassed)
	calcNextType = models.SanitizeTribulationTypeForRank(rank, calcNextType)

	triggerYear := guardNextTribulationYear(calcNextYear, ap.InternalYearsPassed)
	storedYear := guardNextTribulationYear(ap.NextTribulationYear, ap.InternalYearsPassed)
	if storedYear >= ap.InternalYearsPassed || (ap.InternalYearsPassed-storedYear < 10.0) {
		if storedYear < triggerYear {
			triggerYear = storedYear
			if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, storedYear); scheduledType != "" {
				calcNextType = models.SanitizeTribulationTypeForRank(rank, scheduledType)
			}
		} else if math.Abs(storedYear-triggerYear) < 0.001 {
			if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, triggerYear); scheduledType != "" {
				calcNextType = models.SanitizeTribulationTypeForRank(rank, scheduledType)
			}
		}
	}

	if ap.InternalYearsPassed >= triggerYear {
		spawnType := calcNextType
		err = spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.ApertureHP, ap.InternalYearsPassed, spawnType, 1.0)
		if err != nil {
			return err
		}
	} else {
		_, err = db.DB.Exec(`
			UPDATE immortal_apertures 
			SET next_tribulation_year = $1, updated_at = NOW()
			WHERE id = $2
		`, triggerYear, ap.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func CheckAndTriggerTribulations(ctx context.Context) error {
	// Query stable apertures
	rows, err := db.DB.Query(`
		SELECT id, player_id, time_rate, area_size, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, essence_production_per_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, tribulation_pending_at, state,
			COALESCE(dao_marks, '{}'::jsonb),
			COALESCE(is_placed, false), COALESCE(placed_region, '')
		FROM immortal_apertures
		WHERE player_id IS NOT NULL AND state IN ('STABLE', 'DAMAGED', 'TRIBULATION_PENDING', 'TRIBULATION_ACTIVE')
	`)
	if err != nil {
		return err
	}

	apertures := []ApertureCheck{}
	for rows.Next() {
		var ap ApertureCheck
		if err := rows.Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.AreaSize, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.EssenceProductionPerYear,
			&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.TribulationPendingAt, &ap.State, &ap.DaoMarks,
			&ap.IsPlaced, &ap.PlacedRegion); err == nil {
			apertures = append(apertures, ap)
		}
	}

	if len(apertures) > 0 {
		// ── Batch-read: player lifespan, time_speed, rank (1 query) ──────────
		type pdata struct {
			lifespan float64
			ts       float64
			rank     int
			race     string
		}
		pMap := make(map[string]*pdata, len(apertures))
		{
			pids := make([]string, len(apertures))
			for i, ap := range apertures {
				pids[i] = ap.PlayerID
			}
			if pRows, qErr := db.DB.Query(`SELECT id, lifespan, time_speed_multiplier, rank, COALESCE(race, '') FROM players WHERE id = ANY($1::uuid[])`, pq.Array(pids)); qErr != nil {
				logger.S.Errorf("CheckAndTriggerTribulations: batch-read players error: %v", qErr)
			} else {
				defer pRows.Close()
				for pRows.Next() {
					var id string
					var d pdata
					if err := pRows.Scan(&id, &d.lifespan, &d.ts, &d.rank, &d.race); err == nil {
						pMap[id] = &d
					}
				}
			}
		}

		// ── Batch-read: resource points for affected apertures (1 query) ──────
		pointsByAperture := make(map[string][]models.ApertureResourcePoint)
		{
			aids := make([]string, len(apertures))
			for i, ap := range apertures {
				aids[i] = ap.ID
			}
			if nRows, qErr := db.DB.Query(`
				SELECT aperture_id, id, point_type, name, path, vitality, state, x, y
				FROM aperture_resource_points
				WHERE aperture_id = ANY($1::uuid[])
			`, pq.Array(aids)); qErr != nil {
				logger.S.Errorf("CheckAndTriggerTribulations: batch-read resource points error: %v", qErr)
			} else {
				for nRows.Next() {
					var aid string
					var pt models.ApertureResourcePoint
					if err := nRows.Scan(&aid, &pt.ID, &pt.PointType, &pt.Name, &pt.Path, &pt.Vitality, &pt.State, &pt.X, &pt.Y); err == nil {
						pt.ApertureID = &aid
						pointsByAperture[aid] = append(pointsByAperture[aid], pt)
					}
				}
				nRows.Close()
			}
		}

		// ── Batch-read: equipped time speed bonuses (1 query) ─────────────────
		equippedBonusMap := make(map[string]float64, len(apertures))
		{
			pids := make([]string, len(apertures))
			for i, ap := range apertures {
				pids[i] = ap.PlayerID
			}
			if bRows, bErr := db.DB.Query(`
				SELECT player_id, gu_id, gu_rank
				FROM player_gu
				WHERE player_id = ANY($1::uuid[]) AND equipped = true AND state = 'ACTIVE'
			`, pq.Array(pids)); bErr != nil {
				logger.S.Errorf("CheckAndTriggerTribulations: batch-read equipped Gus error: %v", bErr)
			} else {
				defer bRows.Close()
				bestRankByPlayer := make(map[string]map[string]int)
				for bRows.Next() {
					var pid, guID string
					var rank int
					if err := bRows.Scan(&pid, &guID, &rank); err == nil {
						m, ok := bestRankByPlayer[pid]
						if !ok {
							m = make(map[string]int)
							bestRankByPlayer[pid] = m
						}
						if rank > m[guID] {
							m[guID] = rank
						}
					}
				}
				for pid, bestRank := range bestRankByPlayer {
					var total float64
					for guID, rank := range bestRank {
						if spec, ok := models.GetGu(guID, rank); ok {
							total += models.GuStatValue(spec, "TIME_SPEED")
						}
					}
					equippedBonusMap[pid] = total
				}
			}
		}

		// ── Accumulate batch writes ───────────────────────────────────────────
		type lifespanUp struct {
			id    string
			decay float64
		}
		type manaUp struct {
			id  string
			val float64
		}
		type pointUp struct {
			id        string
			vitality  int
			exhausted bool
		}
		type msgIns struct{ receiver, content string }

		var lifeUps []lifespanUp
		var expiryCandidates []apertureExpiryCandidate
		var manaUps []manaUp
		var pointUps []pointUp
		var msgs []msgIns

		// ── Process each aperture ─────────────────────────────────────────────
		for _, ap := range apertures {
			timeSpeedMultiplier := 1.0
			realYearsPassed := 60.0 / 86400.0

			// Look up player data from batch
			currentLifespan := 100.0
			playerRace := ""
			playerRank := 0
			if pd, ok := pMap[ap.PlayerID]; ok {
				currentLifespan = pd.lifespan
				timeSpeedMultiplier = pd.ts
				playerRace = pd.race
				playerRank = pd.rank
			}
			// Vĩnh Sinh (Rank 10): bất tử tuổi thọ + không còn tai kiếp,
			// các hoạt động Tiên Khiếu khác vẫn diễn ra bình thường.
			vinhSinh := models.IsVinhSinhRank(playerRank)

			dummyP := models.Player{}
			dummyP.Race = playerRace
			dummyP.Lifespan = currentLifespan
			dummyP.Rank = playerRank
			isImmortal := gamefx.Get(&dummyP).Immortal

			// A player in Cõi Chết has no aperture progression to run.
			if gamefx.IsDeadState(&dummyP) {
				continue
			}

			if isImmortal {
				timeSpeedMultiplier = 1.0
			} else {
				if timeSpeedMultiplier <= 0 {
					timeSpeedMultiplier = 1.0
				}
				if bonus, ok := equippedBonusMap[ap.PlayerID]; ok {
					timeSpeedMultiplier += bonus
				}
			}

			if isImmortal {
				// Dead Aperture (Tử Khiếu) Decay: LOW grade (~3M area / ~12.5 rate) takes ~5 days
				decayRate := 0.0017 * (1.0 + realYearsPassed)
				decayArea := 400 * (1.0 + realYearsPassed)
				newRate := math.Max(0.5, ap.TimeRate-decayRate)
				newArea := math.Max(50.0, ap.AreaSize-decayArea)
				if newRate != ap.TimeRate || newArea != ap.AreaSize {
					ap.TimeRate = newRate
					ap.AreaSize = newArea
					_, _ = db.DB.Exec(`UPDATE immortal_apertures SET time_rate = $1, area_size = $2, updated_at = NOW() WHERE id = $3`, newRate, newArea, ap.ID)
				}
			} else if !vinhSinh {
				if currentLifespan <= 0.0 {
					expiryCandidates = append(expiryCandidates, apertureExpiryCandidate{
						apertureID: ap.ID,
						playerID:   ap.PlayerID,
					})
					continue
				}

				lifespanDecay := realYearsPassed * timeSpeedMultiplier
				newLifespan := math.Max(0.0, currentLifespan-lifespanDecay)
				if newLifespan <= 0.0 {
					expiryCandidates = append(expiryCandidates, apertureExpiryCandidate{
						apertureID:    ap.ID,
						playerID:      ap.PlayerID,
						lifespanDecay: lifespanDecay,
					})
					continue
				}
				lifeUps = append(lifeUps, lifespanUp{id: ap.PlayerID, decay: lifespanDecay})
			}
			if ap.State == "TRIBULATION_ACTIVE" {
				continue // Keep lifespan checks running; active combat/event progression owns this state.
			}

			effectiveRate := ap.TimeRate
			if ap.IsPlaced {
				effectiveRate = 1.0 * LayerTribulationTimeRateMultiplier(worldMapIDForRegion(ap.PlacedRegion))
			}

			subjectiveYearsPassed := 60.0 * effectiveRate * timeSpeedMultiplier / 86400.0
			newYearsPassed := ap.InternalYearsPassed + subjectiveYearsPassed
			yearsDiff := math.Floor(newYearsPassed) - math.Floor(ap.InternalYearsPassed)

			if yearsDiff > 0 {
				addedEssence := float64(yearsDiff) * float64(ap.EssenceProductionPerYear)
				// Dao purity suy giảm sản lượng Tiên Nguyên tự nhiên (spec §5.1).
				if _, total, purity := gamefx.DaoPurity(ap.DaoMarks); total > 0 {
					addedEssence *= gamefx.EcologyPurityFactor(purity)
				}
				manaUps = append(manaUps, manaUp{id: ap.PlayerID, val: addedEssence})

				// Conflicts Tick — resource points decay when their path conflicts
				// with the aperture's Dao-mark composition.
				points := pointsByAperture[ap.ID]
				if len(points) > 0 {
					daoProfile, _ := loadApertureDaoCombatProfile(db.DB, ap.PlayerID, ap.DaoMarks)

					for i := range points {
						pt := &points[i]
						conflictMultiplier := apertureDaoConflictMultiplierForProfile(daoProfile, pt.Path)
						decay := int(float64(pt.Vitality) * 0.10 * float64(yearsDiff) * (1.0 - conflictMultiplier))

						if decay > 0 {
							newVitality := pt.Vitality - decay
							exhausted := false
							if newVitality <= 0 {
								newVitality = 0
								exhausted = true
							}
							pointUps = append(pointUps, pointUp{id: pt.ID, vitality: newVitality, exhausted: exhausted})

							if exhausted {
								msgs = append(msgs, msgIns{receiver: ap.PlayerID, content: "Điểm tài nguyên " + pt.Name + " đã cạn kiệt sinh khí do bất ổn sinh thái hoặc xung đột Đạo ngân! Cần Bồi Dưỡng để khôi phục."})
							}
						}
					}

					// Variant human simulation is owned by worker.StartSocietySimulationWorker.
				}
			}

			// Tribulation scheduling
			pd := pMap[ap.PlayerID]
			rank := 6
			if pd != nil && pd.rank > 0 {
				rank = pd.rank
			}

			calcNextYear, calcNextType := models.GetNextTribulationInfo(rank, ap.RankStartYear, ap.InternalYearsPassed)
			calcNextType = models.SanitizeTribulationTypeForRank(rank, calcNextType)

			triggerYear := guardNextTribulationYear(calcNextYear, ap.InternalYearsPassed)
			storedYear := guardNextTribulationYear(ap.NextTribulationYear, ap.InternalYearsPassed)
			if storedYear >= ap.InternalYearsPassed || (ap.InternalYearsPassed-storedYear < 10.0) {
				if storedYear < triggerYear {
					triggerYear = storedYear
					if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, storedYear); scheduledType != "" {
						calcNextType = models.SanitizeTribulationTypeForRank(rank, scheduledType)
					}
				} else if math.Abs(storedYear-triggerYear) < 0.001 {
					if scheduledType := models.GetTribulationTypeAtYear(rank, ap.RankStartYear, triggerYear); scheduledType != "" {
						calcNextType = models.SanitizeTribulationTypeForRank(rank, scheduledType)
					}
				}
			}

			if vinhSinh {
				// Vĩnh Sinh: không còn tai kiếp giáng xuống; mọi hoạt động khác giữ nguyên.
				newState := ap.State
				if newState == "TRIBULATION_PENDING" {
					newState = "STABLE"
				}
				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET internal_years_passed = $1, state = $2, tribulation_pending_at = NULL, updated_at = NOW()
					WHERE id = $3
				`, newYearsPassed, newState, ap.ID)
				continue
			}

			if isImmortal {
				// Tiên Cương: Tử Khiếu ngưng trệ, tai kiếp ngưng giáng xuống
				newState := ap.State
				if newState == "TRIBULATION_PENDING" {
					newState = "STABLE"
				}
				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET internal_years_passed = $1, state = $2, tribulation_pending_at = NULL, updated_at = NOW()
					WHERE id = $3
				`, newYearsPassed, newState, ap.ID)
				continue
			}

			if newYearsPassed >= triggerYear {
				// Nếu đã bị quá giờ bảo lưu (> 24 giờ) dù online hay offline => LUÔN BẠI!
				if ap.TribulationPendingAt.Valid && time.Since(ap.TribulationPendingAt.Time) > tribulationPenaltyAfter {
					failOverduePendingTribulation(ap, newYearsPassed, calcNextType)
					continue
				}

				if !isPlayerOnline(ap.PlayerID) {
					_, _ = db.DB.Exec(`
						UPDATE immortal_apertures
						SET state = 'TRIBULATION_PENDING',
						    tribulation_pending_at = COALESCE(tribulation_pending_at, NOW()),
						    internal_years_passed = $1, next_tribulation_year = $2, updated_at = NOW()
						WHERE id = $3
					`, newYearsPassed, triggerYear, ap.ID)
					continue
				}

				rewardMult := 1.0
				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET internal_years_passed = $1, tribulation_pending_at = NULL, updated_at = NOW()
					WHERE id = $2
				`, newYearsPassed, ap.ID)

				spawnType := calcNextType
				// Dị Biến Tiên Khiếu → Kiếp Nạn Hỗn Loạn (spec §5.2).
				if gamefx.IsChaoticTribulation(ap.DaoMarks, nil) {
					spawnType = "CHAOTIC_TRIBULATION"
				}
				err := spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.ApertureHP, newYearsPassed, spawnType, rewardMult)
				if err != nil {
					logger.S.Errorf("Failed to spawn tribulation event for player %s: %v", ap.PlayerID, err)
				}
			} else {
				_, _ = db.DB.Exec(`
					UPDATE immortal_apertures
					SET internal_years_passed = $1, next_tribulation_year = $2, updated_at = NOW()
					WHERE id = $3
				`, newYearsPassed, triggerYear, ap.ID)
			}
		}

		// ── Batch-write: all accumulated changes (single transaction) ─────────
		if len(lifeUps)+len(manaUps)+len(pointUps)+len(msgs) > 0 {
			if tx, txErr := db.DB.Begin(); txErr == nil {
				defer tx.Rollback()

				// Batch UPDATE player lifespans
				if len(lifeUps) > 0 {
					ids := make([]string, len(lifeUps))
					vals := make([]float64, len(lifeUps))
					for i, u := range lifeUps {
						ids[i] = u.id
						vals[i] = u.decay
					}
					if _, err := tx.Exec(`UPDATE players AS p SET lifespan = GREATEST(0.0, p.lifespan - v.decay) FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::float8[]) AS decay) AS v WHERE p.id = v.id`, pq.Array(ids), pq.Array(vals)); err != nil {
						logger.S.Errorf("CheckAndTriggerTribulations: batch update lifespans failed: %v", err)
					}
				}

				// Batch UPDATE player mana
				if len(manaUps) > 0 {
					ids := make([]string, 0, len(manaUps))
					vals := make([]float64, 0, len(manaUps))
					for _, u := range manaUps {
						if pd, ok := pMap[u.id]; ok {
							dummyP := models.Player{}
							dummyP.Race = pd.race
							if gamefx.Get(&dummyP).DisableManaRegen {
								continue
							}
						}
						ids = append(ids, u.id)
						vals = append(vals, u.val)
					}
					if len(ids) > 0 {
						if _, err := tx.Exec(`UPDATE players AS p SET current_mana = LEAST(p.max_mana, p.current_mana + v.val), last_mana_update = NOW() FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::float8[]) AS val) AS v WHERE p.id = v.id`, pq.Array(ids), pq.Array(vals)); err != nil {
							logger.S.Errorf("CheckAndTriggerTribulations: batch update mana failed: %v", err)
						}
					}
				}

				// Batch UPDATE point vitality
				if len(pointUps) > 0 {
					ids := make([]string, 0, len(pointUps))
					vits := make([]int, 0, len(pointUps))
					for _, u := range pointUps {
						ids = append(ids, u.id)
						vits = append(vits, u.vitality)
					}
					if _, err := tx.Exec(`
						UPDATE aperture_resource_points AS p
						SET vitality = v.vitality,
						    state = CASE WHEN v.vitality <= 0 THEN 'EXHAUSTED' ELSE p.state END,
						    updated_at = NOW()
						FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS vitality) AS v
						WHERE p.id = v.id
					`, pq.Array(ids), pq.Array(vits)); err != nil {
						logger.S.Errorf("CheckAndTriggerTribulations: batch update points failed: %v", err)
					}
				}

				// Batch INSERT messages
				if len(msgs) > 0 {
					receivers := make([]string, len(msgs))
					contents := make([]string, len(msgs))
					for i, m := range msgs {
						receivers[i] = m.receiver
						contents[i] = m.content
					}
					if _, err := tx.Exec(`INSERT INTO messages (sender_id, receiver_id, content, channel) SELECT '11111111-1111-1111-1111-111111111111'::uuid, v.receiver, v.content, 'private' FROM (SELECT unnest($1::uuid[]) AS receiver, unnest($2::text[]) AS content) AS v`, pq.Array(receivers), pq.Array(contents)); err != nil {
						logger.S.Errorf("CheckAndTriggerTribulations: batch insert messages failed: %v", err)
					}
				}

				if err := tx.Commit(); err != nil {
					logger.S.Errorf("CheckAndTriggerTribulations: commit batch writes failed: %v", err)
				}
			} else {
				logger.S.Errorf("CheckAndTriggerTribulations: begin tx for batch writes failed: %v", txErr)
			}
		}

		for _, candidate := range expiryCandidates {
			if _, err := expireApertureOwner(ctx, candidate); err != nil {
				logger.S.Errorf("CheckAndTriggerTribulations: expire player %s aperture %s: %v", candidate.playerID, candidate.apertureID, err)
			}
		}
	}

	// 2. Check and auto-resolve pending active tribulations that exceeded 24 hours
	autoRows, err := db.DB.Query(`
		SELECT id, player_id
		FROM tribulation_events
		WHERE status = 'ACTIVE' AND COALESCE(stage, '') != 'AFTERMATH' AND started_at < NOW() - INTERVAL '24 hours'
	`)
	if err == nil {
		type EventResolve struct {
			EventID  string
			PlayerID string
		}
		toResolve := []EventResolve{}
		for autoRows.Next() {
			var ev EventResolve
			if err := autoRows.Scan(&ev.EventID, &ev.PlayerID); err == nil {
				toResolve = append(toResolve, ev)
			}
		}
		autoRows.Close()

		for _, r := range toResolve {
			logger.S.Infof("Auto-resolving expired tribulation event %s for player %s", r.EventID, r.PlayerID)
			autoResolveEvent(r.EventID, r.PlayerID)
		}
	}

	return nil
}

// chaoticHybridPath joins up to 3 highest-mark paths (dominant first) into a
// comma-separated path list used to display and reward-split the hybrid tribulation (spec §5.2).
func chaoticHybridPath(daoMarks models.DaoMarksMap, fallback string) string {
	type pv struct {
		p string
		v int64
	}
	var ps []pv
	for p, v := range daoMarks {
		if v > 0 {
			ps = append(ps, pv{p, v})
		}
	}
	if len(ps) == 0 {
		return fallback
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].v != ps[j].v {
			return ps[i].v > ps[j].v
		}
		return ps[i].p < ps[j].p
	})
	n := len(ps)
	if n > 3 {
		n = 3
	}
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, strings.TrimSuffix(strings.ToLower(ps[i].p), " path"))
	}
	return strings.Join(parts, ",")
}

// tribulationPathLabel renders a (possibly hybrid) comma-separated path list for display.
func tribulationPathLabel(path string) string {
	if !strings.Contains(path, ",") {
		return strings.Title(path)
	}
	parts := strings.Split(path, ",")
	for i, p := range parts {
		parts[i] = strings.Title(strings.TrimSpace(p))
	}
	return strings.Join(parts, " + ")
}

func spawnTribulationEventWorker(apertureID string, playerID string, currentHP float64, newYearsPassed float64, tribulationType string, rewardMult float64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// A player can only face one tribulation at a time. The aperture state
	// normally prevents a second spawn, but a stale row must never let the
	// panel and the realtime map select different events.
	var hasActive bool
	if err := tx.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM tribulation_events WHERE player_id = $1 AND status = 'ACTIVE')
	`, playerID).Scan(&hasActive); err != nil {
		return err
	}
	if hasActive {
		return nil
	}

	var rank int
	err = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&rank)
	if err != nil {
		return err
	}

	tribulationType = models.SanitizeTribulationTypeForRank(rank, tribulationType)

	var vitalPaths []string
	var vitalGuID string
	var vitalGuRank int
	errVital := tx.QueryRow(`SELECT gu_id, gu_rank FROM player_gu WHERE player_id = $1 AND is_vital = true LIMIT 1`, playerID).Scan(&vitalGuID, &vitalGuRank)
	if errVital == nil {
		if vitalGu, found := models.GetGu(vitalGuID, vitalGuRank); found {
			vitalPaths = vitalGu.Path
		}
	} else if errVital != sql.ErrNoRows {
		return fmt.Errorf("query Vital Gu for tribulation affinity: %w", errVital)
	}

	path, ok := chooseTribulationPath(models.AllGuPaths(), vitalPaths, tribulationType, rand.Float64())
	if !ok {
		return fmt.Errorf("no eligible dao paths available for tribulation")
	}

	var daoMarksRaw []byte
	err = tx.QueryRow(`SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&daoMarksRaw)
	if err != nil {
		return err
	}

	var daoMarks models.DaoMarksMap
	if len(daoMarksRaw) > 0 {
		_ = json.Unmarshal(daoMarksRaw, &daoMarks)
	}

	// Kiếp Nạn Hỗn Loạn: khuếch đại HP/Công quái theo độ tạp (spec §5.2).
	calamityMult := 1.0
	if tribulationType == "CHAOTIC_TRIBULATION" {
		_, _, purity := gamefx.DaoPurity(daoMarks)
		calamityMult = 1.0 + (1.0-purity)*0.75
		path = chaoticHybridPath(daoMarks, path)
	}

	// Thiên Đạo Phẫn Nộ: Cảm ứng trợ thủ bên trong Tiên Khiếu gia tăng uy lực
	var helperCount int
	var helperMaxRank int
	_ = tx.QueryRow(`
		SELECT COUNT(g.player_id), COALESCE(MAX(p.rank), 0)
		FROM aperture_guest_roster g
		JOIN players p ON p.id = g.player_id
		WHERE g.aperture_id = $1 AND g.role IN ('VISITOR', 'ASSISTANT')
	`, apertureID).Scan(&helperCount, &helperMaxRank)

	if helperCount > 0 {
		deltaRank := helperMaxRank - rank + 1
		if deltaRank < 0 {
			deltaRank = 0
		}
		coopMult := 1.0 + float64(helperCount)*0.4 + float64(deltaRank)*0.3
		calamityMult *= coopMult
	}

	rewardMarks := models.CalculateTribulationReward(rank, tribulationType)
	if rewardMult <= 0 {
		rewardMult = 1.0
	}
	rewardMarks = int64(float64(rewardMarks) * rewardMult)

	// Determine Archetype and Stage
	archetypes := []string{"ENVIRONMENTAL", "BEAST_SWARM", "ANOMALOUS_COMBAT"}
	archetype := archetypes[rand.Intn(len(archetypes))]
	stage := "HARBINGER"
	prepDuration := 15 * time.Minute
	prepEndsAt := time.Now().Add(prepDuration)

	hazardMetaObj := map[string]interface{}{
		"archetype":     archetype,
		"hazard_path":   path,
		"calamity_name": fmt.Sprintf("Kiếp Nạn %s (%s)", strings.ToUpper(path), models.GetTribulationTypeName(tribulationType)),
	}

	if archetype == "ENVIRONMENTAL" {
		numCells := 4 + rand.Intn(4)
		cells := make([]map[string]interface{}, 0, numCells)
		for i := 0; i < numCells; i++ {
			cells = append(cells, map[string]interface{}{
				"x":            rand.Intn(20),
				"y":            rand.Intn(20),
				"path":         path,
				"severity":     1 + rand.Intn(3),
				"extinguished": false,
			})
		}
		hazardMetaObj["cells"] = cells
	} else if archetype == "BEAST_SWARM" {
		hazardMetaObj["wave"] = 1
		hazardMetaObj["total_waves"] = 3
	} else {
		hazardModifier := "FIELD_LIGHTNING"
		if path == "ice" || path == "water" {
			hazardModifier = "PERMAFROST"
		} else if path == "fire" || path == "poison" {
			hazardModifier = "GU_RESTRICTION"
		}
		hazardMetaObj["hazard_modifier"] = hazardModifier
	}

	hazardMetaBytes, _ := json.Marshal(hazardMetaObj)

	// Insert Tribulation Event
	eventID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO tribulation_events (id, aperture_id, player_id, tribulation_type, path, status, aperture_hp_before, reward_dao_marks, started_at, archetype, stage, preparation_ends_at, hazard_metadata, spoils_collected)
		VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6, $7, NOW(), $8, $9, $10, $11, false)
	`, eventID, apertureID, playerID, tribulationType, path, currentHP, rewardMarks, archetype, stage, prepEndsAt, hazardMetaBytes)
	if err != nil {
		return err
	}

	// Record participants in tribulation_coop_participants
	if helperCount > 0 {
		_, _ = tx.Exec(`
			INSERT INTO tribulation_coop_participants (event_id, player_id, rank, primary_path, is_alive, created_at)
			SELECT $1, g.player_id, p.rank, COALESCE(p.race, ''), true, NOW()
			FROM aperture_guest_roster g
			JOIN players p ON p.id = g.player_id
			WHERE g.aperture_id = $2 AND g.role IN ('VISITOR', 'ASSISTANT')
			ON CONFLICT (event_id, player_id) DO NOTHING
		`, eventID, apertureID)
	}

	// Spawn Enemies
	bossTemplateID := "monster_earth_calamity_boss"
	switch tribulationType {
	case "EARTH_CALAMITY":
		bossTemplateID = "monster_earth_calamity_boss"
	case "HEAVENLY_TRIBULATION":
		bossTemplateID = "monster_heaven_tribulation_boss"
	case "GRAND_TRIBULATION":
		bossTemplateID = "monster_grand_calamity_boss"
	case "MYRIAD_TRIBULATION":
		bossTemplateID = "monster_myriad_tribulation_boss"
	case "CHAOS_CALAMITY":
		bossTemplateID = "monster_chaos_calamity_boss"
	case "CHAOTIC_TRIBULATION":
		bossTemplateID = "monster_chaos_calamity_boss"
	default:
		if rank >= 9 {
			bossTemplateID = "monster_grand_calamity_boss"
		} else if rank == 8 {
			bossTemplateID = "monster_heaven_tribulation_boss"
		} else {
			bossTemplateID = "monster_earth_calamity_boss"
		}
	}

	bossRank, bossSubRank := models.GetTribulationRankAndSubRank(rank, tribulationType)

	var boss models.MonsterTemplate
	var bossGu []byte
	err = tx.QueryRow(`
		SELECT id, name, type, max_hp, base_attack, base_defense, gu_inventory, ai_profile, loot_table_id, description
		FROM monster_templates WHERE id = $1
	`, bossTemplateID).Scan(
		&boss.ID, &boss.Name, &boss.Type,
		&boss.MaxHP, &boss.BaseAttack, &boss.BaseDefense,
		&bossGu, &boss.AIProfile, &boss.LootTableID, &boss.Description,
	)
	if err != nil {
		return fmt.Errorf("load tribulation boss template %s from database: %w", bossTemplateID, err)
	}

	scaledHP, scaledAtk, scaledDef := models.ScaleMonsterStats(boss.MaxHP, boss.BaseAttack, boss.BaseDefense, bossRank, bossSubRank, boss.Type)
	bossHP := scaledHP * calamityMult
	bossAtk := scaledAtk * calamityMult
	bossDef := scaledDef

	_, err = tx.Exec(`
		INSERT INTO tribulation_enemies (event_id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_boss, is_dead, rank, sub_rank)
		VALUES ($1, $2, 'BOSS', $3, $4, $4, $5, $6, $7, $8, true, false, $9, $10)
	`, eventID, fmt.Sprintf("%s (%s Path)", boss.Name, tribulationPathLabel(path)), path, bossHP, bossAtk, bossDef, rand.Intn(20), rand.Intn(20), bossRank, bossSubRank)
	if err != nil {
		return err
	}

	// Spawn Minions
	var minion models.MonsterTemplate
	var minionGu []byte
	err = tx.QueryRow(`
		SELECT id, name, type, max_hp, base_attack, base_defense, gu_inventory, ai_profile, loot_table_id, description
		FROM monster_templates WHERE id = 'monster_tribulation_beast'
	`).Scan(
		&minion.ID, &minion.Name, &minion.Type,
		&minion.MaxHP, &minion.BaseAttack, &minion.BaseDefense,
		&minionGu, &minion.AIProfile, &minion.LootTableID, &minion.Description,
	)
	if err != nil {
		return fmt.Errorf("load tribulation minion template monster_tribulation_beast from database: %w", err)
	}

	minionCount := 1 + rand.Intn(rank-3)
	minionSubRank := 1
	for i := 0; i < minionCount; i++ {
		scaledHPm, scaledAtkm, scaledDefm := models.ScaleMonsterStats(minion.MaxHP, minion.BaseAttack, minion.BaseDefense, bossRank, minionSubRank, minion.Type)
		mHP := scaledHPm * calamityMult
		mAtk := scaledAtkm * calamityMult
		mDef := scaledDefm

		_, err = tx.Exec(`
			INSERT INTO tribulation_enemies (event_id, name, enemy_type, path, hp, max_hp, attack, defense, x, y, is_dead, rank, sub_rank)
			VALUES ($1, $2, 'MINION', $3, $4, $4, $5, $6, $7, $8, false, $9, $10)
		`, eventID, fmt.Sprintf("%s #%d", minion.Name, i+1), path, mHP, mAtk, mDef, rand.Intn(20), rand.Intn(20), bossRank, minionSubRank)
		if err != nil {
			return err
		}
	}

	// Set aperture status to active tribulation and update internal_years_passed
	_, err = tx.Exec(`
		UPDATE immortal_apertures
		SET state = 'TRIBULATION_ACTIVE', internal_years_passed = $1, updated_at = NOW()
		WHERE id = $2
	`, newYearsPassed, apertureID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func vitalGuPathChance(tribulationType string) float64 {
	switch tribulationType {
	case "EARTH_CALAMITY":
		return 0.80
	case "HEAVENLY_TRIBULATION":
		return 0.85
	case "GRAND_TRIBULATION":
		return 0.90
	case "MYRIAD_TRIBULATION":
		return 0.95
	case "CHAOS_CALAMITY":
		return 1.00
	case "CHAOTIC_TRIBULATION":
		return 1.00
	default:
		return 0
	}
}

// chooseTribulationPath keeps a tier-scaled affinity with the Vital Gu's
// ordinary paths. Heaven Path is never eligible. When the affinity roll fails,
// choosing from non-vital paths keeps the configured probability exact.
func chooseTribulationPath(supportedPaths, vitalPaths []string, tribulationType string, roll float64) (string, bool) {
	eligible := eligibleOrdinaryDaoMarkPaths(supportedPaths)
	if len(eligible) == 0 {
		return "", false
	}

	eligibleSet := make(map[string]struct{}, len(eligible))
	for _, path := range eligible {
		eligibleSet[path] = struct{}{}
	}
	vitalSet := make(map[string]struct{})
	for _, path := range eligibleOrdinaryDaoMarkPaths(vitalPaths) {
		if _, exists := eligibleSet[path]; exists {
			vitalSet[path] = struct{}{}
		}
	}

	matchedVital := make([]string, 0, len(vitalSet))
	fallback := make([]string, 0, len(eligible))
	for _, path := range eligible {
		if _, matched := vitalSet[path]; matched {
			matchedVital = append(matchedVital, path)
		} else {
			fallback = append(fallback, path)
		}
	}
	if len(matchedVital) > 0 && roll < vitalGuPathChance(tribulationType) {
		return matchedVital[rand.Intn(len(matchedVital))], true
	}
	if len(fallback) > 0 {
		return fallback[rand.Intn(len(fallback))], true
	}
	return eligible[rand.Intn(len(eligible))], true
}

// tribulationNodesToDestroy returns how many surviving resource nodes a failed
// tribulation flattens: 30% of them, at least one when any survive.
func tribulationNodesToDestroy(activeCount int) int {
	n := int(float64(activeCount) * 0.3)
	if n < 1 && activeCount > 0 {
		n = 1
	}
	return n
}

// applyTribulationFailurePenalty applies the shared cost of a failed
// tribulation: flatten 30% of the aperture's living resource nodes and charge
// 2000 Nguyên Thạch + 5 Tiên Nguyên. Shared by every failure path so the penalty
// cannot drift between them.
func applyTribulationFailurePenalty(tx *sql.Tx, apertureID, playerID string) {
	var activePoints int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM aperture_resource_points WHERE aperture_id = $1 AND state <> 'EXHAUSTED'`, apertureID).Scan(&activePoints)

	if n := tribulationNodesToDestroy(activePoints); n > 0 {
		// Failed tribulations flatten a random share of the surviving resource
		// points to zero vitality instead of destroying rows outright.
		_, _ = tx.Exec(`
			UPDATE aperture_resource_points
			SET vitality = 0, state = 'EXHAUSTED', updated_at = NOW()
			WHERE id IN (
				SELECT id FROM aperture_resource_points
				WHERE aperture_id = $1 AND state <> 'EXHAUSTED'
				ORDER BY random()
				LIMIT $2
			)
		`, apertureID, n)
	}

	_, _ = tx.Exec(`
		UPDATE players
		SET primeval_stones = GREATEST(0, primeval_stones - 2000), immortal_stones = GREATEST(0.0, immortal_stones - 5.0)
		WHERE id = $1
	`, playerID)
}

func failOverduePendingTribulation(ap ApertureCheck, newYearsPassed float64, calcNextType string) {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var playerRank int
	_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, ap.PlayerID).Scan(&playerRank)

	nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, newYearsPassed)
	nextYear = guardNextTribulationYear(nextYear, newYearsPassed)

	newHP := float64(ap.MaxApertureHP) / 10.0
	if newHP < 100 {
		newHP = 100
	}

	applyTribulationFailurePenalty(tx, ap.ID, ap.PlayerID)

	_, _ = tx.Exec(`
		UPDATE immortal_apertures 
		SET state = 'DAMAGED', next_tribulation_year = $1, aperture_hp = $2,
			internal_years_passed = $3, tribulation_pending_at = NULL,
			area_size = GREATEST(0, CAST(area_size AS numeric) * 0.9),
			time_rate = GREATEST(1.0, time_rate * 0.9),
			updated_at = NOW()
		WHERE id = $4
	`, nextYear, newHP, newYearsPassed, ap.ID)

	eventID := uuid.New()
	_, _ = tx.Exec(`
		INSERT INTO tribulation_events (id, aperture_id, player_id, tribulation_type, path, status, aperture_hp_before, aperture_hp_after, reward_dao_marks, started_at, ended_at)
		VALUES ($1, $2, $3, $4, 'general', 'FAILED', $5, $6, 0, NOW(), NOW())
	`, eventID, ap.ID, ap.PlayerID, calcNextType, ap.ApertureHP, newHP)

	_, _ = tx.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content, channel)
		VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Đạo hữu trốn tránh kiếp nạn quá 24 giờ! Thiên Ý cuồng bạo tàn phá Tiên Khiếu, kiếp nạn bị phán định THẤT BẠI: mất 2,000 Nguyên Thạch, 5 Tiên Nguyên, diện tích và tốc độ thời gian bị giảm 10%, 30% tài nguyên bị san phẳng.', 'private')
	`, ap.PlayerID)

	_ = tx.Commit()
}

func autoResolveEvent(eventID string, playerID string) {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var ev models.TribulationEvent
	err = tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, started_at, aperture_hp_before
		FROM tribulation_events WHERE id = $1 AND status = 'ACTIVE'
	`, eventID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.StartedAt, &ev.ApertureHPBefore)
	if err != nil {
		return
	}

	// Fetch current counters from aperture
	var ap models.ImmortalAperture
	_ = tx.QueryRow(`
		SELECT id, aperture_hp, max_aperture_hp, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, internal_years_passed
		FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.NextTribulationYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount,
		&ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.InternalYearsPassed)
	if err != nil {
		return
	}

	var playerRank int
	_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)

	// Auto-resolve counts as a failure — counters are only incremented on
	// success, but the next year is still pushed to the next milestone.
	nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
	nextYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)

	var feathermanPop int
	_ = tx.QueryRow(`SELECT population FROM aperture_variant_humans WHERE aperture_id = $1 AND race = 'FEATHERMAN'`, ev.ApertureID).Scan(&feathermanPop)
	reduction := float64(feathermanPop) * 0.001
	if reduction > 0.5 {
		reduction = 0.5
	}

	hpFactor := 0.1 + 0.4*reduction
	newHP := float64(ev.ApertureHPBefore) * hpFactor
	if newHP < 100 {
		newHP = 100
	}

	_, _ = tx.Exec(`
		UPDATE immortal_apertures 
		SET state = 'DAMAGED', next_tribulation_year = $1, aperture_hp = $2,
			earth_calamity_count = $3, heavenly_tribulation_count = $4, grand_tribulation_count = $5,
			myriad_tribulation_count = $6, chaos_calamity_count = $7,
			area_size = GREATEST(0, CAST(area_size AS numeric) * 0.9),
			time_rate = GREATEST(1.0, time_rate * 0.9)
		WHERE id = $8
	`, nextYear, newHP,
		ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount,
		ap.MyriadTribulationCount, ap.ChaosCalamityCount, ev.ApertureID)

	_, _ = tx.Exec(`
		UPDATE tribulation_events
		SET status = 'FAILED', aperture_hp_after = $1, ended_at = NOW(), reward_dao_marks = 0
		WHERE id = $2
	`, newHP, ev.ID)

	applyTribulationFailurePenalty(tx, ev.ApertureID, playerID)

	_, _ = tx.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content, channel)
		VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Kiếp nạn quá hạn 24 giờ không giải quyết! Thiên Ý cuồng bạo tàn phá Tiên Khiếu, kiếp nạn bị phán định THẤT BẠI: mất 2,000 Nguyên Thạch, 5 Tiên Nguyên, diện tích và tốc độ thời gian bị giảm 10%, 30% tài nguyên bị san phẳng.', 'private')
	`, playerID)

	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	_ = tx.Commit()
}
