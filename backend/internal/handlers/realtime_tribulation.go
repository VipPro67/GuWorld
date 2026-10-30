package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	realtimeai "github.com/vippro67/GuWorld/backend/internal/realtime/ai"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// apertureNodeTemplateID is the realtime template marker for a resource node.
// Nodes are stationary player-faction entities that tribulation monsters can
// destroy during the live encounter.
const apertureNodeTemplateID = "aperture_node"

// tribulationBossTemplateID mirrors the authored boss selection used when the
// event is created. Realtime spawns must use the same template so the client
// renders the correct monster.
func tribulationBossTemplateID(tribulationType string) string {
	switch tribulationType {
	case "EARTH_CALAMITY":
		return "monster_earth_calamity_boss"
	case "HEAVENLY_TRIBULATION":
		return "monster_heaven_tribulation_boss"
	case "GRAND_TRIBULATION":
		return "monster_grand_calamity_boss"
	case "MYRIAD_TRIBULATION":
		return "monster_myriad_tribulation_boss"
	case "CHAOS_CALAMITY", "CHAOTIC_TRIBULATION":
		return "monster_chaos_calamity_boss"
	default:
		return "monster_earth_calamity_boss"
	}
}

// tribulationNodeDamage mirrors the legacy grid worker's fixed damage per hit
// so switching to realtime does not change node durability pacing.
func tribulationNodeDamage(tribulationType string) float64 {
	switch tribulationType {
	case "HEAVENLY_TRIBULATION":
		return 2
	case "GRAND_TRIBULATION":
		return 4
	case "MYRIAD_TRIBULATION":
		return 8
	case "CHAOS_CALAMITY":
		return 16
	case "CHAOTIC_TRIBULATION":
		return 24
	default:
		return 1
	}
}

// LoadRealtimeTribulationSpawns builds the per-event encounter from the durable
// rows. This is the only source of realtime tribulation stats: the rows already
// carry the shared rank/sub-rank scale used by realtime combat.
// ok is false when the player has no active tribulation event.
func LoadRealtimeTribulationSpawns(ctx context.Context, playerID uuid.UUID) (string, []realtimeworld.MonsterSpawn, bool, error) {
	if db.DB == nil {
		return "", nil, false, nil
	}
	var eventID uuid.UUID
	var apertureID uuid.UUID
	var tribulationType string
	err := db.DB.QueryRowContext(ctx, `
		SELECT id, aperture_id, tribulation_type
		FROM tribulation_events
		WHERE player_id = $1 AND status = 'ACTIVE'
		ORDER BY started_at DESC
		LIMIT 1
	`, playerID).Scan(&eventID, &apertureID, &tribulationType)
	if err == sql.ErrNoRows {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}

	bossTemplateID := tribulationBossTemplateID(tribulationType)
	nodeDamage := tribulationNodeDamage(tribulationType)
	spawns := make([]realtimeworld.MonsterSpawn, 0, 8)

	enemyRows, err := db.DB.QueryContext(ctx, `
		SELECT id, name, enemy_type, hp, max_hp, attack, defense, x, y, is_boss
		FROM tribulation_enemies
		WHERE event_id = $1 AND is_dead = false
		ORDER BY is_boss DESC, id
	`, eventID)
	if err != nil {
		return "", nil, false, err
	}
	defer enemyRows.Close()
	for enemyRows.Next() {
		var (
			enemyID                    uuid.UUID
			name, enemyType            string
			hp, maxHP, attack, defense float64
			x, y                       int
			isBoss                     bool
		)
		if err := enemyRows.Scan(&enemyID, &name, &enemyType, &hp, &maxHP, &attack, &defense, &x, &y, &isBoss); err != nil {
			return "", nil, false, err
		}
		templateID := "monster_tribulation_beast"
		templateType := "Tribulation_Monster"
		damageToNodes := nodeDamage
		if isBoss || strings.EqualFold(enemyType, "BOSS") {
			templateID = bossTemplateID
			templateType = "Tribulation_Boss"
			damageToNodes *= 2
		}
		stats := realtimeai.DefaultStats(templateType)
		spawns = append(spawns, realtimeworld.MonsterSpawn{
			TemplateID: templateID, Name: name,
			X: projectApertureCell(x), Y: projectApertureCell(y),
			MaxHP: math.Max(1, maxHP), CurrentHP: math.Max(1, hp),
			AttackDamage: math.Max(0, attack), NodeDamage: damageToNodes, Defense: math.Max(0, defense),
			Speed: stats.Speed, Radius: stats.Radius,
			AggroRadius: stats.AggroRadius, LeashRadius: stats.LeashRadius,
			AttackRange: stats.AttackRange, PatrolRadius: stats.PatrolRadius,
			AttackCooldownTicks: stats.AttackCooldownTicks,
			SplashRadius:        stats.SplashRadius, SplashDamage: stats.SplashDamage,
			PersistentID: enemyID.String(),
			NoRespawn:    true,
		})
	}
	if err := enemyRows.Err(); err != nil {
		return "", nil, false, err
	}

	nodeRows, err := db.DB.QueryContext(ctx, `
		SELECT id, name, hp, max_hp, defense_power, x, y, blocks_movement, node_category
		FROM aperture_resource_nodes
		WHERE aperture_id = $1 AND is_destroyed = false
		ORDER BY y ASC, x ASC
	`, apertureID)
	if err != nil {
		return "", nil, false, err
	}
	defer nodeRows.Close()
	for nodeRows.Next() {
		var (
			nodeID             uuid.UUID
			name               string
			hp, maxHP, defense int
			x, y               int
			blocksMovement     bool
			nodeCategory       string
		)
		if err := nodeRows.Scan(&nodeID, &name, &hp, &maxHP, &defense, &x, &y, &blocksMovement, &nodeCategory); err != nil {
			return "", nil, false, err
		}
		// Defensive structures are intentionally excluded from the live
		// tribulation encounter; their durable state stays untouched.
		if strings.EqualFold(strings.TrimSpace(nodeCategory), "DEFENSE") {
			continue
		}
		radius := 0.6
		if blocksMovement {
			radius = 0.8
		}
		spawns = append(spawns, realtimeworld.MonsterSpawn{
			TemplateID: apertureNodeTemplateID, Name: name,
			X: projectApertureCell(x), Y: projectApertureCell(y),
			MaxHP: math.Max(1, float64(maxHP)), CurrentHP: math.Max(1, float64(hp)),
			Defense: math.Max(0, float64(defense)),
			Speed:   0, Radius: radius, Passive: true,
			PersistentID: nodeID.String(),
			NoRespawn:    true,
			Node:         true,
		})
	}
	if err := nodeRows.Err(); err != nil {
		return "", nil, false, err
	}

	return eventID.String(), spawns, true, nil
}

func clampApertureCell(value int) float64 {
	// Keep an inner margin on the persistent 20x20 aperture grid.
	if value < 2 {
		return 2
	}
	if value > 17 {
		return 17
	}
	return float64(value)
}

func projectApertureCell(value int) float64 {
	// The realtime encounter is a 40x40 projection of the persistent 20x20 grid.
	return clampApertureCell(value) * 2
}

// PersistRealtimeTribulationSnapshot copies authoritative live HP for the
// event's monsters and nodes back to their durable rows. It is invoked by the
// periodic checkpoint, disconnect flush and every tribulation handoff so a
// crash cannot resurrect destroyed nodes or killed enemies.
func PersistRealtimeTribulationSnapshot(ctx context.Context, playerID uuid.UUID, snapshot sim.Snapshot) error {
	if db.DB == nil || len(snapshot.Entities) == 0 {
		return nil
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, entity := range snapshot.Entities {
		persistentID := strings.TrimSpace(entity.AI.PersistentID)
		if persistentID == "" {
			continue
		}
		hp := int(math.Max(0, math.Round(entity.Vitals.HP)))
		if strings.HasPrefix(string(entity.ID), "n:") {
			if _, err := tx.ExecContext(ctx, `
				UPDATE aperture_resource_nodes
				SET hp = CASE WHEN is_destroyed OR $1 <= 0 THEN 0 ELSE $1 END,
				    is_destroyed = is_destroyed OR $1 <= 0,
				    updated_at = NOW()
				WHERE id = $2
			`, hp, persistentID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE tribulation_enemies
			SET hp = $1, is_dead = ($1 <= 0), updated_at = NOW()
			WHERE id = $2 AND is_dead = false
		`, hp, persistentID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// markRealtimeTribulationEnemyDead persists one death and returns the owning
// event plus the number of enemies still alive.
func markRealtimeTribulationEnemyDead(ctx context.Context, enemyID string) (uuid.UUID, int, error) {
	var eventID uuid.UUID
	if err := db.DB.QueryRowContext(ctx, `
		UPDATE tribulation_enemies
		SET hp = 0, is_dead = true, updated_at = NOW()
		WHERE id = $1 AND is_dead = false
		RETURNING event_id
	`, enemyID).Scan(&eventID); err != nil {
		if err == sql.ErrNoRows {
			if err := db.DB.QueryRowContext(ctx, `SELECT event_id FROM tribulation_enemies WHERE id = $1`, enemyID).Scan(&eventID); err != nil {
				return uuid.Nil, 0, err
			}
		} else {
			return uuid.Nil, 0, err
		}
	}
	var alive int
	if err := db.DB.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, eventID).Scan(&alive); err != nil {
		return uuid.Nil, 0, err
	}
	return eventID, alive, nil
}

// HandleRealtimeTribulationDeath persists a realtime tribulation kill and, when
// it was the last living enemy, settles victory automatically. It is installed
// as part of the realtime manager's death handler.
func HandleRealtimeTribulationDeath(ctx context.Context, death realtimeworld.DeathEvent) error {
	if death.IsNode {
		if death.PersistentID != "" {
			_, _ = db.DB.ExecContext(ctx, `
				UPDATE aperture_resource_nodes
				SET hp = 0, is_destroyed = true, updated_at = NOW()
				WHERE id = $1
			`, death.PersistentID)
		}
		return nil
	}
	if death.PersistentID == "" {
		return nil
	}
	playerID, err := uuid.Parse(string(death.SourceID))
	if err != nil {
		return nil
	}
	eventID, alive, err := markRealtimeTribulationEnemyDead(ctx, death.PersistentID)
	if err != nil {
		return fmt.Errorf("mark tribulation enemy dead: %w", err)
	}
	if alive > 0 {
		return nil
	}
	finishHandoff, abortHandoff, _, handoffStatus, _, handoffMessage := beginTribulationHandoff(playerID)
	if handoffStatus == http.StatusServiceUnavailable {
		return fmt.Errorf("tribulation victory handoff unavailable: %s", handoffMessage)
	}
	defer abortHandoff()

	result, status, settleErr := settleRealtimeTribulationVictory(ctx, playerID, eventID, "realtime")
	if settleErr != nil {
		if status == http.StatusNotFound {
			// Another path (auto-reconcile worker, manual settle or a previous
			// retry) already closed this event. Releasing the handoff and
			// returning nil prevents a permanent retry loop.
			if handoffStatus == 0 {
				finishHandoff()
			}
			return nil
		}
		return fmt.Errorf("settle realtime tribulation victory (%d): %w", status, settleErr)
	}
	if handoffStatus == 0 {
		finishHandoff()
	}
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
	notify.Send(db.DB, playerID, "tribulation_victory", "Vượt Kiếp Thành Công", result.Message)
	logger.S.Infof("Realtime tribulation auto-settled for %s: %s", playerID, result.Message)
	return nil
}

// HandleRealtimeTribulationPlayerDeath exits a defeated character without a
// death drop. The event stays ACTIVE; the saved pre-tribulation position is
// reused and HP is restored to 1 so the character can recover.
func HandleRealtimeTribulationPlayerDeath(ctx context.Context, death realtimeworld.DeathEvent) error {
	playerID, err := uuid.Parse(string(death.TargetID))
	if err != nil {
		return nil
	}
	finishHandoff, abortHandoff, realtimeTransfer, status, _, message := beginTribulationHandoff(playerID)
	if status != 0 {
		return fmt.Errorf("tribulation death handoff unavailable: %s", message)
	}
	defer abortHandoff()

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tribulation defeat exit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	exitRegion, exitX, exitY, err := resolveApertureExitLocation(ctx, tx, playerID)
	if err != nil {
		return fmt.Errorf("resolve tribulation defeat exit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE players
		SET hp = 1,
		    current_region = $1, world_zone_id = $1, map_instance_id = NULL,
		    world_x = $2, world_y = $3, world_safe_x = $2, world_safe_y = $3,
		    x = $4, y = $5, world_location_revision = world_location_revision + 1
		WHERE id = $6
	`, exitRegion, exitX, exitY, int(exitX), int(exitY), playerID); err != nil {
		return fmt.Errorf("persist defeated tribulation exit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit defeated tribulation exit: %w", err)
	}
	if status == 0 {
		finishHandoff()
	}
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
	notify.Send(db.DB, playerID, "tribulation_defeat", "Kiệt Sức Trong Tiên Khiếu",
		"Đạo hữu đã kiệt sức trước quái tai kiếp và được đưa về vị trí cũ. Không rơi rớt vật phẩm.")
	logger.S.Infof("Realtime tribulation defeat exit for %s (realtime=%v)", playerID, realtimeTransfer)
	return nil
}

// RealtimeExitTribulation always returns the character to the saved aperture
// exit. Unlike realtime-escape it does not need an event ID, so it also works
// after a refresh, an expired event or a party/UI desync.
func RealtimeExitTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	finishHandoff, abortHandoff, realtimeTransfer, status, code, message := beginTribulationHandoff(playerID)
	if status != 0 {
		c.JSON(status, gin.H{"code": code, "error": message})
		return
	}
	defer abortHandoff()

	ctx := c.Request.Context()
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu rời Tiên Khiếu"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	exitRegion, exitX, exitY, exitErr := resolveApertureExitLocation(ctx, tx, playerID)
	if exitErr != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "aperture_exit_unavailable", "error": exitErr.Error()})
		return
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE players
		SET current_region = $1, world_zone_id = $1, world_x = $2, world_y = $3, x = $4, y = $5,
		    world_location_revision = world_location_revision + 1
		WHERE id = $6
	`, exitRegion, exitX, exitY, int(exitX), int(exitY), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi rời khỏi Tiên Khiếu"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu vị trí trở về"})
		return
	}
	if status == 0 {
		finishHandoff()
	}
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Đã rời khỏi Tiên Khiếu trở về thế giới thực!",
		"return_region":     exitRegion,
		"return_x":          exitX,
		"return_y":          exitY,
		"realtime_transfer": realtimeTransfer,
	})
}
