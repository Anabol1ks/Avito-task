package main

import (
	"os"
	"reviewer_pr/internal/config"
	"reviewer_pr/internal/database"
	httpapi "reviewer_pr/internal/http"
	"reviewer_pr/internal/logger"
	"reviewer_pr/internal/repository"
	"reviewer_pr/internal/router"
	"reviewer_pr/internal/service"

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

	repos := repository.New(db)
	services := service.New(repos, log)
	handlers := httpapi.New(services, log)

	r := router.Router(handlers, log)
	port := ":" + cfg.Port
	if err := r.Run(port); err != nil {
		log.Fatal("failed to run http server", zap.Error(err))
	}
}
