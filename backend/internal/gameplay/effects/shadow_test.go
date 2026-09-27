package effects

import (
	"context"
	"encoding/json"
	"testing"
)

func TestShadowLegacyRemainsAuthoritativeAndMismatchIsStructured(t *testing.T) {
	legacy := map[string]any{"damage": 10.0}
	var got ShadowMismatch
	s := &ShadowMode{Enabled: true, Log: func(m ShadowMismatch) { got = m }}
	s.Compare(context.Background(), DomainCombat, "gu", "fire_gu", "fixture-1", 7, legacy, func(context.Context) (any, error) { return map[string]any{"damage": 11.0}, nil })
	if legacy["damage"] != 10.0 || s.MismatchCount() != 1 || got.Event != "effect_block_shadow_mismatch" || got.SourceID != "fire_gu" {
		t.Fatalf("legacy=%v count=%d mismatch=%+v", legacy, s.MismatchCount(), got)
	}
}

func TestShadowNormalizationPreservesJSONEncodingSemantics(t *testing.T) {
	type projection struct {
		Params json.RawMessage `json:"params"`
		Bytes  []byte          `json:"bytes,omitempty"`
	}
	s := &ShadowMode{Enabled: true, Log: func(ShadowMismatch) { t.Fatal("unexpected log") }}
	s.Compare(context.Background(), DomainItemUse, "item", "potion", "fixture", 2,
		projection{Params: json.RawMessage(`{"b":2,"a":1}`)}, func(context.Context) (any, error) {
			return map[string]any{"params": map[string]any{"a": 1.0, "b": 2.0}}, nil
		})
	if s.MismatchCount() != 0 {
		t.Fatalf("mismatches=%d", s.MismatchCount())
	}
}

func TestShadowCancellationSkipsEvaluation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	s := &ShadowMode{Enabled: true}
	s.Compare(ctx, DomainGrowth, "gu", "x", "", 0, 1, func(context.Context) (any, error) { called = true; return 2, nil })
	if called || s.MismatchCount() != 0 {
		t.Fatalf("called=%v mismatches=%d", called, s.MismatchCount())
	}
}

func TestShadowExactParityDoesNotLog(t *testing.T) {
	s := &ShadowMode{Enabled: true, Log: func(ShadowMismatch) { t.Fatal("unexpected log") }}
	s.Compare(context.Background(), DomainRefinement, "gu", "x", "", 0, map[string]int{"b": 2, "a": 1}, func(context.Context) (any, error) { return map[string]int{"a": 1, "b": 2}, nil })
	if s.MismatchCount() != 0 {
		t.Fatal("unexpected mismatch")
	}
}

func TestShadowNormalizesFloatsAndNilCollections(t *testing.T) {
	type projection struct {
		Values []float64         `json:"values"`
		Labels map[string]string `json:"labels"`
	}
	s := &ShadowMode{Enabled: true, Log: func(ShadowMismatch) { t.Fatal("unexpected log") }}
	s.CompareDefinition(context.Background(), DomainDerived, "gu", "jade", 3, "player-1", 12,
		projection{}, func(context.Context) (any, error) {
			return projection{Values: []float64{}, Labels: map[string]string{}}, nil
		})
	s.Compare(context.Background(), DomainGrowth, "gu", "jade", "player-1", 12,
		map[string]float64{"value": 0.1 + 0.2}, func(context.Context) (any, error) {
			return map[string]float64{"value": 0.3}, nil
		})
	if s.MismatchCount() != 0 {
		t.Fatalf("mismatches=%d", s.MismatchCount())
	}
}

func TestShadowMismatchIncludesDefinitionRank(t *testing.T) {
	var got ShadowMismatch
	s := &ShadowMode{Enabled: true, Log: func(m ShadowMismatch) { got = m }}
	s.CompareDefinition(context.Background(), DomainCombat, "gu", "fire", 7, "battle", 22, 1, func(context.Context) (any, error) { return 2, nil })
	if got.DefinitionRank != 7 || got.CatalogRevision != 22 {
		t.Fatalf("mismatch=%+v", got)
	}
}
