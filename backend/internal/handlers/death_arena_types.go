package handlers

import "time"

const (
	// MaxTournamentTeams is the maximum number of teams allowed in a tournament (16 teams = 4 rounds)
	MaxTournamentTeams = 16
	// MaxTournamentPlayers is the maximum number of participants allowed (16 teams * 6 members = 96)
	MaxTournamentPlayers = MaxTournamentTeams * 6
)

// Tournament represents a death arena tournament
type Tournament struct {
	ID                    string     `json:"id"`
	Status                string     `json:"status"`
	ScheduledAt           time.Time  `json:"scheduled_at"`
	RegistrationStartedAt time.Time  `json:"registration_started_at"`
	RegistrationClosedAt  time.Time  `json:"registration_closed_at"`
	StartedAt             *time.Time `json:"started_at,omitempty"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
	TotalPlayers          int        `json:"total_players"`
	TotalTeams            int        `json:"total_teams"`
	CurrentRound          int        `json:"current_round"`
	TotalRounds           int        `json:"total_rounds"`
	ChampionTeamID        *string    `json:"champion_team_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

// Team represents a team in the tournament
type Team struct {
	ID                string    `json:"id"`
	TournamentID      string    `json:"tournament_id"`
	TeamName          string    `json:"team_name"`
	SeedNumber        int       `json:"seed_number"`
	Eliminated        bool      `json:"eliminated"`
	EliminatedAtRound *int      `json:"eliminated_at_round,omitempty"`
	FinalPlacement    *int      `json:"final_placement,omitempty"`
	MemberCount       int       `json:"member_count"`
	AvgHP             float64   `json:"avg_hp"`
	CreatedAt         time.Time `json:"created_at"`
}

// TeamMember represents a team member
type TeamMember struct {
	ID                string    `json:"id"`
	TeamID            string    `json:"team_id"`
	PlayerID          string    `json:"player_id"`
	Username          string    `json:"username"`
	AvatarURL         string    `json:"avatar_url"`
	Rank              int       `json:"rank"`
	CurrentHP         float64   `json:"current_hp"`
	MaxHP             float64   `json:"max_hp"`
	CurrentMana       float64   `json:"current_mana"`
	MaxMana           float64   `json:"max_mana"`
	Eliminated        bool      `json:"eliminated"`
	EliminatedAtRound *int      `json:"eliminated_at_round,omitempty"`
	JoinedAt          time.Time `json:"joined_at"`
}

// Match represents a tournament match
type Match struct {
	ID                    string     `json:"id"`
	TournamentID          string     `json:"tournament_id"`
	RoundNumber           int        `json:"round_number"`
	BracketPosition       int        `json:"bracket_position"`
	TeamAID               string     `json:"team_a_id"`
	TeamAName             string     `json:"team_a_name"`
	TeamBID               *string    `json:"team_b_id,omitempty"`
	TeamBName             *string    `json:"team_b_name,omitempty"`
	WinnerTeamID          *string    `json:"winner_team_id,omitempty"`
	Status                string     `json:"status"`
	RewardStonesPerMember int        `json:"reward_stones_per_member"`
	CreatedAt             time.Time  `json:"created_at"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
}

// ChampionReward represents a champion reward
type ChampionReward struct {
	ID                   string    `json:"id"`
	TournamentID         string    `json:"tournament_id"`
	PlayerID             string    `json:"player_id"`
	Username             string    `json:"username"`
	Placement            int       `json:"placement"`
	TeamID               string    `json:"team_id"`
	TeamName             string    `json:"team_name"`
	Title                string    `json:"title"`
	RewardStones         int       `json:"reward_stones"`
	RewardImmortalStones int       `json:"reward_immortal_stones"`
	RewardGuID           *string   `json:"reward_gu_id,omitempty"`
	RewardGuRank         *int      `json:"reward_gu_rank,omitempty"`
	RewardItems          []byte    `json:"reward_items"`
	Claimed              bool      `json:"claimed"`
	CreatedAt            time.Time `json:"created_at"`
}

// GetCurrentTournament returns the current active tournament
