package models

import "time"

type Team struct {
	Name      string    `gorm:"column:team_name;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Users []User `gorm:"foreignKey:TeamName;references:Name"`
}

func (Team) TableName() string {
	return "teams"
}

type User struct {
	ID        string    `gorm:"column:user_id;primaryKey"`
	Username  string    `gorm:"column:username;not null"`
	TeamName  string    `gorm:"column:team_name;not null;index"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Team *Team `gorm:"foreignKey:TeamName;references:Name"`
}

func (User) TableName() string {
	return "users"
}

type PullRequestStatus string

const (
	PRStatusOpen   PullRequestStatus = "OPEN"
	PRStatusMerged PullRequestStatus = "MERGED"
)

type PullRequest struct {
	ID        string            `gorm:"column:pull_request_id;primaryKey"`
	Name      string            `gorm:"column:pull_request_name;not null"`
	AuthorID  string            `gorm:"column:author_id;not null;index"`
	Status    PullRequestStatus `gorm:"column:status;type:text;not null;default:'OPEN'"`
	CreatedAt time.Time         `gorm:"column:created_at;autoCreateTime"`
	MergedAt  *time.Time        `gorm:"column:merged_at"`

	Author    *User        `gorm:"foreignKey:AuthorID;references:ID"`
	Reviewers []PRReviewer `gorm:"foreignKey:PullRequestID;references:ID"`
}

func (PullRequest) TableName() string {
	return "pull_requests"
}

type PRReviewer struct {
	PullRequestID string    `gorm:"column:pull_request_id;primaryKey"`
	ReviewerID    string    `gorm:"column:reviewer_id;primaryKey;index"`
	AssignedAt    time.Time `gorm:"column:assigned_at;autoCreateTime"`

	PullRequest *PullRequest `gorm:"foreignKey:PullRequestID;references:ID"`
	Reviewer    *User        `gorm:"foreignKey:ReviewerID;references:ID"`
}

func (PRReviewer) TableName() string {
	return "pr_reviewers"
}
