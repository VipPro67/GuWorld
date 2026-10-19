package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// RegionRules is the shared, additive restriction format for named WorldMap
// areas and spatial-realm regions. Empty fields preserve existing gameplay.
type RegionRules struct {
	Rank struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"rank"`
	Paths struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
	} `json:"paths"`
	Gu struct {
		MaxRank int      `json:"max_rank"`
		DenyIDs []string `json:"deny_ids"`
	} `json:"gu"`
	Items struct {
		DenyIDs []string `json:"deny_ids"`
	} `json:"items"`
	Actions struct {
		Deny []string `json:"deny"`
	} `json:"actions"`
}

var regionPolicyActions = map[string]bool{
	"pvp": true, "teleport": true, "gu_use": true, "item_use": true,
	"combat": true, "monster_combat": true, "resource_gathering": true, "gu_capture": true,
}

func decodeRegionRules(raw json.RawMessage) (RegionRules, error) {
	var rules RegionRules
	if len(raw) == 0 || string(raw) == "null" {
		return rules, nil
	}
	if err := json.Unmarshal(raw, &rules); err != nil {
		return RegionRules{}, fmt.Errorf("rules phải là JSON object hợp lệ: %w", err)
	}
	return rules, nil
}

func validateRegionRules(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return fmt.Errorf("rules phải là một JSON object")
	}
	rules, err := decodeRegionRules(raw)
	if err != nil {
		return err
	}
	if rules.Rank.Min < 0 || rules.Rank.Min > 9 || rules.Rank.Max < 0 || rules.Rank.Max > 9 || (rules.Rank.Min > 0 && rules.Rank.Max > 0 && rules.Rank.Max < rules.Rank.Min) {
		return fmt.Errorf("rank.min/max phải nằm trong khoảng 0–9 và max không thấp hơn min")
	}
	if rules.Gu.MaxRank < 0 || rules.Gu.MaxRank > 9 {
		return fmt.Errorf("gu.max_rank phải nằm trong khoảng 0–9")
	}
	if err := validatePolicyStrings("paths.allow", rules.Paths.Allow); err != nil {
		return err
	}
	if err := validatePolicyStrings("paths.deny", rules.Paths.Deny); err != nil {
		return err
	}
	if err := validatePolicyStrings("gu.deny_ids", rules.Gu.DenyIDs); err != nil {
		return err
	}
	if err := validatePolicyStrings("items.deny_ids", rules.Items.DenyIDs); err != nil {
		return err
	}
	if err := validatePolicyStrings("actions.deny", rules.Actions.Deny); err != nil {
		return err
	}
	for _, action := range rules.Actions.Deny {
		if !regionPolicyActions[normalizePolicyToken(action)] {
			return fmt.Errorf("actions.deny chứa hành động chưa hỗ trợ: %s", action)
		}
	}
	return nil
}

func validatePolicyStrings(field string, values []string) error {
	if len(values) > 200 {
		return fmt.Errorf("%s tối đa 200 giá trị", field)
	}
	for _, value := range values {
		if value = strings.TrimSpace(value); value == "" || len(value) > 80 {
			return fmt.Errorf("%s không được chứa giá trị trống hoặc dài hơn 80 ký tự", field)
		}
	}
	return nil
}

func normalizePolicyToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizePolicyPath(value string) string {
	value = normalizePolicyToken(value)
	return strings.TrimSpace(strings.TrimSuffix(value, " path"))
}

// regionRulesDecode caches the expensive half of regionRulesFor: the per-region
// JSON decode and the group-presence scan. The world-monster tick calls
// regionRulesFor twice per online player every 250ms, so without this the same
// handful of regions is unmarshalled thousands of times per second.
type regionRulesDecode struct {
	raw      []byte
	groups   map[string]json.RawMessage
	rules    RegionRules
	hasRules bool
}

// regionRulesDecodeCache keeps only the per-region decode. The parent-lineage
// merge still runs per call, so a parent rule change is picked up immediately.
// Entries are keyed by region id and revalidated against the region's raw Rules
// bytes, so an admin import (or a test that rewrites zoneReg) invalidates them
// transparently without a generation counter.
var regionRulesDecodeCache struct {
	mu      sync.RWMutex
	entries map[string]regionRulesDecode
}

func decodedRegionRules(region *models.Region) (regionRulesDecode, error) {
	regionRulesDecodeCache.mu.RLock()
	cached, ok := regionRulesDecodeCache.entries[region.ID]
	regionRulesDecodeCache.mu.RUnlock()
	if ok && bytes.Equal(cached.raw, region.Rules) {
		return cached, nil
	}

	entry := regionRulesDecode{raw: append([]byte(nil), region.Rules...)}
	if len(region.Rules) > 0 && string(region.Rules) != "null" {
		var groups map[string]json.RawMessage
		if err := json.Unmarshal(region.Rules, &groups); err != nil || groups == nil {
			return regionRulesDecode{}, fmt.Errorf("rules của khu vực %s không hợp lệ", region.ID)
		}
		rules, err := decodeRegionRules(region.Rules)
		if err != nil {
			return regionRulesDecode{}, err
		}
		entry.groups, entry.rules, entry.hasRules = groups, rules, true
	}

	regionRulesDecodeCache.mu.Lock()
	if regionRulesDecodeCache.entries == nil {
		regionRulesDecodeCache.entries = make(map[string]regionRulesDecode)
	}
	regionRulesDecodeCache.entries[region.ID] = entry
	regionRulesDecodeCache.mu.Unlock()
	return entry, nil
}

func regionRulesFor(regionID string) (RegionRules, error) {
	region := ZoneByID(regionID)
	if region == nil {
		return RegionRules{}, nil
	}
	lineage := make([]*models.Region, 0, 4)
	seen := make(map[string]struct{}, 4)
	for region != nil {
		if _, exists := seen[region.ID]; exists {
			break
		}
		seen[region.ID] = struct{}{}
		lineage = append(lineage, region)
		if region.ParentID == "" {
			break
		}
		region = ZoneByID(region.ParentID)
	}

	var effective RegionRules
	for i := len(lineage) - 1; i >= 0; i-- {
		decoded, err := decodedRegionRules(lineage[i])
		if err != nil {
			return RegionRules{}, err
		}
		if !decoded.hasRules {
			continue
		}
		// Group presence is meaningful: an omitted group inherits, while an
		// explicitly present empty group clears that group from its parent.
		if _, ok := decoded.groups["rank"]; ok {
			effective.Rank = decoded.rules.Rank
		}
		if _, ok := decoded.groups["paths"]; ok {
			effective.Paths = decoded.rules.Paths
		}
		if _, ok := decoded.groups["gu"]; ok {
			effective.Gu = decoded.rules.Gu
		}
		if _, ok := decoded.groups["items"]; ok {
			effective.Items = decoded.rules.Items
		}
		if _, ok := decoded.groups["actions"]; ok {
			effective.Actions = decoded.rules.Actions
		}
	}
	return effective, nil
}

func checkRegionAction(regionID, action string) error {
	rules, err := regionRulesFor(regionID)
	if err != nil {
		return err
	}
	action = normalizePolicyToken(action)
	for _, denied := range rules.Actions.Deny {
		if normalizePolicyToken(denied) == action {
			return fmt.Errorf("Khu vực này cấm hành động %s", action)
		}
	}
	return nil
}

func checkRegionEntry(regionID string, rank int, inventory models.GuInventory) error {
	rules, err := regionRulesFor(regionID)
	if err != nil {
		return err
	}
	if rules.Rank.Min > 0 && rank < rules.Rank.Min {
		return fmt.Errorf("Cảnh giới chưa đạt mức tối thiểu để vào khu vực (cần rank %d)", rules.Rank.Min)
	}
	if rules.Rank.Max > 0 && rank > rules.Rank.Max {
		return fmt.Errorf("Cảnh giới vượt quá giới hạn của khu vực (tối đa rank %d)", rules.Rank.Max)
	}
	return checkRegionEquippedGus(regionID, inventory)
}

func checkRegionGu(regionID string, guID string, rank int, paths []string) error {
	rules, err := regionRulesFor(regionID)
	if err != nil {
		return err
	}
	for _, denied := range rules.Gu.DenyIDs {
		if strings.EqualFold(strings.TrimSpace(denied), strings.TrimSpace(guID)) {
			return fmt.Errorf("Khu vực này cấm Cổ trùng %s", guID)
		}
	}
	if rules.Gu.MaxRank > 0 && rank > rules.Gu.MaxRank {
		return fmt.Errorf("Khu vực này chỉ cho phép Cổ trùng tối đa rank %d", rules.Gu.MaxRank)
	}
	guPaths := map[string]bool{}
	for _, path := range paths {
		guPaths[normalizePolicyPath(path)] = true
	}
	for _, denied := range rules.Paths.Deny {
		if guPaths[normalizePolicyPath(denied)] {
			return fmt.Errorf("Khu vực này cấm sử dụng Cổ trùng thuộc đạo %s", denied)
		}
	}
	if len(rules.Paths.Allow) > 0 {
		allowed := false
		for _, path := range rules.Paths.Allow {
			if guPaths[normalizePolicyPath(path)] {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("Khu vực này chỉ cho phép Cổ trùng thuộc các đạo đã quy định")
		}
	}
	return nil
}

func checkRegionEquippedGus(regionID string, inventory models.GuInventory) error {
	for _, gu := range inventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			continue
		}
		if err := checkRegionGu(regionID, gu.GuID, gu.Rank, spec.Path); err != nil {
			return err
		}
	}
	return nil
}

func checkRegionItem(regionID, itemID string) error {
	rules, err := regionRulesFor(regionID)
	if err != nil {
		return err
	}
	for _, denied := range rules.Items.DenyIDs {
		if strings.EqualFold(strings.TrimSpace(denied), strings.TrimSpace(itemID)) {
			return fmt.Errorf("Khu vực này cấm sử dụng vật phẩm %s", itemID)
		}
	}
	return nil
}
