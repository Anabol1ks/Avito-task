package repository

import (
	"context"
	"reviewer_pr/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UsersRepo interface {
	UpsertUser(ctx context.Context, u *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	SetUserActive(ctx context.Context, id string, active bool) error
	GetActiveTeamMembersExcept(ctx context.Context, teamName, exceptUserID string) ([]models.User, error)
}

type usersRepo struct {
	db *gorm.DB
}

func NewUsersRepo(db *gorm.DB) UsersRepo {
	return &usersRepo{db: db}
}

func (r *usersRepo) UpsertUser(ctx context.Context, u *models.User) error {
	return r.db.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"username", "team_name", "is_active", "updated_at"}),
		},
	).Create(u).Error
}

func (r *usersRepo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Where("user_id = ?", id).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *usersRepo) SetUserActive(ctx context.Context, id string, active bool) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("user_id = ?", id).Update("is_active", active).Error
}

func (r *usersRepo) GetActiveTeamMembersExcept(ctx context.Context, teamName, exceptUserID string) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Where("team_name = ? AND is_active = TRUE AND user_id <> ?", teamName, exceptUserID).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
