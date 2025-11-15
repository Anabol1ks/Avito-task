package service_test

import (
	"context"
	"reviewer_pr/internal/repository"
	"reviewer_pr/internal/service"
	"reviewer_pr/internal/testhelpers"
	"testing"

	"go.uber.org/zap"
)

func TestUserService_SetIsActive(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	userService := service.NewUserService(repo, log)

	ctx := context.Background()

	// Создаем тестовых пользователей
	users := testhelpers.CreateTestTeam(t, db, "backend", 3)

	// Деактивируем пользователя
	user, err := userService.SetIsActive(ctx, users[0].ID, false)
	if err != nil {
		t.Fatalf("SetIsActive failed: %v", err)
	}

	if user.IsActive {
		t.Error("expected user to be inactive")
	}

	// Активируем обратно
	user, err = userService.SetIsActive(ctx, users[0].ID, true)
	if err != nil {
		t.Fatalf("SetIsActive failed: %v", err)
	}

	if !user.IsActive {
		t.Error("expected user to be active")
	}

	// Проверяем несуществующего пользователя
	_, err = userService.SetIsActive(ctx, "nonexistent", true)
	if err == nil {
		t.Error("expected error when setting status for nonexistent user, got nil")
	}
	if serr, ok := err.(*service.Error); ok {
		if serr.Code != service.ErrorCodeNotFound {
			t.Errorf("expected ErrorCodeNotFound, got %s", serr.Code)
		}
	}
}

func TestUserService_GetUser(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	userService := service.NewUserService(repo, log)

	ctx := context.Background()

	// Создаем тестовых пользователей
	users := testhelpers.CreateTestTeam(t, db, "frontend", 2)

	// Получаем пользователя
	user, err := userService.GetUser(ctx, users[0].ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if user.ID != users[0].ID {
		t.Errorf("expected user ID %s, got %s", users[0].ID, user.ID)
	}

	if user.TeamName != "frontend" {
		t.Errorf("expected team name 'frontend', got %s", user.TeamName)
	}

	// Проверяем несуществующего пользователя
	_, err = userService.GetUser(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error when getting nonexistent user, got nil")
	}
	if serr, ok := err.(*service.Error); ok {
		if serr.Code != service.ErrorCodeNotFound {
			t.Errorf("expected ErrorCodeNotFound, got %s", serr.Code)
		}
	}
}
