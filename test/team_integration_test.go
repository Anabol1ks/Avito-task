package service_test

import (
	"context"
	"reviewer_pr/internal/repository"
	"reviewer_pr/internal/service"
	"reviewer_pr/internal/testhelpers"
	"testing"

	"go.uber.org/zap"
)

func TestTeamService_AddTeam(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	teamService := service.NewTeamService(repo, log)

	ctx := context.Background()

	input := service.CreateTeamInput{
		TeamName: "backend",
		Members: []service.CreateTeamMemberInput{
			{UserID: "user1", Username: "Alice", IsActive: true},
			{UserID: "user2", Username: "Bob", IsActive: true},
			{UserID: "user3", Username: "Charlie", IsActive: false},
		},
	}

	result, err := teamService.AddTeam(ctx, input)
	if err != nil {
		t.Fatalf("AddTeam failed: %v", err)
	}

	if result.Team.Name != "backend" {
		t.Errorf("expected team name 'backend', got %s", result.Team.Name)
	}

	if len(result.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(result.Members))
	}

	// Проверяем, что второй вызов с тем же именем команды возвращает ошибку
	_, err = teamService.AddTeam(ctx, input)
	if err == nil {
		t.Error("expected error when adding duplicate team, got nil")
	}
	if serr, ok := err.(*service.Error); ok {
		if serr.Code != service.ErrorCodeTeamExists {
			t.Errorf("expected ErrorCodeTeamExists, got %s", serr.Code)
		}
	}
}

func TestTeamService_GetTeam(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	teamService := service.NewTeamService(repo, log)

	ctx := context.Background()

	// Создаем команду
	input := service.CreateTeamInput{
		TeamName: "frontend",
		Members: []service.CreateTeamMemberInput{
			{UserID: "user1", Username: "Alice", IsActive: true},
			{UserID: "user2", Username: "Bob", IsActive: true},
		},
	}

	_, err := teamService.AddTeam(ctx, input)
	if err != nil {
		t.Fatalf("AddTeam failed: %v", err)
	}

	// Получаем команду
	result, err := teamService.GetTeam(ctx, "frontend")
	if err != nil {
		t.Fatalf("GetTeam failed: %v", err)
	}

	if result.Team.Name != "frontend" {
		t.Errorf("expected team name 'frontend', got %s", result.Team.Name)
	}

	if len(result.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(result.Members))
	}

	// Проверяем несуществующую команду
	_, err = teamService.GetTeam(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error when getting nonexistent team, got nil")
	}
	if serr, ok := err.(*service.Error); ok {
		if serr.Code != service.ErrorCodeNotFound {
			t.Errorf("expected ErrorCodeNotFound, got %s", serr.Code)
		}
	}
}
