package handlers

import (
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// CombatOutcome represents the pure, deterministic consequences of a combat simulation.
// It separates gameplay consequence calculations from database mutation adapters,
// enabling full unit testing of battle rules without a live database transaction.
type CombatOutcome struct {
	Winner               string
	AlliesAlive          bool
	EnemiesAlive         bool
	FinalPlayerHP        float64
	FinalPlayerMana      float64
	HumanQiReward        float64
	ShouldClearCell      bool
	ClearedCellKey       string
	ArenaCoinsAwarded    int
	ArenaSwapEligible    bool
	ArenaSwapTargetID    string
	IsVirtual            bool
	IsTribulation        bool
	TargetSectID         string
	MineSlotID           int
	RawEnemyType         string
	EnemyName            string
	EnemyTemplateID      string
	Summary              map[string]interface{}
	Logs                 []simulator.CombatLog
	PendingNotifications []notify.Notification
}

// CombatOutcomeInput contains the read-only inputs required to evaluate combat consequences.
type CombatOutcomeInput struct {
	SimState      *simulator.CombatState
	PlayerID      string
	PlayerX       int
	PlayerY       int
	CurrentRegion string
	ActiveCombat  *models.CombatState
	IsVirtual     bool
	IsTribulation bool
}

// EvaluateCombatOutcome computes all gameplay consequences from a completed simulator run.
// It is pure and does not perform I/O, SQL, or network operations.
func EvaluateCombatOutcome(input CombatOutcomeInput, initialLogs []simulator.CombatLog) *CombatOutcome {
	enemiesAlive, alliesAlive, finalPlayerHP := combatOutcome(input.SimState, input.PlayerID)

	winner := "Phe Địch"
	if alliesAlive {
		winner = "Phe Ta"
	}

	var finalMana float64
	for _, a := range input.SimState.Allies {
		if a.ID == input.PlayerID {
			finalMana = a.Mana
			break
		}
	}

	summary := make(map[string]interface{})
	logs := append([]simulator.CombatLog(nil), initialLogs...)

	var arenaCoins int
	var arenaSwapEligible bool
	var arenaSwapTargetID string
	var shouldClearCell bool
	var clearedCellKey string

	enemyName := ""
	rawEnemyType := ""
	targetSectID := ""
	mineSlotID := 0
	enemyTemplateID := ""
	enemyRank := 1

	if input.ActiveCombat != nil {
		enemyName = input.ActiveCombat.EnemyName
		rawEnemyType = input.ActiveCombat.EnemyType
		targetSectID = input.ActiveCombat.TargetSectID
		mineSlotID = input.ActiveCombat.MineSlotID
		enemyTemplateID = input.ActiveCombat.EnemyTemplateID
		enemyRank = input.ActiveCombat.EnemyRank
	}

	if alliesAlive {
		if input.IsVirtual {
			arenaCoins = 50
			summary["reward"] = "50 Xu Đấu Trường"
			if input.ActiveCombat != nil && input.ActiveCombat.TargetPlayerID != "" && input.ActiveCombat.TargetPlayerID != "dummy" {
				arenaSwapEligible = true
				arenaSwapTargetID = input.ActiveCombat.TargetPlayerID
			}
		} else {
			if !input.IsTribulation && targetSectID == "" {
				shouldClearCell = true
				clearedCellKey = fmt.Sprintf("%d,%d", input.PlayerX, input.PlayerY)
			}
		}
	} else {
		if input.IsVirtual {
			arenaCoins = 1
		}
	}

	qiReward := 2.0
	if alliesAlive {
		qiReward = combatHumanQiReward(enemyRank)
	}

	return &CombatOutcome{
		Winner:            winner,
		AlliesAlive:       alliesAlive,
		EnemiesAlive:      enemiesAlive,
		FinalPlayerHP:     finalPlayerHP,
		FinalPlayerMana:   finalMana,
		HumanQiReward:     qiReward,
		ShouldClearCell:   shouldClearCell,
		ClearedCellKey:    clearedCellKey,
		ArenaCoinsAwarded: arenaCoins,
		ArenaSwapEligible: arenaSwapEligible,
		ArenaSwapTargetID: arenaSwapTargetID,
		IsVirtual:         input.IsVirtual,
		IsTribulation:     input.IsTribulation,
		TargetSectID:      targetSectID,
		MineSlotID:        mineSlotID,
		RawEnemyType:      rawEnemyType,
		EnemyName:         enemyName,
		EnemyTemplateID:   enemyTemplateID,
		Summary:           summary,
		Logs:              logs,
	}
}

// BuildFormationLogs produces the standard Turn 0 formation announcement log.
func BuildFormationLogs(allies []*simulator.CombatantAuto) []simulator.CombatLog {
	allyParts := make([]string, 0, len(allies))
	for _, a := range allies {
		allyParts = append(allyParts, fmt.Sprintf("%s (%s)", a.Name, a.Kind))
	}
	return []simulator.CombatLog{{
		Turn:   0,
		Action: "FORMATION",
		Detail: fmt.Sprintf("Phe ta xuất trận (%d): %s", len(allyParts), strings.Join(allyParts, ", ")),
	}}
}

// BuildCombatTypeLog produces the standard Turn 0 combat type header log.
func BuildCombatTypeLog(activeCombat *models.CombatState) simulator.CombatLog {
	label := combatTypeLabel(activeCombat)
	return simulator.CombatLog{
		Turn:   0,
		Action: "COMBAT_TYPE",
		Detail: fmt.Sprintf("=== %s ===", label),
	}
}
