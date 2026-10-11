package handlers

import (
	"encoding/json"
	"time"
)

const (
	// MaxMartialPlayersPerRank is the largest bracket size opened for a single rank.
	MaxMartialPlayersPerRank = 128
	// MinMartialPlayersPerRank is the minimum online players required to open a rank bracket.
	MinMartialPlayersPerRank = 8
)

// MartialTournament represents a single rank bracket of the martial arts event.
type MartialTournament struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	Rank             int        `json:"rank"`
	ScheduledAt      time.Time  `json:"scheduled_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	TotalPlayers     int        `json:"total_players"`
	CurrentRound     int        `json:"current_round"`
	TotalRounds      int        `json:"total_rounds"`
	ChampionPlayerID *string    `json:"champion_player_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// MartialCombatant is one player participating in one rank bracket.
type MartialCombatant struct {
	ID                string    `json:"id"`
	TournamentID      string    `json:"tournament_id"`
	PlayerID          string    `json:"player_id"`
	Username          string    `json:"username"`
	AvatarURL         string    `json:"avatar_url"`
	SeedNumber        int       `json:"seed_number"`
	Rank              int       `json:"rank"`
	SubRank           int       `json:"sub_rank"`
	Eliminated        bool      `json:"eliminated"`
	EliminatedAtRound *int      `json:"eliminated_at_round,omitempty"`
	FinalPlacement    *int      `json:"final_placement,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// MartialMatch is one 1v1 match in a rank bracket.
type MartialMatch struct {
	ID                string     `json:"id"`
	TournamentID      string     `json:"tournament_id"`
	RoundNumber       int        `json:"round_number"`
	BracketPosition   int        `json:"bracket_position"`
	CombatantAID      string     `json:"combatant_a_id"`
	CombatantAName    string     `json:"combatant_a_name"`
	CombatantBID      *string    `json:"combatant_b_id,omitempty"`
	CombatantBName    *string    `json:"combatant_b_name,omitempty"`
	WinnerCombatantID *string    `json:"winner_combatant_id,omitempty"`
	Status            string     `json:"status"`
	RewardStones      int        `json:"reward_stones"`
	CreatedAt         time.Time  `json:"created_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}

// MartialChampionReward is a claimable placement reward.
type MartialChampionReward struct {
	ID                   string          `json:"id"`
	TournamentID         string          `json:"tournament_id"`
	PlayerID             string          `json:"player_id"`
	Username             string          `json:"username"`
	Placement            int             `json:"placement"`
	CombatantID          string          `json:"combatant_id"`
	Title                string          `json:"title"`
	RewardStones         int             `json:"reward_stones"`
	RewardImmortalStones int             `json:"reward_immortal_stones"`
	RewardGuID           *string         `json:"reward_gu_id,omitempty"`
	RewardGuRank         *int            `json:"reward_gu_rank,omitempty"`
	RewardItems          json.RawMessage `json:"reward_items"`
	Claimed              bool            `json:"claimed"`
	CreatedAt            time.Time       `json:"created_at"`
}
