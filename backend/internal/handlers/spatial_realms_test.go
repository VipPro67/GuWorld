package handlers

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseRealmRulesDefaults(t *testing.T) {
	rules := parseRealmRules(nil, 60)
	if rules.DurationMinutes != 60 {
		t.Errorf("expected DurationMinutes 60, got %d", rules.DurationMinutes)
	}
	if rules.PvPMode != "SAFE" {
		t.Errorf("expected PvPMode SAFE, got %s", rules.PvPMode)
	}
	if rules.DeathPenalty != "EVICT" {
		t.Errorf("expected DeathPenalty EVICT, got %s", rules.DeathPenalty)
	}
	if rules.AllowTeleport != false {
		t.Errorf("expected AllowTeleport false, got %v", rules.AllowTeleport)
	}
	if rules.EntryItemCount != 1 {
		t.Errorf("expected EntryItemCount 1, got %d", rules.EntryItemCount)
	}
}

func TestParseRealmRulesCustom(t *testing.T) {
	raw := []byte(`{
		"duration_minutes": 180,
		"pvp_mode": "FREE_PK",
		"max_allowed_rank": 5,
		"apply_rank_suppression": true,
		"death_penalty": "SAFE_RESPAWN",
		"allow_teleport": true,
		"entry_item": "token_heritage_1",
		"entry_item_count": 2
	}`)
	rules := parseRealmRules(raw, 60)
	if rules.DurationMinutes != 180 {
		t.Errorf("expected DurationMinutes 180, got %d", rules.DurationMinutes)
	}
	if rules.PvPMode != "FREE_PK" {
		t.Errorf("expected PvPMode FREE_PK, got %s", rules.PvPMode)
	}
	if rules.MaxAllowedRank != 5 {
		t.Errorf("expected MaxAllowedRank 5, got %d", rules.MaxAllowedRank)
	}
	if !rules.ApplyRankSuppression {
		t.Errorf("expected ApplyRankSuppression true, got false")
	}
	if rules.DeathPenalty != "SAFE_RESPAWN" {
		t.Errorf("expected DeathPenalty SAFE_RESPAWN, got %s", rules.DeathPenalty)
	}
	if !rules.AllowTeleport {
		t.Errorf("expected AllowTeleport true, got false")
	}
	if rules.EntryItem != "token_heritage_1" {
		t.Errorf("expected EntryItem token_heritage_1, got %s", rules.EntryItem)
	}
	if rules.EntryItemCount != 2 {
		t.Errorf("expected EntryItemCount 2, got %d", rules.EntryItemCount)
	}
}

func TestAltarMetadataUnmarshal(t *testing.T) {
	raw := []byte(`{
		"altar_name": "Đài Truyền Thừa: Hoa Tửu Hành Giả",
		"path": "Food Path",
		"tier": "IMMORTAL",
		"guardian_coord": {"x": 1015, "y": 1010},
		"req_rank": 3,
		"req_dao_marks": {
			"Food Path": 50
		},
		"required_item": "wine_pot_token",
		"required_item_count": 1,
		"cooldown_hours": 0,
		"rewards": {
			"gus": ["GU_FOOD_WINE_POT_4", "GU_SOUL_CLEANSE_3"],
			"recipes": ["recipe_wine_gu_3"],
			"primeval_stones": 50000,
			"materials": {
				"green_bamboo_wine": 10
			},
			"legacy_text": "Ta - Hoa Tửu Hành Giả, một đời phong lưu tiêu sái..."
		}
	}`)

	var meta AltarMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("failed to unmarshal AltarMetadata: %v", err)
	}

	if meta.AltarName != "Đài Truyền Thừa: Hoa Tửu Hành Giả" {
		t.Errorf("unexpected AltarName: %s", meta.AltarName)
	}
	if meta.Tier != "IMMORTAL" {
		t.Errorf("unexpected Tier: %s", meta.Tier)
	}
	if meta.GuardianCoord == nil || meta.GuardianCoord.X != 1015 || meta.GuardianCoord.Y != 1010 {
		t.Errorf("unexpected GuardianCoord: %+v", meta.GuardianCoord)
	}
	if meta.ReqRank != 3 {
		t.Errorf("unexpected ReqRank: %d", meta.ReqRank)
	}
	if meta.ReqDaoMarks["Food Path"] != 50 {
		t.Errorf("unexpected ReqDaoMarks: %+v", meta.ReqDaoMarks)
	}
	if meta.RequiredItem != "wine_pot_token" || meta.RequiredItemCount != 1 {
		t.Errorf("unexpected RequiredItem: %s x%d", meta.RequiredItem, meta.RequiredItemCount)
	}
	if len(meta.Rewards.Gus) != 2 || meta.Rewards.Gus[0] != "GU_FOOD_WINE_POT_4" {
		t.Errorf("unexpected Rewards.Gus: %+v", meta.Rewards.Gus)
	}
	if meta.Rewards.PrimevalStones != 50000 {
		t.Errorf("unexpected PrimevalStones: %d", meta.Rewards.PrimevalStones)
	}
	if meta.Rewards.Materials["green_bamboo_wine"] != 10 {
		t.Errorf("unexpected Materials: %+v", meta.Rewards.Materials)
	}
}

func TestFormatCooldownDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Minute, "45 phút"},
		{2*time.Hour + 15*time.Minute, "2 giờ 15 phút"},
		{1 * time.Hour, "1 giờ 0 phút"},
	}
	for _, tc := range cases {
		got := formatCooldownDuration(tc.d)
		if got != tc.want {
			t.Errorf("formatCooldownDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
