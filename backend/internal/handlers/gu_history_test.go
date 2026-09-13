package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

func TestGuHistoryLogStruct(t *testing.T) {
	instanceID := uuid.New().String()
	guID := "GU_SPRING_AUTUMN_CICADA"
	rank := 6
	eventType := "CREATED_REFINE"
	actorID := uuid.New()
	summary := "Test history log summary"
	meta := map[string]interface{}{"test_key": "test_val"}

	logEntry := db.GuHistoryLog{
		ID:         uuid.New().String(),
		InstanceID: instanceID,
		GuID:       guID,
		Rank:       rank,
		EventType:  eventType,
		Summary:    summary,
		Metadata:   meta,
		CreatedAt:  time.Now(),
	}

	if logEntry.InstanceID != instanceID {
		t.Errorf("InstanceID = %s, want %s", logEntry.InstanceID, instanceID)
	}
	if logEntry.Rank != 6 {
		t.Errorf("Rank = %d, want 6", logEntry.Rank)
	}
	if logEntry.EventType != "CREATED_REFINE" {
		t.Errorf("EventType = %s, want CREATED_REFINE", logEntry.EventType)
	}
	_ = actorID
}
