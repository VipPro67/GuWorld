package effects

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

// BlockSet is the runtime projection of a canonical Block v2 envelope. It
// keeps evaluator call sites slice based while ensuring product snapshots
// serialize with an explicit schema version. Unmarshal accepts the earlier
// flat-array shape at read seams so existing saved products remain readable.
type BlockSet []Block

func (blocks BlockSet) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal([]Block(blocks))
	if err != nil {
		return nil, err
	}
	canonical, issues := effectcatalog.CanonicalBlockV2Envelope(raw, effectcatalog.Location{Source: "product_effects"})
	if len(issues) > 0 {
		return nil, &effectcatalog.ValidationError{Issues: issues}
	}
	return canonical, nil
}

func (blocks *BlockSet) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		// Product rows written before envelope v2 used flat Block arrays. Keep
		// this read projection until each product storage group is migrated.
		var legacy []Block
		if err := json.Unmarshal(trimmed, &legacy); err != nil {
			return err
		}
		*blocks = BlockSet(legacy)
		return nil
	}
	canonical, issues := effectcatalog.CanonicalBlockV2Envelope(raw, effectcatalog.Location{Source: "product_effects"})
	if len(issues) > 0 {
		return &effectcatalog.ValidationError{Issues: issues}
	}
	var envelope struct {
		SchemaVersion int     `json:"schema_version"`
		Blocks        []Block `json:"blocks"`
	}
	if err := json.Unmarshal(canonical, &envelope); err != nil {
		return fmt.Errorf("decode Block v2 product envelope: %w", err)
	}
	*blocks = BlockSet(envelope.Blocks)
	return nil
}

func DecodeBlockSet(raw []byte) (BlockSet, error) {
	var blocks BlockSet
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}
