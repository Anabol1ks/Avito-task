package testhelpers

import (
	"reviewer_pr/internal/database"
	"reviewer_pr/internal/models"
	"testing"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Создаем in-memory SQLite базу данных
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		PrepareStmt: false,
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	logger := zap.NewNop()
	if err := database.AutoMigrate(db, logger); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	})

	return db
}

func CleanDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	db.Exec("DELETE FROM pr_reviewers")
	db.Exec("DELETE FROM pull_requests")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM teams")
}

// CreateTestTeam создает тестовую команду с пользователями
func CreateTestTeam(t *testing.T, db *gorm.DB, teamName string, userCount int) []models.User {
	t.Helper()

	team := &models.Team{Name: teamName}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create test team: %v", err)
	}

	users := make([]models.User, userCount)
	for i := 0; i < userCount; i++ {
		users[i] = models.User{
			ID:       teamName + "-user-" + string(rune('A'+i)),
			Username: teamName + "-User-" + string(rune('A'+i)),
			TeamName: teamName,
			IsActive: true,
		}
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}
	}

	return users
}
