package service

import (
	"context"
	"reviewer_pr/internal/repository"

	"go.uber.org/zap"
)

type StatsService interface {
	GetStats(ctx context.Context) (*Stats, error)
}

type Stats struct {
	ByUser []UserStats
	ByPR   []PRStats
}

type UserStats struct {
	UserID      string
	Username    string
	TeamName    string
	ReviewCount int64
}

type PRStats struct {
	PullRequestID string
	ReviewerCount int64
}

type statsService struct {
	repo *repository.Repository
	log  *zap.Logger
}

func NewStatsService(repo *repository.Repository, log *zap.Logger) StatsService {
	return &statsService{repo: repo, log: log}
}

func (s *statsService) GetStats(ctx context.Context) (*Stats, error) {
	userStats, err := s.repo.PRs.GetUserReviewStats(ctx)
	if err != nil {
		return nil, err
	}

	prStats, err := s.repo.PRs.GetPRReviewStats(ctx)
	if err != nil {
		return nil, err
	}

	res := &Stats{
		ByUser: make([]UserStats, 0, len(userStats)),
		ByPR:   make([]PRStats, 0, len(prStats)),
	}

	for _, u := range userStats {
		res.ByUser = append(res.ByUser, UserStats{
			UserID:      u.UserID,
			Username:    u.Username,
			TeamName:    u.TeamName,
			ReviewCount: u.ReviewCount,
		})
	}

	for _, p := range prStats {
		res.ByPR = append(res.ByPR, PRStats{
			PullRequestID: p.PullRequestID,
			ReviewerCount: p.ReviewerCount,
		})
	}

	return res, nil
}
