package effects

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

// ShadowMode compares pure legacy and block projections. The legacy value is
// always returned by callers; ShadowMode intentionally has no Executor or
// Applicator, which makes side effects impossible on the comparison path.
type ShadowMode struct {
	Enabled bool
	Log     func(ShadowMismatch)
	count   atomic.Uint64
}

type ShadowMismatch struct {
	Event           string          `json:"event"`
	Domain          Domain          `json:"domain"`
	SourceKind      string          `json:"source_kind"`
	SourceID        string          `json:"source_id"`
	CatalogRevision uint64          `json:"catalog_revision,omitempty"`
	ContextID       string          `json:"context_id,omitempty"`
	Legacy          json.RawMessage `json:"legacy"`
	Block           json.RawMessage `json:"block"`
	Error           string          `json:"error,omitempty"`
}

func ShadowFromEnv(log func(ShadowMismatch)) *ShadowMode {
	return &ShadowMode{Enabled: os.Getenv("EFFECT_BLOCK_SHADOW") == "1", Log: log}
}

func (s *ShadowMode) MismatchCount() uint64 {
	if s == nil {
		return 0
	}
	return s.count.Load()
}

// Compare evaluates blockFn synchronously under the caller's context. It does
// not spawn work, start transactions, or apply commands. Cancellation is
// checked before evaluation and before logging.
func (s *ShadowMode) Compare(ctx context.Context, domain Domain, sourceKind, sourceID, contextID string, revision uint64, legacy any, blockFn func(context.Context) (any, error)) {
	if s == nil || !s.Enabled {
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}
	block, err := blockFn(ctx)
	if ctx.Err() != nil {
		return
	}
	legacyJSON, legacyErr := canonicalJSON(legacy)
	blockJSON, blockErr := canonicalJSON(block)
	if err == nil && legacyErr == nil && blockErr == nil && bytes.Equal(legacyJSON, blockJSON) {
		return
	}
	m := ShadowMismatch{Event: "effect_block_shadow_mismatch", Domain: domain, SourceKind: sourceKind, SourceID: sourceID, CatalogRevision: revision, ContextID: contextID, Legacy: legacyJSON, Block: blockJSON}
	if err != nil {
		m.Error = err.Error()
	} else if legacyErr != nil {
		m.Error = legacyErr.Error()
	} else if blockErr != nil {
		m.Error = blockErr.Error()
	}
	s.count.Add(1)
	if s.Log != nil {
		s.Log(m)
	}
}

func canonicalJSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal shadow value: %w", err)
	}
	var normalized any
	if err := json.Unmarshal(b, &normalized); err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}
