package database

import (
	"errors"
	"reviewer_pr/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB, log *zap.Logger) error {
	if err := db.AutoMigrate(
		&models.Team{},
		&models.User{},
		&models.PullRequest{},
		&models.PRReviewer{},
	); err != nil {
		var pgErr *pgconn.PgError
		if ok := errors.As(err, &pgErr); ok {
			log.Error("ошибка миграции", zap.String("pg_code", pgErr.Code), zap.Error(err))
		} else {
			log.Error("ошибка миграции", zap.Error(err))
		}
		return err
	}

	log.Info("Миграция выполнена успешно")
	return nil
}
