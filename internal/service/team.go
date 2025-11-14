package service

import (
	"context"
	"errors"
	"reviewer_pr/internal/models"
	"reviewer_pr/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TeamService interface {
	AddTeam(ctx context.Context, in CreateTeamInput) (*TeamWithMembers, error)
}

type teamService struct {
	repo *repository.Repository
	log  *zap.Logger
}

func NewTeamService(repo *repository.Repository, log *zap.Logger) TeamService {
	return &teamService{repo: repo, log: log}
}

type CreateTeamInput struct {
	TeamName string
	Members  []CreateTeamMemberInput
}

type CreateTeamMemberInput struct {
	UserID   string
	Username string
	IsActive bool
}

type TeamWithMembers struct {
	Team    *models.Team
	Members []models.User
}

func (s *teamService) AddTeam(ctx context.Context, in CreateTeamInput) (*TeamWithMembers, error) {
	var result *TeamWithMembers

	err := s.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := s.repo.Teams.GetTeamByName(ctx, in.TeamName)
		if err != nil && existing != nil {
			return NewErr(ErrorCodeTeamExists, "team already exists")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		team := &models.Team{
			Name: in.TeamName,
		}

		if err := s.repo.Teams.Create(ctx, team); err != nil {
			return err
		}

		members := make([]models.User, 0, len(in.Members))
		for _, m := range in.Members {
			u := &models.User{
				ID:       m.UserID,
				Username: m.Username,
				TeamName: in.TeamName,
				IsActive: m.IsActive,
			}
			if err := s.repo.Users.UpsertUser(ctx, u); err != nil {
				return err
			}
			members = append(members, *u)
		}

		result = &TeamWithMembers{
			Team:    team,
			Members: members,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *teamService) GetTeam(ctx context.Context, teamName string) (*TeamWithMembers, error) {
	team, err := s.repo.Teams.GetTeamByName(ctx, teamName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewErr(ErrorCodeNotFound, "team not found")
		}
		return nil, err
	}

	users, err := s.repo.Teams.GetTeamMembers(ctx, teamName)
	if err != nil {
		return nil, err
	}

	return &TeamWithMembers{
		Team:    team,
		Members: users,
	}, nil
}
