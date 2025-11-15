package service_test

import (
	"context"
	"reviewer_pr/internal/models"
	"reviewer_pr/internal/repository"
	"reviewer_pr/internal/service"
	"reviewer_pr/internal/testhelpers"
	"testing"

	"go.uber.org/zap"
)

func TestStatsService_GetStats(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	teamService := service.NewTeamService(repo, log)
	statsService := service.NewStatsService(repo, log)

	ctx := context.Background()

	// Создаем команду с пользователями
	teamInput := service.CreateTeamInput{
		TeamName: "stats-team",
		Members: []service.CreateTeamMemberInput{
			{UserID: "user1", Username: "User1", IsActive: true},
			{UserID: "user2", Username: "User2", IsActive: true},
			{UserID: "user3", Username: "User3", IsActive: true},
		},
	}

	_, err := teamService.AddTeam(ctx, teamInput)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Создаем несколько PR с рецензентами
	pr1 := &models.PullRequest{
		ID:       "PR-501",
		Name:     "PR 1",
		AuthorID: "user1",
		Status:   models.PRStatusOpen,
	}
	if err := repo.PRs.Create(ctx, pr1); err != nil {
		t.Fatalf("Failed to create PR1: %v", err)
	}
	if err := repo.PRs.AddReviewers(ctx, "PR-501", []string{"user2", "user3"}); err != nil {
		t.Fatalf("Failed to add reviewers to PR1: %v", err)
	}

	pr2 := &models.PullRequest{
		ID:       "PR-502",
		Name:     "PR 2",
		AuthorID: "user2",
		Status:   models.PRStatusOpen,
	}
	if err := repo.PRs.Create(ctx, pr2); err != nil {
		t.Fatalf("Failed to create PR2: %v", err)
	}
	if err := repo.PRs.AddReviewers(ctx, "PR-502", []string{"user1"}); err != nil {
		t.Fatalf("Failed to add reviewers to PR2: %v", err)
	}

	pr3 := &models.PullRequest{
		ID:       "PR-503",
		Name:     "PR 3",
		AuthorID: "user3",
		Status:   models.PRStatusMerged,
	}
	if err := repo.PRs.Create(ctx, pr3); err != nil {
		t.Fatalf("Failed to create PR3: %v", err)
	}
	if err := repo.PRs.AddReviewers(ctx, "PR-503", []string{"user1", "user2"}); err != nil {
		t.Fatalf("Failed to add reviewers to PR3: %v", err)
	}

	// Получаем статистику
	stats, err := statsService.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Проверяем статистику по пользователям
	if len(stats.ByUser) == 0 {
		t.Error("Expected user stats, got empty list")
	}

	// Подсчитываем ожидаемое количество ревью для каждого пользователя
	// user1: рецензент в PR-502 и PR-503 = 2
	// user2: рецензент в PR-501 и PR-503 = 2
	// user3: рецензент в PR-501 = 1
	expectedReviews := map[string]int64{
		"user1": 2,
		"user2": 2,
		"user3": 1,
	}

	for _, userStat := range stats.ByUser {
		expected, ok := expectedReviews[userStat.UserID]
		if !ok {
			t.Errorf("Unexpected user in stats: %s", userStat.UserID)
			continue
		}

		if userStat.ReviewCount != expected {
			t.Errorf("User %s: expected %d reviews, got %d", userStat.UserID, expected, userStat.ReviewCount)
		}

		if userStat.TeamName != "stats-team" {
			t.Errorf("User %s: expected team 'stats-team', got %s", userStat.UserID, userStat.TeamName)
		}
	}

	// Проверяем статистику по PR
	if len(stats.ByPR) != 3 {
		t.Errorf("Expected 3 PRs in stats, got %d", len(stats.ByPR))
	}

	expectedPRReviewers := map[string]int64{
		"PR-501": 2,
		"PR-502": 1,
		"PR-503": 2,
	}

	for _, prStat := range stats.ByPR {
		expected, ok := expectedPRReviewers[prStat.PullRequestID]
		if !ok {
			t.Errorf("Unexpected PR in stats: %s", prStat.PullRequestID)
			continue
		}

		if prStat.ReviewerCount != expected {
			t.Errorf("PR %s: expected %d reviewers, got %d", prStat.PullRequestID, expected, prStat.ReviewerCount)
		}
	}

	t.Logf("Stats returned correctly")
	t.Logf("  - User stats: %d users", len(stats.ByUser))
	t.Logf("  - PR stats: %d PRs", len(stats.ByPR))
}

func TestStatsService_EmptyStats(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	statsService := service.NewStatsService(repo, log)

	ctx := context.Background()

	// Получаем статистику для пустой БД
	stats, err := statsService.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if len(stats.ByUser) != 0 {
		t.Errorf("Expected 0 user stats, got %d", len(stats.ByUser))
	}

	if len(stats.ByPR) != 0 {
		t.Errorf("Expected 0 PR stats, got %d", len(stats.ByPR))
	}

	t.Logf("Empty stats handled correctly")
}

func TestStatsService_OnlyPRsWithoutReviewers(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	teamService := service.NewTeamService(repo, log)
	statsService := service.NewStatsService(repo, log)

	ctx := context.Background()

	// Создаем команду с одним пользователем
	teamInput := service.CreateTeamInput{
		TeamName: "solo-stats-team",
		Members: []service.CreateTeamMemberInput{
			{UserID: "solo-user", Username: "SoloUser", IsActive: true},
		},
	}

	_, err := teamService.AddTeam(ctx, teamInput)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Создаем PR без рецензентов
	pr := &models.PullRequest{
		ID:       "PR-600",
		Name:     "Solo PR",
		AuthorID: "solo-user",
		Status:   models.PRStatusOpen,
	}
	if err := repo.PRs.Create(ctx, pr); err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	// Получаем статистику
	stats, err := statsService.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Не должно быть статистики по пользователям (нет рецензентов)
	if len(stats.ByUser) != 0 {
		t.Errorf("Expected 0 user stats (no reviewers), got %d", len(stats.ByUser))
	}

	// PR без рецензентов не попадает в статистику (GetPRReviewStats возвращает только PR с рецензентами)
	if len(stats.ByPR) != 0 {
		t.Errorf("Expected 0 PR stats (PR without reviewers not included), got %d", len(stats.ByPR))
	}

	t.Logf("Stats for PRs without reviewers handled correctly (not included in stats)")
}
