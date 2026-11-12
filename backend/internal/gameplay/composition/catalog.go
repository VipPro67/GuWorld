package composition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

const GrantCatalogVersion = 1

var capabilityID = regexp.MustCompile(`^[A-Z][A-Z0-9_.]*$`)

// CapabilityGrant is an explicit designer-authored declaration. It binds one
// Block v2 operation to a composition capability; no Gu role/tag is consulted.
type CapabilityGrant struct {
	EffectBlockID  string `json:"effect_block_id"`
	OperationIndex int    `json:"operation_index"`
	Capability     string `json:"capability"`
}

type GrantCatalog struct {
	SchemaVersion int               `json:"schema_version"`
	Grants        []CapabilityGrant `json:"grants"`
}

// ParseGrantCatalog validates explicit grants against the source effect
// blocks, then returns a canonical ordering for stable fingerprints.
func ParseGrantCatalog(raw []byte, blocks []gameplayeffects.Block) (GrantCatalog, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return GrantCatalog{SchemaVersion: GrantCatalogVersion, Grants: []CapabilityGrant{}}, nil
	}
	var catalog GrantCatalog
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return GrantCatalog{}, fmt.Errorf("composition_contributions must be a JSON envelope: %w", err)
	}
	if catalog.SchemaVersion != GrantCatalogVersion || catalog.Grants == nil {
		return GrantCatalog{}, fmt.Errorf("composition_contributions schema_version must be %d and grants must be an array", GrantCatalogVersion)
	}
	blocksByID := make(map[string]gameplayeffects.Block, len(blocks))
	for _, block := range blocks {
		blocksByID[block.ID] = block
	}
	seen := make(map[string]bool, len(catalog.Grants))
	for index := range catalog.Grants {
		grant := &catalog.Grants[index]
		grant.EffectBlockID = strings.TrimSpace(grant.EffectBlockID)
		grant.Capability = strings.ToUpper(strings.TrimSpace(grant.Capability))
		if grant.EffectBlockID == "" || grant.OperationIndex < 0 || !capabilityID.MatchString(grant.Capability) {
			return GrantCatalog{}, fmt.Errorf("composition grant must reference a block, non-negative operation index, and canonical capability ID")
		}
		block, exists := blocksByID[grant.EffectBlockID]
		if !exists || grant.OperationIndex >= len(block.Operations) {
			return GrantCatalog{}, fmt.Errorf("composition grant %s[%d] does not reference an authored effect operation", grant.EffectBlockID, grant.OperationIndex)
		}
		if err := validateCapabilityGrant(grant.Capability, block, block.Operations[grant.OperationIndex]); err != nil {
			return GrantCatalog{}, fmt.Errorf("composition grant %s[%d]: %w", grant.EffectBlockID, grant.OperationIndex, err)
		}
		key := fmt.Sprintf("%s:%d", grant.EffectBlockID, grant.OperationIndex)
		if seen[key] {
			return GrantCatalog{}, fmt.Errorf("composition grant source %s is duplicated", key)
		}
		seen[key] = true
	}
	sort.Slice(catalog.Grants, func(i, j int) bool {
		if catalog.Grants[i].EffectBlockID == catalog.Grants[j].EffectBlockID {
			return catalog.Grants[i].OperationIndex < catalog.Grants[j].OperationIndex
		}
		return catalog.Grants[i].EffectBlockID < catalog.Grants[j].EffectBlockID
	})
	return catalog, nil
}

func (catalog GrantCatalog) Value() ([]byte, error) {
	if catalog.SchemaVersion == 0 {
		catalog.SchemaVersion = GrantCatalogVersion
	}
	if catalog.Grants == nil {
		catalog.Grants = []CapabilityGrant{}
	}
	return json.Marshal(catalog)
}
