package handlers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Position updates and watchtower damage are processed per event in isolated transactions with timeouts.
func ProcessActiveTribulationGrid() error {
	// 1. Fetch all active tribulation events outside of long-held transactions
	rows, err := db.DB.Query(`
		SELECT id, aperture_id, player_id, tribulation_type, started_at
		FROM tribulation_events 
		WHERE status = 'ACTIVE'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []ActiveTribulationEvent
	for rows.Next() {
		var ev ActiveTribulationEvent
		if err := rows.Scan(&ev.EventID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.StartedAt); err == nil {
			events = append(events, ev)
		}
	}
	rows.Close()

	// 2. Process each event in its own transaction so a single failing or slow event doesn't block the rest
	for _, ev := range events {
		if err := processActiveTribulationEvent(ev); err != nil {
			logger.S.Errorf("Error processing tribulation event %s for player %s: %v", ev.EventID, ev.PlayerID, err)
		}
	}
	return nil
}

func processActiveTribulationEvent(ev ActiveTribulationEvent) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout = '30000'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '10000'`); err != nil {
		return err
	}
	suppressed, err := suppressTribulationsForDeadAperture(tx, ev.PlayerID)
	if err != nil {
		return err
	}
	if suppressed {
		return tx.Commit()
	}

	// Batch accumulators – reset per event.
	type posUpdate struct {
		ID string
		X  int
		Y  int
	}
	type nodeDmgAcc struct {
		HP          int
		IsDestroyed bool
	}

	// Freeze the tribulation while the player is offline: enemies hold their
	// position and never damage nodes. Bump started_at so the 24h auto-resolve
	// window only counts online time.
	if !isPlayerOnline(ev.PlayerID) {
		_, _ = tx.Exec(`UPDATE tribulation_events SET started_at = NOW() WHERE id = $1`, ev.EventID)
		return tx.Commit()
	}

	// Grace period after spawn/return: enemies stay idle for a short window.
	if time.Since(ev.StartedAt) < tribulationGracePeriod {
		return tx.Commit()
	}

	// Fixed node damage per hit, based on tribulation type
	nodeDmgPerHit := 1 // EARTH_CALAMITY
	switch ev.TribulationType {
	case "HEAVENLY_TRIBULATION":
		nodeDmgPerHit = 2
	case "GRAND_TRIBULATION":
		nodeDmgPerHit = 4
	case "MYRIAD_TRIBULATION":
		nodeDmgPerHit = 8
	case "CHAOS_CALAMITY":
		nodeDmgPerHit = 16
	case "CHAOTIC_TRIBULATION":
		// Hybrid kiếp nạn dồn hỏa lực phá node (spec §5.2): x1.5 above CHAOS.
		nodeDmgPerHit = 24
	}

	// Fetch living enemies
	enRows, err := tx.Query(`
		SELECT id, name, enemy_type, x, y 
		FROM tribulation_enemies 
		WHERE event_id = $1 AND is_dead = false
	`, ev.EventID)
	if err != nil {
		return fmt.Errorf("failed to query tribulation enemies: %w", err)
	}

	type TEnemy struct {
		ID        string
		Name      string
		EnemyType string
		X         int
		Y         int
		NodeDmg   int
	}
	var enemies []TEnemy
	for enRows.Next() {
		var en TEnemy
		if err := enRows.Scan(&en.ID, &en.Name, &en.EnemyType, &en.X, &en.Y); err != nil {
			enRows.Close()
			return fmt.Errorf("failed to scan enemy row: %w", err)
		}
		// Minions chip away at the mine; the boss deals double the minion damage
		en.NodeDmg = nodeDmgPerHit
		if en.EnemyType == "BOSS" {
			en.NodeDmg = nodeDmgPerHit * 2
		}
		enemies = append(enemies, en)
	}
	enRows.Close()

	if len(enemies) == 0 {
		return tx.Commit()
	}

	// Fetch non-destroyed resource nodes
	nodeRows, err := tx.Query(`
		SELECT id, name, hp, x, y,
			COALESCE(node_category, 'RESOURCE'), COALESCE(blocks_movement, false),
			COALESCE(defense_power, 0), COALESCE(attack_range, 0),
			COALESCE(hp_bonus, 0), COALESCE(damage_reduction, 0)
		FROM aperture_resource_nodes 
		WHERE aperture_id = $1 AND is_destroyed = false
	`, ev.ApertureID)
	if err != nil {
		return fmt.Errorf("failed to query aperture resource nodes: %w", err)
	}

	type TNode struct {
		ID              string
		Name            string
		HP              int
		X               int
		Y               int
		NodeCategory    string
		BlocksMovement  bool
		DefensePower    int
		AttackRange     int
		HPBonus         int
		DamageReduction int
		Cells           []models.ApertureGridCell
	}
	var nodes []TNode
	for nodeRows.Next() {
		var nd TNode
		if err := nodeRows.Scan(&nd.ID, &nd.Name, &nd.HP, &nd.X, &nd.Y,
			&nd.NodeCategory, &nd.BlocksMovement,
			&nd.DefensePower, &nd.AttackRange,
			&nd.HPBonus, &nd.DamageReduction); err != nil {
			nodeRows.Close()
			return fmt.Errorf("failed to scan resource node row: %w", err)
		}
		nodes = append(nodes, nd)
	}
	nodeRows.Close()
	cellsByNode, err := loadApertureNodeCells(tx, ev.ApertureID)
	if err != nil {
		return fmt.Errorf("failed to load aperture node cells: %w", err)
	}
	for i := range nodes {
		nodes[i].Cells = cellsByNode[nodes[i].ID]
		if len(nodes[i].Cells) == 0 {
			nodes[i].Cells = []models.ApertureGridCell{{X: nodes[i].X, Y: nodes[i].Y}}
		}
	}

	// Accumulate batch updates for this event
	var pendingPos []posUpdate
	pendingNodeDmg := make(map[string]*nodeDmgAcc)
	type pendingMsg struct {
		ReceiverID string
		Content    string
	}
	var pendingMsgs []pendingMsg
	// Track tower damage to enemies (enemyID -> total damage)
	pendingTowerDmg := make(map[string]int)

	if len(nodes) == 0 {
		// No nodes to attack: enemies wander around aimlessly
		for _, en := range enemies {
			dx, dy := 0, 0
			switch rand.Intn(4) {
			case 0:
				dx = 1
			case 1:
				dx = -1
			case 2:
				dy = 1
			case 3:
				dy = -1
			}
			newX := en.X + dx
			if newX < 0 {
				newX = 0
			} else if newX > 19 {
				newX = 19
			}
			newY := en.Y + dy
			if newY < 0 {
				newY = 0
			} else if newY > 19 {
				newY = 19
			}
			pendingPos = append(pendingPos, posUpdate{ID: en.ID, X: newX, Y: newY})
		}
	} else {
		applyAreaDamage := func(en TEnemy) bool {
			hitAny := false
			var destroyedIDs []string
			for i := range nodes {
				target := &nodes[i]
				if target.HP <= 0 {
					continue
				}
				hitCells := 0
				for _, cell := range target.Cells {
					dx, dy := cell.X-en.X, cell.Y-en.Y
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}
					if dx <= 1 && dy <= 1 {
						hitCells++
					}
				}
				if hitCells == 0 {
					continue
				}
				hitAny = true
				damageReduction := 0
				for j := range nodes {
					buff := &nodes[j]
					if buff.ID == target.ID || buff.HP <= 0 || buff.DamageReduction <= 0 {
						continue
					}
					dx, dy := buff.X-target.X, buff.Y-target.Y
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}
					distance := dx + dy
					if buff.AttackRange == 0 || distance <= buff.AttackRange {
						damageReduction += buff.DamageReduction
					}
				}
				if damageReduction > 80 {
					damageReduction = 80
				}
				damage := en.NodeDmg * hitCells * (100 - damageReduction) / 100
				if damage < 1 {
					damage = 1
				}
				target.HP -= damage
				isDestroyed := target.HP <= 0
				if isDestroyed {
					target.HP = 0
					destroyedIDs = append(destroyedIDs, target.ID)
					pendingMsgs = append(pendingMsgs, pendingMsg{ReceiverID: ev.PlayerID, Content: fmt.Sprintf("Tai kiếp đã phá hủy hoàn toàn %s!", target.Name)})
				}
				pendingNodeDmg[target.ID] = &nodeDmgAcc{HP: target.HP, IsDestroyed: isDestroyed}
			}
			if len(destroyedIDs) > 0 {
				remaining := nodes[:0]
				for _, node := range nodes {
					destroyed := false
					for _, id := range destroyedIDs {
						if node.ID == id {
							destroyed = true
							break
						}
					}
					if !destroyed {
						remaining = append(remaining, node)
					}
				}
				nodes = remaining
			}
			return hitAny
		}

		// Process each enemy
		for _, en := range enemies {
			// Find nearest non-wall node (prefer defense nodes, then resource)
			var nearestNode *TNode
			var nearestCell models.ApertureGridCell
			minDist := 999999
			for i := range nodes {
				nd := &nodes[i]
				nodeDist := 999999
				var nodeCell models.ApertureGridCell
				for _, cell := range nd.Cells {
					dx, dy := cell.X-en.X, cell.Y-en.Y
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}
					dist := dx + dy
					if dist < nodeDist {
						nodeDist, nodeCell = dist, cell
					}
				}
				if nodeDist < minDist || (nodeDist == minDist && nearestNode != nil && nd.NodeCategory == "DEFENSE" && nearestNode.NodeCategory != "DEFENSE") {
					minDist, nearestNode, nearestCell = nodeDist, nd, nodeCell
				}
			}

			if nearestNode == nil {
				continue
			}

			if applyAreaDamage(en) {
				continue
			}

			dx, dy := nearestCell.X-en.X, nearestCell.Y-en.Y
			if dx > 0 {
				en.X++
			} else if dx < 0 {
				en.X--
			}
			if dy > 0 {
				en.Y++
			} else if dy < 0 {
				en.Y--
			}
			pendingPos = append(pendingPos, posUpdate{ID: en.ID, X: en.X, Y: en.Y})
		}

		// Watchtower attacks: towers shoot at enemies in range
		for i := range nodes {
			tower := &nodes[i]
			if tower.NodeCategory != "DEFENSE" || tower.DefensePower <= 0 || tower.AttackRange <= 0 || tower.HP <= 0 {
				continue
			}
			for _, en := range enemies {
				wdx := tower.X - en.X
				if wdx < 0 {
					wdx = -wdx
				}
				wdy := tower.Y - en.Y
				if wdy < 0 {
					wdy = -wdy
				}
				wdist := wdx + wdy
				if wdist <= tower.AttackRange {
					pendingTowerDmg[en.ID] += tower.DefensePower
				}
			}
		}
	}

	// --- Batch: update enemy positions ---
	if len(pendingPos) > 0 {
		ids := make([]string, len(pendingPos))
		xs := make([]int, len(pendingPos))
		ys := make([]int, len(pendingPos))
		for i, p := range pendingPos {
			ids[i] = p.ID
			xs[i] = p.X
			ys[i] = p.Y
		}
		_, err = tx.Exec(`
			UPDATE tribulation_enemies AS e
			SET x = v.x, y = v.y, updated_at = NOW()
			FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS x, unnest($3::int[]) AS y) AS v
			WHERE e.id = v.id
		`, pq.Array(ids), pq.Array(xs), pq.Array(ys))
		if err != nil {
			return fmt.Errorf("batch update enemy positions: %w", err)
		}
	}

	// --- Batch: update resource node HPs ---
	if len(pendingNodeDmg) > 0 {
		ndIDs := make([]string, 0, len(pendingNodeDmg))
		ndHPs := make([]int, 0, len(pendingNodeDmg))
		destroyedIDs := make([]string, 0, len(pendingNodeDmg))
		for id, acc := range pendingNodeDmg {
			if acc.IsDestroyed {
				destroyedIDs = append(destroyedIDs, id)
				continue
			}
			ndIDs = append(ndIDs, id)
			ndHPs = append(ndHPs, acc.HP)
		}
		if len(ndIDs) > 0 {
			if _, err = tx.Exec(`UPDATE aperture_resource_nodes AS n SET hp = v.hp, updated_at = NOW() FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS hp) AS v WHERE n.id = v.id`, pq.Array(ndIDs), pq.Array(ndHPs)); err != nil {
				return fmt.Errorf("batch update node HP: %w", err)
			}
		}
		if len(destroyedIDs) > 0 {
			if _, err = tx.Exec(`DELETE FROM aperture_resource_nodes WHERE id = ANY($1::uuid[])`, pq.Array(destroyedIDs)); err != nil {
				return fmt.Errorf("delete destroyed nodes: %w", err)
			}
		}
	}

	// --- Batch: apply watchtower damage to enemies ---
	if len(pendingTowerDmg) > 0 {
		tIDs := make([]string, 0, len(pendingTowerDmg))
		tDmg := make([]int, 0, len(pendingTowerDmg))
		for enemyID, dmg := range pendingTowerDmg {
			tIDs = append(tIDs, enemyID)
			tDmg = append(tDmg, dmg)
		}
		_, err = tx.Exec(`
			UPDATE tribulation_enemies AS e
			SET hp = GREATEST(e.hp - v.dmg, 0),
			    is_dead = CASE WHEN e.hp - v.dmg <= 0 THEN true ELSE false END,
			    updated_at = NOW()
			FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS dmg) AS v
			WHERE e.id = v.id
		`, pq.Array(tIDs), pq.Array(tDmg))
		if err != nil {
			return fmt.Errorf("batch watchtower damage: %w", err)
		}
	}

	// --- Batch: insert system messages ---
	if len(pendingMsgs) > 0 {
		for _, m := range pendingMsgs {
			_, err = tx.Exec(`
				INSERT INTO messages (sender_id, receiver_id, content, channel)
				VALUES ('11111111-1111-1111-1111-111111111111', $1, $2, 'private')
			`, m.ReceiverID, m.Content)
			if err != nil {
				return fmt.Errorf("insert tribulation message: %w", err)
			}
		}
	}

	return tx.Commit()
}

func RegenerateImmortalApertures() error {
	// Cheap guard: skip the UPDATE entirely when no aperture needs healing.
	var needsHealing bool
	if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM immortal_apertures WHERE state != 'DESTROYED' AND aperture_hp < max_aperture_hp)`).Scan(&needsHealing); err != nil {
		return err
	}
	if !needsHealing {
		return nil
	}

	// Heal aperture_hp by 0.5% of max_aperture_hp per minute (minimum 10 HP per minute)
	// If state is DAMAGED and aperture_hp reaches max_aperture_hp, set state back to STABLE
	_, err := db.DB.Exec(`
			UPDATE immortal_apertures
			SET 
				aperture_hp = LEAST(max_aperture_hp, aperture_hp + GREATEST(10, max_aperture_hp / 200)),
				state = CASE 
					WHEN (aperture_hp + GREATEST(10, max_aperture_hp / 200)) >= max_aperture_hp THEN
						CASE 
							WHEN NOT EXISTS (SELECT 1 FROM tribulation_events WHERE aperture_id = immortal_apertures.id AND status = 'ACTIVE')
							     AND NOT EXISTS (SELECT 1 FROM tribulation_enemies te JOIN tribulation_events ev ON te.event_id = ev.id WHERE ev.aperture_id = immortal_apertures.id AND te.is_dead = false)
							THEN 'STABLE'
							ELSE 'TRIBULATION_ACTIVE'
						END
					ELSE state
				END,
				updated_at = NOW()
			WHERE state != 'DESTROYED' AND aperture_hp < max_aperture_hp
		`)
	return err
}

func StartTribulationWorker(ctx context.Context) {
	ticker60 := time.NewTicker(60 * time.Second)
	ticker15 := time.NewTicker(15 * time.Second)

	go func() {
		defer ticker60.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.S.Info("Tribulation worker (60s) stopped")
				return
			case <-ticker60.C:
				err := CheckAndTriggerTribulations()
				if err != nil {
					logger.S.Errorf("Tribulation background worker error: %v", err)
				}
				errRegen := RegenerateImmortalApertures()
				if errRegen != nil {
					logger.S.Errorf("Aperture HP regeneration background worker error: %v", errRegen)
				}
			}
		}
	}()

	go func() {
		defer ticker15.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.S.Info("Tribulation worker (15s) stopped")
				return
			case <-ticker15.C:
				err := ProcessActiveTribulationGrid()
				if err != nil {
					logger.S.Errorf("Tribulation grid worker error: %v", err)
				}
			}
		}
	}()

	logger.S.Info("Tribulation and internal year worker started successfully (Interval: 60s)")
}
