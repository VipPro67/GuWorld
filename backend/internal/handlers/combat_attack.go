package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type CombatAttackRequest struct {
	PlayerID      string   `json:"player_id" binding:"required"`
	TargetID      string   `json:"target_id"`
	EquippedGuIDs []string `json:"equipped_gu_ids"`
	IsCharge      bool     `json:"is_charge"`
	TargetX       int      `json:"target_x"`
	TargetY       int      `json:"target_y"`
	ItemID        string   `json:"item_id"`
	UseStones     string   `json:"use_stones"`
}

// enemyPathResistance returns the defender's maximum path-resistance fraction
// for the given paths (PVP enemies may carry character effects).
func enemyPathResistance(player *models.Player, paths []string) float64 {
	if player == nil || player.ActiveCombat == nil || len(player.ActiveCombat.EnemyPathResistance) == 0 {
		return 0
	}
	if len(paths) == 0 {
		paths = []string{"strength"}
	}
	maxR := 0.0
	for _, p := range paths {
		clean := strings.TrimSuffix(strings.ToLower(p), " path")
		if r := player.ActiveCombat.EnemyPathResistance[clean]; r > maxR {
			maxR = r
		}
	}
	return maxR
}

// enemyResistsControl returns true if the PVP defender's mental resistance
// causes a control effect to be resisted.
func enemyResistsControl(player *models.Player) bool {
	if player == nil || player.ActiveCombat == nil || player.ActiveCombat.EnemyMentalResistance <= 0 {
		return false
	}
	return rand.Float64() < player.ActiveCombat.EnemyMentalResistance
}

// applyPathMasteryGain adds Chân Ý points to a path while respecting the
// per-battle cap (gamefx.PathMasteryBattleGainLimit(rank) * battle). The amount
// earned in the current battle is tracked on the active combat state; once the
// cap is reached further gains are suppressed. Returns the points actually
// granted (may be less than gain, or 0 when capped).
func applyPathMasteryGain(player *models.Player, path string, gain int) int {
	if player.PathMasteries == nil {
		player.PathMasteries = make(models.PathMasteriesMap)
	}
	if gain <= 0 {
		return 0
	}
	if player.ActiveCombat == nil {
		player.PathMasteries[path] += gain
		return gain
	}
	capLimit := gamefx.PathMasteryBattleGainLimit(player.Rank)
	remaining := capLimit - player.ActiveCombat.PathMasteryGained
	if remaining <= 0 {
		return 0
	}
	if gain > remaining {
		gain = remaining
	}
	player.ActiveCombat.PathMasteryGained += gain
	player.PathMasteries[path] += gain
	return gain
}

func CombatAttack(c *gin.Context) {
	var req CombatAttackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}

	if player.ActiveCombat == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không trong trạng thái chiến đấu."})
		return
	}

	if player.HP <= 0 && player.ActiveCombat.EnemyType != "PVP" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã kiệt sức. Dưỡng thương trước khi chiến đấu."})
		return
	}

	if len(player.ActiveCombat.Allies) == 0 {
		player.ActiveCombat = InitializeCombatState(tx, player.ActiveCombat, &player)
	}

	if player.ActiveCombat.SkillCooldowns != nil {
		for k, v := range player.ActiveCombat.SkillCooldowns {
			if v > 0 {
				player.ActiveCombat.SkillCooldowns[k] = v - 1
			}
		}
	}

	var turnLogs []string

	// ── Tick status effects for all allies ──
	for i := range player.ActiveCombat.Allies {
		c := &player.ActiveCombat.Allies[i]
		if c.HP <= 0 {
			continue
		}
		var activeEffs []models.StatusEffect
		for j := 0; j < len(c.StatusEffects); j++ {
			eff := c.StatusEffects[j]
			if eff.Turns > 0 {
				switch eff.Type {
				case "poison", "bleed", "slow_damage":
					c.HP -= eff.Value
					if c.HP < 0 {
						c.HP = 0
					}
					turnLogs = append(turnLogs, fmt.Sprintf("%s mất %.1f HP do %s (Còn %d lượt).", c.Name, eff.Value, eff.Type, eff.Turns))
				case "regen":
					healVal := eff.Value
					c.HP += healVal
					if c.HP > c.MaxHP {
						c.HP = c.MaxHP
					}
					turnLogs = append(turnLogs, fmt.Sprintf("%s hồi %.1f HP do Hồi Sinh (Còn %d lượt).", c.Name, healVal, eff.Turns))
				}
				eff.Turns--
			}
			if eff.Turns > 0 {
				activeEffs = append(activeEffs, eff)
			}
		}
		c.StatusEffects = activeEffs
	}

	// ── Tick status effects for all enemies ──
	for i := range player.ActiveCombat.Enemies {
		c := &player.ActiveCombat.Enemies[i]
		if c.HP <= 0 {
			continue
		}
		var activeEffs []models.StatusEffect
		for j := 0; j < len(c.StatusEffects); j++ {
			eff := c.StatusEffects[j]
			if eff.Turns > 0 {
				switch eff.Type {
				case "poison", "bleed", "slow_damage":
					c.HP -= eff.Value
					if c.HP < 0 {
						c.HP = 0
					}
					turnLogs = append(turnLogs, fmt.Sprintf("%s mất %.1f HP do %s (Còn %d lượt).", c.Name, eff.Value, eff.Type, eff.Turns))
				case "regen":
					healVal := eff.Value
					c.HP += healVal
					if c.HP > c.MaxHP {
						c.HP = c.MaxHP
					}
					turnLogs = append(turnLogs, fmt.Sprintf("%s hồi %.1f HP do Hồi Sinh (Còn %d lượt).", c.Name, healVal, eff.Turns))
				}
				eff.Turns--
			}
			if eff.Turns > 0 {
				activeEffs = append(activeEffs, eff)
			}
		}
		c.StatusEffects = activeEffs
	}

	var casterIdx = -1
	for idx, ally := range player.ActiveCombat.Allies {
		if ally.ID == player.ID.String() {
			casterIdx = idx
			if player.ActiveCombat.EnemyType != "PVP" {
				player.ActiveCombat.Allies[idx].HP = player.HP
			}
			player.ActiveCombat.Allies[idx].CurrentMana = player.CurrentMana
			break
		}
	}
	if casterIdx == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find player in combatants"})
		return
	}

	// Virtual mode: auto-restore mana to max each turn
	isVirtual := player.ActiveCombat.CombatMode == "virtual"
	if isVirtual {
		player.CurrentMana = player.MaxMana
		for i := range player.ActiveCombat.Allies {
			player.ActiveCombat.Allies[i].CurrentMana = player.ActiveCombat.Allies[i].MaxMana
		}
		for i := range player.ActiveCombat.Enemies {
			player.ActiveCombat.Enemies[i].CurrentMana = player.ActiveCombat.Enemies[i].MaxMana
		}
	}

	playerStats := CalculatePlayerStats(&player)
	player.ActiveCombat.Allies[casterIdx].BaseAttack = playerStats["attack"]
	player.ActiveCombat.Allies[casterIdx].BaseDefense = playerStats["defense"]

	playerCombatant := player.ActiveCombat.Allies[casterIdx]

	var selectedGus []models.GuInstance
	var selectedSpecs []models.GuDictionaryItem
	totalManaCost := 0.0
	looseGuFailureChance := 0.0

	isChargingThisTurn := false
	isUnleashingCharge := false
	isUsingItem := false
	isUsingStones := false

	if req.UseStones != "" {
		isUsingStones = true
		player.ActiveCombat.ChargePoints = 0
	} else if req.ItemID != "" {
		isUsingItem = true
		player.ActiveCombat.ChargePoints = 0

		if !isVirtual {
			if player.ActiveCombat.ItemsUsedCount >= 3 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ được dùng tối đa 3 vật phẩm trong mỗi trận chiến!"})
				return
			}
		}
		if player.ActiveCombat.ItemCooldowns == nil {
			player.ActiveCombat.ItemCooldowns = make(map[string]int)
		}
		if !isVirtual && player.ActiveCombat.ItemCooldowns[req.ItemID] > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm đang hồi chiêu (còn %d lượt)", player.ActiveCombat.ItemCooldowns[req.ItemID])})
			return
		}

		if !isVirtual {
			qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), req.ItemID)
			if err != nil || qty <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm này trong túi đồ"})
				return
			}
		}
		item, ok := models.ItemRegistry[req.ItemID]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại"})
			return
		}
		if item.Type != "Consumable" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này không thể dùng trong chiến đấu"})
			return
		}

		if !isVirtual {
			if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, "materials", 1); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hao vật phẩm"})
				return
			}
			player.ActiveCombat.ItemsUsedCount++
			player.ActiveCombat.ItemCooldowns[req.ItemID] = 3
		}

	} else if req.IsCharge {
		player.ActiveCombat.ChargePoints++
		if player.ActiveCombat.ChargePoints >= 2 {
			isUnleashingCharge = true
			player.ActiveCombat.ChargePoints = 0
		} else {
			isChargingThisTurn = true
		}
	} else {
		player.ActiveCombat.ChargePoints = 0

		for _, instID := range req.EquippedGuIDs {
			found := false
			for _, inst := range player.GuInventory {
				if inst.InstanceID == instID {
					if inst.Status == "DEAD" {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể chiến đấu bằng Cổ trùng ĐÃ CHẾT: " + inst.GuID})
						return
					}
					spec, ok := models.GetGu(inst.GuID, inst.Rank)
					if !ok {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy định nghĩa của Cổ trùng: " + inst.GuID})
						return
					}

					if player.CurrentRegion == "three_kings_inheritance" {
						hasAllowedPath := false
						for _, p := range spec.Path {
							if p == "Star Path" || p == "Earth Path" {
								hasAllowedPath = true
								break
							}
						}
						if !hasAllowedPath {
							c.JSON(http.StatusBadRequest, gin.H{"error": "Bí cảnh Tam Vương Truyền Thừa áp chế Cổ trùng không thuộc Nguyệt đạo hoặc Thổ đạo: " + spec.Name})
							return
						}
					}

					selectedGus = append(selectedGus, inst)
					selectedSpecs = append(selectedSpecs, spec)

					_, _, cost, errVal := CalculateGuEffectForCombatant(&playerCombatant, &spec, true, gamefx.Get(&player).DaoNganBonus, gamefx.Get(&player).PathDamageBonus)
					if errVal == nil {
						totalManaCost += cost
					}
					found = true
					break
				}
			}
			if !found {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ: " + instID})
				return
			}
		}

		if playerCombatant.CurrentMana < totalManaCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
			return
		}
		looseGuFailureChance = calculateLooseGuFailureChance(selectedSpecs)
	}

	targetID := req.TargetID
	targetIdx := -1
	for idx, enemy := range player.ActiveCombat.Enemies {
		if enemy.HP > 0 {
			if targetID == "" || enemy.ID == targetID {
				targetIdx = idx
				break
			}
		}
	}
	if targetIdx == -1 {
		for idx, enemy := range player.ActiveCombat.Enemies {
			if enemy.HP > 0 {
				targetIdx = idx
				break
			}
		}
	}

	if len(selectedSpecs) > 1 {
		turnLogs = append(turnLogs, fmt.Sprintf("=== DÙNG CỔ RIÊNG LẺ (rủi ro hụt %.0f%%) ===", looseGuFailureChance*100))
	} else {
		turnLogs = append(turnLogs, "=== HÀNH ĐỘNG CHIẾN ĐẤU ===")
	}

	type CombatAction struct {
		InitiatorID string
		Speed       float64
		Execute     func() []string
	}
	var actions []CombatAction

	casterID := player.ID.String()
	primaryEnemyID := ""
	hasDuel := false
	var duelSeq []func() []string

	if targetIdx != -1 {
		primaryEnemyID = player.ActiveCombat.Enemies[targetIdx].ID
		casterCombatant := &player.ActiveCombat.Allies[casterIdx]
		prefX := req.TargetX
		prefY := req.TargetY
		if prefX <= 0 {
			prefX = 1
		}
		if prefY <= 0 {
			prefY = 2
		}

		// ── Interleaved exchange ("đánh trả từng phát"): the caster's strikes
		// resolve one by one and the primary target answers each strike with a
		// riposte. Effective speed decides who opens the exchange. ──
		numStrikes := 0
		switch {
		case isUsingStones, isUsingItem, isChargingThisTurn:
			numStrikes = 0
		default:
			numStrikes = 1
			if !isUnleashingCharge && len(selectedSpecs) > 1 {
				numStrikes = len(selectedSpecs)
			}
		}
		strikeFired := make([]bool, numStrikes)

		// guCastLogs resolves a single equipped Gu against the battlefield,
		// appending its logs (per-Gu behavior identical to the previous
		// all-at-once resolution).
		guCastLogs := func(logs *[]string, spec models.GuDictionaryItem) {
			refCaster, _, _ := findCombatant(player.ActiveCombat, player.ID.String())
			if refCaster == nil || refCaster.HP <= 0 {
				return
			}
			{
				dmgBase, defBase, _, _ := CalculateGuEffectForCombatant(refCaster, &spec, false, gamefx.Get(&player).DaoNganBonus, gamefx.Get(&player).PathDamageBonus)

				if !isVirtual {
					if len(spec.Path) > 0 {
						cleanPath := strings.TrimSuffix(strings.ToLower(spec.Path[0]), " path")
						pathGain := 1
						if spec.Rank >= 6 {
							pathGain = 10
						}
						gained := applyPathMasteryGain(&player, cleanPath, pathGain)
						if gained > 0 {
							newLevel := gamefx.PathMasteryLevel(player.PathMasteries[cleanPath])
							*logs = append(*logs, fmt.Sprintf("-> Ngộ Đạo tăng tiến (%s)! +%d điểm Chân Ý %s (Cảnh giới: %s).", spec.Name, gained, strings.Title(cleanPath), gamefx.PathMasteryLevelName(newLevel)))
						}
					}
				}
				dmg, def, cost, errGu := CalculateGuEffectForCombatant(refCaster, &spec, true, gamefx.Get(&player).DaoNganBonus, gamefx.Get(&player).PathDamageBonus)
				if errGu == nil && refCaster.CurrentMana >= cost {
					refCaster.CurrentMana -= cost
					if refCaster.Rank < spec.Rank {
						backlashHP := 5.0
						if refCaster.HP <= backlashHP {
							refCaster.HP = 0
						} else {
							refCaster.HP -= backlashHP
						}
						*logs = append(*logs, fmt.Sprintf("[Phản Phệ] Cổ trùng Rank cao phản phệ %s, trừ %.0f HP (còn %.0f HP)!", refCaster.Name, backlashHP, refCaster.HP))
					}

					// Apply weak debuff to damage
					weakMult := 1.0
					for _, eff := range refCaster.StatusEffects {
						if eff.Type == "weak" && eff.Turns > 0 {
							weakMult *= (1.0 - eff.Value/100.0)
						}
					}
					dmg *= weakMult
					dmgBase *= weakMult

					isHealing := false
					isDefense := spec.Type == "Defense"
					isAttack := spec.Type == "Attack" || spec.Type == "Active"
					if spec.Effects.Active != nil {
						isHealing = spec.Effects.Active.HealFlat > 0
						if spec.Effects.Active.ShieldFlat > 0 {
							isDefense = true
						}
						if spec.Effects.Active.DamageFlat > 0 || spec.Effects.Active.DamageMult > 0 {
							isAttack = true
						}
					}

					if isHealing {
						healVal := def
						if false {
							healVal = refCaster.BaseAttack * 1.5
						}
						if healVal <= 0 {
							healVal = 30.0
						}
						healVal *= (1.0 + gamefx.Get(&player).HealingReceivedBonus)
						refCaster.HP += healVal
						if refCaster.HP > refCaster.MaxHP {
							refCaster.HP = refCaster.MaxHP
						}
						healBase := defBase
						if healBase <= 0 {
							healBase = 30.0
						}
						healDao := healVal - healBase
						if healDao > 0 {
							*logs = append(*logs, fmt.Sprintf("-> %s kích hoạt %s, khôi phục %.1f HP bản thân (HP hiện tại: %.1f) [Gốc: %.1f, Đạo Ngân: +%.1f].", refCaster.Name, spec.Name, healVal, refCaster.HP, healBase, healDao))
						} else {
							*logs = append(*logs, fmt.Sprintf("-> %s kích hoạt %s, khôi phục %.1f HP bản thân (HP hiện tại: %.1f).", refCaster.Name, spec.Name, healVal, refCaster.HP))
						}
					} else if isDefense {
						shieldVal := def * 10.0
						if shieldVal <= 0 {
							shieldVal = 150.0
						}
						refCaster.VirtualShield += shieldVal
						shieldBase := defBase * 10.0
						if shieldBase <= 0 {
							shieldBase = 150.0
						}
						shieldDao := shieldVal - shieldBase
						if shieldDao > 0 {
							*logs = append(*logs, fmt.Sprintf("-> %s thi triển %s phòng thủ, nhận %.1f Khiên hộ thể (Virtual Shield: %.1f) [Gốc: %.1f, Đạo Ngân: +%.1f].", refCaster.Name, spec.Name, shieldVal, refCaster.VirtualShield, shieldBase, shieldDao))
						} else {
							*logs = append(*logs, fmt.Sprintf("-> %s thi triển %s phòng thủ, nhận %.1f Khiên hộ thể (Virtual Shield: %.1f).", refCaster.Name, spec.Name, shieldVal, refCaster.VirtualShield))
						}
					} else if isAttack {
						hasBypass := spec.Effects.Active != nil && spec.Effects.Active.CurveAttack
						targetType := spec.Effects.Active.TargetSelector
						targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, targetType, prefX, prefY, hasBypass)
						if len(targetsIdx) > 0 {
							dmgDao := dmg - dmgBase
							if dmgDao > 0 {
								*logs = append(*logs, fmt.Sprintf("-> %s kích hoạt %s! (Sức mạnh: %.1f [Gốc: %.1f, Đạo Ngân: +%.1f])", refCaster.Name, spec.Name, dmg, dmgBase, dmgDao))
							} else {
								*logs = append(*logs, fmt.Sprintf("-> %s kích hoạt %s!", refCaster.Name, spec.Name))
							}
							for _, eIdx := range targetsIdx {
								tEnemy := &player.ActiveCombat.Enemies[eIdx]
								netDmg := dmg - tEnemy.BaseDefense
								netDmg *= (1.0 - enemyPathResistance(&player, spec.Path))
								if netDmg < 1.0 {
									netDmg = 1.0
								}
								dmgLog := applyDamageWithLogs(refCaster.Name, tEnemy.Name, tEnemy, netDmg)
								*logs = append(*logs, dmgLog)

								// Apply status effects from Gu to target
								if spec.Effects.Active != nil {
									active := spec.Effects.Active
									resisted := enemyResistsControl(&player)
									switch active.ApplyStatus {
									case "SILENCE":
										if !resisted {
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "silence", Value: 1, Turns: active.StatusDurationTicks,
											})
											*logs = append(*logs, fmt.Sprintf("%s bị Im lặng %d lượt!", tEnemy.Name, active.StatusDurationTicks))
										}
									case "STUN":
										if !resisted {
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "stun", Value: 1, Turns: active.StatusDurationTicks,
											})
											*logs = append(*logs, fmt.Sprintf("%s bị Choáng %d lượt!", tEnemy.Name, active.StatusDurationTicks))
										}
									case "FREEZE":
										if !resisted {
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "freeze", Value: 1, Turns: active.StatusDurationTicks,
											})
											*logs = append(*logs, fmt.Sprintf("%s bị Đóng băng %d lượt!", tEnemy.Name, active.StatusDurationTicks))
										}
									case "SLOW":
										if !resisted {
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "slow", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
											})
											*logs = append(*logs, fmt.Sprintf("%s bị Chậm %d%% %d lượt!", tEnemy.Name, int(active.StatusTickDamage), active.StatusDurationTicks))
										}
									case "SLOW1":
										tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
											Type: "slow1", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
										})
										*logs = append(*logs, fmt.Sprintf("%s bị Chậm %d%% %d lượt (không thể kháng)!", tEnemy.Name, int(active.StatusTickDamage), active.StatusDurationTicks))
									case "SLOW_DAMAGE":
										if !resisted {
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "slow_damage", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
											})
											*logs = append(*logs, fmt.Sprintf("%s bị Chậm sát thương %d%% và mất %d HP/lượt trong %d lượt!", tEnemy.Name, int(active.StatusTickDamage), int(active.StatusTickDamage), active.StatusDurationTicks))
										}
									case "WEAK":
										if !resisted {
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "weak", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
											})
											*logs = append(*logs, fmt.Sprintf("%s bị Yếu %d%% %d lượt!", tEnemy.Name, int(active.StatusTickDamage), active.StatusDurationTicks))
										}
									case "BLEED":
										tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
											Type: "bleed", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
										})
										*logs = append(*logs, fmt.Sprintf("%s bị Chảy máu (%.0f/lượt %d lượt)!", tEnemy.Name, active.StatusTickDamage, active.StatusDurationTicks))
									case "POISON":
										tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
											Type: "poison", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
										})
										*logs = append(*logs, fmt.Sprintf("%s bị Độc (%.0f/lượt %d lượt)!", tEnemy.Name, active.StatusTickDamage, active.StatusDurationTicks))
									}
								}
							}
						}
					} else {
						*logs = append(*logs, fmt.Sprintf("-> %s kích hoạt %s (Hỗ trợ)!", refCaster.Name, spec.Name))
					}
				}
			}
		}

		// basicAttackLogs performs a plain basic attack against the preferred
		// target (used for empty selections and when silenced).
		basicAttackLogs := func(logs *[]string, grantsMastery bool) {
			refCaster, _, _ := findCombatant(player.ActiveCombat, player.ID.String())
			if refCaster == nil || refCaster.HP <= 0 {
				return
			}
			targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", prefX, prefY, false)
			if len(targetsIdx) == 0 {
				return
			}
			targetEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
			daoMult := gamefx.DaoMultiplierForPaths(refCaster.DaoMarks, []string{"strength"}, gamefx.Get(&player).DaoNganBonus)
			dmgVal := refCaster.BaseAttack
			if dmgVal <= 0 {
				dmgVal = 5.0
			}
			// Apply weak debuff
			for _, eff := range refCaster.StatusEffects {
				if eff.Type == "weak" && eff.Turns > 0 {
					dmgVal *= (1.0 - eff.Value/100.0)
				}
			}
			fullDmg := dmgVal * daoMult
			netDmg := (fullDmg - targetEnemy.BaseDefense) * (1.0 - enemyPathResistance(&player, []string{"strength"}))
			if netDmg < 1.0 {
				netDmg = 1.0
			}
			dmgLog := applyDamageWithLogs(refCaster.Name, targetEnemy.Name, targetEnemy, netDmg)

			strengthNote := ""
			if grantsMastery {
				strengthGain := applyPathMasteryGain(&player, "strength", 1)
				if strengthGain > 0 {
					pts := player.PathMasteries["strength"]
					lvlName := "Sơ Học"
					if pts >= 10000 {
						lvlName = "Vô Thượng Tông Sư"
					} else if pts >= 2500 {
						lvlName = "Đại Tông Sư"
					} else if pts >= 500 {
						lvlName = "Tông Sư"
					} else if pts >= 100 {
						lvlName = "Đại Sư"
					}
					strengthNote = fmt.Sprintf(" Ngộ Đạo: +1 Chân Ý Lực (%s)", lvlName)
				}
			}

			if daoMult > 1.0 {
				dmgDao := fullDmg - dmgVal
				*logs = append(*logs, fmt.Sprintf("-> %s đánh thường (Basic Attack) [Gốc: %.1f, Đạo Ngân: +%.1f]!%s", refCaster.Name, dmgVal, dmgDao, strengthNote), dmgLog)
			} else {
				*logs = append(*logs, fmt.Sprintf("-> %s đánh thường (Basic Attack)!%s", refCaster.Name, strengthNote), dmgLog)
			}
		}

		// casterStrike builds the k-th caster exchange step. Step 0 carries the
		// turn guards (stun/silence/loose-Gu failure); later steps stay silent
		// when those conditions block them.
		casterStrike := func(k int) func() []string {
			return func() []string {
				refCaster, _, _ := findCombatant(player.ActiveCombat, casterID)
				if refCaster == nil || refCaster.HP <= 0 {
					return nil
				}
				stunned := false
				silenced := false
				for _, eff := range refCaster.StatusEffects {
					if eff.Turns > 0 {
						if eff.Type == "stun" || eff.Type == "freeze" {
							stunned = true
						}
						if eff.Type == "silence" {
							silenced = true
						}
					}
				}
				var logs []string
				if k == 0 {
					if stunned {
						return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refCaster.Name)}
					}
					if len(selectedSpecs) > 0 && silenced {
						strikeFired[k] = true
						basicAttackLogs(&logs, false)
						return logs
					}
					if len(selectedSpecs) > 0 && looseGuFailureChance > 0 && rand.Float64() < looseGuFailureChance {
						wastedMana := totalManaCost * 0.5
						if wastedMana > refCaster.CurrentMana {
							wastedMana = refCaster.CurrentMana
						}
						refCaster.CurrentMana -= wastedMana
						return []string{fmt.Sprintf("-> %s cưỡng ép vận nhiều Cổ riêng lẻ, đạo ngân hỗn loạn khiến chiêu thất bại! Hao phí %.1f nguyên khí.", refCaster.Name, wastedMana)}
					}
					strikeFired[k] = true
					if len(selectedSpecs) > 0 {
						guCastLogs(&logs, selectedSpecs[k])
					} else {
						basicAttackLogs(&logs, true)
					}
					return logs
				}
				if stunned || silenced {
					return nil
				}
				strikeFired[k] = true
				guCastLogs(&logs, selectedSpecs[k])
				return logs
			}
		}

		// counterStrike builds the riposte answering the k-th caster strike.
		counterStrike := func(k int, enemyID string) func() []string {
			return func() []string {
				if !strikeFired[k] {
					return nil
				}
				refEnemy, _, _ := findCombatant(player.ActiveCombat, enemyID)
				if refEnemy == nil || refEnemy.HP <= 0 {
					return nil
				}
				refCaster, _, _ := findCombatant(player.ActiveCombat, casterID)
				if refCaster == nil || refCaster.HP <= 0 {
					return nil
				}
				for _, eff := range refEnemy.StatusEffects {
					if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
						return []string{fmt.Sprintf("-> %s bị choáng/đóng băng, không thể đáp trả!", refEnemy.Name)}
					}
				}
				dmgVal := refEnemy.BaseAttack
				if dmgVal <= 0 {
					dmgVal = 5.0
				}
				netDmg := dmgVal - refCaster.BaseDefense
				if netDmg < 1.0 {
					netDmg = 1.0
				}
				dmgLog := applyDamageWithLogs(refEnemy.Name, refCaster.Name, refCaster, netDmg)
				return []string{fmt.Sprintf("-> %s không chậm trễ, đáp trả %s một đòn!", refEnemy.Name, refCaster.Name), dmgLog}
			}
		}

		switch {
		case isUsingStones:
			actions = append(actions, CombatAction{
				InitiatorID: casterID,
				Speed:       casterCombatant.Speed,
				Execute: func() []string {
					var logs []string
					refCaster, _, _ := findCombatant(player.ActiveCombat, casterID)
					if refCaster == nil || refCaster.HP <= 0 {
						return nil
					}
					for _, eff := range refCaster.StatusEffects {
						if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refCaster.Name)}
						}
					}
					if isVirtual {
						refCaster.CurrentMana = refCaster.MaxMana
						logs = append(logs, fmt.Sprintf("-> [Ảnh Chiếu] %s vận chuyển Chân Nguyên đầy đủ từ thân xác thật!", refCaster.Name))
					} else if req.UseStones == "immortal" {
						if player.ImmortalStones >= 1 {
							player.ImmortalStones -= 1
							refCaster.CurrentMana += 100.0
							if refCaster.CurrentMana > refCaster.MaxMana {
								refCaster.CurrentMana = refCaster.MaxMana
							}
							logs = append(logs, fmt.Sprintf("-> %s nghiền nát 1 Tiên Nguyên Thạch, khôi phục mạnh mẽ tiên nguyên!", refCaster.Name))
						} else {
							logs = append(logs, fmt.Sprintf("-> %s định dùng Tiên Nguyên Thạch nhưng không đủ!", refCaster.Name))
						}
					} else {
						if player.PrimevalStones >= 100 {
							player.PrimevalStones -= 100
							refCaster.CurrentMana += 50.0
							if refCaster.CurrentMana > refCaster.MaxMana {
								refCaster.CurrentMana = refCaster.MaxMana
							}
							logs = append(logs, fmt.Sprintf("-> %s hấp thu 100 Nguyên Thạch, khôi phục chân nguyên!", refCaster.Name))
						} else {
							logs = append(logs, fmt.Sprintf("-> %s định dùng Nguyên Thạch nhưng không đủ!", refCaster.Name))
						}
					}
					return logs
				},
			})
		case isUsingItem:
			actions = append(actions, CombatAction{
				InitiatorID: casterID,
				Speed:       casterCombatant.Speed,
				Execute: func() []string {
					var logs []string
					refCaster, _, _ := findCombatant(player.ActiveCombat, casterID)
					if refCaster == nil || refCaster.HP <= 0 {
						return nil
					}
					for _, eff := range refCaster.StatusEffects {
						if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refCaster.Name)}
						}
					}
					itemSpec := models.ItemRegistry[req.ItemID]
					healVal := itemSpec.Effects["restore_hp"]
					if healVal == 0 {
						healVal = 50.0
					}

					if refCaster.Rank >= 6 {
						refCaster.ApertureDmg -= healVal
						if refCaster.ApertureDmg < 0 {
							refCaster.ApertureDmg = 0
						}
						logs = append(logs, fmt.Sprintf("-> %s sử dụng %s, giảm %.1f hư tổn tiên khiếu!", refCaster.Name, itemSpec.Name, healVal))
					} else {
						refCaster.HP += healVal
						if refCaster.HP > refCaster.MaxHP {
							refCaster.HP = refCaster.MaxHP
						}
						logs = append(logs, fmt.Sprintf("-> %s sử dụng %s, khôi phục %.1f HP (HP hiện tại: %.1f).", refCaster.Name, itemSpec.Name, healVal, refCaster.HP))
					}
					return logs
				},
			})
		case isChargingThisTurn:
			actions = append(actions, CombatAction{
				InitiatorID: casterID,
				Speed:       casterCombatant.Speed,
				Execute: func() []string {
					refCaster, _, _ := findCombatant(player.ActiveCombat, casterID)
					if refCaster == nil || refCaster.HP <= 0 {
						return nil
					}
					for _, eff := range refCaster.StatusEffects {
						if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refCaster.Name)}
						}
					}
					return []string{fmt.Sprintf("%s tích tụ Nguyên Khí, vận chuyển Chân Nguyên chuẩn bị cho Đại Sát Chiêu!", refCaster.Name)}
				},
			})
		default:
			// Duel paths: unleash charge, Gu casts, or basic attack — each
			// strike is answered by a riposte from the primary target.
			hasDuel = true
			casterSteps := make([]func() []string, 0, numStrikes)
			if isUnleashingCharge {
				casterSteps = append(casterSteps, func() []string {
					refCaster, _, _ := findCombatant(player.ActiveCombat, casterID)
					if refCaster == nil || refCaster.HP <= 0 {
						return nil
					}
					for _, eff := range refCaster.StatusEffects {
						if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refCaster.Name)}
						}
					}
					var logs []string
					strikeFired[0] = true
					targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", prefX, prefY, false)
					if len(targetsIdx) > 0 {
						targetEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
						dmgValue := playerStats["attack"] * 2.5
						// Apply weak debuff
						for _, eff := range refCaster.StatusEffects {
							if eff.Type == "weak" && eff.Turns > 0 {
								dmgValue *= (1.0 - eff.Value/100.0)
							}
						}
						netDmg := dmgValue - targetEnemy.BaseDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						dmgLog := applyDamageWithLogs(refCaster.Name, targetEnemy.Name, targetEnemy, netDmg)
						logs = append(logs, fmt.Sprintf("-> %s kích hoạt Bộc Phá Sát Chiêu (Unleash Charge)!", refCaster.Name), dmgLog)
					}
					return logs
				})
			} else {
				for k := 0; k < numStrikes; k++ {
					casterSteps = append(casterSteps, casterStrike(k))
				}
			}

			// Assemble the interleaved sequence: effective speed decides who
			// opens, then both sides alternate strictly.
			openedByCaster := effectiveCombatSpeed(player.ActiveCombat, casterID) >= effectiveCombatSpeed(player.ActiveCombat, primaryEnemyID)
			duelSeq = make([]func() []string, 0, numStrikes*2)
			for k := 0; k < numStrikes; k++ {
				counter := counterStrike(k, primaryEnemyID)
				if openedByCaster {
					duelSeq = append(duelSeq, casterSteps[k], counter)
				} else {
					duelSeq = append(duelSeq, counter, casterSteps[k])
				}
			}

			// Caster slot participates in the global speed ordering; its
			// content already lives inside the duel sequence.
			actions = append(actions, CombatAction{
				InitiatorID: casterID,
				Speed:       casterCombatant.Speed,
				Execute:     func() []string { return nil },
			})
		}
	}

	for aIdx := range player.ActiveCombat.Allies {
		ally := &player.ActiveCombat.Allies[aIdx]
		if ally.ID != player.ID.String() && ally.HP > 0 {
			allyID := ally.ID
			actions = append(actions, CombatAction{
				InitiatorID: allyID,
				Speed:       ally.Speed,
				Execute: func() []string {
					var logs []string
					refAlly, _, _ := findCombatant(player.ActiveCombat, allyID)
					if refAlly == nil || refAlly.HP <= 0 {
						return nil
					}
					for _, eff := range refAlly.StatusEffects {
						if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refAlly.Name)}
						}
					}

					targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", 2, 3, false)
					if len(targetsIdx) > 0 {
						tEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
						dmgVal := refAlly.BaseAttack
						if dmgVal <= 0 {
							dmgVal = 8.0
						}
						netDmg := dmgVal - tEnemy.BaseDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						dmgLog := applyDamageWithLogs(refAlly.Name, tEnemy.Name, tEnemy, netDmg)
						logs = append(logs, fmt.Sprintf("-> Đồng minh %s đánh thường lên %s.", refAlly.Name, tEnemy.Name), dmgLog)
					}
					return logs
				},
			})
		}
	}

	for eIdx := range player.ActiveCombat.Enemies {
		enemy := &player.ActiveCombat.Enemies[eIdx]
		if enemy.HP > 0 {
			enemyID := enemy.ID
			actions = append(actions, CombatAction{
				InitiatorID: enemyID,
				Speed:       enemy.Speed,
				Execute: func() []string {
					var logs []string
					refEnemy, _, _ := findCombatant(player.ActiveCombat, enemyID)
					if refEnemy == nil || refEnemy.HP <= 0 {
						return nil
					}
					for _, eff := range refEnemy.StatusEffects {
						if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
							return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refEnemy.Name)}
						}
					}

					targetsIdx := ResolveTargets(player.ActiveCombat.Allies, "SINGLE", 2, 3, false)
					if len(targetsIdx) > 0 {
						targetAlly := &player.ActiveCombat.Allies[targetsIdx[0]]
						enemyCtx := &CombatContext{
							Attacker: refEnemy,
							Defender: targetAlly,
							Logs:     []string{},
						}
						enemyActions := BuildEnemyActions(enemyCtx, refEnemy)
						if len(enemyActions) > 0 {
							act := enemyActions[0]
							if act.ManaCost > 0 {
								if player.ActiveCombat.EnemyType != "PVP" && refEnemy.Kind != "Player" {
									hpCost := refEnemy.MaxHP * 0.05
									if hpCost < 1.0 {
										hpCost = 1.0
									}
									refEnemy.HP -= hpCost
									logs = append(logs, fmt.Sprintf("-> %s tiêu hao sinh lực (Mất %.1f HP) để thúc dục Cổ ký sinh!", refEnemy.Name, hpCost))
									if refEnemy.HP <= 0 {
										logs = append(logs, fmt.Sprintf("-> %s bị Cổ ký sinh vắt kiệt sinh mệnh đến chết!", refEnemy.Name))
										return logs
									}
								} else if refEnemy.CurrentMana >= act.ManaCost {
									refEnemy.CurrentMana -= act.ManaCost
								}
							}

							isHealing := act.Def > 0 && (act.IsUtility || (len(act.StatusEffects) == 0 && act.Dmg == 0))
							isAttack := act.Dmg > 0 || len(act.StatusEffects) > 0

							if isHealing {
								healVal := act.Def
								refEnemy.HP += healVal
								if refEnemy.HP > refEnemy.MaxHP {
									refEnemy.HP = refEnemy.MaxHP
								}
								logs = append(logs, fmt.Sprintf("-> %s kích hoạt phòng ngự/trị liệu %s, hồi %.1f HP.", refEnemy.Name, act.Name, healVal))
							} else if isAttack {
								netDmg := act.Dmg - targetAlly.BaseDefense
								if netDmg < 1.0 {
									netDmg = 1.0
								}
								dmgLog := applyDamageWithLogs(refEnemy.Name, targetAlly.Name, targetAlly, netDmg)
								logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s tấn công!", refEnemy.Name, act.Name), dmgLog)
							} else {
								logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s (Hỗ trợ)!", refEnemy.Name, act.Name))
							}
						} else {
							dmgVal := refEnemy.BaseAttack
							if dmgVal <= 0 {
								dmgVal = 5.0
							}
							netDmg := dmgVal - targetAlly.BaseDefense
							if netDmg < 1.0 {
								netDmg = 1.0
							}
							dmgLog := applyDamageWithLogs(refEnemy.Name, targetAlly.Name, targetAlly, netDmg)
							logs = append(logs, fmt.Sprintf("-> %s đánh thường!", refEnemy.Name), dmgLog)
						}
					}
					return logs
				},
			})
		}
	}

	sort.SliceStable(actions, func(i, j int) bool {
		return effectiveCombatSpeed(player.ActiveCombat, actions[i].InitiatorID) > effectiveCombatSpeed(player.ActiveCombat, actions[j].InitiatorID)
	})

	// runDuel executes the interleaved exchange between the caster and the
	// primary target: each caster strike is answered by a riposte.
	runDuel := func() {
		for _, step := range duelSeq {
			if stepLogs := step(); len(stepLogs) > 0 {
				turnLogs = append(turnLogs, stepLogs...)
			}
		}
	}

	duelRan := false
	for _, act := range actions {
		refActor, _, _ := findCombatant(player.ActiveCombat, act.InitiatorID)
		if refActor == nil || refActor.HP <= 0 {
			continue
		}
		if hasDuel && !duelRan && (act.InitiatorID == casterID || act.InitiatorID == primaryEnemyID) {
			duelRan = true
			runDuel()
			if act.InitiatorID == casterID {
				// Caster content was executed inside the duel sequence.
				continue
			}
		}
		resLogs := act.Execute()
		if len(resLogs) > 0 {
			turnLogs = append(turnLogs, resLogs...)
		}
	}
	if hasDuel && !duelRan {
		runDuel()
	}

	if !isVirtual {
		player.HP = player.ActiveCombat.Allies[casterIdx].HP
		player.CurrentMana = player.ActiveCombat.Allies[casterIdx].CurrentMana

		for _, ally := range player.ActiveCombat.Allies {
			if ally.ID != player.ID.String() {
				if ally.Kind == "Pet" {
					// Pet ally — resolve death or update HP
					if ally.HP <= 0 {
						_, _ = tx.Exec(`DELETE FROM player_pets WHERE id = $1 AND player_id = $2`, ally.ID, player.ID)
						_, _ = tx.Exec(`
							INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
							VALUES ($1, 'pet.died_in_combat', 'player_pet', $2, $3, '{}', '', false, 'Pet chết trong chiến đấu', NOW())
						`, player.ID, ally.ID, fmt.Sprintf("%s chết khi chiến đấu", ally.Name))
					} else {
						_, _ = tx.Exec(`UPDATE player_pets SET hp = $1, updated_at = NOW() WHERE id = $2 AND player_id = $3`,
							ally.HP, ally.ID, player.ID)
					}
					continue
				}
				allyUUID, errParse := uuid.Parse(ally.ID)
				if errParse == nil {
					allyHP := ally.HP
					if allyHP < 0 {
						allyHP = 0
					}
					_, _ = tx.Exec("UPDATE players SET hp = $1 WHERE id = $2", allyHP, allyUUID)
				}
			}
		}

		// Persist challenged/defender pet damage & death (mine / real PVP)
		defenderID := ""
		if player.ActiveCombat != nil {
			defenderID = player.ActiveCombat.TargetPlayerID
		}
		if defenderID != "" && defenderID != "dummy" {
			for _, enemy := range player.ActiveCombat.Enemies {
				if enemy.Kind != "Pet" {
					continue
				}
				if enemy.HP <= 0 {
					_, _ = tx.Exec(`DELETE FROM player_pets WHERE id = $1 AND player_id = $2`, enemy.ID, defenderID)
					_, _ = tx.Exec(`
						INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
						VALUES ($1, 'pet.died_in_combat', 'player_pet', $2, $3, '{}', '', false, 'Pet chết trong chiến đấu', NOW())
					`, defenderID, enemy.ID, fmt.Sprintf("%s chết khi phòng thủ trước %s", enemy.Name, player.ActiveCombat.EnemyName))
					turnLogs = append(turnLogs, fmt.Sprintf("%s đã chết trong chiến đấu!", enemy.Name))
				} else {
					_, _ = tx.Exec(`UPDATE player_pets SET hp = $1, updated_at = NOW() WHERE id = $2 AND player_id = $3`,
						enemy.HP, enemy.ID, defenderID)
				}
			}
		}
	}

	victory := true
	for _, enemy := range player.ActiveCombat.Enemies {
		if enemy.HP > 0 {
			victory = false
			break
		}
	}

	defeat := player.HP <= 0
	isPVP := player.ActiveCombat.TargetPlayerID != ""

	ctx := &CombatContext{
		Attacker: &playerCombatant,
		Logs:     turnLogs,
	}

	// Capture history data before ActiveCombat is cleared
	var histEnemyName, histEnemyType string
	var histTargetPlayerID string
	var histItemsUsed int
	if player.ActiveCombat != nil {
		histEnemyName = player.ActiveCombat.EnemyName
		histEnemyType = player.ActiveCombat.EnemyType
		histTargetPlayerID = player.ActiveCombat.TargetPlayerID
		histItemsUsed = player.ActiveCombat.ItemsUsedCount
	}

	if victory {
		ctx.Logs = append(ctx.Logs, "=== CHIẾN THẮNG VANG DỘI ===")

		qiGained := AddHumanQi(&player, 20.0)
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiến thắng đem lại sự tin tưởng của bá tánh. Nhân Khí +%.0f.", qiGained))

		IntegratePassiveGus(&player, &ctx.Logs)

		var partyMembers []uuid.UUID
		if player.PartyID != nil {
			var membersRaw []byte
			errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
			if errPart == nil {
				var members []PartyMember
				json.Unmarshal(membersRaw, &members)
				for _, m := range members {
					mUUID, errParse := uuid.Parse(m.ID)
					if errParse == nil {
						var sameCell bool
						errCell := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE id = $1 AND x = $2 AND y = $3 AND current_region = $4)", mUUID, player.X, player.Y, player.CurrentRegion).Scan(&sameCell)
						if errCell == nil && sameCell {
							partyMembers = append(partyMembers, mUUID)
						}
					}
				}
			}
		}

		if isPVP && !isVirtual {
			if player.ActiveCombat.TargetPlayerID != "dummy" {
				targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
				if parseErr == nil {
					var targetHP float64
					if errHP := tx.QueryRow("SELECT hp FROM players WHERE id = $1 FOR UPDATE", targetUUID).Scan(&targetHP); errHP == nil {
						targetHP -= 30.0
						if targetHP < 10.0 {
							targetHP = 10.0
						}
						_, _ = tx.Exec("UPDATE players SET hp = $1 WHERE id = $2", targetHP, targetUUID)
					}
					dropSummary := applyDeathDropRecord(tx, &targetUUID, &player.ID)
					if dropSummary != "" {
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiến thắng PVP vang dội! Đạo hữu cướp đoạt từ bại binh: %s.", dropSummary))
					} else {
						ctx.Logs = append(ctx.Logs, "Chiến thắng PVP vang dội! Bại binh không còn gì đáng giá để cướp đoạt.")
					}
				}
			}
		} else {
			if player.ActiveCombat != nil && player.ActiveCombat.LootTableID != "" {
				lt, err := db.LoadLootTable(tx, player.ActiveCombat.LootTableID)
				if err == nil {
					loot := RollLoot(lt, player.ActiveCombat.EnemyRank)
					if player.ActiveCombat.EnemyGuInventory != nil && len(player.ActiveCombat.EnemyGuInventory) > 0 {
						for _, eg := range player.ActiveCombat.EnemyGuInventory {
							if rand.Intn(100) < 10 {
								loot.Gus = append(loot.Gus, eg.GuID)
								ctx.Logs = append(ctx.Logs, fmt.Sprintf("Cơ duyên! Đạo hữu bóc tách thành công cổ trùng ký sinh %s từ thi thể kẻ địch!", eg.GuID))
							}
						}
					}
					if len(partyMembers) > 1 {
						ctx.Logs = append(ctx.Logs, "=== PHÂN CHIA CHIẾN LỢI PHẨM TỔ ĐỘI ===")
						stoneQty := loot.Items["primeval_stone"]
						if stoneQty > 0 {
							splitStones := stoneQty / len(partyMembers)
							if splitStones < 1 {
								splitStones = 1
							}
							for _, memberID := range partyMembers {
								if memberID == player.ID {
									player.PrimevalStones += int64(splitStones)
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", splitStones))
								} else {
									_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", splitStones, memberID)
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", memberID).Scan(&uname)
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s nhận được %d Nguyên Thạch.", uname, splitStones))
								}
							}
						}

						for matID, qty := range loot.Items {
							if matID == "primeval_stone" {
								continue
							}
							for q := 0; q < qty; q++ {
								winnerID := partyMembers[rand.Intn(len(partyMembers))]
								if winnerID == player.ID {
									if err := InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", 1); err != nil {
										logger.S.Errorf("Failed to add loot item %s to player %s: %v", matID, player.ID, err)
									}
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhặt được 1x %s.", matID))
								} else {
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)
									if err := InventorySvc.AddItem(tx, winnerID.String(), matID, "materials", 1); err != nil {
										logger.S.Errorf("Failed to distribute loot %s to party member %s: %v", matID, winnerID, err)
									}
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s nhặt được 1x %s.", uname, matID))
								}
							}
						}

						for _, gID := range loot.Gus {
							winnerID := partyMembers[rand.Intn(len(partyMembers))]
							guSpec, _ := models.GetGuAny(gID)
							newGu := models.GuInstance{
								InstanceID: uuid.New().String(),
								GuID:       gID,
								Rank:       guSpec.Rank,
								Hunger:     100,
								Status:     "ACTIVE",
								Equipped:   false,
							}
							if winnerID == player.ID {
								player.GuInventory = append(player.GuInventory, newGu)
								ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn bắt được 1x %s!", gID))
							} else {
								var uname string
								_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)
								_ = db.AddPlayerGu(tx, winnerID.String(), newGu)
								ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s thu phục thành công Cổ trùng: %s!", uname, gID))
							}
						}
					} else {
						lootLogs := ApplyLoot(tx, &player, loot)
						ctx.Logs = append(ctx.Logs, lootLogs...)
					}
					markBossCleared(tx, &player)
				} else {
					logger.S.Errorf("Failed to load loot table %s: %v", player.ActiveCombat.LootTableID, err)
				}
			} else {
				ctx.Logs = append(ctx.Logs, "Chiến đấu thắng lợi vang dội!")
			}
		}

		if isPVP {
			if player.ActiveCombat.TargetPlayerID != "dummy" {
				targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
				if parseErr == nil {
					var playerRank, targetRank int
					errPR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", player.ID).Scan(&playerRank)
					errTR := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", targetUUID).Scan(&targetRank)
					if errPR == nil && errTR == nil && playerRank > targetRank {
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = -1 WHERE player_id = $1", player.ID)
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", targetRank, player.ID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", playerRank, targetUUID)
						_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", targetRank, player.ID)
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chúc mừng! Đạo hữu đã vượt cấp khiêu chiến thành công, thăng lên Hạng %d Võ Đài!", targetRank))
					}
				}
			}
		}

		if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
			slotID := player.ActiveCombat.MineSlotID
			var rarity string
			var mineType string
			var currentOccupierStr *string
			var monsterIDStr *string
			var lastClaimedAt *time.Time
			err = tx.QueryRow("SELECT rarity, mine_type, occupier_id, monster_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE", slotID).Scan(&rarity, &mineType, &currentOccupierStr, &monsterIDStr, &lastClaimedAt)
			if err == nil && currentOccupierStr != nil && lastClaimedAt != nil {
				stones, qi, chests := CalculateMineRewards(slotID, rarity, *lastClaimedAt)
				stonesHalf := stones / 2
				qiHalf := qi / 2.0
				chestsHalf := chests / 2
				if stonesHalf > 0 || qiHalf > 0.0 || chestsHalf > 0 {
					if errMine := AddMineRewardsToPlayerTx(tx, player.ID, stonesHalf, qiHalf, chestsHalf, mineType); errMine != nil {
						logger.S.Errorf("Failed to add mine rewards to player %s (slot %d): %v", player.ID, slotID, errMine)
					} else {
						if mineType == "immortal" {
							player.ImmortalStones += int64(stonesHalf)
						} else {
							player.PrimevalStones += int64(stonesHalf)
						}
						player.HumanQi += qiHalf
						
						stoneName := "Nguyên Thạch"
						if mineType == "immortal" {
							stoneName = "Tiên Nguyên Thạch"
						}
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Cướp mỏ thành công! +%d %s, +%.0f Nhân Khí, +%d Rương Mỏ", stonesHalf, stoneName, qiHalf, chestsHalf))
					}
				} else {
					ctx.Logs = append(ctx.Logs, "Mỏ đối thủ chưa tích lũy đủ tài nguyên để cướp.")
				}
			} else if err == nil {
				ctx.Logs = append(ctx.Logs, "Mỏ không có người chiếm giữ, không thể cướp tài nguyên.")
			}

			now := time.Now()
			_, errUpdate := tx.Exec(`
					UPDATE mine_slots
					SET occupier_id = $1, monster_id = NULL, occupied_at = $2, last_claimed_at = $3
					WHERE id = $4
				`, player.ID, now, now, slotID)
			if errUpdate == nil {
				ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiếm lĩnh thành công mỏ phẩm chất %s #%d!", rarity, slotID))
			} else {
				logger.S.Errorf("Failed to update mine slot occupier in combat victory: %v", errUpdate)
			}
		}

		if player.ClearedCells == nil {
			player.ClearedCells = make(models.ClearedCellsMap)
		}
		coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
		player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)

		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil {
				if allyUUID != player.ID {
					_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID)
				}
			}
		}
		player.ActiveCombat = nil
	} else if defeat {
		player.HP = 1.0
		defeat = true
		qiGained := AddHumanQi(&player, 5.0)
		ctx.Logs = append(ctx.Logs, fmt.Sprintf("Dù bại trận, đạo hữu vẫn giữ được chí khí. Nhân Khí +%.0f.", qiGained))

		isTribulation := player.ActiveCombat != nil && player.ActiveCombat.EnemyTemplateID == "monster_heavenly_tribulation"

		if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
			ctx.Logs = append(ctx.Logs, "Thách đấu mỏ linh thạch thất bại, không thể chiếm lĩnh mỏ.")
		} else if !isTribulation {
			ctx.Logs = append(ctx.Logs, "Đại bại! Đạo hữu bất tỉnh nhân sự trước áp lực của đối thủ. May mắn thoát chết.")
		}

		player.CurrentRegion, player.X, player.Y = NearestHub(player.X, player.Y)

		if isTribulation {
			player.ImmortalStones /= 2
			_, _ = tx.Exec(`
				UPDATE immortal_apertures 
				SET tribulation_count = 0, earth_calamity_count = 0, heavenly_tribulation_count = 0, grand_tribulation_count = 0, myriad_tribulation_count = 0, chaos_calamity_count = 0 
				WHERE player_id = $1
			`, player.ID)
			nextTrib := time.Now().Add(10 * time.Minute)
			player.NextTribulationTime = &nextTrib

			player.HumanQi += 200.0
			if player.HumanQi > 100000.0 {
				player.HumanQi = 100000.0
			}
			ctx.Logs = append(ctx.Logs, "Độ kiếp thất bại nhưng may mắn giữ được mạng sống! Nhân khí +200. Tiên Nguyên Thạch bị hao tổn 50% và reset số lần độ kiếp.")
		}

		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil {
				if allyUUID != player.ID {
					_, _ = tx.Exec("UPDATE players SET active_combat = NULL WHERE id = $1", allyUUID)
				}
			}
		}
		player.ActiveCombat = nil
	} else {
		combatBytes, _ := json.Marshal(player.ActiveCombat)
		for _, ally := range player.ActiveCombat.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil && allyUUID != player.ID {
				_, _ = tx.Exec("UPDATE players SET active_combat = $1 WHERE id = $2", combatBytes, allyUUID)
			}
		}
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update combat results"})
		return
	}
	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, primeval_stones = $3, cleared_cells = $4, active_combat = $5,
		    current_region = $6, x = $7, y = $8, human_qi = $9, path_masteries = $10
		WHERE id = $11
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.PrimevalStones, player.ClearedCells, player.ActiveCombat,
		player.CurrentRegion, player.X, player.Y, player.HumanQi, player.PathMasteries, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update combat results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	if victory || defeat {
		action := "combat.defeat"
		if victory {
			action = "combat.victory"
		}
		auditlog.Logf(db.DB, &authID, action, "enemy=%s enemyType=%s", histEnemyName, histEnemyType)
	}

	setWorldPosFromPlayer(&player)

	// Save combat history asynchronously
	if victory || defeat {
		winner := "Phe Địch"
		if victory {
			winner = "Phe Ta"
		}
		go func() {
			logsIface := make([]interface{}, len(ctx.Logs))
			for i, l := range ctx.Logs {
				logsIface[i] = l
			}
			summary := CombatSummary{
				FinalPlayerHP: int(player.HP),
				ItemsUsed:     histItemsUsed,
			}
			SaveCombatHistory(player.ID, winner, logsIface, summary, nil, histEnemyName, histEnemyType, player.CurrentRegion)

			if isPVP && histTargetPlayerID != "" && histTargetPlayerID != "dummy" {
				pvpWinner := "Phe Ta"
				if defeat {
					pvpWinner = "Phe Địch"
				}
				SaveCombatHistory(uuid.MustParse(histTargetPlayerID), pvpWinner, logsIface, summary, nil, player.Username, "PVP", player.CurrentRegion)
			}
		}()
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"logs":    ctx.Logs,
		"enemy":   player.ActiveCombat,
		"victory": victory,
		"defeat":  defeat,
	})
}
