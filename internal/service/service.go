package service

import (
	"reviewer_pr/internal/repository"

	"go.uber.org/zap"
)

type Services struct {
	Teams TeamService
	Users UserService
	PRs   PRService
}

func New(repo *repository.Repository, log *zap.Logger) *Services {
	return buildServices(repo, log)
}

func buildServices(repo *repository.Repository, log *zap.Logger) *Services {
	return &Services{
		Teams: NewTeamService(repo, log),
		Users: NewUserService(repo, log),
		PRs:   NewPRService(repo, log),
	}
}
