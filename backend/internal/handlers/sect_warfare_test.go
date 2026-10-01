package handlers

import (
	"database/sql"
	"testing"
	"time"
)

func TestSectArrayRallyTiming(t *testing.T) {
	if sectArrayRallyWindow != 10*time.Minute {
		t.Fatalf("rally window = %v", sectArrayRallyWindow)
	}
	if sectArrayCooldown != 12*time.Hour {
		t.Fatalf("cooldown = %v", sectArrayCooldown)
	}
}

func TestSectArrayCanIntercept(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		readyAt sql.NullTime
		want    bool
	}{
		{name: "never triggered", want: true},
		{name: "cooldown elapsed", readyAt: sql.NullTime{Time: now, Valid: true}, want: true},
		{name: "cooldown active", readyAt: sql.NullTime{Time: now.Add(time.Minute), Valid: true}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sectArrayCanIntercept(now, tt.readyAt); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
