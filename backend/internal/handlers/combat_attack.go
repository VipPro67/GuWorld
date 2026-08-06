package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
				case "poison", "bleed":
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
				case "poison", "bleed":
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
			qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), req.ItemID, "materials")
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
							if p == "Moon Path" || p == "Earth Path" {
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

					_, _, cost, errVal := CalculateGuEffectForCombatant(&playerCombatant, &spec, true)
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

	if targetIdx != -1 {
		casterCombatant := &player.ActiveCombat.Allies[casterIdx]
		prefX := req.TargetX
		prefY := req.TargetY
		if prefX <= 0 {
			prefX = 1
		}
		if prefY <= 0 {
			prefY = 2
		}

		actions = append(actions, CombatAction{
			InitiatorID: player.ID.String(),
			Speed:       casterCombatant.Speed,
			Execute: func() []string {
				var logs []string
				refCaster, _, _ := findCombatant(player.ActiveCombat, player.ID.String())
				if refCaster == nil || refCaster.HP <= 0 {
					return nil
				}
				for _, eff := range refCaster.StatusEffects {
					if (eff.Type == "stun" || eff.Type == "freeze") && eff.Turns > 0 {
						return []string{fmt.Sprintf("-> %s bị choáng/đóng băng và mất lượt!", refCaster.Name)}
					}
				}
				// Check silence — force basic attack only
				isSilenced := false
				for _, eff := range refCaster.StatusEffects {
					if eff.Type == "silence" && eff.Turns > 0 {
						isSilenced = true
						break
					}
				}

				if isUsingStones {
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
				} else if isUsingItem {
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
				} else if isChargingThisTurn {
					logs = append(logs, fmt.Sprintf("%s tích tụ Nguyên Khí, vận chuyển Chân Nguyên chuẩn bị cho Đại Sát Chiêu!", refCaster.Name))
				} else if isUnleashingCharge {
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
				} else if isSilenced {
					// Silence: force basic attack only
					targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", prefX, prefY, false)
					if len(targetsIdx) > 0 {
						targetEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
						daoMult := 1.0
						if refCaster.DaoMarks != nil {
							if amt, ok := refCaster.DaoMarks["strength"]; ok && amt > 0 {
								if amt <= 10000 {
									daoMult = 1.0 + float64(amt)/100.0
								} else {
									daoMult = 1.0 + 100.0 + float64(amt-10000)/500.0
								}
							}
						}
						dmgVal := refCaster.BaseAttack
						if dmgVal <= 0 {
							dmgVal = 5.0
						}
						fullDmg := dmgVal * daoMult
						netDmg := fullDmg - targetEnemy.BaseDefense
						if netDmg < 1.0 {
							netDmg = 1.0
						}
						dmgLog := applyDamageWithLogs(refCaster.Name, targetEnemy.Name, targetEnemy, netDmg)
						logs = append(logs, fmt.Sprintf("-> %s bị Im Lặng, chỉ có thể đánh thường!", refCaster.Name), dmgLog)
					}
				} else {
					if looseGuFailureChance > 0 && rand.Float64() < looseGuFailureChance {
						wastedMana := totalManaCost * 0.5
						if wastedMana > refCaster.CurrentMana {
							wastedMana = refCaster.CurrentMana
						}
						refCaster.CurrentMana -= wastedMana
						logs = append(logs, fmt.Sprintf("-> %s cưỡng ép vận nhiều Cổ riêng lẻ, đạo ngân hỗn loạn khiến chiêu thất bại! Hao phí %.1f nguyên khí.", refCaster.Name, wastedMana))
						return logs
					}

					for _, spec := range selectedSpecs {
						dmgBase, defBase, _, _ := CalculateGuEffectForCombatant(refCaster, &spec, false)
						dmg, def, cost, errGu := CalculateGuEffectForCombatant(refCaster, &spec, true)
						if errGu == nil && refCaster.CurrentMana >= cost {
							refCaster.CurrentMana -= cost
							if refCaster.Rank < spec.Rank {
								backlashHP := 5.0
								if refCaster.HP <= backlashHP {
									refCaster.HP = 0
								} else {
									refCaster.HP -= backlashHP
								}
								logs = append(logs, fmt.Sprintf("[Phản Phệ] Cổ trùng Rank cao phản phệ %s, trừ %.0f HP (còn %.0f HP)!", refCaster.Name, backlashHP, refCaster.HP))
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
								isHealing = spec.Effects.Active.HealFlat > 0 || spec.ID == "GU_NINE_LEAF_VITALITY_GRASS"
								if spec.Effects.Active.ShieldFlat > 0 { isDefense = true }
								if spec.Effects.Active.DamageFlat > 0 || spec.Effects.Active.DamageMult > 0 { isAttack = true }
							}

							if isHealing {
								healVal := def
								if false {
									healVal = refCaster.BaseAttack * 1.5
								}
								if healVal <= 0 {
									healVal = 30.0
								}
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
									logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s, khôi phục %.1f HP bản thân (HP hiện tại: %.1f) [Gốc: %.1f, Đạo Ngân: +%.1f].", refCaster.Name, spec.Name, healVal, refCaster.HP, healBase, healDao))
								} else {
									logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s, khôi phục %.1f HP bản thân (HP hiện tại: %.1f).", refCaster.Name, spec.Name, healVal, refCaster.HP))
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
									logs = append(logs, fmt.Sprintf("-> %s thi triển %s phòng thủ, nhận %.1f Khiên hộ thể (Virtual Shield: %.1f) [Gốc: %.1f, Đạo Ngân: +%.1f].", refCaster.Name, spec.Name, shieldVal, refCaster.VirtualShield, shieldBase, shieldDao))
								} else {
									logs = append(logs, fmt.Sprintf("-> %s thi triển %s phòng thủ, nhận %.1f Khiên hộ thể (Virtual Shield: %.1f).", refCaster.Name, spec.Name, shieldVal, refCaster.VirtualShield))
								}
							} else if isAttack {
								hasBypass := spec.Effects.Active != nil && spec.Effects.Active.CurveAttack
								targetType := spec.Effects.Active.TargetSelector;
								targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, targetType, prefX, prefY, hasBypass)
								if len(targetsIdx) > 0 {
									dmgDao := dmg - dmgBase
									if dmgDao > 0 {
										logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s! (Sức mạnh: %.1f [Gốc: %.1f, Đạo Ngân: +%.1f])", refCaster.Name, spec.Name, dmg, dmgBase, dmgDao))
									} else {
										logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s!", refCaster.Name, spec.Name))
									}
								for _, eIdx := range targetsIdx {
									tEnemy := &player.ActiveCombat.Enemies[eIdx]
									netDmg := dmg - tEnemy.BaseDefense
									if netDmg < 1.0 {
										netDmg = 1.0
									}
									dmgLog := applyDamageWithLogs(refCaster.Name, tEnemy.Name, tEnemy, netDmg)
									logs = append(logs, dmgLog)

									// Apply status effects from Gu to target
									if spec.Effects.Active != nil {
										active := spec.Effects.Active
										switch active.ApplyStatus {
										case "SILENCE":
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "silence", Value: 1, Turns: active.StatusDurationTicks,
											})
											logs = append(logs, fmt.Sprintf("%s bị Im lặng %d lượt!", tEnemy.Name, active.StatusDurationTicks))
										case "STUN":
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "stun", Value: 1, Turns: active.StatusDurationTicks,
											})
											logs = append(logs, fmt.Sprintf("%s bị Choáng %d lượt!", tEnemy.Name, active.StatusDurationTicks))
										case "FREEZE":
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "freeze", Value: 1, Turns: active.StatusDurationTicks,
											})
											logs = append(logs, fmt.Sprintf("%s bị Đóng băng %d lượt!", tEnemy.Name, active.StatusDurationTicks))
										case "SLOW":
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "slow", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
											})
											logs = append(logs, fmt.Sprintf("%s bị Chậm %d%% %d lượt!", tEnemy.Name, int(active.StatusTickDamage), active.StatusDurationTicks))
										case "WEAK":
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "weak", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
											})
											logs = append(logs, fmt.Sprintf("%s bị Yếu %d%% %d lượt!", tEnemy.Name, int(active.StatusTickDamage), active.StatusDurationTicks))
										case "BLEED":
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "bleed", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
											})
											logs = append(logs, fmt.Sprintf("%s bị Chảy máu (%.0f/lượt %d lượt)!", tEnemy.Name, active.StatusTickDamage, active.StatusDurationTicks))
										case "POISON":
											tEnemy.StatusEffects = append(tEnemy.StatusEffects, models.StatusEffect{
												Type: "poison", Value: active.StatusTickDamage, Turns: active.StatusDurationTicks,
											})
											logs = append(logs, fmt.Sprintf("%s bị Độc (%.0f/lượt %d lượt)!", tEnemy.Name, active.StatusTickDamage, active.StatusDurationTicks))
										}
									}
								}
								}
							} else {
								logs = append(logs, fmt.Sprintf("-> %s kích hoạt %s (Hỗ trợ)!", refCaster.Name, spec.Name))
							}
						}
					}
					if len(selectedSpecs) == 0 {
						targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", prefX, prefY, false)
						if len(targetsIdx) > 0 {
							targetEnemy := &player.ActiveCombat.Enemies[targetsIdx[0]]
							daoMult := 1.0
							if refCaster.DaoMarks != nil {
								if amt, ok := refCaster.DaoMarks["strength"]; ok && amt > 0 {
									if amt <= 10000 {
										daoMult = 1.0 + float64(amt)/100.0
									} else {
										daoMult = 1.0 + 100.0 + float64(amt-10000)/500.0
									}
								}
							}
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
							netDmg := fullDmg - targetEnemy.BaseDefense
							if netDmg < 1.0 {
								netDmg = 1.0
							}
							dmgLog := applyDamageWithLogs(refCaster.Name, targetEnemy.Name, targetEnemy, netDmg)
							if daoMult > 1.0 {
								dmgDao := fullDmg - dmgVal
								logs = append(logs, fmt.Sprintf("-> %s đánh thường (Basic Attack) [Gốc: %.1f, Đạo Ngân: +%.1f]!", refCaster.Name, dmgVal, dmgDao), dmgLog)
							} else {
								logs = append(logs, fmt.Sprintf("-> %s đánh thường (Basic Attack)!", refCaster.Name), dmgLog)
							}
						}
					}
				}
				return logs
			},
		})
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

					targetsIdx := ResolveTargets(player.ActiveCombat.Enemies, "SINGLE", 1, 2, false)
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

					targetsIdx := ResolveTargets(player.ActiveCombat.Allies, "SINGLE", 2, 2, false)
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

	sort.Slice(actions, func(i, j int) bool {
		speedI := actions[i].Speed
		speedJ := actions[j].Speed
		// Apply slow debuff to effective speed
		refI, _, _ := findCombatant(player.ActiveCombat, actions[i].InitiatorID)
		if refI != nil {
			for _, eff := range refI.StatusEffects {
				if eff.Type == "slow" && eff.Turns > 0 {
					speedI *= (1.0 - eff.Value/100.0)
				}
			}
		}
		refJ, _, _ := findCombatant(player.ActiveCombat, actions[j].InitiatorID)
		if refJ != nil {
			for _, eff := range refJ.StatusEffects {
				if eff.Type == "slow" && eff.Turns > 0 {
					speedJ *= (1.0 - eff.Value/100.0)
				}
			}
		}
		return speedI > speedJ
	})

	for _, act := range actions {
		refActor, _, _ := findCombatant(player.ActiveCombat, act.InitiatorID)
		if refActor == nil || refActor.HP <= 0 {
			continue
		}
		resLogs := act.Execute()
		if len(resLogs) > 0 {
			turnLogs = append(turnLogs, resLogs...)
		}
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
			stolenStones := 10.0
			if player.ActiveCombat.TargetPlayerID != "dummy" {
				targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
				if parseErr == nil {
					var targetStones float64
					var targetHP float64
					var targetLootBytes []byte
					err = tx.QueryRow("SELECT primeval_stones, hp, temp_loot FROM players WHERE id = $1 FOR UPDATE", targetUUID).Scan(&targetStones, &targetHP, &targetLootBytes)
					if err == nil {
						stolenStones = targetStones * 0.10
						if stolenStones < 1 {
							stolenStones = 1
						}
						if targetStones < stolenStones {
							stolenStones = targetStones
						}
						targetStones -= stolenStones
						targetHP -= 30.0
						if targetHP < 10.0 {
							targetHP = 10.0
						}

						var targetLoot models.TempLoot
						if len(targetLootBytes) > 0 {
							json.Unmarshal(targetLootBytes, &targetLoot)
						}

						player.TempLoot.PrimevalStones += targetLoot.PrimevalStones
						if player.TempLoot.MaterialsInventory == nil {
							player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
						}
						for matID, qty := range targetLoot.MaterialsInventory {
							player.TempLoot.MaterialsInventory[matID] += qty
						}
						player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, targetLoot.GuInventory...)

						emptyLoot := models.TempLoot{
							GuInventory:        models.GuInventory{},
							MaterialsInventory: models.MaterialsInventory{},
							PrimevalStones:     0,
						}

						_, _ = tx.Exec("UPDATE players SET primeval_stones = $1, hp = $2, temp_loot = $3 WHERE id = $4", targetStones, targetHP, emptyLoot, targetUUID)

						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Thu hoạch toàn bộ chiến lợi phẩm tạm thời của bại binh! Hút được %.0f Nguyên Thạch, %d Cổ trùng, và các nguyên liệu khác từ đối phương.", targetLoot.PrimevalStones, len(targetLoot.GuInventory)))
					}

					}
			}
			player.PrimevalStones += stolenStones
			ctx.Logs = append(ctx.Logs, fmt.Sprintf("Chiến thắng PVP vang dội! Đạo hữu cướp đoạt %.0f Nguyên Thạch từ đối phương!", stolenStones))
		} else {
			isExplore := player.CurrentRegion != "Southern_Border" && player.CurrentRegion != "Lobby"
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
									if isExplore {
										player.TempLoot.PrimevalStones += float64(splitStones)
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch (giữ tạm thời).", splitStones))
									} else {
										player.PrimevalStones += float64(splitStones)
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhận được %d Nguyên Thạch.", splitStones))
									}
								} else {
									if isExplore {
										var targetLoot models.TempLoot
										var targetLootBytes []byte
										_ = tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", memberID).Scan(&targetLootBytes)
										if len(targetLootBytes) > 0 {
											json.Unmarshal(targetLootBytes, &targetLoot)
										}
										targetLoot.PrimevalStones += float64(splitStones)
										_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, memberID)
									} else {
										_, _ = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", splitStones, memberID)
									}
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
									if isExplore {
										if player.TempLoot.MaterialsInventory == nil {
											player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
										}
										player.TempLoot.MaterialsInventory[matID]++
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhặt được 1x %s (giữ tạm thời).", matID))
									} else {
										if err := InventorySvc.AddItem(tx, player.ID.String(), matID, "materials", 1); err != nil {
											logger.S.Errorf("Failed to add loot item %s to player %s: %v", matID, player.ID, err)
										}
										ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn nhặt được 1x %s.", matID))
									}
								} else {
									var uname string
									_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)

									if isExplore {
										var targetLoot models.TempLoot
										var targetLootBytes []byte
										_ = tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", winnerID).Scan(&targetLootBytes)
										if len(targetLootBytes) > 0 {
											json.Unmarshal(targetLootBytes, &targetLoot)
										}
										if targetLoot.MaterialsInventory == nil {
											targetLoot.MaterialsInventory = make(models.MaterialsInventory)
										}
										targetLoot.MaterialsInventory[matID]++
										_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, winnerID)
									} else {
										if err := InventorySvc.AddItem(tx, winnerID.String(), matID, "materials", 1); err != nil {
											logger.S.Errorf("Failed to distribute loot %s to party member %s: %v", matID, winnerID, err)
										}
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
								if isExplore {
									player.TempLoot.GuInventory = append(player.TempLoot.GuInventory, newGu)
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn bắt được 1x %s (giữ tạm thời)!", gID))
								} else {
									player.GuInventory = append(player.GuInventory, newGu)
									ctx.Logs = append(ctx.Logs, fmt.Sprintf("Bạn bắt được 1x %s!", gID))
								}
							} else {
								var uname string
								_ = tx.QueryRow("SELECT username FROM players WHERE id = $1", winnerID).Scan(&uname)

								if isExplore {
									var targetLoot models.TempLoot
									var targetLootBytes []byte
									_ = tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", winnerID).Scan(&targetLootBytes)
									if len(targetLootBytes) > 0 {
										json.Unmarshal(targetLootBytes, &targetLoot)
									}
									targetLoot.GuInventory = append(targetLoot.GuInventory, newGu)
									_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, winnerID)
								} else {
									targetGuInv, err := db.LoadPlayerGus(tx, winnerID.String(), false)
									if err != nil {
										targetGuInv = models.GuInventory{}
									}
									targetGuInv = append(targetGuInv, newGu)
									_ = db.SavePlayerGus(tx, winnerID.String(), targetGuInv)
								}
								ctx.Logs = append(ctx.Logs, fmt.Sprintf("Đồng đội %s thu phục thành công Cổ trùng: %s!", uname, gID))
							}
						}
					} else {
						lootLogs := ApplyLoot(tx, &player, loot)
						ctx.Logs = append(ctx.Logs, lootLogs...)
					}
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
			var currentOccupierStr *string
			var monsterIDStr *string
			var lastClaimedAt *time.Time
			err = tx.QueryRow("SELECT rarity, occupier_id, monster_id, last_claimed_at FROM mine_slots WHERE id = $1 FOR UPDATE", slotID).Scan(&rarity, &currentOccupierStr, &monsterIDStr, &lastClaimedAt)
			if err == nil && currentOccupierStr != nil && lastClaimedAt != nil {
				stones, qi, chests := CalculateMineRewards(slotID, rarity, *lastClaimedAt)
				stonesHalf := stones / 2
				qiHalf := qi / 2.0
				chestsHalf := chests / 2
				if stonesHalf > 0 || qiHalf > 0.0 || chestsHalf > 0 {
					if errMine := AddMineRewardsToPlayerTx(tx, player.ID, stonesHalf, qiHalf, chestsHalf); errMine != nil {
						logger.S.Errorf("Failed to add mine rewards to player %s (slot %d): %v", player.ID, slotID, errMine)
					} else {
						player.PrimevalStones += float64(stonesHalf)
						player.HumanQi += qiHalf
						ctx.Logs = append(ctx.Logs, fmt.Sprintf("Cướp mỏ thành công! +%d Nguyên Thạch, +%.0f Nhân Khí, +%d Rương Mỏ", stonesHalf, qiHalf, chestsHalf))
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
		if player.ActiveCombat != nil && player.ActiveCombat.MineSlotID > 0 {
			ctx.Logs = append(ctx.Logs, "Thách đấu mỏ linh thạch thất bại, không thể chiếm lĩnh mỏ.")
		} else {
			ctx.Logs = append(ctx.Logs, "Đại bại! Đạo hữu bất tỉnh nhân sự trước áp lực của đối thủ. May mắn thoát chết nhưng rơi rớt mất sạch chiến lợi phẩm thu thập trong chuyến đi này.")
		}

		if isPVP && !isVirtual && player.ActiveCombat.TargetPlayerID != "dummy" {
			targetUUID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID)
			if parseErr == nil {
				var targetLootBytes []byte
				errLoot := tx.QueryRow("SELECT temp_loot FROM players WHERE id = $1 FOR UPDATE", targetUUID).Scan(&targetLootBytes)
				if errLoot == nil {
					var targetLoot models.TempLoot
					if len(targetLootBytes) > 0 {
						json.Unmarshal(targetLootBytes, &targetLoot)
					}
					targetLoot.PrimevalStones += player.TempLoot.PrimevalStones
					if targetLoot.MaterialsInventory == nil {
						targetLoot.MaterialsInventory = make(models.MaterialsInventory)
					}
					for matID, qty := range player.TempLoot.MaterialsInventory {
						targetLoot.MaterialsInventory[matID] += qty
					}
					targetLoot.GuInventory = append(targetLoot.GuInventory, player.TempLoot.GuInventory...)

					_, _ = tx.Exec("UPDATE players SET temp_loot = $1 WHERE id = $2", targetLoot, targetUUID)
				}
			}
		}

		player.TempLoot = models.TempLoot{
			GuInventory:        models.GuInventory{},
			MaterialsInventory: models.MaterialsInventory{},
			PrimevalStones:     0,
		}

		player.CurrentRegion = "Southern_Border"
		player.X = 50
		player.Y = 50
		player.ExploredCells = models.ExploredCellsList{"50,50"}

		if player.ActiveCombat != nil && player.ActiveCombat.EnemyTemplateID == "monster_heavenly_tribulation" {
			player.ImmortalStones = player.ImmortalStones * 0.5
			player.TribulationCount = 0
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

	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update combat results"})
		return
	}
	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, primeval_stones = $3, cleared_cells = $4, active_combat = $5,
		    temp_loot = $6, current_region = $7, x = $8, y = $9, explored_cells = $10, human_qi = $11
		WHERE id = $12
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.PrimevalStones, player.ClearedCells, player.ActiveCombat,
		player.TempLoot, player.CurrentRegion, player.X, player.Y, player.ExploredCells, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update combat results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

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
