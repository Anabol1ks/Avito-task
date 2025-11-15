package service_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	httpapi "reviewer_pr/internal/http"
	"reviewer_pr/internal/repository"
	"reviewer_pr/internal/router"
	"reviewer_pr/internal/service"
	"reviewer_pr/internal/testhelpers"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestHandlers_FullWorkflow - полный сквозной тест через HTTP API
// Проверяет весь стек: HTTP → Handler → Service → Repository → DB
func TestHandlers_FullWorkflow(t *testing.T) {
	// Setup
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	services := service.New(repo, log)
	handler := httpapi.New(services, log)
	r := router.Router(handler)

	// Шаг 1: Создаем команду
	teamPayload := map[string]interface{}{
		"team_name": "backend",
		"members": []map[string]interface{}{
			{"user_id": "alice", "username": "Alice", "is_active": true},
			{"user_id": "bob", "username": "Bob", "is_active": true},
			{"user_id": "charlie", "username": "Charlie", "is_active": true},
			{"user_id": "dave", "username": "Dave", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamPayload)
	req := httptest.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Expected 201 Created")

	var teamResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &teamResp)
	assert.NotNil(t, teamResp["team"], "Expected team in response")

	t.Log("✓ Team created via API")

	// Шаг 2: Получаем команду
	req = httptest.NewRequest("GET", "/team/get?team_name=backend", nil)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var getTeamResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getTeamResp)
	assert.Equal(t, "backend", getTeamResp["team_name"], "Expected team name 'backend'")

	t.Log("✓ Team retrieved via API")

	// Шаг 3: Создаем PR
	prPayload := map[string]interface{}{
		"pull_request_id":   "PR-001",
		"pull_request_name": "Feature: Add new API",
		"author_id":         "alice",
	}

	body, _ = json.Marshal(prPayload)
	req = httptest.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Expected 201 Created")

	var prResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &prResp)
	pr := prResp["pr"].(map[string]interface{})

	assert.Equal(t, "PR-001", pr["pull_request_id"], "Expected PR ID 'PR-001'")
	assert.Equal(t, "OPEN", pr["status"], "Expected status 'OPEN'")

	reviewers := pr["assigned_reviewers"].([]interface{})
	assert.GreaterOrEqual(t, len(reviewers), 0, "Expected 0 or more reviewers")
	assert.LessOrEqual(t, len(reviewers), 2, "Expected max 2 reviewers")

	t.Logf("✓ PR created via API with %d reviewer(s)", len(reviewers))

	// Шаг 4: Переназначаем рецензента (если есть)
	if len(reviewers) > 0 {
		oldReviewerID := reviewers[0].(string)

		reassignPayload := map[string]interface{}{
			"pull_request_id": "PR-001",
			"old_reviewer_id": oldReviewerID,
		}

		body, _ = json.Marshal(reassignPayload)
		req = httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK for reassign")

		var reassignResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &reassignResp)
		reassignedPR := reassignResp["pr"].(map[string]interface{})

		newReviewers := reassignedPR["assigned_reviewers"].([]interface{})

		// Проверяем что старого рецензента нет
		for _, r := range newReviewers {
			assert.NotEqual(t, oldReviewerID, r.(string), "Old reviewer should not be in list")
		}

		t.Logf("✓ Reviewer reassigned via API from %s", oldReviewerID)
	}

	// Шаг 5: Мержим PR
	mergePayload := map[string]interface{}{
		"pull_request_id": "PR-001",
	}

	body, _ = json.Marshal(mergePayload)
	req = httptest.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK for merge")

	var mergeResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &mergeResp)
	mergedPR := mergeResp["pr"].(map[string]interface{})

	assert.Equal(t, "MERGED", mergedPR["status"], "Expected status 'MERGED'")
	assert.NotNil(t, mergedPR["mergedAt"], "Expected mergedAt to be set")

	t.Log("✓ PR merged via API")

	// Шаг 6: Пытаемся переназначить после merge (должна быть ошибка)
	if len(reviewers) > 0 {
		oldReviewerID := reviewers[0].(string)

		reassignPayload := map[string]interface{}{
			"pull_request_id": "PR-001",
			"old_reviewer_id": oldReviewerID,
		}

		body, _ = json.Marshal(reassignPayload)
		req = httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code, "Expected 409 Conflict for reassign after merge")

		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		errorBody := errResp["error"].(map[string]interface{})

		assert.Equal(t, "PR_MERGED", errorBody["code"], "Expected error code 'PR_MERGED'")

		t.Log("✓ Reassignment after merge correctly rejected via API")
	}
}

// TestHandlers_UserDeactivation - тест деактивации пользователя через API
func TestHandlers_UserDeactivation(t *testing.T) {
	// Setup
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	services := service.New(repo, log)
	handler := httpapi.New(services, log)
	r := router.Router(handler)

	// Создаем команду
	teamPayload := map[string]interface{}{
		"team_name": "qa-team",
		"members": []map[string]interface{}{
			{"user_id": "eve", "username": "Eve", "is_active": true},
			{"user_id": "frank", "username": "Frank", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamPayload)
	req := httptest.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Деактивируем пользователя
	deactivatePayload := map[string]interface{}{
		"user_id":   "eve",
		"is_active": false,
	}

	body, _ = json.Marshal(deactivatePayload)
	req = httptest.NewRequest("POST", "/users/setIsActive", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var userResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &userResp)
	user := userResp["user"].(map[string]interface{})

	assert.Equal(t, "eve", user["user_id"], "Expected user_id 'eve'")
	assert.Equal(t, false, user["is_active"], "Expected is_active to be false")

	t.Log("✓ User deactivated via API")

	// Активируем обратно
	activatePayload := map[string]interface{}{
		"user_id":   "eve",
		"is_active": true,
	}

	body, _ = json.Marshal(activatePayload)
	req = httptest.NewRequest("POST", "/users/setIsActive", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	json.Unmarshal(w.Body.Bytes(), &userResp)
	user = userResp["user"].(map[string]interface{})

	assert.Equal(t, true, user["is_active"], "Expected is_active to be true")

	t.Log("✓ User activated via API")
}

// TestHandlers_GetStats - тест получения статистики через API
func TestHandlers_GetStats(t *testing.T) {
	// Setup
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	services := service.New(repo, log)
	handler := httpapi.New(services, log)
	r := router.Router(handler)

	// Создаем команду
	teamPayload := map[string]interface{}{
		"team_name": "stats-team",
		"members": []map[string]interface{}{
			{"user_id": "user1", "username": "User1", "is_active": true},
			{"user_id": "user2", "username": "User2", "is_active": true},
			{"user_id": "user3", "username": "User3", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamPayload)
	req := httptest.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Создаем PR
	prPayload := map[string]interface{}{
		"pull_request_id":   "PR-100",
		"pull_request_name": "Feature",
		"author_id":         "user1",
	}

	body, _ = json.Marshal(prPayload)
	req = httptest.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Получаем статистику
	req = httptest.NewRequest("GET", "/stats", nil)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var statsResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &statsResp)

	assert.NotNil(t, statsResp["by_user"], "Expected by_user in response")
	assert.NotNil(t, statsResp["by_pr"], "Expected by_pr in response")

	t.Log("✓ Stats retrieved via API")
}

// TestHandlers_GetUserReviews - тест получения PR для пользователя через API
func TestHandlers_GetUserReviews(t *testing.T) {
	// Setup
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	services := service.New(repo, log)
	handler := httpapi.New(services, log)
	r := router.Router(handler)

	// Создаем команду
	teamPayload := map[string]interface{}{
		"team_name": "review-team",
		"members": []map[string]interface{}{
			{"user_id": "reviewer1", "username": "Reviewer1", "is_active": true},
			{"user_id": "author1", "username": "Author1", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamPayload)
	req := httptest.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Создаем PR от author1
	prPayload := map[string]interface{}{
		"pull_request_id":   "PR-200",
		"pull_request_name": "Feature X",
		"author_id":         "author1",
	}

	body, _ = json.Marshal(prPayload)
	req = httptest.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Получаем PR для reviewer1
	req = httptest.NewRequest("GET", "/users/getReview?user_id=reviewer1", nil)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var reviewResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &reviewResp)

	assert.Equal(t, "reviewer1", reviewResp["user_id"], "Expected user_id 'reviewer1'")
	assert.NotNil(t, reviewResp["pull_requests"], "Expected pull_requests in response")

	t.Log("✓ User reviews retrieved via API")
}

// TestHandlers_ErrorCases - тест обработки ошибок через API
func TestHandlers_ErrorCases(t *testing.T) {
	// Setup
	db := testhelpers.SetupTestDB(t)
	repo := repository.New(db)
	log := zap.NewNop()
	services := service.New(repo, log)
	handler := httpapi.New(services, log)
	r := router.Router(handler)

	t.Run("Team already exists", func(t *testing.T) {
		teamPayload := map[string]interface{}{
			"team_name": "duplicate-team",
			"members": []map[string]interface{}{
				{"user_id": "user1", "username": "User1", "is_active": true},
			},
		}

		// Первый запрос - успешно
		body, _ := json.Marshal(teamPayload)
		req := httptest.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Второй запрос - ошибка
		body, _ = json.Marshal(teamPayload)
		req = httptest.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		errorBody := errResp["error"].(map[string]interface{})

		assert.Equal(t, "TEAM_EXISTS", errorBody["code"])
	})

	t.Run("Team not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/team/get?team_name=nonexistent", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		errorBody := errResp["error"].(map[string]interface{})

		assert.Equal(t, "NOT_FOUND", errorBody["code"])
	})

	t.Run("User not found", func(t *testing.T) {
		payload := map[string]interface{}{
			"user_id":   "nonexistent",
			"is_active": true,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/users/setIsActive", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		errorBody := errResp["error"].(map[string]interface{})

		assert.Equal(t, "NOT_FOUND", errorBody["code"])
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/team/add", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
