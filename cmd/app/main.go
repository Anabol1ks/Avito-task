package main

import (
	"os"
	"reviewer_pr/internal/config"
	"reviewer_pr/internal/database"
	"reviewer_pr/internal/logger"

	"github.com/joho/godotenv"
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
}
