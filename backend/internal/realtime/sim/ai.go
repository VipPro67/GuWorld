package sim

import (
	"math"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// AIStateKind also supplies a read-only combat presentation state to clients.
type AIStateKind string

const (
	AIIdle    AIStateKind = "idle"
	AIPatrol  AIStateKind = "patrol"
	AIAggro   AIStateKind = "aggro"
	AIChase   AIStateKind = "chase"
	AIAttack  AIStateKind = "attack"
	AILeash   AIStateKind = "leash"
	AIDead    AIStateKind = "dead"
	AIRespawn AIStateKind = "respawn"
)

const (
	DefaultAIThinkTicks     = 5
	MonsterActivationRadius = 48.0
)

// AIState is server-owned monster behavior. All timers are zone ticks, never
// wall clock. Threat is an entity ID and is revalidated every tick.
type AIState struct {
	Active bool
	// Passive monsters never acquire a target by proximity. They still use the
	// threat table, so a player who attacks first can opt into the fight.
	Passive             bool
	TemplateID          string
	Name                string
	State               AIStateKind
	Home                Vec2
	AggroRadius         float64
	LeashRadius         float64
	AttackRange         float64
	PatrolRadius        float64
	AttackDamage        float64
	AttackCooldownTicks uint32
	RespawnTicks        uint32
	SplashRadius        float64
	SplashDamage        float64
	ThreatTable         map[EntityID]float64
	NextAttackTick      uint64
	NextThinkTick       uint64
	ThinkTicks          uint32
	RespawnTick         uint64
	Threat              EntityID
	PatrolTarget        *Vec2
}

func (w *World) stepMonsterAI() {
	if w.dormantWhenEmpty && w.activePlayers == 0 {
		return
	}

	var playerPositions []Vec2
	for _, entity := range w.entities {
		if entity.Kind == EntityPlayer && entity.Vitals.HP > 0 && !w.hasStatus(entity, "STEALTH") {
			playerPositions = append(playerPositions, entity.Transform.Position)
		}
	}

	for _, entity := range w.entities {
		if entity.Kind != EntityMonster || !entity.AI.Active {
			continue
		}
		if entity.Vitals.HP <= 0 {
			entity.moveIntent = Vec2{}
			if entity.AI.RespawnTick > 0 && w.tick >= entity.AI.RespawnTick {
				w.respawnMonsterLocked(entity)
			}
			continue
		}

		if len(playerPositions) > 0 && entity.AI.Threat == "" {
			nearPlayer := false
			for _, pPos := range playerPositions {
				dx := entity.Transform.Position.X - pPos.X
				dy := entity.Transform.Position.Y - pPos.Y
				if dx*dx+dy*dy <= MonsterActivationRadius*MonsterActivationRadius {
					nearPlayer = true
					break
				}
			}
			if !nearPlayer {
				entity.moveIntent = Vec2{}
				entity.moveTarget = nil
				continue
			}
		}

		thinkTicks := entity.AI.ThinkTicks
		if thinkTicks == 0 {
			thinkTicks = DefaultAIThinkTicks
		}
		if w.tick >= entity.AI.NextThinkTick {
			w.thinkMonsterLocked(entity)
			entity.AI.NextThinkTick = w.tick + uint64(thinkTicks)
		}
		w.actMonsterLocked(entity)
	}
}

func (w *World) thinkMonsterLocked(monster *Entity) {
	distanceFromHome := math.Hypot(monster.Transform.Position.X-monster.AI.Home.X, monster.Transform.Position.Y-monster.AI.Home.Y)
	if distanceFromHome > monster.AI.LeashRadius {
		monster.AI.Threat = ""
		monster.AI.PatrolTarget = nil
		clear(monster.AI.ThreatTable)
		return
	}
	target := w.selectMonsterTargetLocked(monster)
	if target != nil {
		monster.AI.Threat = target.ID
		monster.AI.PatrolTarget = nil
		monster.AI.State = AIAggro
		return
	}
	monster.AI.Threat = ""
	if monster.AI.PatrolRadius <= 0 {
		monster.AI.State = AIIdle
		return
	}
	if monster.AI.PatrolTarget != nil {
		return
	}
	for attempt := 0; attempt < 4; attempt++ {
		angle := w.rng.Float64() * 2 * math.Pi
		radius := w.rng.Float64() * monster.AI.PatrolRadius
		target := Vec2{X: monster.AI.Home.X + math.Cos(angle)*radius, Y: monster.AI.Home.Y + math.Sin(angle)*radius}
		if !w.blocked(target, monster.Radius) {
			monster.AI.PatrolTarget = &target
			monster.AI.State = AIPatrol
			return
		}
	}
	monster.AI.State = AIIdle
}

func (w *World) actMonsterLocked(monster *Entity) {
	if w.isStunned(monster) {
		monster.moveIntent = Vec2{}
		return
	}
	distanceFromHome := math.Hypot(monster.Transform.Position.X-monster.AI.Home.X, monster.Transform.Position.Y-monster.AI.Home.Y)
	if distanceFromHome > monster.AI.LeashRadius {
		monster.AI.Threat = ""
		monster.AI.PatrolTarget = nil
		clear(monster.AI.ThreatTable)
		monster.AI.State = AILeash
		w.moveMonsterTowardLocked(monster, monster.AI.Home)
		return
	}

	if monster.AI.Threat != "" {
		target := w.entities[monster.AI.Threat]
		if target == nil || target.Kind != EntityPlayer || target.Vitals.HP <= 0 || w.hasStatus(target, "STEALTH") {
			monster.AI.Threat = ""
			monster.AI.State = AIPatrol
		} else {
			distance := math.Hypot(target.Transform.Position.X-monster.Transform.Position.X, target.Transform.Position.Y-monster.Transform.Position.Y)
			if distance <= monster.AI.AttackRange {
				monster.moveIntent = Vec2{}
				monster.AI.State = AIAttack
				cooldown := monster.AI.AttackCooldownTicks
				if cooldown == 0 {
					cooldown = 20
				}
				if w.tick >= monster.AI.NextAttackTick {
					monster.AI.NextAttackTick = w.tick + uint64(cooldown)
					w.applyDamageLocked(monster, target, monster.AI.AttackDamage, "")
					if monster.AI.SplashRadius > 0 {
						splash := monster.AI.SplashDamage
						if splash <= 0 {
							splash = monster.AI.AttackDamage * 0.5
						}
						for _, other := range w.entities {
							if other.Kind != EntityPlayer || other.ID == target.ID || other.Vitals.HP <= 0 || w.hasStatus(other, "STEALTH") {
								continue
							}
							if math.Hypot(other.Transform.Position.X-target.Transform.Position.X, other.Transform.Position.Y-target.Transform.Position.Y) <= monster.AI.SplashRadius {
								w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(monster.ID), TargetID: string(other.ID), X: other.Transform.Position.X, Y: other.Transform.Position.Y, Reason: "splash"})
								w.applyDamageLocked(monster, other, splash, "")
							}
						}
					}
				}
				return
			}
			monster.AI.State = AIChase
			w.moveMonsterTowardLocked(monster, target.Transform.Position)
			return
		}
	}

	if monster.AI.PatrolTarget != nil {
		target := *monster.AI.PatrolTarget
		if math.Hypot(target.X-monster.Transform.Position.X, target.Y-monster.Transform.Position.Y) <= 0.4 {
			monster.AI.PatrolTarget = nil
			monster.moveIntent = Vec2{}
			monster.AI.State = AIIdle
			return
		}
		monster.AI.State = AIPatrol
		w.moveMonsterTowardLocked(monster, target)
		return
	}
	monster.moveIntent = Vec2{}
	if monster.AI.State != AIIdle {
		monster.AI.State = AIIdle
	}
}

func (w *World) moveMonsterTowardLocked(monster *Entity, target Vec2) {
	dx := target.X - monster.Transform.Position.X
	dy := target.Y - monster.Transform.Position.Y
	distance := math.Hypot(dx, dy)
	if distance <= 0.05 {
		monster.moveIntent = Vec2{}
		return
	}
	monster.moveIntent = Vec2{X: dx / distance, Y: dy / distance}
}

func (w *World) selectMonsterTargetLocked(monster *Entity) *Entity {
	// Threat table wins while entries are alive and within leash range.
	var best *Entity
	bestThreat := 0.0
	for candidateID, threat := range monster.AI.ThreatTable {
		candidate := w.entities[candidateID]
		if candidate == nil || candidate.Kind != EntityPlayer || candidate.Vitals.HP <= 0 || w.hasStatus(candidate, "STEALTH") {
			delete(monster.AI.ThreatTable, candidateID)
			continue
		}
		if math.Hypot(candidate.Transform.Position.X-monster.Transform.Position.X, candidate.Transform.Position.Y-monster.Transform.Position.Y) > monster.AI.LeashRadius {
			continue
		}
		if best == nil || threat > bestThreat || (threat == bestThreat && candidate.ID < best.ID) {
			best = candidate
			bestThreat = threat
		}
	}
	if best != nil {
		return best
	}
	if monster.AI.Passive {
		return nil
	}
	var nearest *Entity
	var nearestDistance float64
	for _, candidate := range w.entities {
		if candidate.Kind != EntityPlayer || candidate.Vitals.HP <= 0 || w.hasStatus(candidate, "STEALTH") {
			continue
		}
		distance := math.Hypot(candidate.Transform.Position.X-monster.Transform.Position.X, candidate.Transform.Position.Y-monster.Transform.Position.Y)
		if distance > monster.AI.AggroRadius {
			continue
		}
		if nearest == nil || distance < nearestDistance || (distance == nearestDistance && candidate.ID < nearest.ID) {
			nearest = candidate
			nearestDistance = distance
		}
	}
	return nearest
}

func (w *World) recordThreatLocked(monster *Entity, source *Entity, amount float64) {
	if monster == nil || source == nil || monster.Kind != EntityMonster || !monster.AI.Active || source.Kind != EntityPlayer {
		return
	}
	if monster.AI.ThreatTable == nil {
		monster.AI.ThreatTable = map[EntityID]float64{}
	}
	monster.AI.ThreatTable[source.ID] += amount
}

func (w *World) respawnMonsterLocked(monster *Entity) {
	monster.Transform.Position = monster.AI.Home
	monster.Vitals.HP = monster.Vitals.MaxHP
	monster.Revision++
	monster.moveIntent = Vec2{}
	monster.AI.Threat = ""
	monster.AI.PatrolTarget = nil
	clear(monster.AI.ThreatTable)
	monster.AI.RespawnTick = 0
	monster.AI.NextThinkTick = 0
	monster.AI.NextAttackTick = 0
	monster.AI.State = AIRespawn
	monster.Combat.Statuses = nil
}

// QueueDamage applies server-side damage on the zone tick owner. It is used by
// combat/effect systems and deterministic tests; clients cannot call it.
func (w *World) QueueDamage(target EntityID, amount float64) (<-chan error, error) {
	request := lifecycleCommand{apply: func() error {
		entity := w.entities[target]
		if entity == nil {
			return ErrInvalidEntity
		}
		w.applyDamageLocked(nil, entity, amount, "queued_damage")
		return nil
	}, result: make(chan error, 1)}
	select {
	case w.lifecycleInbox <- request:
		return request.result, nil
	default:
		return nil, ErrLifecycleQueueFull
	}
}
