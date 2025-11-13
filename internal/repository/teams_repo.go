package repository

import (
	"context"
	"reviewer_pr/internal/models"

	"gorm.io/gorm"
)

type TeamsRepo interface {
	Create(ctx context.Context, team *models.Team) error
	GetTeamByName(ctx context.Context, name string) (*models.Team, error)
	GetTeamMembers(ctx context.Context, teamName string) ([]models.User, error)
}

type teamsRepo struct {
	db *gorm.DB
}

func NewTeamsRepo(db *gorm.DB) TeamsRepo {
	return &teamsRepo{db: db}
}

func (r *teamsRepo) Create(ctx context.Context, team *models.Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

func (r *teamsRepo) GetTeamByName(ctx context.Context, name string) (*models.Team, error) {
	var team models.Team
	err := r.db.WithContext(ctx).Where("team_name = ?", name).First(&team).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *teamsRepo) GetTeamMembers(ctx context.Context, teamName string) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Where("team_name = ?", teamName).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
