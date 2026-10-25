// Package loot turns authoritative monster deaths into exactly-once durable
// rewards. Zone ticks never call this package; a manager death worker invokes it
// off the tick with a bounded queue.
package loot

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const RewardVersion = 1

var (
	ErrInvalidGrant = errors.New("invalid loot grant")
	ErrNoRecipient  = errors.New("loot recipient is missing")
)

type GrantKey struct {
	ZoneID        string
	DeathID       string
	RecipientID   uuid.UUID
	TemplateID    string
	MonsterRank   int
	RewardVersion int
}

type Reward struct {
	Kind     string `json:"kind"` // item | gu
	ID       string `json:"id"`
	Quantity int64  `json:"quantity,omitempty"`
	Rank     int    `json:"rank,omitempty"`
}

type Result struct {
	IdempotencyKey string   `json:"idempotency_key"`
	AlreadyGranted bool     `json:"already_granted"`
	Rewards        []Reward `json:"rewards,omitempty"`
}

type Service struct {
	db        *sql.DB
	inventory *services.InventoryService

	mu  sync.Mutex
	rng *rand.Rand
}

func NewService(sqlDB *sql.DB, inventory *services.InventoryService) *Service {
	seed := time.Now().UnixNano()
	var bytes [8]byte
	if _, err := cryptorand.Read(bytes[:]); err == nil {
		seed = int64(binary.LittleEndian.Uint64(bytes[:]))
	}
	return &Service{db: sqlDB, inventory: inventory, rng: rand.New(rand.NewSource(seed))}
}

func NewServiceWithRand(sqlDB *sql.DB, inventory *services.InventoryService, rng *rand.Rand) *Service {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	return &Service{db: sqlDB, inventory: inventory, rng: rng}
}

func (k GrantKey) validate() error {
	if strings.TrimSpace(k.ZoneID) == "" || strings.TrimSpace(k.DeathID) == "" || strings.TrimSpace(k.TemplateID) == "" {
		return ErrInvalidGrant
	}
	if k.RecipientID == uuid.Nil {
		return ErrNoRecipient
	}
	if k.RewardVersion <= 0 {
		return fmt.Errorf("%w: reward version must be positive", ErrInvalidGrant)
	}
	return nil
}

func (k GrantKey) IdempotencyKey() string {
	return fmt.Sprintf("%s:%s:%s:%d", k.ZoneID, k.DeathID, k.RecipientID.String(), k.RewardVersion)
}

// GrantKill claims the idempotency key and grants inventory/Gu in one
// transaction. Replaying the same key returns AlreadyGranted with no mutation.
func (s *Service) GrantKill(ctx context.Context, key GrantKey) (Result, error) {
	if s == nil || s.db == nil {
		return Result{}, errors.New("loot service has no database")
	}
	if err := key.validate(); err != nil {
		return Result{}, err
	}
	idempotencyKey := key.IdempotencyKey()
	result := Result{IdempotencyKey: idempotencyKey}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback() }()

	insert, err := tx.ExecContext(ctx, `
		INSERT INTO realtime_loot_grants
			(idempotency_key, zone_id, death_id, recipient_id, template_id, reward_version, payload)
		VALUES ($1,$2,$3,$4,$5,$6,'{}'::jsonb)
		ON CONFLICT (idempotency_key) DO NOTHING
	`, idempotencyKey, key.ZoneID, key.DeathID, key.RecipientID, key.TemplateID, key.RewardVersion)
	if err != nil {
		return result, err
	}
	affected, err := insert.RowsAffected()
	if err != nil {
		return result, err
	}
	if affected == 0 {
		result.AlreadyGranted = true
		return result, nil
	}

	template, ok := models.MonsterRegistry[key.TemplateID]
	if !ok {
		return s.finishGrant(ctx, tx, key, result, nil)
	}

	rank := key.MonsterRank
	if rank <= 0 {
		rank = 1
	}

	var allRewards []Reward

	// 1. Roll monster-specific loot table (Loot theo quái)
	if template.LootTableID != "" {
		if table, ok := models.GetLootTable(template.LootTableID); ok {
			allRewards = append(allRewards, s.roll(table, rank)...)
		} else if table, err := db.LoadLootTable(tx, template.LootTableID); err == nil {
			allRewards = append(allRewards, s.roll(table, rank)...)
		} else {
			logger.S.Warnf("GrantKill: failed to load monster loot table %q: %v", template.LootTableID, err)
		}
	}

	// 2. Roll region-specific loot table (Loot theo vùng)
	regionID := key.ZoneID
	if parts := strings.Split(regionID, "/"); len(parts) > 1 {
		regionID = strings.Trim(parts[0], "\"")
	}
	if regionID != "" && template.Type != "" {
		regRules := models.GetRegionLootRules(regionID, template.Type, rank)
		if len(regRules) > 0 {
			for _, entry := range regRules {
				s.mu.Lock()
				rollVal := s.rng.Float64()
				s.mu.Unlock()
				if rollVal <= entry.Chance {
					if regTable, ok := models.GetLootTable(entry.LootTableID); ok {
						allRewards = append(allRewards, s.roll(regTable, rank)...)
					} else if regTable, err := db.LoadLootTable(tx, entry.LootTableID); err == nil {
						allRewards = append(allRewards, s.roll(regTable, rank)...)
					}
				}
			}
		} else {
			// Fallback to DB query if in-memory registry has no rules for this region
			rows, err := tx.QueryContext(ctx, `
				SELECT loot_table_id, chance FROM region_loot_tables
				WHERE (region_id = $1 OR region_id IN (SELECT parent_id FROM regions WHERE id = $1))
				  AND lower(monster_type) = lower($2)
				  AND min_rank <= $3 AND max_rank >= $3
				  AND is_active = TRUE AND chance > 0`, regionID, template.Type, rank)
			if err == nil {
				type regLootEntry struct {
					tableID string
					chance  float64
				}
				var entries []regLootEntry
				for rows.Next() {
					var tableID string
					var chance float64
					if err := rows.Scan(&tableID, &chance); err == nil {
						entries = append(entries, regLootEntry{tableID: tableID, chance: chance})
					}
				}
				rows.Close()
				for _, entry := range entries {
					s.mu.Lock()
					rollVal := s.rng.Float64()
					s.mu.Unlock()
					if rollVal <= entry.chance {
						if regTable, err := db.LoadLootTable(tx, entry.tableID); err == nil {
							allRewards = append(allRewards, s.roll(regTable, rank)...)
						}
					}
				}
			}
		}
	}

	if len(allRewards) == 0 {
		return s.finishGrant(ctx, tx, key, result, nil)
	}

	if err := s.apply(ctx, tx, key.RecipientID, allRewards); err != nil {
		return result, err
	}
	return s.finishGrant(ctx, tx, key, result, allRewards)
}

func (s *Service) finishGrant(ctx context.Context, tx *sql.Tx, key GrantKey, result Result, rewards []Reward) (Result, error) {
	payload, err := json.Marshal(rewards)
	if err != nil {
		return result, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_loot_grants SET payload=$2::jsonb WHERE idempotency_key=$1`, result.IdempotencyKey, payload); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	result.Rewards = rewards
	return result, nil
}

func (s *Service) roll(table models.LootTable, monsterRank int) []Reward {
	s.mu.Lock()
	defer s.mu.Unlock()
	rewards := make([]Reward, 0)
	for _, entry := range table.Entries {
		if entry.Chance <= 0 || s.rng.Float64() > entry.Chance {
			continue
		}
		if entry.ItemID != "" {
			quantity := rollQuantity(s.rng, entry.MinQty, entry.MaxQty)
			if quantity <= 0 {
				continue
			}
			if _, ok := models.ItemRegistry[entry.ItemID]; !ok {
				continue
			}
			rewards = append(rewards, Reward{Kind: "item", ID: entry.ItemID, Quantity: quantity})
		}
		if entry.GuID != "" {
			rank := entry.Rank
			if rank <= 0 {
				rank = 1
			}
			if _, ok := models.GuRegistry[entry.GuID][rank]; !ok {
				continue
			}
			rewards = append(rewards, Reward{Kind: "gu", ID: entry.GuID, Rank: rank, Quantity: 1})
		}
	}
	return rewards
}

func (s *Service) apply(ctx context.Context, tx *sql.Tx, playerID uuid.UUID, rewards []Reward) error {
	items := make([]services.ItemBatchEntry, 0)
	for _, reward := range rewards {
		switch reward.Kind {
		case "item":
			category := "materials"
			if spec, ok := models.ItemRegistry[reward.ID]; ok {
				category = models.ItemCategory(spec.Type)
			}
			items = append(items, services.ItemBatchEntry{ItemID: reward.ID, Category: category, Quantity: reward.Quantity})
		case "gu":
			if err := grantGu(ctx, tx, playerID, reward); err != nil {
				return err
			}
		}
	}
	if len(items) > 0 {
		if s.inventory == nil {
			s.inventory = services.NewInventoryService(s.db)
		}
		if err := s.inventory.AddItemsBatch(tx, playerID.String(), items); err != nil {
			return fmt.Errorf("grant loot items: %w", err)
		}
	}
	return nil
}

func grantGu(ctx context.Context, tx *sql.Tx, playerID uuid.UUID, reward Reward) error {
	rank := reward.Rank
	if rank <= 0 {
		rank = 1
	}
	if _, ok := models.GuRegistry[reward.ID][rank]; !ok {
		return nil
	}
	if rank >= 6 {
		var exists bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM player_gu WHERE gu_id=$1 AND (gu_rank >= 6 OR rank >= 6))
		`, reward.ID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return nil
		}
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO player_gu
			(id, player_id, gu_id, gu_rank, rank, level, hunger, state, location, equipped, is_vital, integration_time, growth_hours)
		VALUES ($1,$2,$3,$4,$4,1,100,'ACTIVE','INVENTORY',false,false,0,0)
		ON CONFLICT (id) DO NOTHING
	`, uuid.NewString(), playerID, reward.ID, rank)
	return err
}

func rollQuantity(rng *rand.Rand, min, max int) int64 {
	if min <= 0 && max <= 0 {
		return 1
	}
	if max < min {
		max = min
	}
	if max <= 0 {
		return 0
	}
	if min <= 0 {
		min = 1
	}
	return int64(min + rng.Intn(max-min+1))
}
