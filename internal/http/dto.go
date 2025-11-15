package httpapi

import "time"

type TeamMemberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type TeamDTO struct {
	TeamName string          `json:"team_name"`
	Members  []TeamMemberDTO `json:"members"`
}

type UserDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type PullRequestDTO struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"` // "OPEN" / "MERGED"
	AssignedReviewers []string `json:"assigned_reviewers"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	MergedAt  *time.Time `json:"mergedAt,omitempty"`
}

type PullRequestShortDTO struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type UserStatsDTO struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	TeamName    string `json:"team_name"`
	ReviewCount int64  `json:"review_count"`
}

type PRStatsDTO struct {
	PullRequestID string `json:"pull_request_id"`
	ReviewerCount int64  `json:"reviewer_count"`
}

type StatsResponseDTO struct {
	ByUser []UserStatsDTO `json:"by_user"`
	ByPR   []PRStatsDTO   `json:"by_pr"`
}
