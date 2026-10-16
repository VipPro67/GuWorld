package handlers

import (
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func applyDamageWithLogs(initiatorName, targetName string, target *models.Combatant, damage float64) string {
	if (target.Kind == "Player" || target.Kind == "Ally") && target.Rank >= 6 {
		apertureDmg := damage * 0.5
		target.ApertureDmg += apertureDmg
		damage = damage / 200.0
	}

	if target.VirtualShield > 0 {
		if target.VirtualShield >= damage {
			target.VirtualShield -= damage
			return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (Bị hấp thụ hoàn toàn bởi Khiên ảo, còn %.1f Khiên).",
				initiatorName, damage, targetName, target.VirtualShield)
		} else {
			shieldDmg := target.VirtualShield
			hpDmg := damage - shieldDmg
			target.VirtualShield = 0
			target.HP -= hpDmg
			if target.HP < 0 {
				target.HP = 0
			}
			return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (Lớp Khiên ảo bị đánh vỡ, %.1f sát thương trừ thẳng vào HP, còn %.1f HP).",
				initiatorName, damage, targetName, hpDmg, target.HP)
		}
	} else {
		target.HP -= damage
		if target.HP < 0 {
			target.HP = 0
		}
		return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (%.1f sát thương trừ thẳng vào HP, còn %.1f HP).",
			initiatorName, damage, targetName, damage, target.HP)
	}
}
