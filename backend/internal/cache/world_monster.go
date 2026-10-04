package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// WorldMonster is the ephemeral, server-authoritative state of one ordinary
// world-map monster. Static data remains in PostgreSQL; this value is kept in
// Redis only for as long as the spawn is active or waiting to respawn.
type WorldMonster struct {
	InstanceID        string              `json:"instance_id"`
	SpawnID           string              `json:"spawn_id"`
	MonsterTemplateID string              `json:"monster_template_id"`
	Rank              int                 `json:"rank"`
	SubRank           int                 `json:"sub_rank"`
	RegionID          string              `json:"region_id"`
	X                 float64             `json:"x"`
	Y                 float64             `json:"y"`
	SpawnX            float64             `json:"spawn_x"`
	SpawnY            float64             `json:"spawn_y"`
	HP                float64             `json:"hp"`
	MaxHP             float64             `json:"max_hp"`
	State             string              `json:"state"`
	TargetID          string              `json:"target_id,omitempty"`
	Revision          int64               `json:"revision"`
	LastHitAt         time.Time           `json:"last_hit_at,omitempty"`
	LastSeenAt        time.Time           `json:"last_seen_at,omitempty"`
	LastUpdateAt      time.Time           `json:"last_update_at"`
	NextAttackAt      time.Time           `json:"next_attack_at,omitempty"`
	RespawnAt         time.Time           `json:"respawn_at,omitempty"`
	Statuses          []WorldStatusEffect `json:"statuses"`
	RespawnSeconds    int                 `json:"respawn_seconds"`
	AggroRadius       float64             `json:"aggro_radius"`
	AttackRange       float64             `json:"attack_range"`
	LeashRadius       float64             `json:"leash_radius"`
	Speed             float64             `json:"speed"`
	AttackIntervalMS  int64               `json:"attack_interval_ms"`
	BaseAttack        float64             `json:"base_attack"`
	BaseDefense       float64             `json:"base_defense"`
}

type WorldStatusEffect struct {
	ID        string    `json:"id"`
	Value     float64   `json:"value,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

type WorldMonsterHitResult string

const (
	WorldMonsterHit              WorldMonsterHitResult = "HIT"
	WorldMonsterFirstKill        WorldMonsterHitResult = "FIRST_KILL"
	WorldMonsterAlreadyDefeated  WorldMonsterHitResult = "ALREADY_DEFEATED"
	WorldMonsterRevisionConflict WorldMonsterHitResult = "REVISION_CONFLICT"
)

func worldMonsterKey(regionID, instanceID string) string {
	return fmt.Sprintf("world:monster:%s:%s", regionID, instanceID)
}

func worldMonsterSpawnKey(spawnID string) string { return "world:monster:spawn:" + spawnID }

func worldMonsterBucketKey(regionID string, bx, by int) string {
	return fmt.Sprintf("world:monsters:aoi:%s:%d:%d", regionID, bx, by)
}

// MaterializeWorldMonster installs one instance for a stable spawn identity.
// SETNX on the spawn pointer means concurrent AOI snapshots converge on the
// same instance rather than manufacturing duplicates.
func MaterializeWorldMonster(ctx context.Context, monster WorldMonster, bucketX, bucketY int, ttl time.Duration) (WorldMonster, bool, error) {
	if Client == nil {
		return WorldMonster{}, false, errors.New("redis unavailable")
	}
	data, err := json.Marshal(monster)
	if err != nil {
		return WorldMonster{}, false, err
	}
	keys := []string{worldMonsterSpawnKey(monster.SpawnID), worldMonsterKey(monster.RegionID, monster.InstanceID), worldMonsterBucketKey(monster.RegionID, bucketX, bucketY)}
	result, err := Client.Eval(ctx, `
		local existing = redis.call('GET', KEYS[1])
		if existing then
			if redis.call('EXISTS', ARGV[4] .. existing) == 1 then return {0, existing} end
			redis.call('DEL', KEYS[1])
		end
		redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[3])
		redis.call('SET', KEYS[2], ARGV[2], 'PX', ARGV[3])
		redis.call('SADD', KEYS[3], ARGV[1])
		redis.call('PEXPIRE', KEYS[3], ARGV[3])
		return {1, ARGV[1]}
	`, keys, monster.InstanceID, data, ttl.Milliseconds(), "world:monster:"+monster.RegionID+":").Slice()
	if err != nil {
		return WorldMonster{}, false, err
	}
	created := result[0].(int64) == 1
	instanceID, _ := result[1].(string)
	if created {
		return monster, true, nil
	}
	return GetWorldMonster(ctx, monster.RegionID, instanceID)
}

func GetWorldMonster(ctx context.Context, regionID, instanceID string) (WorldMonster, bool, error) {
	var monster WorldMonster
	if Client == nil {
		return monster, false, errors.New("redis unavailable")
	}
	raw, err := Client.Get(ctx, worldMonsterKey(regionID, instanceID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return monster, false, nil
	}
	if err != nil {
		return monster, false, err
	}
	if err := json.Unmarshal(raw, &monster); err != nil {
		return monster, false, err
	}
	return monster, true, nil
}

func WorldMonstersInBucket(ctx context.Context, regionID string, bx, by int) ([]WorldMonster, error) {
	if Client == nil {
		return nil, errors.New("redis unavailable")
	}
	ids, err := Client.SMembers(ctx, worldMonsterBucketKey(regionID, bx, by)).Result()
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	pipe := Client.Pipeline()
	cmds := make([]*redis.StringCmd, len(ids))
	for i, id := range ids {
		cmds[i] = pipe.Get(ctx, worldMonsterKey(regionID, id))
	}
	_, _ = pipe.Exec(ctx)
	out := make([]WorldMonster, 0, len(ids))
	stale := make([]interface{}, 0)
	for i, cmd := range cmds {
		raw, getErr := cmd.Bytes()
		if errors.Is(getErr, redis.Nil) {
			stale = append(stale, ids[i])
			continue
		}
		if getErr != nil {
			return nil, getErr
		}
		var monster WorldMonster
		if err := json.Unmarshal(raw, &monster); err != nil {
			return nil, err
		}
		out = append(out, monster)
	}
	if len(stale) > 0 {
		_ = Client.SRem(ctx, worldMonsterBucketKey(regionID, bx, by), stale...).Err()
	}
	return out, nil
}

// CASWorldMonster replaces state only if the revision still matches.
func CASWorldMonster(ctx context.Context, previous, monster WorldMonster, expectedRevision int64, oldBucketX, oldBucketY, bucketX, bucketY int, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(monster)
	if err != nil {
		return false, err
	}
	key := worldMonsterKey(monster.RegionID, monster.InstanceID)
	result, err := Client.Eval(ctx, `
		local raw = redis.call('GET', KEYS[1]); if not raw then return 0 end
		local current = cjson.decode(raw); if current.revision ~= tonumber(ARGV[1]) then return -1 end
		redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[4])
		if KEYS[2] ~= KEYS[3] then redis.call('SREM', KEYS[2], ARGV[3]) end
		redis.call('SADD', KEYS[3], ARGV[3]); redis.call('PEXPIRE', KEYS[3], ARGV[4])
		return 1
	`, []string{key, worldMonsterBucketKey(monster.RegionID, oldBucketX, oldBucketY), worldMonsterBucketKey(monster.RegionID, bucketX, bucketY)}, expectedRevision, data, monster.InstanceID, ttl.Milliseconds()).Int()
	return result == 1, err
}

// RespawnWorldMonster atomically replaces a defeated instance and its stable
// spawn pointer. The old tombstone is retained until its existing TTL expires.
func RespawnWorldMonster(ctx context.Context, defeated, replacement WorldMonster, oldBucketX, oldBucketY, bucketX, bucketY int, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(replacement)
	if err != nil {
		return false, err
	}
	result, err := Client.Eval(ctx, `
		local raw = redis.call('GET', KEYS[1]); if not raw then return 0 end
		local old = cjson.decode(raw)
		if old.state ~= 'DEFEATED' or old.revision ~= tonumber(ARGV[1]) then return -1 end
		if redis.call('GET', KEYS[2]) ~= ARGV[2] then return -1 end
		redis.call('SET', KEYS[2], ARGV[3], 'PX', ARGV[5])
		redis.call('SET', KEYS[3], ARGV[4], 'PX', ARGV[5])
		redis.call('SREM', KEYS[4], ARGV[2]); redis.call('SADD', KEYS[5], ARGV[3]); redis.call('PEXPIRE', KEYS[5], ARGV[5])
		return 1
	`, []string{worldMonsterKey(defeated.RegionID, defeated.InstanceID), worldMonsterSpawnKey(defeated.SpawnID), worldMonsterKey(replacement.RegionID, replacement.InstanceID), worldMonsterBucketKey(defeated.RegionID, oldBucketX, oldBucketY), worldMonsterBucketKey(replacement.RegionID, bucketX, bucketY)}, defeated.Revision, defeated.InstanceID, replacement.InstanceID, data, ttl.Milliseconds()).Int()
	return result == 1, err
}

// HitWorldMonster atomically applies damage and is the sole transition into
// DEFEATED. The command key and tombstone outlive the respawn window, making
// retries and simultaneous finishing blows unable to grant a second reward.
func HitWorldMonster(ctx context.Context, regionID, instanceID, commandID string, expectedRevision int64, damage float64, now time.Time, respawnAfter, ttl time.Duration) (WorldMonsterHitResult, WorldMonster, error) {
	var out WorldMonster
	if damage <= 0 || math.IsNaN(damage) || math.IsInf(damage, 0) {
		return "", out, fmt.Errorf("invalid monster damage %v", damage)
	}
	if ttl <= 0 {
		return "", out, fmt.Errorf("invalid monster ttl %s", ttl)
	}
	if Client == nil {
		return "", out, errors.New("redis unavailable")
	}
	key := worldMonsterKey(regionID, instanceID)
	commandKey := "world:monster:command:" + commandID
	var outcome WorldMonsterHitResult
	apply := func(tx *redis.Tx) error {
		if exists, err := tx.Exists(ctx, commandKey).Result(); err != nil {
			return err
		} else if exists > 0 {
			outcome = WorldMonsterAlreadyDefeated
			return nil
		}
		raw, err := tx.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			outcome = WorldMonsterAlreadyDefeated
			return nil
		}
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return err
		}
		if out.State == "DEFEATED" {
			outcome = WorldMonsterAlreadyDefeated
			return nil
		}
		if out.Revision != expectedRevision {
			outcome = WorldMonsterRevisionConflict
			return nil
		}
		outcome = applyWorldMonsterHit(&out, damage, now, respawnAfter)
		updated, err := json.Marshal(out)
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, updated, ttl)
			pipe.Set(ctx, commandKey, outcome, ttl)
			return nil
		})
		return err
	}
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		out, outcome = WorldMonster{}, WorldMonsterHit
		err = Client.Watch(ctx, apply, key, commandKey)
		if !errors.Is(err, redis.TxFailedErr) {
			return outcome, out, err
		}
	}
	if errors.Is(err, redis.TxFailedErr) {
		if exists, existsErr := Client.Exists(ctx, commandKey).Result(); existsErr != nil {
			return "", WorldMonster{}, existsErr
		} else if exists > 0 {
			return WorldMonsterAlreadyDefeated, WorldMonster{}, nil
		}
		latest, exists, getErr := GetWorldMonster(ctx, regionID, instanceID)
		if getErr != nil {
			return "", WorldMonster{}, getErr
		}
		if exists {
			return WorldMonsterRevisionConflict, latest, nil
		}
		return WorldMonsterAlreadyDefeated, WorldMonster{}, nil
	}
	return outcome, out, err
}

func applyWorldMonsterHit(monster *WorldMonster, damage float64, now time.Time, respawnAfter time.Duration) WorldMonsterHitResult {
	monster.HP = max(0, monster.HP-damage)
	monster.Revision++
	monster.LastHitAt, monster.LastUpdateAt = now, now
	if monster.HP > 0 {
		return WorldMonsterHit
	}
	monster.State = "DEFEATED"
	monster.TargetID = ""
	monster.RespawnAt = now.Add(respawnAfter)
	return WorldMonsterFirstKill
}
