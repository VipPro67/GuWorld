package effects

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	DefinitionRank  int             `json:"definition_rank,omitempty"`
	CatalogRevision uint64          `json:"catalog_revision,omitempty"`
	ContextID       string          `json:"context_id,omitempty"`
	Legacy          json.RawMessage `json:"legacy"`
	Block           json.RawMessage `json:"block"`
	Error           string          `json:"error,omitempty"`
}

// CompareDefinition is Compare with the complete catalog identity used by
// ranked definitions. Keeping rank out of ContextID makes mismatch records
// queryable without parsing an opaque string.
func (s *ShadowMode) CompareDefinition(ctx context.Context, domain Domain, sourceKind, sourceID string, rank int, contextID string, revision uint64, legacy any, blockFn func(context.Context) (any, error)) {
	s.compare(ctx, ShadowMismatch{Domain: domain, SourceKind: sourceKind, SourceID: sourceID, DefinitionRank: rank, CatalogRevision: revision, ContextID: contextID}, legacy, blockFn)
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
	s.compare(ctx, ShadowMismatch{Domain: domain, SourceKind: sourceKind, SourceID: sourceID, CatalogRevision: revision, ContextID: contextID}, legacy, blockFn)
}

func (s *ShadowMode) compare(ctx context.Context, identity ShadowMismatch, legacy any, blockFn func(context.Context) (any, error)) {
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
	if err == nil && legacyErr == nil && blockErr == nil && shadowJSONEqual(legacyJSON, blockJSON) {
		return
	}
	m := identity
	m.Event, m.Legacy, m.Block = "effect_block_shadow_mismatch", legacyJSON, blockJSON
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
	return json.Marshal(normalizeShadowJSON(normalized))
}

const shadowFloatPrecision = 1e-9

// normalizeShadowValue removes representation-only differences before the
// comparison: nil collections equal empty collections and floats are rounded
// to a stable precision. JSON marshaling provides deterministic map ordering.
func normalizeShadowJSON(v any) any {
	switch value := v.(type) {
	case float64:
		if math.Abs(value) > math.MaxFloat64*shadowFloatPrecision {
			return value
		}
		return math.Round(value/shadowFloatPrecision) * shadowFloatPrecision
	case []any:
		for i := range value {
			value[i] = normalizeShadowJSON(value[i])
		}
	case map[string]any:
		for key := range value {
			value[key] = normalizeShadowJSON(value[key])
		}
	}
	return v
}

func shadowJSONEqual(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}
	var left, right any
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return false
	}
	return shadowValuesEqual(left, right)
}

func shadowValuesEqual(a, b any) bool {
	if a == nil {
		return emptyJSONCollection(b)
	}
	if b == nil {
		return emptyJSONCollection(a)
	}
	switch left := a.(type) {
	case float64:
		right, ok := b.(float64)
		return ok && left == right
	case string:
		right, ok := b.(string)
		return ok && left == right
	case bool:
		right, ok := b.(bool)
		return ok && left == right
	case []any:
		right, ok := b.([]any)
		if !ok || len(left) != len(right) {
			return false
		}
		for i := range left {
			if !shadowValuesEqual(left[i], right[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		right, ok := b.(map[string]any)
		if !ok || len(left) != len(right) {
			return false
		}
		for key, value := range left {
			other, ok := right[key]
			if !ok || !shadowValuesEqual(value, other) {
				return false
			}
		}
		return true
	}
	return false
}

func emptyJSONCollection(v any) bool {
	switch value := v.(type) {
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	}
	return v == nil
}
