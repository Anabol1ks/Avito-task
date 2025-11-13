package main

import (
	"os"
	"reviewer_pr/internal/config"
	"reviewer_pr/internal/database"
	"reviewer_pr/internal/logger"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()
	isDev := os.Getenv("ENV") == "development"
	if err := logger.Init(isDev); err != nil {
		panic(err)
	}

	defer logger.Sync()

	log := logger.L()

	cfg := config.Load(log)

	db := database.ConnectDB(&cfg.DB, log)
	defer database.CloseDB(db, log)

	if err := database.AutoMigrate(db, log); err != nil {
		log.Fatal("ошибка запуска автомиграции", zap.Error(err))
	}
}
