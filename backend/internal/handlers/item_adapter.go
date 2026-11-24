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
	"RESTORE_HP":                  "restore_hp",
	"RESTORE_MANA":                "restore_mana",
	"INCREASE_HUMAN_QI":           "increase_human_qi",
	"RESTORE_EARTH_QI":            "restore_earth_qi",
	"RESTORE_HEAVEN_QI":           "restore_heaven_qi",
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
	case effects.OpRequire:
		if cmd.Target != effects.TargetSelf || cmd.SubType != "IMMORTAL" {
			return fmt.Errorf("item %s: unsupported REQUIRE command", a.Item.ID)
		}
		_, err := immortalOnlyEffect(a.Tx, a.Player, a.Item, 1)
		return err // also checked before any effect mutation in ExecuteItemConsumption
	case effects.OpGrantResource:
		if cmd.Target != effects.TargetSelf || cmd.Value <= 0 || cmd.Path != "" {
			return fmt.Errorf("item %s: unsupported GRANT_RESOURCE command", a.Item.ID)
		}
		var msg string
		var err error
		switch cmd.Resource {
		case "DAO_MARK":
			if a.Tx == nil {
				return fmt.Errorf("item %s: GRANT_RESOURCE DAO_MARK requires a transaction", a.Item.ID)
			}
			msg, err = addDaoMarksEffect(a.Tx, a.Player, a.Item, cmd.Value)
		case "HUMAN_QI":
			msg, err = increaseHumanQi(a.Tx, a.Player, a.Item, cmd.Value)
		default:
			return fmt.Errorf("item %s: unsupported GRANT_RESOURCE resource %q", a.Item.ID, cmd.Resource)
		}
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil
	case effects.OpRestoreResource:
		if cmd.Target != effects.TargetSelf || cmd.Value <= 0 || cmd.Path != "" {
			return fmt.Errorf("item %s: unsupported RESTORE_RESOURCE command", a.Item.ID)
		}
		if a.Tx == nil {
			return fmt.Errorf("item %s: RESTORE_RESOURCE requires a transaction", a.Item.ID)
		}
		var msg string
		var err error
		switch cmd.Resource {
		case "EARTH_QI":
			msg, err = restoreEarthQiEffect(a.Tx, a.Player, a.Item, cmd.Value)
		case "HEAVEN_QI":
			msg, err = restoreHeavenQiEffect(a.Tx, a.Player, a.Item, cmd.Value)
		default:
			return fmt.Errorf("item %s: unsupported RESTORE_RESOURCE resource %q", a.Item.ID, cmd.Resource)
		}
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil
	case effects.OpModifyAperture:
		if cmd.Target != effects.TargetSelf || cmd.Value <= 0 || cmd.Resource != "" || cmd.Path != "" {
			return fmt.Errorf("item %s: unsupported MODIFY_APERTURE command", a.Item.ID)
		}
		if a.Tx == nil {
			return fmt.Errorf("item %s: MODIFY_APERTURE requires a transaction", a.Item.ID)
		}
		var msg string
		var err error
		switch cmd.SubType {
		case "AREA_SIZE":
			msg, err = increaseApertureAreaEffect(a.Tx, a.Player, a.Item, cmd.Value)
		case "TIME_RATE":
			msg, err = increaseApertureTimeRateEffect(a.Tx, a.Player, a.Item, cmd.Value)
		default:
			return fmt.Errorf("item %s: unsupported MODIFY_APERTURE field %q", a.Item.ID, cmd.SubType)
		}
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil
	case effects.OpRandomizeTrait:
		if cmd.Target != effects.TargetSelf || cmd.Value != 0 || cmd.Resource != "" || cmd.Path != "" {
			return fmt.Errorf("item %s: unsupported RANDOMIZE_TRAIT command", a.Item.ID)
		}
		var msg string
		var err error
		switch cmd.SubType {
		case "CONSTITUTION":
			msg, err = randomConstitutionEffect(a.Tx, a.Player, a.Item, 0)
		case "RACE":
			msg, err = randomRaceEffect(a.Tx, a.Player, a.Item, 0)
		default:
			return fmt.Errorf("item %s: unsupported RANDOMIZE_TRAIT type %q", a.Item.ID, cmd.SubType)
		}
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil
	case effects.OpRepairAperture:
		if cmd.Target != effects.TargetSelf || cmd.Value <= 0 || cmd.SubType != "" || cmd.Resource != "" || cmd.Path != "" {
			return fmt.Errorf("item %s: unsupported REPAIR_APERTURE command", a.Item.ID)
		}
		if a.Tx == nil {
			return fmt.Errorf("item %s: REPAIR_APERTURE requires a transaction", a.Item.ID)
		}
		msg, err := healApertureEffect(a.Tx, a.Player, a.Item, cmd.Value)
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil
	case effects.OpOpenRewardContainer:
		if cmd.Target != effects.TargetSelf || cmd.Value != 0 || cmd.Resource != "" || cmd.Path != "" {
			return fmt.Errorf("item %s: unsupported OPEN_REWARD_CONTAINER command", a.Item.ID)
		}
		if a.Tx == nil {
			return fmt.Errorf("item %s: OPEN_REWARD_CONTAINER requires a transaction", a.Item.ID)
		}
		var msg string
		var err error
		switch cmd.SubType {
		case "MORTAL_MINE":
			msg, err = openMineChestEffect(a.Tx, a.Player, a.Item, 0)
		case "IMMORTAL_MINE":
			msg, err = openImmortalMineChestEffect(a.Tx, a.Player, a.Item, 0)
		default:
			return fmt.Errorf("item %s: unsupported reward container %q", a.Item.ID, cmd.SubType)
		}
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil
	case effects.OpPlaceWorldObject:
		if cmd.Target != effects.TargetSelf || cmd.Value != 0 || cmd.Resource != "" || cmd.Path != "" || cmd.SubType != "PRIMEVAL_SPRING" {
			return fmt.Errorf("item %s: unsupported PLACE_WORLD_OBJECT command", a.Item.ID)
		}
		if a.Tx == nil {
			return fmt.Errorf("item %s: PLACE_WORLD_OBJECT requires a transaction", a.Item.ID)
		}
		msg, err := springEggHatchEffect(a.Tx, a.Player, a.Item, 0)
		if err != nil {
			return err
		}
		if msg != "" {
			a.Messages = append(a.Messages, msg)
		}
		return nil
	case effects.OpHeal:
		if cmd.Target != effects.TargetSelf || cmd.SubType != "" && cmd.SubType != "FLAT" || cmd.Value <= 0 || cmd.StackPolicy != "" || cmd.Resource != "" && !strings.EqualFold(cmd.Resource, "MANA") && !strings.EqualFold(cmd.Resource, "HP") {
			return fmt.Errorf("item %s: unsupported HEAL command", a.Item.ID)
		}
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
		if _, ok := ValidateItemEffects[key]; ok {
			return nil
		}
		return fmt.Errorf("item %s: no executor for native effect %q", a.Item.ID, code)

	default:
		return fmt.Errorf("item %s: unsupported effect operation %q", a.Item.ID, cmd.Operation)
	}
}

func validateItemCommands(item models.ItemSpec, commands []effects.Command) error {
	for _, cmd := range commands {
		if cmd.Target != effects.TargetSelf {
			return fmt.Errorf("item %s: unsupported effect target %q", item.ID, cmd.Target)
		}
		switch cmd.Operation {
		case effects.OpRequire:
			if cmd.SubType != "IMMORTAL" || cmd.Resource != "" || cmd.Value != 0 {
				return fmt.Errorf("item %s: unsupported REQUIRE parameters", item.ID)
			}
		case effects.OpGrantResource:
			if cmd.Resource != "DAO_MARK" && cmd.Resource != "HUMAN_QI" || cmd.Value <= 0 || cmd.SubType != "" || cmd.Path != "" {
				return fmt.Errorf("item %s: unsupported GRANT_RESOURCE parameters", item.ID)
			}
		case effects.OpRestoreResource:
			if cmd.Resource != "EARTH_QI" && cmd.Resource != "HEAVEN_QI" || cmd.Value <= 0 || cmd.SubType != "" || cmd.Path != "" {
				return fmt.Errorf("item %s: unsupported RESTORE_RESOURCE parameters", item.ID)
			}
		case effects.OpModifyAperture:
			if cmd.SubType != "AREA_SIZE" && cmd.SubType != "TIME_RATE" || cmd.Value <= 0 || cmd.Resource != "" || cmd.Path != "" || cmd.StackPolicy != "" {
				return fmt.Errorf("item %s: unsupported MODIFY_APERTURE parameters", item.ID)
			}
		case effects.OpRandomizeTrait:
			if cmd.SubType != "CONSTITUTION" && cmd.SubType != "RACE" || cmd.Value != 0 || cmd.Resource != "" || cmd.Path != "" || cmd.StackPolicy != "" {
				return fmt.Errorf("item %s: unsupported RANDOMIZE_TRAIT parameters", item.ID)
			}
		case effects.OpRepairAperture:
			if cmd.Value <= 0 || cmd.SubType != "" || cmd.Resource != "" || cmd.Path != "" || cmd.StackPolicy != "" {
				return fmt.Errorf("item %s: unsupported REPAIR_APERTURE parameters", item.ID)
			}
		case effects.OpOpenRewardContainer:
			if cmd.SubType != "MORTAL_MINE" && cmd.SubType != "IMMORTAL_MINE" || cmd.Value != 0 || cmd.Resource != "" || cmd.Path != "" || cmd.StackPolicy != "" {
				return fmt.Errorf("item %s: unsupported OPEN_REWARD_CONTAINER parameters", item.ID)
			}
		case effects.OpPlaceWorldObject:
			if cmd.SubType != "PRIMEVAL_SPRING" || cmd.Value != 0 || cmd.Resource != "" || cmd.Path != "" || cmd.StackPolicy != "" {
				return fmt.Errorf("item %s: unsupported PLACE_WORLD_OBJECT parameters", item.ID)
			}
		case effects.OpHeal:
			if cmd.SubType != "" && cmd.SubType != "FLAT" || cmd.Value <= 0 || cmd.StackPolicy != "" || cmd.Resource != "" && !strings.EqualFold(cmd.Resource, "HP") && !strings.EqualFold(cmd.Resource, "MANA") {
				return fmt.Errorf("item %s: unsupported HEAL parameters", item.ID)
			}
		case effects.OpUnique:
			code := strings.ToUpper(strings.TrimSpace(cmd.Code))
			key, ok := itemNativeActionToKey[code]
			if !ok {
				key = strings.ToLower(code)
			}
			if _, ok := ApplyItemEffects[key]; ok {
				continue
			}
			if _, ok := ValidateItemEffects[key]; !ok {
				return fmt.Errorf("item %s: no executor for native effect %q", item.ID, code)
			}
		default:
			return fmt.Errorf("item %s: unsupported effect operation %q", item.ID, cmd.Operation)
		}
	}
	return nil
}

// ExecuteItemConsumption validates and applies item effects through the typed evaluator.
func ExecuteItemConsumption(ctx context.Context, tx *sql.Tx, player *models.Player, item models.ItemSpec) ([]string, error) {
	if len(item.Blocks) > 0 {
		if err := models.ValidateItemBlocks(item.Blocks); err != nil {
			return nil, fmt.Errorf("item %s: %w", item.ID, err)
		}
	}
	for _, command := range item.EffectCommands() {
		if _, ok := ApplyItemEffects[command.Code]; !ok {
			if _, ok := ValidateItemEffects[command.Code]; !ok {
				return nil, fmt.Errorf("item %s: no executor for effect %q", item.ID, command.Code)
			}
		}
	}
	if len(item.Blocks) > 0 && item.Compiled() == nil {
		return nil, fmt.Errorf("item %s: effect compilation failed", item.ID)
	}
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
			if op.Type == effects.OpRequire && op.SubType == "IMMORTAL" {
				if _, err := immortalOnlyEffect(tx, player, item, 1); err != nil {
					return nil, err
				}
			}
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
				Race:    player.Race,
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
		if result.Truncated {
			return nil, fmt.Errorf("item %s: effect evaluation exceeded limits", item.ID)
		}
		if err := validateItemCommands(item, result.Commands); err != nil {
			return nil, err
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
			} else if _, ok := ValidateItemEffects[effect.Code]; !ok {
				return nil, fmt.Errorf("item %s: no executor for effect %q", item.ID, effect.Code)
			}
		}
	}

	if len(applicator.Messages) == 0 {
		applicator.Messages = append(applicator.Messages, fmt.Sprintf("Đạo hữu sử dụng thành công %s.", item.Name))
	}
	return applicator.Messages, nil
}
