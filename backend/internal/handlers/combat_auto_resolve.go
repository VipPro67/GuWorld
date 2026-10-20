package handlers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// CombatResolution is everything a caller needs once a combat has been
// simulated and persisted: what to show the player, plus the work that must run
// only after the surrounding transaction commits.
type CombatResolution struct {
	Winner        string
	Logs          []simulator.CombatLog
	Summary       map[string]interface{}
	Notifications []notify.Notification
	Combatants    []CombatantInfo
	ApertureHP    float64
	ApertureMaxHP float64
	Turn          int
	ItemsUsed     int
	StonesSpent   float64
	AfterCommit   func()
}

// ResolveCombat simulates a combat and applies the outcome to the database. It
// owns everything from building the snapshot through the victory/defeat branch
// to persisting the result; callers own the transaction and their own
// preconditions.
//
// loadout is the loadout to simulate with; nil means use the player's saved
// combat settings. loadoutGuIDs is the explicit Gu selection for the player
// combatant (nil means "whatever the settings/enemy selection implies").
//
// winner is always derived from the simulation, never from log text.
func ResolveCombat(tx *sql.Tx, player *models.Player, loadout *simulator.CombatLoadout, loadoutGuIDs []string) (*CombatResolution, error) {
	var pendingNotifications []notify.Notification
	var publishClearedCell bool
	if player.ActiveCombat == nil {
		return nil, fmt.Errorf("no active combat")
	}
	if err := checkRegionAction(player.CurrentRegion, "combat"); err != nil {
		return nil, err
	}
	if err := checkRegionEquippedGus(player.CurrentRegion, player.GuInventory); err != nil {
		return nil, err
	}

	isVirtual := player.ActiveCombat.CombatMode == "virtual"
	isTribulation := player.ActiveCombat.TribulationEventID != ""

	// Build loadout from the player's saved combat settings when the caller did
	// not supply one.
	if loadout == nil {
		loadout = &simulator.CombatLoadout{
			AutoHealHP:      player.CombatSettings.AutoHealHP,
			HealThreshold:   player.CombatSettings.HealThreshold,
			AutoRestoreMana: player.CombatSettings.AutoRestoreMana,
			ManaThreshold:   player.CombatSettings.ManaThreshold,
		}
	}

	simState, err := buildSimCombatState(tx, player, loadout, isVirtual)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize initiator stats: %w", err)
	}
	simPlayer := simState.Allies[0]

	var combatantInfos []CombatantInfo
	for _, ally := range simState.Allies {
		cInst := CombatantInfo{
			ID:            ally.ID,
			Name:          ally.Name,
			Kind:          "ally",
			MaxHP:         ally.MaxHP,
			HP:            ally.HP,
			MaxMana:       ally.MaxMana,
			Mana:          ally.Mana,
			GridX:         ally.GridX,
			GridY:         ally.GridY,
			Shield:        ally.VirtualShield,
			Damage:        ally.BaseDmg,
			Defense:       ally.BaseDef,
			Speed:         ally.Speed,
			KindDetail:    ally.Kind,
			Rank:          ally.Rank,
			SubRank:       ally.SubRank,
			DaoMarks:      models.DaoMarksMap(ally.DaoMarks),
			PathMasteries: ally.PathMasteries,
		}
		if ally.ID == player.ID.String() {
			if loadout.NoGuSkills {
				cInst.GuInventory = nil
			} else {
				cInst.GuInventory = filterCombatGuInventory(player.GuInventory, loadoutGuIDs, player.CombatSettings)
			}
		}
		combatantInfos = append(combatantInfos, cInst)
	}
	if player.ActiveCombat != nil {
		for _, e := range player.ActiveCombat.Enemies {
			combatantInfos = append(combatantInfos, CombatantInfo{
				ID:            e.ID,
				Name:          e.Name,
				Kind:          "enemy",
				MaxHP:         e.MaxHP,
				HP:            e.HP,
				MaxMana:       e.MaxMana,
				Mana:          e.CurrentMana,
				GridX:         e.GridX,
				GridY:         e.GridY,
				Damage:        e.BaseAttack,
				Defense:       e.BaseDefense,
				Speed:         e.Speed,
				KindDetail:    e.Kind,
				Rank:          e.Rank,
				SubRank:       e.SubRank,
				DaoMarks:      e.DaoMarks,
				PathMasteries: e.PathMasteries,
				GuInventory:   e.GuInventory,
			})
		}
	}

	// Run Simulator
	simState.CaptureTurnSnapshots = true
	logs := simulator.Simulate(simState)

	formationLogs := BuildFormationLogs(simState.Allies)
	logs = append(formationLogs, logs...)

	applyLoveGuPenalties(player, logs)

	itemsUsed, stonesSpent, _, usageLogs := applySimConsumption(tx, player, simPlayer, isVirtual, simState.Turn)
	// Snapshot after consumption so finalizeAutoCombatOutcome only diffs
	// post-consumption changes (e.g. loot).
	before := append(models.GuInventory(nil), player.GuInventory...)

	logs = append(logs, usageLogs...)

	typeLog := BuildCombatTypeLog(player.ActiveCombat)
	logs = append([]simulator.CombatLog{typeLog}, logs...)

	outcome := EvaluateCombatOutcome(CombatOutcomeInput{
		SimState:      simState,
		PlayerID:      player.ID.String(),
		PlayerX:       player.X,
		PlayerY:       player.Y,
		CurrentRegion: player.CurrentRegion,
		ActiveCombat:  player.ActiveCombat,
		IsVirtual:     isVirtual,
		IsTribulation: isTribulation,
	}, logs)

	alliesAlive := outcome.AlliesAlive
	finalPlayerHP := outcome.FinalPlayerHP
	finalMana := outcome.FinalPlayerMana
	enemyName := outcome.EnemyName
	summary := make(map[string]interface{})
	for k, v := range outcome.Summary {
		summary[k] = v
	}

	var updateQuery string
	var args []interface{}
	rawEnemyType := outcome.RawEnemyType

	// Capture the battle region before the outcome branch: a defeat respawns the
	// player (applyAutoCombatDefeat mutates CurrentRegion), and combat history
	// must record where the fight happened, not the respawn hub.
	historyRegion := player.CurrentRegion

	if alliesAlive {
		if isVirtual {
			if err := InventorySvc.AddItem(tx, player.ID.String(), "arena_coin", "materials", int64(outcome.ArenaCoinsAwarded)); err != nil {
				logger.S.Errorf("Failed to award arena_coin to %s: %v", player.ID, err)
			} else {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "ARENA_REWARD",
					Detail: "Chiến thắng Võ Đài! Nhận thưởng 50 Xu Đấu Trường.",
				})
			}
			// Beating a higher-ranked opponent swaps arena ranks.
			if player.ActiveCombat != nil && player.ActiveCombat.TargetPlayerID != "" && player.ActiveCombat.TargetPlayerID != "dummy" {
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
						logs = append(logs, simulator.CombatLog{
							Turn:   simState.Turn,
							Action: "ARENA_RANK_UP",
							Detail: fmt.Sprintf("Chúc mừng! Đạo hữu đã vượt cấp khiêu chiến thành công, thăng lên Hạng %d Võ Đài!", targetRank),
						})
					}
				}
			}
			updateQuery = `UPDATE players SET active_combat = NULL WHERE id = $1`
			args = append(args, player.ID)
			summary["reward"] = "50 Xu Đấu Trường"
		} else {
			var partyMembers []uuid.UUID
			for _, ally := range player.ActiveCombat.Allies {
				if ally.Kind == "Pet" || ally.Kind == "VariantNPC" {
					continue
				}
				allyUUID, errParse := uuid.Parse(ally.ID)
				if errParse == nil {
					partyMembers = append(partyMembers, allyUUID)
				}
			}

			lootDetailLogs, lootNotifications := applyCombatVictoryLoot(tx, player, player.ActiveCombat, partyMembers, isVirtual)
			pendingNotifications = append(pendingNotifications, lootNotifications...)

			markBossCleared(tx, player)

			for _, lMsg := range lootDetailLogs {
				logs = append(logs, simulator.CombatLog{
					Turn:   simState.Turn,
					Action: "LOOT",
					Detail: lMsg,
				})
			}

			mineLogs, mineOccupied := applyAutoCombatMineOccupation(tx, player, player.ActiveCombat, simState.Turn)
			logs = append(logs, mineLogs...)
			if mineOccupied {
				summary["mine_occupied"] = player.ActiveCombat.MineSlotID
			}

			// Sect War victory. This branch only runs when alliesAlive (outer
			// branch), and alliesAlive implies every enemy is down, so the
			// attacker always won here — no log-text matching needed. The
			// attacker-defeat case is handled by applyAutoCombatDefeat.
			if player.ActiveCombat.TargetSectID != "" {
				sectWarLogs := ApplySectWarVictory(tx, player, player.ActiveCombat)
				for _, swMsg := range sectWarLogs {
					logs = append(logs, simulator.CombatLog{
						Turn:   simState.Turn,
						Action: "SECT_WAR",
						Detail: swMsg,
					})
				}
				summary["sect_war_won"] = true
			}

			if outcome.ShouldClearCell {
				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[outcome.ClearedCellKey] = time.Now().Format(time.RFC3339)
				publishClearedCell = true
			}
			// player.ClearedCells implements driver.Valuer, so it goes straight
			// into the jsonb column without a manual marshal.

			player.HP = float64(finalPlayerHP)
			updateQuery = `UPDATE players SET hp = $1, current_mana = $2, active_combat = NULL, primeval_stones = $3, immortal_stones = $4, path_masteries = $5, aptitude_percent = $6, lifespan = $7, cleared_cells = $8, version = version + 1 WHERE id = $9`
			args = append(args, player.HP, finalMana, player.PrimevalStones, player.ImmortalStones, player.PathMasteries, player.AptitudePercent, player.Lifespan, player.ClearedCells, player.ID)

			summary["loot"] = lootDetailLogs
		}
	} else {
		defUpdateQuery, defArgs, defLogs := applyAutoCombatDefeat(tx, player, simState, isVirtual, isTribulation, finalMana)
		updateQuery = defUpdateQuery
		args = append(args, defArgs...)
		logs = append(logs, defLogs...)
	}

	logs, err = finalizeAutoCombatOutcome(tx, player, simState, isVirtual, isTribulation, alliesAlive, updateQuery, args, finalPlayerHP, finalMana, before, logs)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize combat outcome: %w", err)
	}

	qiGained := AddHumanQi(player, outcome.HumanQiReward)
	_, err = tx.Exec("UPDATE players SET human_qi = $1 WHERE id = $2", player.HumanQi, player.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update human_qi: %w", err)
	}
	summary["human_qi_gained"] = qiGained

	// Cổ mang PERSIST_STATUS_ON_HIT: đòn đánh trúng ở PvP thật thì defender
	// nhận persistent status (data-driven, chung helper với AutoCombatStart).
	for _, g := range maybeGrantPersistentHitStatuses(tx, player, simState, logs) {
		if strings.EqualFold(g.StatusKey, gamefx.StatusSinhKhiTan) {
			logs = append(logs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "SINH_KHI_TAN_APPLIED",
				Detail: fmt.Sprintf("Sát khí tàn lụi xâm nhập! Kẻ địch nhận [%s] (công/thủ/tốc giảm %.0f%% trong %d giờ).",
					gamefx.SinhKhiTanTierName(g.Tier),
					-gamefx.SinhKhiTanValueForTier(g.Tier)*100, gamefx.SinhKhiTanDurationMinutes/60),
			})
		} else {
			logs = append(logs, simulator.CombatLog{
				Turn:   simState.Turn,
				Action: "PERSIST_STATUS_APPLIED",
				Detail: fmt.Sprintf("Kẻ địch trúng dư âm Cổ trùng, nhận [%s].",
					gamefx.StatusDisplayName(g.StatusKey)),
			})
		}
	}

	if fakeWallLog := maybeGrantFakeRegionalWallStatus(tx, player); fakeWallLog != nil {
		fakeWallLog.Turn = simState.Turn
		logs = append(logs, *fakeWallLog)
	}

	winner := outcome.Winner

	// Queue external effects for the caller to run only after its transaction commits.
	targetPlayerID := ""
	clearedRegion, clearedX, clearedY := player.CurrentRegion, player.X, player.Y
	clearedPlayerID := player.ID.String()
	clearedEnemyName, clearedEnemyTemplateID := "", ""
	targetSectID := ""
	var sectDefenderIDs []uuid.UUID
	if player.ActiveCombat != nil {
		targetPlayerID = player.ActiveCombat.TargetPlayerID
		clearedEnemyName = player.ActiveCombat.EnemyName
		clearedEnemyTemplateID = player.ActiveCombat.EnemyTemplateID
		targetSectID = player.ActiveCombat.TargetSectID
		if targetSectID != "" {
			for _, enemy := range player.ActiveCombat.Enemies {
				if enemy.Kind != "Player" {
					continue
				}
				if id, parseErr := uuid.Parse(enemy.ID); parseErr == nil && id != player.ID {
					sectDefenderIDs = append(sectDefenderIDs, id)
				}
			}
		}
	}
	historyPlayerID, historyUsername := player.ID, player.Username
	res := &CombatResolution{
		Winner:        winner,
		Logs:          logs,
		Summary:       summary,
		Notifications: pendingNotifications,
		Combatants:    combatantInfos,
		ApertureHP:    simState.ApertureHP,
		ApertureMaxHP: simState.ApertureMaxHP,
		Turn:          simState.Turn,
		ItemsUsed:     itemsUsed,
		StonesSpent:   stonesSpent,
	}
	res.AfterCommit = func() {
		if publishClearedCell {
			publishCombatClearedCell(clearedRegion, clearedX, clearedY, clearedPlayerID, clearedEnemyName, clearedEnemyTemplateID)
		}
		if !shouldRecordCombatHistory(rawEnemyType, winner) {
			return
		}
		go func() {
			summaryStr := buildCombatSummaryFromLogs(logs, finalPlayerHP, itemsUsed, int64(stonesSpent))
			if rawEnemyType == "PVP" {
				attackerIDs, defenderIDs := pvpHistoryPlayerIDs(historyPlayerID, simState.Allies, targetPlayerID)
				savePVPCombatHistory(attackerIDs, defenderIDs, winner, toInterfaceSlice(logs), summaryStr, combatantInfos, enemyName, historyUsername, historyRegion)
			} else {
				// PVE: boss fights and defeats, plus every sect war so both the
				// attacker and the defenders can review the run.
				SaveCombatHistory(historyPlayerID, winner, toInterfaceSlice(logs), summaryStr, combatantInfoInterfaces(combatantInfos, false), enemyName, rawEnemyType, historyRegion)
				if targetSectID != "" {
					saveSectWarDefenderHistory(sectDefenderIDs, winner, toInterfaceSlice(logs), summaryStr, combatantInfos, historyUsername, historyRegion)
				}
			}
		}()
	}

	summary["winner"] = winner
	summary["enemy_name"] = enemyName
	summary["enemy_type"] = rawEnemyType
	allySummary := make([]map[string]interface{}, 0, len(simState.Allies))
	for _, a := range simState.Allies {
		allySummary = append(allySummary, map[string]interface{}{
			"id": a.ID, "name": a.Name, "kind": a.Kind,
			"hp": a.HP, "max_hp": a.MaxHP,
		})
	}
	summary["allies"] = allySummary

	player.ActiveCombat = nil
	return res, nil
}

// ── helpers ────────────────────────────────────────────────────────────

func buildSimActionFromGu(actor *models.Combatant, spec *models.GuDictionaryItem, actionType string) (simulator.SimAction, error) {
	// Mortal users cannot use immortal Gu that cost mana (skip the action).
	if !gamefx.CanUseGuWithMana(actor.Rank, spec) {
		return simulator.SimAction{}, fmt.Errorf("Tu vi %s chưa đủ để sử dụng Tiên Cổ %s (Rank %d) có hao phí!", models.GetRankName(actor.Rank), spec.Name, spec.Rank)
	}

	dmg, def, cost, err := CalculateGuEffectForCombatant(actor, spec, false)
	if err != nil {
		return simulator.SimAction{
			ActionType: actionType,
			Name:       spec.Name,
			Path:       spec.Path,
			Rank:       spec.Rank,
			ManaCost:   cost,
		}, nil
	}

	scale, reason := GetGuAttainmentScale(actor, spec)
	act := simulator.SimAction{
		ActionType: actionType,
		Name:       spec.Name,
		SourceID:   spec.ID,
		Path:       spec.Path,
		Rank:       spec.Rank,
		Damage:     dmg,
		DamageMult: scale,
		Defense:    def,
		ShieldMult: scale,
		Reason:     reason,
		ManaCost:   cost,
	}
	if compiled := spec.Effects.Compiled(); compiled != nil {
		act.CompiledOperations = compiled.CompileCombatAction().Operations
	}
	act.Mechanics = append(act.Mechanics, combatMechanicsForSpec(*spec, true)...)
	resolveSummonMonsterStats(&act)
	if statusKey, pvpOnly, hasPersistent := models.GuPersistentHitStatus(*spec); hasPersistent {
		act.PersistentStatusKey = statusKey
		act.PersistentPvPOnly = pvpOnly
	}

	if actor.Rank < spec.Rank {
		act.BacklashDamage = gamefx.HigherRankHPCost(spec.Rank, actor.Rank, actor.MaxHP)
	}

	if spec.Effects.Active != nil {
		active := spec.Effects.Active
		if active.DamagePath != "" {
			act.Path = []string{active.DamagePath}
		}
		if active.StatBaseAttackFlat > 0 {
			act.Damage += active.StatBaseAttackFlat
		}
		if active.DamageHPPercent > 0 {
			act.DamageHPPercent = active.DamageHPPercent
		}
		if active.ShieldFlatReducePercent > 0 {
			act.ShieldStripPercent = active.ShieldFlatReducePercent
		}
		if active.ForgetRealityChance > 0 {
			act.ProcChance = active.ForgetRealityChance
		}
		if active.TargetDamageReducePct > 0 {
			act.TargetWeakPct = active.TargetDamageReducePct
		}
		if active.DisableGuNextTurn {
			act.DisableGuTurns = 1
		}
		if active.TargetDamageReducePct > 0 || active.DisableGuNextTurn {
			act.BonusTurns = active.StatusDurationTicks
			if act.BonusTurns <= 0 {
				act.BonusTurns = 2
			}
		}
		if active.ShieldMult > 0 {
			// Cộng dồn bonus: act.ShieldMult (đã gồm attainment) + (value-1).
			act.ShieldMult += active.ShieldMult - 1.0
		}
		// Registry-driven statuses (BLEED/POISON/BURN/FEAR/CONFUSE/STEALTH/
		// SLEEP/BLIND/CHARM/BIND + catalog). Entries with chance opt into the
		// rank-scaled proc (chance +0.05/rank, +(rank-1)/3 turns).
		for _, op := range active.ResolveStatusApplicationsForRank(spec.Rank) {
			act.StatusOps = append(act.StatusOps, op)
		}
		if active.LifeStealRate > 0 {
			act.LifeStealPct += active.LifeStealRate
		}
		if active.HealFlat > 0 {
			act.Healing += active.HealFlat
		}
		if active.BloodDrain {
			act.LifeStealPct += 0.5
		}
		if active.SummonPhantom {
			act.PhantomDmg += actor.BaseAttack
		}
		if active.RefineManaTier > 0 {
			boost := active.RefineManaTier * 0.20
			act.Damage += act.Damage * boost
		}
		if active.ManaRegen > 0 {
			act.ManaRestore += active.ManaRegen
		}
		if active.CleansingPower > 0 {
			act.CleanseDebuffs = true
		}
		if active.WoodTransformation {
			act.Defense += 30.0
		}
		if active.DamageMult > 0 {
			// Cộng dồn bonus: act.DamageMult (đã gồm attainment) + (value-1).
			act.DamageMult += active.DamageMult - 1.0
		}
		if active.TargetSelector != "" {
			act.TargetType = active.TargetSelector
			act.TargetX = actor.GridX
			act.TargetY = actor.GridY
		}
		act.Bypass = active.CurveAttack
	} else if len(spec.Effects.Blocks) > 0 {
		for _, block := range spec.Effects.Blocks {
			if block.Trigger == gameplayeffects.TriggerOnAction {
				if block.Target.Selector != "" {
					act.TargetType = string(block.Target.Selector)
					act.TargetX = actor.GridX
					act.TargetY = actor.GridY
				}
				if block.Target.BypassFrontline {
					act.Bypass = true
				}
				for _, op := range block.Operations {
					if op.Target != nil && op.Target.Selector != "" {
						if !strings.EqualFold(string(op.Target.Selector), "SELF") || act.TargetType == "" {
							act.TargetType = string(op.Target.Selector)
						}
						if op.Target.BypassFrontline {
							act.Bypass = true
						}
					}
					switch op.Op {
					case gameplayeffects.OpDamage:
						if op.Path != "" {
							act.Path = []string{op.Path}
						}
						if strings.EqualFold(op.SubType, "MULTIPLIER") && op.Value > 0 {
							act.DamageMult += op.Value - 1.0
						} else if strings.EqualFold(op.SubType, "MAX_HP_PERCENT") && op.Value > 0 {
							if op.Value <= 1.0 {
								act.DamageHPPercent = op.Value
							} else {
								act.DamageHPPercent = op.Value / 100.0
							}
						}
					case gameplayeffects.OpShield:
						if strings.EqualFold(op.SubType, "MULTIPLIER") && op.Value > 0 {
							act.ShieldMult += op.Value - 1.0
						}
					case gameplayeffects.OpHeal:
						if strings.EqualFold(op.SubType, "LIFESTEAL") {
							if op.Value > 1.0 {
								act.LifeStealPct += op.Value / 100.0
							} else {
								act.LifeStealPct += op.Value
							}
						} else if (strings.EqualFold(op.SubType, "RESOURCE") && strings.EqualFold(op.Resource, "MANA")) || strings.EqualFold(op.SubType, "MANA") {
							act.ManaRestore += op.Value
						} else if strings.EqualFold(op.SubType, "PERCENT") {
							if op.Value <= 1.0 {
								act.Healing += actor.MaxHP * op.Value
							} else {
								act.Healing += actor.MaxHP * (op.Value / 100.0)
							}
						} else if strings.EqualFold(op.SubType, "FLAT") || op.SubType == "" {
							act.Healing += op.Value
						}
					case gameplayeffects.OpDirectHP:
						if strings.EqualFold(op.SubType, "MAX_HP_PERCENT") {
							if op.Value <= 1.0 {
								act.DamageHPPercent = op.Value
							} else {
								act.DamageHPPercent = op.Value / 100.0
							}
						} else if strings.EqualFold(op.SubType, "SHIELD_STRIP") {
							if op.Value <= 1.0 {
								act.ShieldStripPercent = op.Value
							} else {
								act.ShieldStripPercent = op.Value / 100.0
							}
						}
					case gameplayeffects.OpBarrier:
						if strings.EqualFold(op.SubType, "PERCENT") {
							if op.Value <= 1.0 {
								act.Defense += actor.MaxHP * op.Value
							} else {
								act.Defense += actor.MaxHP * (op.Value / 100.0)
							}
						} else {
							act.Defense += op.Value
						}
					case gameplayeffects.OpCleanse:
						act.CleanseDebuffs = true
					case gameplayeffects.OpStatus:
						turns := op.Turns
						if turns <= 0 {
							turns = 1
						}
						chance := op.Chance
						if chance <= 0 {
							chance = 1.0
						}
						if op.StatusID != "" {
							if resolved, err := statusfx.ResolveStatus(op.StatusID, op.Value, turns); err == nil {
								if chance > 0 && spec.Rank > 0 {
									resolved.Chance = chance + 0.05*float64(spec.Rank-1)
									if resolved.Chance > 0.95 {
										resolved.Chance = 0.95
									}
									resolved.Turns += (spec.Rank - 1) / 3
								} else {
									resolved.Chance = chance
								}
								act.StatusOps = append(act.StatusOps, resolved)
							}
						}
					case gameplayeffects.OpStatMod:
						switch strings.ToUpper(op.Stat) {
						case "ATTACK":
							act.Damage += op.Value
						case "DEFENSE":
							act.Defense += op.Value
						}
					case gameplayeffects.OpUnique:
						switch strings.ToUpper(op.Code) {
						case "SUMMON_PHANTOM":
							act.PhantomDmg += actor.BaseAttack
						case "WOOD_TRANSFORMATION":
							act.Defense += 30.0
						case "BYPASS_SHIELD":
							act.Bypass = true
						case "GUARANTEED_HIT":
							act.Mechanics = append(act.Mechanics, simulator.CombatMechanic{Trigger: "ON_HIT", ActionCode: "GUARANTEED_HIT"})
						}
					}
				}
			}
		}
	}

	if act.TargetType == "" {
		act.TargetType = "SINGLE"
	}

	return act, nil
}

// hasCombatActiveEffects returns true if the Gu's active effects produce a meaningful combat action.
func hasCombatActiveEffects(spec *models.GuDictionaryItem) bool {
	for _, mechanic := range spec.Effects.Other {
		if mechanic.EventTrigger == "ON_CAST" || mechanic.EventTrigger == "ON_HIT" {
			return true
		}
	}
	if spec.Effects.Active != nil {
		active := spec.Effects.Active
		if active.DamageFlat > 0 || active.DamageMult > 0 || active.ShieldFlat > 0 || active.ShieldMult > 0 ||
			len(active.ApplyStatus) > 0 || active.LifeStealRate > 0 || active.HealFlat > 0 ||
			active.BloodDrain || active.SummonPhantom || active.RefineManaTier > 0 ||
			active.WoodTransformation || active.ManaRegen > 0 || active.CleansingPower > 0 ||
			active.SummonGrowthPhantoms || active.DamageHPPercent > 0 || active.ShieldFlatReducePercent > 0 ||
			active.DisableGuNextTurn || active.ForgetRealityChance > 0 ||
			active.TargetDamageReducePct > 0 || active.StatBaseAttackFlat > 0 || active.DamagePath != "" {
			return true
		}
	}
	for _, block := range spec.Effects.Blocks {
		if block.Trigger == gameplayeffects.TriggerOnAction && len(block.Operations) > 0 {
			for _, op := range block.Operations {
				if op.Op == gameplayeffects.OpDamage || op.Op == gameplayeffects.OpShield || op.Op == gameplayeffects.OpHeal ||
					op.Op == gameplayeffects.OpDirectHP || op.Op == gameplayeffects.OpStatus ||
					op.Op == gameplayeffects.OpBarrier || op.Op == gameplayeffects.OpCleanse ||
					(op.Op == gameplayeffects.OpStatMod && (strings.ToUpper(op.Stat) == "ATTACK" || strings.ToUpper(op.Stat) == "DEFENSE")) ||
					op.Op == gameplayeffects.OpUnique {
					return true
				}
			}
		}
		if block.Trigger == gameplayeffects.TriggerOnCast || block.Trigger == gameplayeffects.TriggerOnHit {
			return true
		}
	}
	return false
}

// hasCombatEffects returns true if the Gu has any active action, passive mechanic, or combat-relevant passive stat.
func hasCombatEffects(spec *models.GuDictionaryItem) bool {
	if hasCombatActiveEffects(spec) {
		return true
	}
	if spec.Effects.Active != nil {
		active := spec.Effects.Active
		if active.SpeedBoostFlat > 0 || active.CritRateBoost > 0 || active.IgnoreDefense > 0 || active.IgnoreDefensePct > 0 || active.AllStatsBuffPct > 0 {
			return true
		}
	}
	for _, mechanic := range spec.Effects.Other {
		if mechanic.EventTrigger != "ON_CAST" && mechanic.EventTrigger != "ON_HIT" {
			return true
		}
	}
	if models.GuStatValue(*spec, "ATTACK") > 0 || models.GuStatValue(*spec, "DEFENSE") > 0 ||
		models.GuStatValue(*spec, "HP") > 0 || models.GuStatValue(*spec, "SPEED") > 0 ||
		models.GuStatValueBySubtype(*spec, "DAMAGE_DEALT", "MULTIPLY_PERCENT") > 0 ||
		models.GuStatValue(*spec, "PATH_CONFLICT_REDUCTION") > 0 ||
		len(models.GuPathResistance(*spec)) > 0 || models.GuImmuneDream(*spec) {
		return true
	}
	for _, block := range spec.Effects.Blocks {
		if block.Trigger == gameplayeffects.TriggerBeforeHit || block.Trigger == gameplayeffects.TriggerOnTurnStart ||
			block.Trigger == gameplayeffects.TriggerOnDeath || block.Trigger == gameplayeffects.TriggerCombatPassive ||
			block.Trigger == gameplayeffects.TriggerOnBattleStart || block.Trigger == gameplayeffects.TriggerBeforeEnemyAttack {
			return true
		}
		if block.Trigger == gameplayeffects.TriggerAlwaysOn {
			for _, op := range block.Operations {
				if op.Op == gameplayeffects.OpStatMod {
					stat := strings.ToUpper(op.Stat)
					if stat == "ATTACK" || stat == "DEFENSE" || stat == "HP" || stat == "SPEED" ||
						stat == "DAMAGE_DEALT" || stat == "PATH_CONFLICT_REDUCTION" ||
						stat == "CRIT_RATE" || stat == "ALL" {
						return true
					}
				}
			}
		}
	}
	return false
}

// computeGlobalIgnoreDefense aggregates ignore_defense from all equipped active Gu (additive, capped at 0.8).
func computeGlobalIgnoreDefense(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) (flat float64, pct float64) {
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok {
			continue
		}
		if spec.Effects.Active != nil {
			flat += spec.Effects.Active.IgnoreDefense
			pct += spec.Effects.Active.IgnoreDefensePct
		}
	}
	if pct > 0.8 {
		pct = 0.8
	}
	return flat, pct
}

// computeGlobalCritRate aggregates crit_rate_boost from all equipped active Gu (additive, capped at 0.8).
func computeGlobalCritRate(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok {
			continue
		}
		if spec.Effects.Active != nil {
			total += spec.Effects.Active.CritRateBoost
		} else {
			for _, block := range spec.Effects.Blocks {
				if block.Trigger == gameplayeffects.TriggerAlwaysOn || block.Trigger == gameplayeffects.TriggerOnAction {
					for _, op := range block.Operations {
						if op.Op == gameplayeffects.OpStatMod && strings.ToUpper(op.Stat) == "CRIT_RATE" {
							total += op.Value
						}
					}
				}
			}
		}
	}
	if total > 0.8 {
		total = 0.8
	}
	return total
}

// computeGlobalAllStatsBuff aggregates all_stats_buff_percent from all equipped active Gu (additive, capped at 0.5).
func computeGlobalAllStatsBuff(selectedGuIDs []string, resolveSpec func(string) (models.GuDictionaryItem, bool)) float64 {
	total := 0.0
	seen := make(map[string]struct{}, len(selectedGuIDs))
	for _, guID := range selectedGuIDs {
		if _, ok := seen[guID]; ok {
			continue
		}
		seen[guID] = struct{}{}
		spec, ok := resolveSpec(guID)
		if !ok {
			continue
		}
		if spec.Effects.Active != nil {
			total += spec.Effects.Active.AllStatsBuffPct
		} else {
			for _, block := range spec.Effects.Blocks {
				if block.Trigger == gameplayeffects.TriggerAlwaysOn || block.Trigger == gameplayeffects.TriggerOnAction {
					for _, op := range block.Operations {
						if op.Op == gameplayeffects.OpStatMod && strings.ToUpper(op.Stat) == "ALL" {
							total += op.Value
						}
					}
				}
			}
		}
	}
	if total > 0.5 {
		total = 0.5
	}
	return total
}

// resolveSummonMonsterMechanics injects the monster template stats referenced by
// a SUMMON_MONSTER mechanic into the compiled mechanic values, so the simulator
// never needs the monster registry. Both the per-turn ON_CAST form and the
// battle-start ON_BATTLE_START form resolve here. Unknown template ids are
// dropped with a warning instead of spawning an empty minion.
func resolveSummonMonsterMechanics(mechanics []simulator.CombatMechanic) []simulator.CombatMechanic {
	if len(mechanics) == 0 {
		return mechanics
	}
	out := mechanics[:0]
	for _, m := range mechanics {
		if m.ActionCode != "SUMMON_MONSTER" || (m.Trigger != "ON_CAST" && m.Trigger != "ON_BATTLE_START") {
			out = append(out, m)
			continue
		}
		id, _ := m.Values["monster_id"].(string)
		tpl, ok := models.MonsterRegistry[id]
		if !ok || tpl.MaxHP <= 0 {
			logger.S.Warnf("SUMMON_MONSTER: unknown monster template %q", id)
			continue
		}
		if m.Values == nil {
			m.Values = map[string]any{}
		}
		m.Values["hp"] = tpl.MaxHP
		m.Values["damage"] = tpl.BaseAttack
		m.Values["defense"] = tpl.BaseDefense
		m.Values["name"] = tpl.Name
		out = append(out, m)
	}
	return out
}

// resolveSummonMonsterStats keeps the SimAction signature used by the builder.
func resolveSummonMonsterStats(act *simulator.SimAction) {
	act.Mechanics = resolveSummonMonsterMechanics(act.Mechanics)
}

// combatMechanicsForSpec is the migration adapter. Registry-loaded definitions
// always take the compiled path; the legacy fallback supports synthetic specs in
// tests and can be removed with the legacy effect fields after parity. Source
// rank/paths are stamped onto each mechanic so battle-start hooks (summons) can
// scale without a SimAction, and SUMMON_MONSTER templates are resolved eagerly.
func combatMechanicsForSpec(spec models.GuDictionaryItem, action bool) []simulator.CombatMechanic {
	var result []simulator.CombatMechanic
	if compiled := spec.Effects.Compiled(); compiled != nil {
		result = simulator.CompiledCombatMechanics(compiled, action)
	} else {
		for _, mechanic := range spec.Effects.Other {
			isAction := mechanic.EventTrigger == "ON_CAST" || mechanic.EventTrigger == "ON_HIT"
			if isAction == action {
				result = append(result, simulator.NewCombatMechanic(mechanic.EventTrigger, mechanic.ActionCode, mechanic.Params))
			}
		}
	}
	for i := range result {
		result[i].Rank = spec.Rank
		result[i].Path = spec.Path
	}
	return resolveSummonMonsterMechanics(result)
}

// battleStartSummonMechanics returns only the ON_BATTLE_START summon hooks of a
// Gu. Arena builders intentionally do not load passive mechanics, but summoning
// Gu used to contribute through their per-turn action; this keeps the
// battle-start summon working there without pulling in unrelated passives.
func battleStartSummonMechanics(spec models.GuDictionaryItem) []simulator.CombatMechanic {
	var result []simulator.CombatMechanic
	for _, m := range combatMechanicsForSpec(spec, false) {
		if m.Trigger == "ON_BATTLE_START" && (m.ActionCode == "SUMMON_MONSTER" || m.ActionCode == "SUMMON_WOLF_FRONTLINE") {
			result = append(result, m)
		}
	}
	return result
}
