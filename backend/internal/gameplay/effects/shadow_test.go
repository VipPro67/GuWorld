package effects

import (
	"context"
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
