package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func playerRankScaledDebuff(player *models.Player) (effectcatalog.RankScaledDebuffParams, bool) {
	if player == nil {
		return effectcatalog.RankScaledDebuffParams{}, false
	}
	for _, gu := range player.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" || gu.Hunger <= 0 {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			spec, ok = models.GetGuAny(gu.GuID)
		}
		if !ok {
			continue
		}
		compiled := models.CompiledGuEffects(spec)
		if compiled == nil {
			continue
		}
		for _, operation := range compiled.Operations(gameplayeffects.TriggerOnBattleStart) {
			if operation.Type != gameplayeffects.OpRankScaledDebuff {
				continue
			}
			params, err := effectcatalog.ParseRankScaledDebuffParams(operation.SubType, operation.Params)
			if err == nil {
				return params, true
			}
		}
	}
	return effectcatalog.RankScaledDebuffParams{}, false
}

func rankScaledDebuffValues(params effectcatalog.RankScaledDebuffParams, rank, subRank int) (float64, time.Duration) {
	if rank < 1 {
		rank = 1
	}
	if subRank < 1 {
		subRank = 1
	}
	pct, _ := gamefx.RegionalWallDebuff(rank, subRank)
	minutes := params.DurationBaseMinutes + (rank-1)*params.DurationPerRankMinutes
	if minutes < params.DurationBaseMinutes {
		minutes = params.DurationBaseMinutes
	}
	return pct * params.StrengthFactor, time.Duration(minutes) * time.Minute
}

func applyPlayerBattleStartStatusOperation(player *models.Player, op gameplayeffects.CompiledOperation) error {
	if player == nil || op.Type != gameplayeffects.OpRevokeStatus || op.Trigger != gameplayeffects.TriggerOnBattleStart {
		return nil
	}
	if !gamefx.ValidStatuses()[op.StatusID] {
		return fmt.Errorf("REVOKE_STATUS references unregistered player status %q", op.StatusID)
	}
	player.Statuses.Remove(op.StatusID)
	return nil
}

func persistBattleStartStatuses(q execQuerier, player *models.Player) error {
	if q == nil || player == nil {
		return nil
	}
	encoded, err := player.Statuses.Value()
	if err != nil {
		return fmt.Errorf("encode player statuses after battle-start effects: %w", err)
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, encoded, player.ID); err != nil {
		return fmt.Errorf("persist player statuses after battle-start effects: %w", err)
	}
	return nil
}

func applyBattleStartStatusEffects(tx *sql.Tx, player *models.Player, isVirtual bool) error {
	if player == nil {
		return nil
	}
	statusCount := len(player.Statuses)
	for _, gu := range player.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" || gu.Hunger <= 0 {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			spec, ok = models.GetGuAny(gu.GuID)
		}
		if !ok {
			continue
		}
		compiled := models.CompiledGuEffects(spec)
		for _, op := range compiled.AllOperations() {
			if err := applyPlayerBattleStartStatusOperation(player, op); err != nil {
				return err
			}
		}
		for _, mechanic := range compiled.NativeMechanics() {
			if mechanic.Trigger == gameplayeffects.TriggerOnBattleStart && mechanic.Code == "CLEANSE_REGIONAL_WALL" {
				player.Statuses.Remove("regional_wall_suppression")
			}
		}
	}
	if !isVirtual && tx != nil && len(player.Statuses) != statusCount {
		if err := persistBattleStartStatuses(tx, player); err != nil {
			return err
		}
	}
	return nil
}
