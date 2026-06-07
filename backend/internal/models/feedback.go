package models

import (
	"time"

	"github.com/google/uuid"
)

type Feedback struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	Username          string    `json:"username,omitempty"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`           // 'BUG', 'GU', 'ITEM', 'RECIPE', 'BALANCE', 'FEATURE', 'OTHER'
	Status            string    `json:"status"`             // 'pending', 'active', 'in_progress', 'completed', 'rejected'
	TotalVotes        int       `json:"total_votes"`
	DeveloperResponse *string   `json:"developer_response"`
	Rewarded          bool      `json:"rewarded"`
	CreatedAt         time.Time `json:"created_at"`
	UserVote          *int      `json:"user_vote,omitempty"` // 1 for Upvote, -1 for Downvote, nil for none
}

type FeedbackVote struct {
	FeedbackID uuid.UUID `json:"feedback_id"`
	UserID     uuid.UUID `json:"user_id"`
	VoteType   int       `json:"vote_type"` // 1 or -1
}
