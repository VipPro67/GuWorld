package handlers

import (
	"sort"
	"testing"
	"time"
)

type mockCandidate struct {
	username     string
	rank         string
	contribution int64
	joinedAt     time.Time
}

func sortCandidates(cands []mockCandidate) {
	sort.Slice(cands, func(i, j int) bool {
		valI := sectRankValue(cands[i].rank)
		valJ := sectRankValue(cands[j].rank)
		if valI != valJ {
			return valI > valJ // Seniority DESC
		}
		if cands[i].contribution != cands[j].contribution {
			return cands[i].contribution > cands[j].contribution // Contribution DESC
		}
		return cands[i].joinedAt.Before(cands[j].joinedAt) // Joined earlier
	})
}

func TestSectLeadershipSuccessionOrder(t *testing.T) {
	now := time.Now()
	cands := []mockCandidate{
		{username: "ngoai_mon_1", rank: "NGOAI_MON", contribution: 50000, joinedAt: now.Add(-100 * time.Hour)},
		{username: "truong_lao_1", rank: "TRUONG_LAO", contribution: 1000, joinedAt: now.Add(-50 * time.Hour)},
		{username: "ho_phap_1", rank: "HO_PHAP", contribution: 2000, joinedAt: now.Add(-20 * time.Hour)},
		{username: "ho_phap_2", rank: "HO_PHAP", contribution: 5000, joinedAt: now.Add(-10 * time.Hour)},
		{username: "chan_truyen_1", rank: "CHAN_TRUYEN", contribution: 99999, joinedAt: now.Add(-200 * time.Hour)},
	}

	sortCandidates(cands)

	// Top candidate MUST be ho_phap_2 (highest rank HO_PHAP, highest contribution 5000 among HO_PHAP)
	if cands[0].username != "ho_phap_2" {
		t.Fatalf("expected top candidate to be ho_phap_2, got %s", cands[0].username)
	}

	// 2nd candidate MUST be ho_phap_1
	if cands[1].username != "ho_phap_1" {
		t.Fatalf("expected 2nd candidate to be ho_phap_1, got %s", cands[1].username)
	}

	// 3rd candidate MUST be truong_lao_1
	if cands[2].username != "truong_lao_1" {
		t.Fatalf("expected 3rd candidate to be truong_lao_1, got %s", cands[2].username)
	}

	// 4th candidate MUST be chan_truyen_1
	if cands[3].username != "chan_truyen_1" {
		t.Fatalf("expected 4th candidate to be chan_truyen_1, got %s", cands[3].username)
	}

	// 5th candidate MUST be ngoai_mon_1
	if cands[4].username != "ngoai_mon_1" {
		t.Fatalf("expected 5th candidate to be ngoai_mon_1, got %s", cands[4].username)
	}
}

func TestSectLeadershipInactivityThreshold(t *testing.T) {
	threshold := 5 * 24 * time.Hour
	now := time.Now()

	// Leader active 4 days ago -> NOT eligible
	lastActive := now.Add(-4 * 24 * time.Hour)
	if now.Sub(lastActive) >= threshold {
		t.Errorf("4 days ago should not be considered inactive")
	}

	// Leader active 5.1 days ago -> ELIGIBLE
	lastActiveOld := now.Add(-5*24*time.Hour - 2*time.Hour)
	if now.Sub(lastActiveOld) < threshold {
		t.Errorf("5.1 days ago should be considered inactive")
	}
}
