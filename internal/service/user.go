package service

import (
	"context"
	"errors"
	"reviewer_pr/internal/models"
	"reviewer_pr/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*models.User, error)
	GetUser(ctx context.Context, userID string) (*models.User, error)
}

type userService struct {
	repo *repository.Repository
	log  *zap.Logger
}

func NewUserService(repo *repository.Repository, log *zap.Logger) UserService {
	return &userService{repo: repo, log: log}
}

func (s *userService) SetIsActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	u, err := s.repo.Users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewErr(ErrorCodeNotFound, "user not found")
		}
		return nil, err
	}

	if err := s.repo.Users.SetUserActive(ctx, userID, isActive); err != nil {
		return nil, err
	}
	u.IsActive = isActive
	return u, nil
}

func (s *userService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	u, err := s.repo.Users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewErr(ErrorCodeNotFound, "user not found")
		}
		return nil, err
	}
	return u, nil
}
