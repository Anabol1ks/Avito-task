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

func TestPRService_FullWorkflow(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	teamService := service.NewTeamService(repo, log)
	prService := service.NewPRService(repo, log)

	ctx := context.Background()

	// Шаг 1: Создаем команду с 4 пользователями (чтобы было кого переназначать)
	teamInput := service.CreateTeamInput{
		TeamName: "backend",
		Members: []service.CreateTeamMemberInput{
			{UserID: "alice", Username: "Alice", IsActive: true},
			{UserID: "bob", Username: "Bob", IsActive: true},
			{UserID: "charlie", Username: "Charlie", IsActive: true},
			{UserID: "dave", Username: "Dave", IsActive: true},
		},
	}

	teamResult, err := teamService.AddTeam(ctx, teamInput)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	if len(teamResult.Members) != 4 {
		t.Errorf("Expected 4 team members, got %d", len(teamResult.Members))
	}

	// Шаг 2: Создаем PR от alice (автор не должен быть рецензентом)
	prInput := service.CreatePRInput{
		ID:       "PR-001",
		Name:     "Feature: Add new API endpoint",
		AuthorID: "alice",
	}

	prResult, err := prService.CreateWithAutoAssign(ctx, prInput)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	if prResult.PR.ID != "PR-001" {
		t.Errorf("Expected PR ID 'PR-001', got %s", prResult.PR.ID)
	}

	if prResult.PR.Status != models.PRStatusOpen {
		t.Errorf("Expected PR status OPEN, got %s", prResult.PR.Status)
	}

	// Проверяем, что назначено 0-2 рецензента (максимум 2, так как автор исключен)
	reviewerCount := len(prResult.Reviewers)
	if reviewerCount < 0 || reviewerCount > 2 {
		t.Errorf("Expected 0-2 reviewers, got %d", reviewerCount)
	}

	// Проверяем, что автор не является рецензентом
	for _, reviewer := range prResult.Reviewers {
		if reviewer.ID == "alice" {
			t.Error("Author should not be assigned as reviewer")
		}
	}

	t.Logf("PR created with %d reviewer(s)", reviewerCount)

	// Если есть рецензенты, проверяем  переназначение
	if reviewerCount > 0 {
		// Шаг 3: Переназначаем рецензента
		oldReviewerID := prResult.Reviewers[0].ID
		t.Logf("Attempting to reassign reviewer %s", oldReviewerID)

		reassignInput := service.ReassignInput{
			PRID:          "PR-001",
			OldReviewerID: oldReviewerID,
		}

		reassignResult, err := prService.ReassignReviewer(ctx, reassignInput)
		if err != nil {
			t.Fatalf("Failed to reassign reviewer: %v", err)
		}

		if reassignResult.ReplacedByID == "" {
			t.Error("Expected new reviewer to be assigned")
		}

		if reassignResult.ReplacedByID == oldReviewerID {
			t.Error("New reviewer should be different from old reviewer")
		}

		if reassignResult.ReplacedByID == "alice" {
			t.Error("Author should not be assigned as new reviewer")
		}

		t.Logf("Reviewer reassigned from %s to %s", oldReviewerID, reassignResult.ReplacedByID)

		// Проверяем, что список рецензентов изменился
		reviewers, err := prService.GetReviewersForPR(ctx, "PR-001")
		if err != nil {
			t.Fatalf("Failed to get reviewers: %v", err)
		}

		foundNew := false
		foundOld := false
		for _, r := range reviewers {
			if r.ReviewerID == reassignResult.ReplacedByID {
				foundNew = true
			}
			if r.ReviewerID == oldReviewerID {
				foundOld = true
			}
		}

		if !foundNew {
			t.Error("New reviewer not found in PR reviewers list")
		}

		if foundOld {
			t.Error("Old reviewer should not be in PR reviewers list after reassignment")
		}
	}

	// Шаг 4: Мержим PR
	mergedPR, err := prService.Merge(ctx, "PR-001")
	if err != nil {
		t.Fatalf("Failed to merge PR: %v", err)
	}

	if mergedPR.Status != models.PRStatusMerged {
		t.Errorf("Expected PR status MERGED, got %s", mergedPR.Status)
	}

	if mergedPR.MergedAt == nil {
		t.Error("Expected MergedAt to be set")
	}

	t.Logf("PR merged successfully")

	if reviewerCount > 0 {
		reassignInput := service.ReassignInput{
			PRID:          "PR-001",
			OldReviewerID: prResult.Reviewers[0].ID,
		}

		_, err = prService.ReassignReviewer(ctx, reassignInput)
		if err == nil {
			t.Error("Expected error when reassigning reviewer for merged PR, got nil")
		}

		if serr, ok := err.(*service.Error); ok {
			if serr.Code != service.ErrorCodePRMerged {
				t.Errorf("Expected ErrorCodePRMerged, got %s", serr.Code)
			}
			t.Logf("Reassignment after merge correctly rejected: %s", serr.Msg)
		} else {
			t.Errorf("Expected service.Error, got %T", err)
		}
	}

	// Пытаемся мержить второй раз (должна быть ошибка)
	_, err = prService.Merge(ctx, "PR-001")
	if err == nil {
		t.Error("Expected error when merging already merged PR, got nil")
	}

	if serr, ok := err.(*service.Error); ok {
		if serr.Code != service.ErrorCodePRMerged {
			t.Errorf("Expected ErrorCodePRMerged, got %s", serr.Code)
		}
		t.Logf("Double merge correctly rejected: %s", serr.Msg)
	}
}

// TestPRService_UserDeactivationRemovesFromReviews - тест деактивации пользователя
// Этот тест проверяет, что деактивированный пользователь:
// 1. Не назначается новым рецензентом
// 2. Может быть переназначен с активного PR
func TestPRService_UserDeactivationRemovesFromReviews(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	teamService := service.NewTeamService(repo, log)
	userService := service.NewUserService(repo, log)
	prService := service.NewPRService(repo, log)

	ctx := context.Background()

	// Создаем команду с 4 пользователями для лучшего тестирования
	teamInput := service.CreateTeamInput{
		TeamName: "qa-team",
		Members: []service.CreateTeamMemberInput{
			{UserID: "dave", Username: "Dave", IsActive: true},
			{UserID: "eve", Username: "Eve", IsActive: true},
			{UserID: "frank", Username: "Frank", IsActive: true},
			{UserID: "grace", Username: "Grace", IsActive: true},
		},
	}

	_, err := teamService.AddTeam(ctx, teamInput)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Создаем PR от dave
	prInput := service.CreatePRInput{
		ID:       "PR-100",
		Name:     "Bug fix",
		AuthorID: "dave",
	}

	prResult, err := prService.CreateWithAutoAssign(ctx, prInput)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	initialReviewerCount := len(prResult.Reviewers)
	t.Logf("Initial reviewers: %d", initialReviewerCount)

	// Деактивируем одного из рецензентов (если есть)
	if initialReviewerCount > 0 {
		reviewerToDeactivate := prResult.Reviewers[0].ID
		t.Logf("Deactivating reviewer: %s", reviewerToDeactivate)

		_, err = userService.SetIsActive(ctx, reviewerToDeactivate, false)
		if err != nil {
			t.Fatalf("Failed to deactivate user: %v", err)
		}

		// Переназначаем деактивированного рецензента
		reassignInput := service.ReassignInput{
			PRID:          "PR-100",
			OldReviewerID: reviewerToDeactivate,
		}

		reassignResult, err := prService.ReassignReviewer(ctx, reassignInput)
		if err != nil {
			t.Fatalf("Failed to reassign deactivated reviewer: %v", err)
		}

		// Новый рецензент должен быть активным
		newReviewer, err := userService.GetUser(ctx, reassignResult.ReplacedByID)
		if err != nil {
			t.Fatalf("Failed to get new reviewer: %v", err)
		}

		if !newReviewer.IsActive {
			t.Error("New reviewer should be active")
		}

		if newReviewer.ID == reviewerToDeactivate {
			t.Error("New reviewer should be different from deactivated one")
		}

		t.Logf("Deactivated reviewer %s replaced with active reviewer %s", reviewerToDeactivate, newReviewer.ID)
	}

	// Создаем новый PR и проверяем, что деактивированный пользователь не назначается
	prInput2 := service.CreatePRInput{
		ID:       "PR-101",
		Name:     "Another feature",
		AuthorID: "dave",
	}

	prResult2, err := prService.CreateWithAutoAssign(ctx, prInput2)
	if err != nil {
		t.Fatalf("Failed to create second PR: %v", err)
	}

	// Проверяем, что среди новых рецензентов нет деактивированных
	for _, reviewer := range prResult2.Reviewers {
		user, err := userService.GetUser(ctx, reviewer.ID)
		if err != nil {
			t.Fatalf("Failed to get reviewer user: %v", err)
		}

		if !user.IsActive {
			t.Errorf("Deactivated user %s should not be assigned as reviewer", user.ID)
		}
	}

	t.Logf("New PR created without deactivated users as reviewers")
}

// TestPRService_NoReviewersAvailable - тест когда нет доступных рецензентов
func TestPRService_NoReviewersAvailable(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	teamService := service.NewTeamService(repo, log)
	prService := service.NewPRService(repo, log)

	ctx := context.Background()

	// Создаем команду только с одним пользователем
	teamInput := service.CreateTeamInput{
		TeamName: "solo-team",
		Members: []service.CreateTeamMemberInput{
			{UserID: "solo", Username: "Solo", IsActive: true},
		},
	}

	_, err := teamService.AddTeam(ctx, teamInput)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Создаем PR от единственного пользователя
	prInput := service.CreatePRInput{
		ID:       "PR-200",
		Name:     "Solo PR",
		AuthorID: "solo",
	}

	prResult, err := prService.CreateWithAutoAssign(ctx, prInput)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	// Должно быть 0 рецензентов (автор исключен, других нет)
	if len(prResult.Reviewers) != 0 {
		t.Errorf("Expected 0 reviewers for solo team, got %d", len(prResult.Reviewers))
	}

	t.Logf("PR created with 0 reviewers when no other team members available")
}

// TestPRService_DuplicatePR - тест создания дублирующего PR
func TestPRService_DuplicatePR(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	prService := service.NewPRService(repo, log)

	ctx := context.Background()

	// Создаем команду
	testhelpers.CreateTestTeam(t, db, "dev-team", 3)

	// Создаем PR
	prInput := service.CreatePRInput{
		ID:       "PR-300",
		Name:     "Feature X",
		AuthorID: "dev-team-user-A",
	}

	_, err := prService.CreateWithAutoAssign(ctx, prInput)
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	// Пытаемся создать PR с тем же ID
	_, err = prService.CreateWithAutoAssign(ctx, prInput)
	if err == nil {
		t.Error("Expected error when creating duplicate PR, got nil")
	}

	if serr, ok := err.(*service.Error); ok {
		if serr.Code != service.ErrorCodePRExists {
			t.Errorf("Expected ErrorCodePRExists, got %s", serr.Code)
		}
		t.Logf("Duplicate PR correctly rejected: %s", serr.Msg)
	}
}

// TestPRService_GetReviewsByUser - тест получения всех PR назначенных пользователю
func TestPRService_GetReviewsByUser(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()

	teamService := service.NewTeamService(repo, log)
	prService := service.NewPRService(repo, log)

	ctx := context.Background()

	// Создаем команду
	teamInput := service.CreateTeamInput{
		TeamName: "review-team",
		Members: []service.CreateTeamMemberInput{
			{UserID: "reviewer1", Username: "Reviewer1", IsActive: true},
			{UserID: "reviewer2", Username: "Reviewer2", IsActive: true},
			{UserID: "author1", Username: "Author1", IsActive: true},
		},
	}

	_, err := teamService.AddTeam(ctx, teamInput)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Создаем несколько PR с явным назначением рецензентов
	// (для этого нужно создать PR и добавить рецензентов через репозиторий)
	pr1 := &models.PullRequest{
		ID:       "PR-401",
		Name:     "PR 1",
		AuthorID: "author1",
		Status:   models.PRStatusOpen,
	}
	if err := repo.PRs.Create(ctx, pr1); err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}
	if err := repo.PRs.AddReviewers(ctx, "PR-401", []string{"reviewer1"}); err != nil {
		t.Fatalf("Failed to add reviewers: %v", err)
	}

	pr2 := &models.PullRequest{
		ID:       "PR-402",
		Name:     "PR 2",
		AuthorID: "author1",
		Status:   models.PRStatusOpen,
	}
	if err := repo.PRs.Create(ctx, pr2); err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}
	if err := repo.PRs.AddReviewers(ctx, "PR-402", []string{"reviewer1", "reviewer2"}); err != nil {
		t.Fatalf("Failed to add reviewers: %v", err)
	}

	// Получаем все PR для reviewer1
	prs, err := prService.GetReviewsByUser(ctx, "reviewer1")
	if err != nil {
		t.Fatalf("Failed to get reviews by user: %v", err)
	}

	if len(prs) != 2 {
		t.Errorf("Expected 2 PRs for reviewer1, got %d", len(prs))
	}

	// Получаем все PR для reviewer2
	prs, err = prService.GetReviewsByUser(ctx, "reviewer2")
	if err != nil {
		t.Fatalf("Failed to get reviews by user: %v", err)
	}

	if len(prs) != 1 {
		t.Errorf("Expected 1 PR for reviewer2, got %d", len(prs))
	}

	t.Logf("Successfully retrieved PRs by reviewer")
}
