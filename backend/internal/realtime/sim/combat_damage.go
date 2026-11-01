package sim

import (
	"math"
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// Damage pipeline: the single server-side damage path (shields absorb first,
// then HP; incoming damage interrupts a player's windup), the hidden passive
// counter, projectile stepping and knockback/aim collision. All of these
// mutate the zone through locked *World helpers so casting, projectiles and
// legacy adapters share one damage contract.

// applyDamageLocked is the single server-side damage path. Shields absorb first,
// then HP; a player's windup is interrupted by incoming damage.
func (w *World) applyDamageLocked(source, target *Entity, amount float64, skillID string) float64 {
	if target == nil || target.Vitals.HP <= 0 || !finite(amount) || amount <= 0 {
		return 0
	}
	sourceID := ""
	if source != nil {
		sourceID = string(source.ID)
	}

	// 1. Check EVASION: target dodges the hit completely
	if w.hasStatus(target, "EVASION") {
		w.emitEvent(combat.Event{
			Type: combat.EventDamageApplied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID,
			X: target.Transform.Position.X, Y: target.Transform.Position.Y,
			Amount: 0, HP: target.Vitals.HP, Shield: target.Vitals.Shield, Reason: "evaded",
		})
		return 0
	}

	// 2. Check INVINCIBLE: target takes 0 damage (Bất Hoại)
	if w.hasStatus(target, "INVINCIBLE") {
		w.emitEvent(combat.Event{
			Type: combat.EventDamageApplied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID,
			X: target.Transform.Position.X, Y: target.Transform.Position.Y,
			Amount: 0, HP: target.Vitals.HP, Shield: target.Vitals.Shield, Reason: "immune",
		})
		return 0
	}

	// Mirror the legacy order: defense, natural Dao resistance, path resistance,
	// then the minimum-hit floor. The skill path was resolved at admission.
	path := "strength"
	if source != nil && skillID != "BASIC_ATTACK" {
		for _, skill := range source.Combat.Skills {
			if skill.ID == skillID && skill.Path != "" {
				path = skill.Path
				break
			}
		}
	}
	defense := target.Defense
	if target.Kind == EntityPlayer && target.Combat.FormActive {
		defense += target.Combat.FormDefenseBonus
	}
	remaining := amount - math.Max(0, defense)
	if target.DaoProfile != nil {
		remaining *= 1 - target.DaoProfile.GetDefensiveResistance(path)
	}
	remaining *= 1 - target.PathResistance[gamefx.CleanPath(path)]
	remaining = math.Max(1, remaining)
	totalDamage := remaining
	shieldAbsorbed := 0.0
	if target.Vitals.Shield > 0 {
		absorbed := math.Min(target.Vitals.Shield, remaining)
		target.Vitals.Shield -= absorbed
		remaining -= absorbed
		shieldAbsorbed = absorbed
	}
	hpBefore := target.Vitals.HP
	damageReason := ""
	if remaining > 0 {
		// 3. Check LAST_STAND: HP cannot fall below 1 (Bất Tử)
		if w.hasStatus(target, "LAST_STAND") {
			newHP := math.Max(1.0, target.Vitals.HP-remaining)
			if newHP == 1.0 && (target.Vitals.HP-remaining) <= 0 {
				damageReason = "last_stand"
			}
			target.Vitals.HP = newHP
		} else {
			target.Vitals.HP = math.Max(0, target.Vitals.HP-remaining)
		}
	}
	hpDamage := hpBefore - target.Vitals.HP
	target.Revision++
	if hpDamage > 0 || shieldAbsorbed > 0 {
		w.removeStatusLocked(target, "STEALTH")
	}
	if target.Kind == EntityPlayer && target.Combat.Pending != nil && !target.Combat.Pending.Skill.FormActivation {
		interrupted := target.Combat.Pending.Skill.ID
		target.Combat.Pending = nil
		interruptedBy := ""
		if source != nil {
			interruptedBy = string(source.ID)
		}
		w.emitEvent(combat.Event{Type: combat.EventCastInterrupted, SourceID: interruptedBy, TargetID: string(target.ID), SkillID: interrupted, Reason: "damaged"})
	}
	w.emitEvent(combat.Event{
		Type: combat.EventDamageApplied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID,
		X: target.Transform.Position.X, Y: target.Transform.Position.Y,
		Amount: totalDamage, ShieldAbsorbed: shieldAbsorbed, HPDamage: hpDamage, HP: target.Vitals.HP, Shield: target.Vitals.Shield, Reason: damageReason,
	})
	if source != nil && target.Kind == EntityMonster {
		w.recordThreatLocked(target, source, shieldAbsorbed+hpDamage)
	}
	if w.hasStatus(target, "THORNS") && source != nil && source.ID != target.ID && skillID != "THORNS" && hpDamage > 0 {
		thornsDmg := math.Max(1, hpDamage*0.35)
		w.applyDamageLocked(target, source, thornsDmg, "THORNS")
	}
	if target.Vitals.HP > 0 {
		w.tryHiddenPassiveLocked(target)
	}
	if target.Vitals.HP > 0 {
		return hpDamage
	}
	target.moveIntent = Vec2{}
	target.moveTarget = nil
	target.Combat.Pending = nil
	target.AI.Threat = ""
	target.AI.PatrolTarget = nil
	w.emitEvent(combat.Event{Type: combat.EventEntityDied, SourceID: sourceID, TargetID: string(target.ID), SkillID: skillID, X: target.Transform.Position.X, Y: target.Transform.Position.Y, TemplateID: target.AI.TemplateID, MonsterRank: target.AI.Rank, PersistentID: target.AI.PersistentID, TargetKind: string(target.Kind)})
	if target.Kind == EntityMonster && target.AI.Active {
		target.AI.State = AIDead
		if !target.AI.NoRespawn {
			respawnTicks := target.AI.RespawnTicks
			if respawnTicks == 0 {
				respawnTicks = 20 * 60
			}
			target.AI.RespawnTick = w.tick + uint64(respawnTicks)
		}
	}
	return hpDamage
}

func (w *World) tryHiddenPassiveLocked(target *Entity) {
	passive := &target.Combat.Hidden
	if target.Kind != EntityPlayer || passive.ID == "" || passive.RequiredHits <= 0 {
		return
	}
	if target.Vitals.MaxHP <= 0 || target.Vitals.HP/target.Vitals.MaxHP > passive.HPPercentLTE {
		passive.HitCount = 0
		return
	}
	if w.tick < passive.NextReadyTick {
		return
	}
	passive.HitCount++
	if passive.HitCount < passive.RequiredHits {
		return
	}
	passive.HitCount = 0
	passive.NextReadyTick = w.tick + passive.ICDTicks
	passive.Discovered = true
	passive.TriggeredOnce = true
	amount := 0.0
	if passive.Shield > 0 {
		target.Vitals.Shield += passive.Shield
		amount += passive.Shield
	}
	if passive.Heal > 0 {
		before := target.Vitals.HP
		target.Vitals.HP = math.Min(target.Vitals.MaxHP, target.Vitals.HP+passive.Heal)
		amount += target.Vitals.HP - before
	}
	target.Revision++
	w.emitEvent(combat.Event{
		Type: combat.EventHiddenPassive, SourceID: string(target.ID), TargetID: string(target.ID),
		SkillID: passive.ID, Amount: amount, HP: target.Vitals.HP, Shield: target.Vitals.Shield,
		Reason: passive.ID, X: target.Transform.Position.X, Y: target.Transform.Position.Y,
	})
}

func (w *World) updateProjectilesLocked() {
	remaining := w.projectiles[:0]
	for _, projectile := range w.projectiles {
		if w.tick > projectile.ExpireTick || projectile.MaxTravel <= projectile.Travelled {
			continue
		}
		from := projectile.Position
		stepDistance := math.Hypot(projectile.Velocity.X, projectile.Velocity.Y)
		remainingDistance := projectile.MaxTravel - projectile.Travelled
		reachedMaxRange := stepDistance >= remainingDistance
		if reachedMaxRange {
			stepScale := remainingDistance / stepDistance
			projectile.Velocity.X *= stepScale
			projectile.Velocity.Y *= stepScale
			stepDistance = remainingDistance
		}
		to := Vec2{X: from.X + projectile.Velocity.X, Y: from.Y + projectile.Velocity.Y}
		projectile.Travelled += stepDistance
		projectile.Position = to
		if w.blocked(to, projectile.Radius) {
			continue
		}
		source := w.entities[projectile.SourceID]
		if source == nil {
			continue
		}
		if projectile.HitEntities == nil {
			projectile.HitEntities = make(map[EntityID]bool)
		}
		ids := make([]EntityID, 0, len(w.entities))
		for id := range w.entities {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		hit := false
		for _, id := range ids {
			target := w.entities[id]
			if target.ID == projectile.SourceID || distance(projectile.RangeOrigin, target.Transform.Position) > projectile.MaxTravel ||
				!w.canHarm(source, target) || projectile.HitEntities[target.ID] {
				continue
			}
			if segmentDistance(from, to, target.Transform.Position) <= projectile.Radius+target.Radius {
				projectile.HitEntities[target.ID] = true
				w.emitEvent(combat.Event{Type: combat.EventHitConfirmed, SourceID: string(projectile.SourceID), TargetID: string(target.ID), SkillID: projectile.SkillID, ImpactVFX: projectile.Skill.Visuals.ImpactVFX, X: target.Transform.Position.X, Y: target.Transform.Position.Y})
				w.applyDamageLocked(source, target, projectile.Damage, projectile.SkillID)
				if target.Vitals.HP > 0 {
					w.applyStatusLocked(source, target, projectile.Skill)
					if projectile.Skill.Cleanse {
						w.cleanseDebuffsLocked(target)
					}
					if projectile.Skill.KnockbackDistance > 0 && target.ID != source.ID {
						w.applyKnockbackLocked(source, target, projectile.Skill.KnockbackDistance)
					}
				}
				if projectile.PierceRemaining > 0 {
					projectile.PierceRemaining--
				} else {
					hit = true
					break
				}
			}
		}
		if !hit && !reachedMaxRange {
			remaining = append(remaining, projectile)
		}
	}
	w.projectiles = remaining
}

// firstCollisionAlongAim finds the first eligible body intersected by a narrow
// swing/ray. Distance and ID ordering keep ties deterministic across zone runs.
func (w *World) firstCollisionAlongAim(source *Entity, aim Vec2, reach, hitRadius float64, eligible func(*Entity) bool) *Entity {
	length := math.Hypot(aim.X, aim.Y)
	if source == nil || length <= 0.01 || reach <= 0 {
		return nil
	}
	from := source.Transform.Position
	direction := Vec2{X: aim.X / length, Y: aim.Y / length}
	to := Vec2{X: from.X + direction.X*reach, Y: from.Y + direction.Y*reach}
	var first *Entity
	firstAlong := math.Inf(1)
	for _, target := range w.entities {
		if !eligible(target) || segmentDistance(from, to, target.Transform.Position) > hitRadius+target.Radius {
			continue
		}
		along := (target.Transform.Position.X-from.X)*direction.X + (target.Transform.Position.Y-from.Y)*direction.Y
		if along < -target.Radius || along > reach+target.Radius {
			continue
		}
		if first == nil || along < firstAlong || (along == firstAlong && target.ID < first.ID) {
			first = target
			firstAlong = along
		}
	}
	return first
}

func (w *World) applyKnockbackLocked(source, target *Entity, dist float64) {
	if source == nil || target == nil || target.Vitals.HP <= 0 || dist <= 0 {
		return
	}
	dx := target.Transform.Position.X - source.Transform.Position.X
	dy := target.Transform.Position.Y - source.Transform.Position.Y
	l := math.Hypot(dx, dy)
	if l <= 0.01 {
		dx = 1
		dy = 0
		l = 1
	}
	dest := Vec2{
		X: target.Transform.Position.X + (dx/l)*dist,
		Y: target.Transform.Position.Y + (dy/l)*dist,
	}
	if !w.blocked(dest, target.Radius) {
		target.Transform.Position = dest
		target.Revision++
	}
}
