package handlers

import (
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
		err = spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.EarthCalamityCount+ap.HeavenlyTribulationCount+ap.GrandTribulationCount+ap.MyriadTribulationCount+ap.ChaosCalamityCount+1, ap.ApertureHP, ap.InternalYearsPassed, spawnType, 1.0)
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

func CheckAndTriggerTribulations() error {
	// Query stable apertures
	rows, err := db.DB.Query(`
		SELECT id, player_id, time_rate, area_size, aperture_hp, max_aperture_hp, internal_years_passed, next_tribulation_year, essence_production_per_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, tribulation_pending_at, state,
			COALESCE(dao_marks, '{}'::jsonb)
		FROM immortal_apertures
		WHERE state IN ('STABLE', 'DAMAGED', 'TRIBULATION_PENDING')
	`)
	if err != nil {
		return err
	}

	apertures := []ApertureCheck{}
	for rows.Next() {
		var ap ApertureCheck
		if err := rows.Scan(&ap.ID, &ap.PlayerID, &ap.TimeRate, &ap.AreaSize, &ap.ApertureHP, &ap.MaxApertureHP, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.EssenceProductionPerYear,
			&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.TribulationPendingAt, &ap.State, &ap.DaoMarks); err == nil {
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

		// ── Batch-read: all nodes for affected apertures (1 query) ────────────
		nodesByAperture := make(map[string][]models.ApertureResourceNode)
		{
			aids := make([]string, len(apertures))
			for i, ap := range apertures {
				aids[i] = ap.ID
			}
			if nRows, qErr := db.DB.Query(`
				SELECT aperture_id, id, node_type, name, path, hp, max_hp, x, y, is_destroyed, zone_type, production_amount
				FROM aperture_resource_nodes
				WHERE aperture_id = ANY($1::uuid[]) AND is_destroyed = false
			`, pq.Array(aids)); qErr != nil {
				logger.S.Errorf("CheckAndTriggerTribulations: batch-read resource nodes error: %v", qErr)
			} else {
				defer nRows.Close()
				for nRows.Next() {
					var nd models.ApertureResourceNode
					var aid string
					if err := nRows.Scan(&aid, &nd.ID, &nd.NodeType, &nd.Name, &nd.Path, &nd.HP, &nd.MaxHP, &nd.X, &nd.Y, &nd.IsDestroyed, &nd.ZoneType, &nd.ProductionAmount); err == nil {
						nd.ApertureID = aid
						nodesByAperture[aid] = append(nodesByAperture[aid], nd)
					}
				}
			}
			if cRows, qErr := db.DB.Query(`
				SELECT aperture_id, node_id, x, y FROM aperture_resource_node_cells
				WHERE aperture_id = ANY($1::uuid[]) ORDER BY aperture_id, node_id, y, x
			`, pq.Array(aids)); qErr != nil {
				logger.S.Errorf("CheckAndTriggerTribulations: batch-read node areas error: %v", qErr)
			} else {
				cellsByNode := make(map[string][]models.ApertureGridCell)
				for cRows.Next() {
					var aid, nodeID string
					var cell models.ApertureGridCell
					if err := cRows.Scan(&aid, &nodeID, &cell.X, &cell.Y); err == nil {
						cellsByNode[nodeID] = append(cellsByNode[nodeID], cell)
					}
				}
				cRows.Close()
				for aid := range nodesByAperture {
					for i := range nodesByAperture[aid] {
						node := &nodesByAperture[aid][i]
						node.Cells = cellsByNode[node.ID]
						if len(node.Cells) == 0 {
							node.Cells = []models.ApertureGridCell{{X: node.X, Y: node.Y}}
						}
					}
				}
			}
		}

		// ── Batch-read: all variant humans (1 query) ──────────────────────────
		type vhRow struct {
			id, race, laborIntensity string
			population               int
			happiness, oppression    float64
		}
		vhByAperture := make(map[string][]vhRow)
		{
			aids := make([]string, len(apertures))
			for i, ap := range apertures {
				aids[i] = ap.ID
			}
			if vRows, qErr := db.DB.Query(`
				SELECT aperture_id, id, race, population, happiness, oppression, labor_intensity
				FROM aperture_variant_humans
				WHERE aperture_id = ANY($1::uuid[])
			`, pq.Array(aids)); qErr != nil {
				logger.S.Errorf("CheckAndTriggerTribulations: batch-read variant humans error: %v", qErr)
			} else {
				defer vRows.Close()
				for vRows.Next() {
					var v vhRow
					var aid string
					if err := vRows.Scan(&aid, &v.id, &v.race, &v.population, &v.happiness, &v.oppression, &v.laborIntensity); err == nil {
						vhByAperture[aid] = append(vhByAperture[aid], v)
					}
				}
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
		type nodeUp struct {
			id        string
			hp        int
			destroyed bool
		}
		type vhUp struct {
			id             string
			pop            int
			happy, oppress float64
		}
		type msgIns struct{ receiver, content string }

		var lifeUps []lifespanUp
		var manaUps []manaUp
		var nodeUps []nodeUp
		var vhUps []vhUp
		var msgs []msgIns

		// ── Process each aperture ─────────────────────────────────────────────
		for _, ap := range apertures {
			timeSpeedMultiplier := 1.0
			realYearsPassed := 60.0 / 36000.0

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
					continue
				}

				lifespanDecay := realYearsPassed * timeSpeedMultiplier
				newLifespan := math.Max(0.0, currentLifespan-lifespanDecay)
				lifeUps = append(lifeUps, lifespanUp{id: ap.PlayerID, decay: lifespanDecay})

				if newLifespan <= 0.0 {
					// Freeze aperture (keep per-aperture Exec: different SET clause)
					_, _ = db.DB.Exec(`
						UPDATE immortal_apertures
						SET time_rate = 1.0, aperture_hp = max_aperture_hp / 10, state = 'DAMAGED', updated_at = NOW()
						WHERE id = $1
					`, ap.ID)
					msgs = append(msgs, msgIns{receiver: ap.PlayerID, content: "Đạo hữu đã hết thọ nguyên! Tiên Khiếu suy kiệt, dòng thời gian ngưng trệ. Hãy mau kiếm Thọ Cổ để hồi sinh!"})
					logger.S.Warnf("Player %s lifespan depleted, aperture %s frozen", ap.PlayerID, ap.ID)
					continue
				}
			}

			subjectiveYearsPassed := 60.0 * ap.TimeRate * timeSpeedMultiplier / 36000.0
			newYearsPassed := ap.InternalYearsPassed + subjectiveYearsPassed
			yearsDiff := math.Floor(newYearsPassed) - math.Floor(ap.InternalYearsPassed)

			if yearsDiff > 0 {
				addedEssence := float64(yearsDiff) * float64(ap.EssenceProductionPerYear)
				// Dao purity suy giảm sản lượng Tiên Nguyên tự nhiên (spec §5.1).
				if _, total, purity := gamefx.DaoPurity(ap.DaoMarks); total > 0 {
					addedEssence *= gamefx.EcologyPurityFactor(purity)
				}
				manaUps = append(manaUps, manaUp{id: ap.PlayerID, val: addedEssence})

				// Ecosystem & Conflicts Tick — read from pre-loaded nodes
				nodes := nodesByAperture[ap.ID]
				if len(nodes) > 0 {
					daoProfile, _ := loadApertureDaoCombatProfile(db.DB, ap.PlayerID, ap.DaoMarks)
					_, _, _, overgrazing, _, _ := EvaluateEcosystem(nodes)

					for _, nd := range nodes {
						decayDamage := 0
						conflictMultiplier := apertureDaoConflictMultiplierForProfile(daoProfile, nd.Path)
						decayDamage += int(float64(nd.MaxHP) * 0.10 * float64(yearsDiff) * (1.0 - conflictMultiplier))
						if overgrazing && (nd.NodeType == "FARMING_ZONE" || nd.ZoneType == "khu_trong" || nd.NodeType == "MOON_ORCHID_FIELD" || nd.NodeType == "GREEN_BAMBOO_GROVE" || nd.NodeType == "SWORD_LEAF_GRASS") {
							decayDamage += int(float64(nd.MaxHP) * 0.05 * float64(yearsDiff))
						}

						if decayDamage > 0 {
							newHP := nd.HP - decayDamage
							isDestroyed := false
							if newHP <= 0 {
								newHP = 0
								isDestroyed = true
							}
							nodeUps = append(nodeUps, nodeUp{id: nd.ID, hp: newHP, destroyed: isDestroyed})

							if isDestroyed {
								msgs = append(msgs, msgIns{receiver: ap.PlayerID, content: "Điểm tài nguyên " + nd.Name + " đã bị phá hủy do bất ổn sinh thái hoặc xung đột Đạo ngân!"})
							}
						}
					}

					// Variant Human Tick — read from pre-loaded data
					vhs := vhByAperture[ap.ID]
					if len(vhs) > 0 {
						totalFoodProd := int64(0)
						for _, nd := range nodes {
							if nd.NodeType == "FARMING_ZONE" || nd.NodeType == "BREEDING_ZONE" || nd.ZoneType == "khu_trong" || nd.ZoneType == "khu_nuoi" || nd.NodeType == "MOON_ORCHID_FIELD" || nd.NodeType == "GREEN_BAMBOO_GROVE" || nd.NodeType == "SWORD_LEAF_GRASS" || nd.NodeType == "SPIRIT_DEER_PASTURE" {
								totalFoodProd += nd.ProductionAmount
							}
						}

						for _, v := range vhs {
							happinessChange := 0.0
							oppressionChange := 0.0

							foodNeeded := float64(v.population) / 100.0
							if float64(totalFoodProd) < foodNeeded {
								happinessChange -= 10.0 * float64(yearsDiff)
							} else {
								happinessChange += 2.0 * float64(yearsDiff)
							}

							switch v.laborIntensity {
							case "LOW":
								happinessChange += 5.0 * float64(yearsDiff)
								oppressionChange -= 5.0 * float64(yearsDiff)
							case "NORMAL":
							case "HIGH":
								happinessChange -= 10.0 * float64(yearsDiff)
								oppressionChange += 10.0 * float64(yearsDiff)
							case "EXPLOIT":
								happinessChange -= 25.0 * float64(yearsDiff)
								oppressionChange += 25.0 * float64(yearsDiff)
							}

							newHappiness := math.Max(0.0, math.Min(100.0, v.happiness+happinessChange))
							newOppression := math.Max(0.0, math.Min(100.0, v.oppression+oppressionChange))

							rebellionRisk := (100.0 - newHappiness + newOppression) / 4.0
							newPop := v.population
							if newHappiness < 30.0 || newOppression > 70.0 {
								if rand.Float64() < (rebellionRisk / 100.0) {
									newPop = int(float64(v.population) * 0.8)
									// Rebellion damage is terminal at zero HP: delete the ruin.
									_, _ = db.DB.Exec(`
						WITH victims AS MATERIALIZED (
							SELECT id FROM aperture_resource_nodes
							WHERE aperture_id = $1 AND is_destroyed = false
							ORDER BY random() LIMIT 2
						), deleted AS (
							DELETE FROM aperture_resource_nodes n
							USING victims v
							WHERE n.id = v.id AND n.hp <= 1
							RETURNING n.id
						)
						UPDATE aperture_resource_nodes n
						SET hp = n.hp / 2, updated_at = NOW()
						FROM victims v
						WHERE n.id = v.id AND n.hp > 1
					`, ap.ID)
									ap.ApertureHP = math.Max(0.0, ap.ApertureHP-ap.MaxApertureHP*0.05)
									_, _ = db.DB.Exec(`UPDATE immortal_apertures SET aperture_hp = $1 WHERE id = $2`, ap.ApertureHP, ap.ID)
									msgs = append(msgs, msgIns{receiver: ap.PlayerID, content: "Bộ tộc " + v.race + " trong Tiên Khiếu đã nổi loạn do bị áp bức dữ dội! Một số điểm tài nguyên và Tiên Khiếu đã bị hư hại."})
								}
							}

							settlementCount := 0
							targetSettlementType := ""
							switch v.race {
							case "HAIRY_MAN":
								targetSettlementType = "HAIRY_MAN_SETTLEMENT"
							case "ROCKMAN":
								targetSettlementType = "ROCKMAN_TRIBE"
							case "FEATHERMAN":
								targetSettlementType = "FEATHERMAN_CANYON"
							}
							for _, nd := range nodes {
								if nd.NodeType == targetSettlementType {
									settlementCount++
								}
							}

							maxPop := 200 * settlementCount
							if settlementCount > 0 {
								if newHappiness > 60.0 {
									newPop = int(float64(v.population) * math.Pow(1.05, float64(yearsDiff)))
								}
								if newPop > maxPop {
									newPop = maxPop
								}
							} else {
								newPop = int(float64(v.population) * math.Pow(0.90, float64(yearsDiff)))
							}

							vhUps = append(vhUps, vhUp{id: v.id, pop: newPop, happy: newHappiness, oppress: newOppression})
						}
					}
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
				err := spawnTribulationEventWorker(ap.ID, ap.PlayerID, ap.EarthCalamityCount+ap.HeavenlyTribulationCount+ap.GrandTribulationCount+ap.MyriadTribulationCount+ap.ChaosCalamityCount+1, ap.ApertureHP, newYearsPassed, spawnType, rewardMult)
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
		if len(lifeUps)+len(manaUps)+len(nodeUps)+len(vhUps)+len(msgs) > 0 {
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

				// Batch UPDATE node HP
				if len(nodeUps) > 0 {
					ids := make([]string, 0, len(nodeUps))
					hps := make([]int, 0, len(nodeUps))
					destroyedIDs := make([]string, 0, len(nodeUps))
					for _, u := range nodeUps {
						if u.destroyed {
							destroyedIDs = append(destroyedIDs, u.id)
							continue
						}
						ids = append(ids, u.id)
						hps = append(hps, u.hp)
					}
					if len(ids) > 0 {
						if _, err := tx.Exec(`UPDATE aperture_resource_nodes AS n SET hp = v.hp, updated_at = NOW() FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS hp) AS v WHERE n.id = v.id`, pq.Array(ids), pq.Array(hps)); err != nil {
							logger.S.Errorf("CheckAndTriggerTribulations: batch update nodes failed: %v", err)
						}
					}
					if len(destroyedIDs) > 0 {
						if _, err := tx.Exec(`DELETE FROM aperture_resource_nodes WHERE id = ANY($1::uuid[])`, pq.Array(destroyedIDs)); err != nil {
							logger.S.Errorf("CheckAndTriggerTribulations: delete destroyed nodes failed: %v", err)
						}
					}
				}

				// Batch UPDATE variant humans
				if len(vhUps) > 0 {
					ids := make([]string, len(vhUps))
					pops := make([]int, len(vhUps))
					happys := make([]float64, len(vhUps))
					oppresses := make([]float64, len(vhUps))
					for i, u := range vhUps {
						ids[i] = u.id
						pops[i] = u.pop
						happys[i] = u.happy
						oppresses[i] = u.oppress
					}
					if _, err := tx.Exec(`UPDATE aperture_variant_humans AS v SET population = d.pop, happiness = d.happy, oppression = d.oppress, updated_at = NOW() FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS pop, unnest($3::float8[]) AS happy, unnest($4::float8[]) AS oppress) AS d WHERE v.id = d.id`, pq.Array(ids), pq.Array(pops), pq.Array(happys), pq.Array(oppresses)); err != nil {
						logger.S.Errorf("CheckAndTriggerTribulations: batch update variant humans failed: %v", err)
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

	}

	// 2. Check and auto-resolve pending active tribulations that exceeded 24 hours
	autoRows, err := db.DB.Query(`
		SELECT id, player_id
		FROM tribulation_events
		WHERE status = 'ACTIVE' AND started_at < NOW() - INTERVAL '24 hours'
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

func spawnTribulationEventWorker(apertureID string, playerID string, tribCount int, currentHP float64, newYearsPassed float64, tribulationType string, rewardMult float64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	maxDao := int64(0)
	for _, val := range daoMarks {
		if val > maxDao {
			maxDao = val
		}
	}

	expectedDao := 40000.0
	switch rank {
	case 6:
		expectedDao = 40000.0
	case 7:
		expectedDao = 220000.0
	case 8:
		expectedDao = 1400000.0
	case 9:
		expectedDao = 8500000.0
	default:
		if rank < 6 {
			expectedDao = 1000.0
		} else {
			expectedDao = 8500000.0
		}
	}

	daoBalancingMult := 1.0
	if expectedDao > 0 {
		ratio := float64(maxDao) / expectedDao
		if ratio < 0.25 {
			ratio = 0.25
		}
		if rank < 9 && ratio > 3.0 {
			ratio = 3.0
		}
		daoBalancingMult = ratio
	}

	// Kiếp Nạn Hỗn Loạn: khuếch đại HP/Công quái theo độ tạp (spec §5.2).
	calamityMult := 1.0
	if tribulationType == "CHAOTIC_TRIBULATION" {
		_, _, purity := gamefx.DaoPurity(daoMarks)
		calamityMult = 1.0 + (1.0-purity)*0.75
		path = chaoticHybridPath(daoMarks, path)
	}

	difficulty := 1 + tribCount
	rewardMarks := models.CalculateTribulationReward(rank, tribulationType, difficulty)
	if rewardMult <= 0 {
		rewardMult = 1.0
	}
	rewardMarks = int64(float64(rewardMarks) * rewardMult)

	// Insert Tribulation Event
	eventID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO tribulation_events (id, aperture_id, player_id, tribulation_type, path, difficulty, status, aperture_hp_before, reward_dao_marks, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE', $7, $8, NOW())
	`, eventID, apertureID, playerID, tribulationType, path, difficulty, currentHP, rewardMarks)
	if err != nil {
		return err
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

	// Apply difficulty scaling
	diffMultiplier := 1.0 + float64(difficulty)*0.2
	scaledHP, scaledAtk, scaledDef := models.ScaleMonsterStats(boss.MaxHP, boss.BaseAttack, boss.BaseDefense, bossRank, bossSubRank, boss.Type)
	bossHP := scaledHP * diffMultiplier * daoBalancingMult * calamityMult
	bossAtk := scaledAtk * diffMultiplier * daoBalancingMult * calamityMult
	bossDef := scaledDef * diffMultiplier * daoBalancingMult

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
		mHP := scaledHPm * diffMultiplier * daoBalancingMult * calamityMult
		mAtk := scaledAtkm * diffMultiplier * daoBalancingMult * calamityMult
		mDef := scaledDefm * diffMultiplier * daoBalancingMult

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

	var activeNodesCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM aperture_resource_nodes WHERE aperture_id = $1 AND is_destroyed = false`, ap.ID).Scan(&activeNodesCount)
	nodesToDestroy := int(float64(activeNodesCount) * 0.3)
	if nodesToDestroy < 1 && activeNodesCount > 0 {
		nodesToDestroy = 1
	}
	if nodesToDestroy > 0 {
		_, _ = tx.Exec(`
			DELETE FROM aperture_resource_nodes
			WHERE id IN (
				SELECT id FROM aperture_resource_nodes 
				WHERE aperture_id = $1 AND is_destroyed = false 
				ORDER BY random() 
				LIMIT $2
			)
		`, ap.ID, nodesToDestroy)
	}

	_, _ = tx.Exec(`
		UPDATE players 
		SET primeval_stones = GREATEST(0, primeval_stones - 2000), immortal_stones = GREATEST(0.0, immortal_stones - 5.0)
		WHERE id = $1
	`, ap.PlayerID)

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
		INSERT INTO tribulation_events (id, aperture_id, player_id, tribulation_type, path, difficulty, status, aperture_hp_before, aperture_hp_after, reward_dao_marks, started_at, ended_at)
		VALUES ($1, $2, $3, $4, 'general', 1, 'FAILED', $5, $6, 0, NOW(), NOW())
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
		SELECT id, aperture_id, player_id, tribulation_type, path, difficulty, started_at, aperture_hp_before
		FROM tribulation_events WHERE id = $1 AND status = 'ACTIVE'
	`, eventID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.Difficulty, &ev.StartedAt, &ev.ApertureHPBefore)
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

	var activeNodesCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM aperture_resource_nodes WHERE aperture_id = $1 AND is_destroyed = false`, ev.ApertureID).Scan(&activeNodesCount)
	nodesToDestroy := int(float64(activeNodesCount) * 0.3)
	if nodesToDestroy < 1 && activeNodesCount > 0 {
		nodesToDestroy = 1
	}
	if nodesToDestroy > 0 {
		_, _ = tx.Exec(`
			DELETE FROM aperture_resource_nodes
			WHERE id IN (
				SELECT id FROM aperture_resource_nodes 
				WHERE aperture_id = $1 AND is_destroyed = false 
				ORDER BY random() LIMIT $2
			)
		`, ev.ApertureID, nodesToDestroy)
	}

	_, _ = tx.Exec(`
		UPDATE players 
		SET primeval_stones = GREATEST(0, primeval_stones - 2000), immortal_stones = GREATEST(0.0, immortal_stones - 5.0)
		WHERE id = $1
	`, playerID)

	_, _ = tx.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content, channel)
		VALUES ('11111111-1111-1111-1111-111111111111', $1, 'Kiếp nạn quá hạn 24 giờ không giải quyết! Thiên Ý cuồng bạo tàn phá Tiên Khiếu, kiếp nạn bị phán định THẤT BẠI: mất 2,000 Nguyên Thạch, 5 Tiên Nguyên, diện tích và tốc độ thời gian bị giảm 10%, 30% tài nguyên bị san phẳng.', 'private')
	`, playerID)

	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	_ = tx.Commit()
}

type ActiveTribulationEvent struct {
	EventID         string
	ApertureID      string
	PlayerID        string
	TribulationType string
	StartedAt       time.Time
}
