package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRetryLoginEvaluationRetriesHungerLockTimeout(t *testing.T) {
	attempts := 0
	player, err := retryLoginEvaluation(context.Background(), func() (models.Player, error) {
		attempts++
		if attempts < 3 {
			return models.Player{}, fmt.Errorf("evaluate player hunger: %w", &pq.Error{
				Code:    "55P03",
				Message: "canceling statement due to lock timeout",
			})
		}
		player := models.Player{}
		player.Username = "lock-survivor"
		return player, nil
	})
	if err != nil {
		t.Fatalf("expected transient lock timeout to recover, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 transaction attempts, got %d", attempts)
	}
	if player.Username != "lock-survivor" {
		t.Fatalf("expected successful retry result, got username %q", player.Username)
	}
}

func TestRetryLoginEvaluationDoesNotRetryPermanentError(t *testing.T) {
	attempts := 0
	wantErr := fmt.Errorf("invalid player data")
	_, err := retryLoginEvaluation(context.Background(), func() (models.Player, error) {
		attempts++
		return models.Player{}, wantErr
	})
	if err != wantErr {
		t.Fatalf("expected permanent error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected one transaction attempt, got %d", attempts)
	}
}
