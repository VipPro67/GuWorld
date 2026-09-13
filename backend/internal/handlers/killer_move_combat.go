package handlers

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func applyKillerMoveToCombat(combat *models.CombatState, casterID, targetID string, damage, defense float64, effects map[string]float64, compounds []CompoundEffectInfo, logs *[]string) {
	// Apply shield to caster
	for i := range combat.Allies {
		if combat.Allies[i].ID == casterID {
			combat.Allies[i].VirtualShield += defense
			break
		}
	}

	// Track compound-derived special effects
	hasIgnoreArmor := false
	hasLifesteal := false
	lifestealPct := 0.0
	hasThorns := false
	thornsDmg := 0.0
	hasMultiHit := false
	multiHitFactor := 1.0
	hasPhantomAttack := false
	phantomDmg := 0.0
	hasRegen := false
	regenVal := 0.0
	hasHemorrhage := false
	hemorrhageDmg := 0.0
	hasToxicWound := false
	toxicDmg := 0.0

	for _, c := range compounds {
		switch c.Type {
		case "hemorrhage":
			hasHemorrhage = true
			hemorrhageDmg = c.Value * 0.3
		case "regenerative_shield":
			hasRegen = true
			regenVal = c.Value * 0.15
		case "toxic_wound":
			hasToxicWound = true
			toxicDmg = c.Value * 0.3
		case "bloodthirsty_strength":
			hasLifesteal = true
			if v := effectValue(effects, "life_steal", "lifesteal"); v > 0 {
				lifestealPct = v
			} else {
				lifestealPct = 15.0
			}
		case "chain_damage":
			hasMultiHit = true
			if v, ok := effects["multi_hit"]; ok && v > 0 {
				multiHitFactor = 1.0 + v
			}
		case "counter_thorns":
			hasThorns = true
			if v, ok := effects["thorns_damage"]; ok && v > 0 {
				thornsDmg = v
			}
		case "phantom_onslaught":
			hasPhantomAttack = true
			if v, ok := effects["phantom_attack"]; ok && v > 0 {
				phantomDmg = v
			}
		case "concussive_blow":
			if v, ok := effects["stun"]; ok && v > 0 {
				combat.Enemies = applyStunToTargets(combat.Enemies, targetID, v, logs)
			}
		}
	}

	// Apply to enemies
	for i := range combat.Enemies {
		if combat.Enemies[i].HP <= 0 {
			continue
		}
		if targetID == "" || combat.Enemies[i].ID == targetID {
			finalDamage := damage

			// Armor penetration from compounds
			if hasIgnoreArmor {
				finalDamage -= combat.Enemies[i].BaseDefense * 0.2
			} else {
				finalDamage -= combat.Enemies[i].BaseDefense * 0.5
			}
			if finalDamage < 1.0 {
				finalDamage = 1.0
			}

			// Multi-hit (chain damage)
			if hasMultiHit && multiHitFactor > 1.0 {
				hitCount := int(multiHitFactor)
				if hitCount < 1 {
					hitCount = 1
				}
				totalHitDmg := 0.0
				for h := 0; h < hitCount; h++ {
					hitDmg := finalDamage * 0.6
					if h > 0 {
						hitDmg *= 0.7 // diminishing on chain hits
					}
					combat.Enemies[i].HP = math.Max(0, combat.Enemies[i].HP-hitDmg)
					totalHitDmg += hitDmg
				}
				*logs = append(*logs, fmt.Sprintf("%s trúng %d đòn liên hoàn, tổng %.1f sát thương!", combat.Enemies[i].Name, hitCount, totalHitDmg))
				finalDamage = totalHitDmg
			} else {
				combat.Enemies[i].HP = math.Max(0, combat.Enemies[i].HP-finalDamage)
				*logs = append(*logs, fmt.Sprintf("%s chịu %.1f sát thương từ sát chiêu.", combat.Enemies[i].Name, finalDamage))
			}

			// Phantom bonus attack
			if hasPhantomAttack && phantomDmg > 0 {
				combat.Enemies[i].HP = math.Max(0, combat.Enemies[i].HP-phantomDmg)
				*logs = append(*logs, fmt.Sprintf("Hư ảnh tấn công thêm %.1f sát thương!", phantomDmg))
				finalDamage += phantomDmg
			}

			// Lifesteal – heal HP first, excess becomes VirtualShield
			if hasLifesteal && lifestealPct > 0 {
				healAmount := finalDamage * (lifestealPct / 100.0)
				for a := range combat.Allies {
					if combat.Allies[a].ID == casterID {
						hpNeeded := combat.Allies[a].MaxHP - combat.Allies[a].HP
						if healAmount <= hpNeeded {
							combat.Allies[a].HP = math.Min(combat.Allies[a].MaxHP, combat.Allies[a].HP+healAmount)
							*logs = append(*logs, fmt.Sprintf("%s hút %.1f máu từ sát chiêu, hồi phục HP!", combat.Allies[a].Name, healAmount))
						} else {
							combat.Allies[a].HP = combat.Allies[a].MaxHP
							excess := healAmount - hpNeeded
							combat.Allies[a].VirtualShield += excess
							*logs = append(*logs, fmt.Sprintf("%s hút %.1f máu từ sát chiêu, HP đầy, chuyển %.1f thành khiên hộ thể!", combat.Allies[a].Name, healAmount, excess))
						}
						break
					}
				}
			}

			// Thorns damage (reflect)
			if hasThorns && thornsDmg > 0 {
				*logs = append(*logs, fmt.Sprintf("Khiên phản thương gây %.1f sát thương lên %s!", thornsDmg, combat.Enemies[i].Name))
			}

			// Apply debuffs from effects
			if effects != nil {
				if hasHemorrhage && hemorrhageDmg > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "bleed", Value: hemorrhageDmg, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị xuất huyết trọng thương!", combat.Enemies[i].Name))
				}
				if hasToxicWound && toxicDmg > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "poison", Value: toxicDmg, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị trúng độc từ vết thương!", combat.Enemies[i].Name))
				}
				if bleed, ok := effects["bleed"]; ok && bleed > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "bleed", Value: bleed, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị chảy máu!", combat.Enemies[i].Name))
				}
				if poison, ok := effects["poison"]; ok && poison > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "poison", Value: poison, Turns: 3})
					*logs = append(*logs, fmt.Sprintf("%s bị trúng độc!", combat.Enemies[i].Name))
				}
				if stun, ok := effects["stun"]; ok && stun > 0 {
					combat.Enemies[i].StatusEffects = append(combat.Enemies[i].StatusEffects, models.StatusEffect{Type: "stun", Value: stun, Turns: 1})
					*logs = append(*logs, fmt.Sprintf("%s bị choáng!", combat.Enemies[i].Name))
				}
			}

			// Regen to caster (from regenerative shield)
			if hasRegen && regenVal > 0 {
				for a := range combat.Allies {
					if combat.Allies[a].ID == casterID {
						combat.Allies[a].HP = math.Min(combat.Allies[a].MaxHP, combat.Allies[a].HP+regenVal)
						break
					}
				}
				*logs = append(*logs, fmt.Sprintf("Khiên sinh mệnh hồi phục %.1f HP.", regenVal))
			}

			return
		}
	}
}

func effectValue(effects map[string]float64, keys ...string) float64 {
	for _, key := range keys {
		if v, ok := effects[key]; ok {
			return v
		}
	}
	return 0
}
func mergeEffects(dst map[string]float64, src map[string]float64) {
	for k, v := range src {
		dst[k] += v
	}
}

// V2: Path Conflict — returns (penalty, isOpposing)
func calculatePathConflict(core models.GuDictionaryItem, support models.GuDictionaryItem) (float64, bool) {
	hasMatch := false
	hasOpposition := false
	hasDifference := false

	for _, cp := range core.Path {
		cPath := normalizePath(cp)
		for _, sp := range support.Path {
			sPath := normalizePath(sp)
			if cPath == sPath {
				hasMatch = true
			} else if isOpposingPath(cPath, sPath) {
				hasOpposition = true
			} else {
				hasDifference = true
			}
		}
	}

	if hasOpposition {
		return killerMovePathOppositionPenalty, true
	}
	if hasMatch {
		return 0, false
	}
	if hasDifference {
		return killerMovePathConflictPenaltyBase, false
	}
	return 0, false
}

func isOpposingPath(a, b string) bool {
	opposingPairs := [][2]string{
		{"fire", "water"},
		{"strength", "wisdom"},
		{"metal", "wood"},
		{"light", "dark"},
		{"lightning", "earth"},
		{"moon", "sun"},
		{"creation", "destruction"},
		{"life", "death"},
	}
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	for _, pair := range opposingPairs {
		if (a == pair[0] && b == pair[1]) || (a == pair[1] && b == pair[0]) {
			return true
		}
	}
	return false
}

func normalizePath(path string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.ToLower(path), " path"))
}
func attainmentLevel(player *models.Player, paths []string) int {
	if player.PathMasteries == nil {
		return 0
	}
	maxLvl := 0
	for _, path := range paths {
		clean := normalizePath(path)
		points := player.PathMasteries[clean]
		lvl := gamefx.PathMasteryLevel(points)
		if lvl > maxLvl {
			maxLvl = lvl
		}
	}
	return maxLvl
}
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
func successRateWarnings(successRate float64) []string {
	if successRate >= 80 {
		return []string{"Ổn định"}
	}
	if successRate >= 50 {
		return []string{"Có rủi ro: sát chiêu có thể phản phệ"}
	}
	return []string{"Nguy hiểm: cấu trúc sát chiêu rất bất ổn"}
}
func applyStunToTargets(enemies []models.Combatant, targetID string, stunValue float64, logs *[]string) []models.Combatant {
	stunTurns := 1
	if stunValue > 50 {
		stunTurns = 2
	}
	for i := range enemies {
		if enemies[i].HP <= 0 {
			continue
		}
		if targetID == "" || enemies[i].ID == targetID {
			enemies[i].StatusEffects = append(enemies[i].StatusEffects, models.StatusEffect{Type: "stun", Value: stunValue, Turns: stunTurns})
			*logs = append(*logs, fmt.Sprintf("%s bị choáng %d lượt!", enemies[i].Name, stunTurns))
			return enemies
		}
	}
	return enemies
}

func inferMoveType(damage, defense float64, effects map[string]float64) string {
	if damage >= defense && damage > 0 {
		return "ATTACK"
	}
	if defense > 0 {
		return "DEFENSE"
	}
	keys := make([]string, 0, len(effects))
	for k := range effects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return strings.ToUpper(keys[0])
	}
	return "UTILITY"
}
