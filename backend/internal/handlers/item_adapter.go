package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

var itemNativeActionToKey = map[string]string{
	"RESTORE_HUNGER":              "hunger",
	"RESTORE_HP":                  "restore_hp",
	"RESTORE_MANA":                "restore_mana",
	"INCREASE_HUMAN_QI":           "increase_human_qi",
	"HEAL_APERTURE":               "heal_aperture",
	"INCREASE_APERTURE_AREA":      "increase_aperture_area",
	"INCREASE_APERTURE_TIME_RATE": "increase_aperture_time_rate",
	"RANDOM_CONSTITUTION":         "random_constitution",
	"RANDOM_RACE":                 "random_race",
	"OPEN_MINE_CHEST":             "open_mine_chest",
	"OPEN_IMMORTAL_MINE_CHEST":    "open_immortal_mine_chest",
	"CHANGE_RACE":                 "change_race",
	"LIFESPAN_BOOST":              "lifespan_boost",
	"SOUL_BOOST":                  "soul_boost",
	"EXIT_SPATIAL_REALM":          "exit_spatial_realm",
	"ADD_DAO_MARKS":               "add_dao_marks",
	"SPRING_EGG_HATCH":            "spring_egg_hatch",
	"REMOVE_NEGATIVE_STATUS":      "remove_negative_status",
	"MORTAL_ONLY":                 "mortal_only",
	"IMMORTAL_ONLY":               "immortal_only",
	"MIN_RANK":                    "min_rank",
	"MAX_RANK":                    "max_rank",
}

// ItemApplicator fulfills the effects.Applicator interface for item consumption.
// It executes commands within a database transaction and collects narrative feedback.
type ItemApplicator struct {
	Tx       *sql.Tx
	Player   *models.Player
	Item     models.ItemSpec
	Messages []string
}

func (a *ItemApplicator) ApplyEffectCommand(ctx context.Context, cmd effects.Command) error {
	val := cmd.Value
	if v, ok := cmd.Values["value"].(float64); ok && v != 0 {
		val = v
	}

	switch cmd.Operation {
	case effects.OpHeal:
		if strings.EqualFold(cmd.Resource, "MANA") {
			msg, err := restoreManaEffect(a.Tx, a.Player, a.Item, val)
			if err != nil {
				return err
			}
			if msg != "" {
				a.Messages = append(a.Messages, msg)
			}
			return nil
		}
		msg, err := restoreHPEffect(a.Tx, a.Player, a.Item, val)
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil

	case effects.OpUnique:
		code := strings.ToUpper(strings.TrimSpace(cmd.Code))
		key, ok := itemNativeActionToKey[code]
		if !ok {
			key = strings.ToLower(code)
		}

		if key == "hunger" {
			a.Messages = append(a.Messages, fmt.Sprintf("Đạo hữu sử dụng %s.", a.Item.Name))
			return nil
		}

		if appHandler, ok := ApplyItemEffects[key]; ok {
			msg, err := appHandler(a.Tx, a.Player, a.Item, val)
			if err != nil {
				return err
			}
			if msg != "" {
				a.Messages = append(a.Messages, msg)
			}
			return nil
		}
		return nil

	default:
		return nil
	}
}

// ExecuteItemConsumption validates and applies item effects through the typed evaluator.
func ExecuteItemConsumption(ctx context.Context, tx *sql.Tx, player *models.Player, item models.ItemSpec) ([]string, error) {
	// 1. Validation phase (restrictions: mortal_only, immortal_only, min_rank, max_rank)
	for _, command := range item.EffectCommands() {
		if valHandler, ok := ValidateItemEffects[command.Code]; ok {
			if _, err := valHandler(tx, player, item, command.Value); err != nil {
				return nil, err
			}
		}
	}

	compiled := item.Compiled()
	if compiled != nil {
		for _, op := range compiled.Operations(effects.TriggerOnConsume) {
			if op.Type == effects.OpUnique {
				code := strings.ToUpper(strings.TrimSpace(op.Code))
				key, ok := itemNativeActionToKey[code]
				if !ok {
					key = strings.ToLower(code)
				}
				if valHandler, ok := ValidateItemEffects[key]; ok {
					val := op.Value
					if v, ok := op.Values["value"].(float64); ok && v != 0 {
						val = v
					}
					if _, err := valHandler(tx, player, item, val); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	applicator := &ItemApplicator{
		Tx:       tx,
		Player:   player,
		Item:     item,
		Messages: make([]string, 0),
	}

	// 2. Application phase
	if compiled != nil {
		evalCtx := effects.Context{
			Actor: effects.Snapshot{
				ID:      player.ID.String(),
				HP:      player.HP,
				MaxHP:   gamefx.EffectiveMaxHP(player),
				Mana:    player.CurrentMana,
				MaxMana: player.MaxMana,
			},
			Rank: player.Rank,
		}
		result, err := compiled.Evaluate(evalCtx, effects.Event{Trigger: effects.TriggerOnConsume})
		if err != nil {
			return nil, fmt.Errorf("đánh giá hiệu ứng vật phẩm thất bại: %w", err)
		}

		for _, cmd := range result.Commands {
			if err := applicator.ApplyEffectCommand(ctx, cmd); err != nil {
				return nil, err
			}
		}
	} else {
		// Fallback for legacy items without compiled blocks
		for _, effect := range item.EffectCommands() {
			if appHandler, ok := ApplyItemEffects[effect.Code]; ok {
				msg, err := appHandler(tx, player, item, effect.Value)
				if err != nil {
					return nil, err
				}
				if msg != "" {
					applicator.Messages = append(applicator.Messages, msg)
				}
			}
		}
	}

	if len(applicator.Messages) == 0 {
		applicator.Messages = append(applicator.Messages, fmt.Sprintf("Đạo hữu sử dụng thành công %s.", item.Name))
	}
	return applicator.Messages, nil
}
