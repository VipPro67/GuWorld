package handlers

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func worldUseOperationAllowed(op gameplayeffects.Operation, target gameplayeffects.Target) bool {
	if op.StackPolicy != "" {
		return false
	}
	if target.Selector != gameplayeffects.TargetSelf && target.Selector != "" {
		return false
	}
	if op.Target != nil && op.Target.Selector != gameplayeffects.TargetSelf && op.Target.Selector != "" {
		return false
	}
	if op.Op == gameplayeffects.OpStatus {
		if op.Turns < 0 || op.Turns > 24*60 || op.Chance < 0 || op.Chance > 1 || math.IsNaN(op.Chance) || math.IsInf(op.Chance, 0) || (op.Chance > 0 && op.Chance < 1) || math.IsNaN(op.Value) || math.IsInf(op.Value, 0) {
			return false
		}
		switch strings.ToUpper(op.StatusID) {
		case statusfx.StatusFlying, statusfx.StatusHaste:
			return op.Value >= 0 && op.Value <= 100
		case statusfx.StatusStealth:
			return op.Value >= 0 && op.Value <= 100
		default:
			return false
		}
	}
	if op.Op != gameplayeffects.OpHeal || op.Value <= 0 || math.IsNaN(op.Value) || math.IsInf(op.Value, 0) {
		return false
	}
	subType := strings.ToUpper(strings.TrimSpace(op.SubType))
	switch subType {
	case "", "FLAT", "PERCENT":
		return op.Resource == ""
	case "RESOURCE":
		return strings.EqualFold(op.Resource, "MANA")
	default:
		return false
	}
}

// validateWorldUseCommand is a pure preflight. Validate the entire command list
// before applying its first mutation so an unsupported later command cannot
// leave an earlier heal/status partially committed.
func validateWorldUseCommand(command gameplayeffects.Command, rank int) error {
	if command.Target != gameplayeffects.TargetSelf || command.Domain != gameplayeffects.DomainCombat || command.StackPolicy != "" {
		return fmt.Errorf("ON_USE adapter only supports self-target combat commands")
	}
	if command.Operation == gameplayeffects.OpStatus {
		status := command.Status
		if status == nil || math.IsNaN(status.Chance) || math.IsInf(status.Chance, 0) || (status.Chance != 0 && status.Chance != 1) ||
			status.Turns < 0 || status.Turns > 24*60 || math.IsNaN(status.Value) || math.IsInf(status.Value, 0) || status.Value < 0 || status.Value > 100 {
			return fmt.Errorf("ON_USE adapter does not support status %q", worldUseStatusID(status))
		}
		durationMinutes := status.Turns
		if durationMinutes <= 0 {
			durationMinutes = rank
		}
		if durationMinutes < 1 {
			durationMinutes = 1
		}
		if durationMinutes > 24*60 {
			return fmt.Errorf("ON_USE status duration is outside world limits")
		}
		switch strings.ToUpper(status.ID) {
		case statusfx.StatusFlying:
			return nil
		case statusfx.StatusHaste, statusfx.StatusStealth:
			if status.Value <= 0 && rank > 100 {
				return fmt.Errorf("ON_USE status fallback value exceeds world limits")
			}
			return nil
		default:
			return fmt.Errorf("ON_USE adapter does not support status %q", status.ID)
		}
	}
	if command.Operation != gameplayeffects.OpHeal || command.Value <= 0 || math.IsNaN(command.Value) || math.IsInf(command.Value, 0) {
		return fmt.Errorf("ON_USE adapter received an unsupported effect command")
	}
	switch strings.ToUpper(command.SubType) {
	case "", "FLAT", "PERCENT":
		if command.Resource != "" {
			return fmt.Errorf("ON_USE adapter does not support heal resource %q", command.Resource)
		}
	case "RESOURCE":
		if !strings.EqualFold(command.Resource, "MANA") {
			return fmt.Errorf("ON_USE adapter does not support resource %q", command.Resource)
		}
	default:
		return fmt.Errorf("ON_USE adapter does not support heal subtype %q", command.SubType)
	}
	return nil
}

// applyWorldUseCommands owns world mutations for the supported ON_USE
// subset: self HP/mana recovery, flight, temporary world concealment, and haste.
func applyWorldUseCommands(player *models.Player, commands []gameplayeffects.Command, rank int, source string) (float64, float64, int, int, int, error) {
	var healedHP, restoredMana float64
	flightDurationMinutes := 0
	stealthConcealment := 0
	hasteDurationMinutes := 0
	for _, command := range commands {
		if err := validateWorldUseCommand(command, rank); err != nil {
			return 0, 0, 0, 0, 0, err
		}
	}
	for _, command := range commands {
		if command.Target != gameplayeffects.TargetSelf {
			return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter only supports self-target commands")
		}
		if command.Operation == gameplayeffects.OpStatus {
			if command.Status == nil || command.Status.Chance < 0 || command.Status.Chance > 1 || math.IsNaN(command.Status.Chance) || math.IsInf(command.Status.Chance, 0) || (command.Status.Chance > 0 && command.Status.Chance < 1) {
				return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter does not support status %q", worldUseStatusID(command.Status))
			}
			durationMinutes := command.Status.Turns
			if durationMinutes <= 0 {
				durationMinutes = rank
			}
			if durationMinutes < 1 {
				durationMinutes = 1
			}
			if durationMinutes > 24*60 || math.IsNaN(command.Status.Value) || math.IsInf(command.Status.Value, 0) {
				return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE status duration or value is outside world limits")
			}
			switch strings.ToUpper(command.Status.ID) {
			case statusfx.StatusFlying:
				player.Statuses.Add(StatusFlying, source, time.Duration(durationMinutes)*time.Minute, command.Status.Value)
				flightDurationMinutes = durationMinutes
			case statusfx.StatusHaste:
				value := command.Status.Value
				if value <= 0 {
					value = float64(rank)
				}
				if value > 100 {
					return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE haste value exceeds world speed cap")
				}
				player.Statuses.Add(statusfx.StatusHaste, source, time.Duration(durationMinutes)*time.Minute, value)
				hasteDurationMinutes = durationMinutes
			case statusfx.StatusStealth:
				value := command.Status.Value
				if value <= 0 {
					value = float64(rank)
				}
				if value > 100 {
					return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE stealth value exceeds world concealment cap")
				}
				stealthConcealment = int(math.Round(value))
				player.Statuses.Add(statusfx.StatusStealth, source, time.Duration(durationMinutes)*time.Minute, float64(stealthConcealment))
			default:
				return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter does not support status %q", command.Status.ID)
			}
			continue
		}
		if command.Domain != gameplayeffects.DomainCombat || command.Operation != gameplayeffects.OpHeal || command.Value <= 0 || math.IsNaN(command.Value) || math.IsInf(command.Value, 0) {
			return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter received an unsupported effect command")
		}
		switch strings.ToUpper(command.SubType) {
		case "", "FLAT":
			if command.Resource != "" {
				return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter does not support heal resource %q", command.Resource)
			}
			before := player.HP
			player.HP = math.Min(gamefx.EffectiveMaxHP(player), player.HP+command.Value)
			healedHP += player.HP - before
		case "PERCENT":
			if command.Resource != "" {
				return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter does not support heal resource %q", command.Resource)
			}
			amount := command.Value
			if amount > 1 {
				amount /= 100
			}
			before := player.HP
			player.HP = math.Min(gamefx.EffectiveMaxHP(player), player.HP+amount*gamefx.EffectiveMaxHP(player))
			healedHP += player.HP - before
		case "RESOURCE":
			if !strings.EqualFold(command.Resource, "MANA") {
				return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter does not support resource %q", command.Resource)
			}
			before := player.CurrentMana
			player.CurrentMana = math.Min(gamefx.EffectiveMaxMana(player), player.CurrentMana+command.Value)
			restoredMana += player.CurrentMana - before
		default:
			return 0, 0, 0, 0, 0, fmt.Errorf("ON_USE adapter does not support heal subtype %q", command.SubType)
		}
	}
	return healedHP, restoredMana, flightDurationMinutes, stealthConcealment, hasteDurationMinutes, nil
}

func worldUseStatusID(status *statusfx.StatusOp) string {
	if status == nil {
		return ""
	}
	return status.ID
}
