package repository

import "gorm.io/gorm"

type Repository struct {
	DB    *gorm.DB
	Teams TeamsRepo
	Users UsersRepo
	PRs   PRRepo
}

func buildRepository(db *gorm.DB) *Repository {
	return &Repository{
		DB:    db,
		Teams: NewTeamsRepo(db),
		Users: NewUsersRepo(db),
		PRs:   NewPRRepo(db),
	}
}

func New(db *gorm.DB) *Repository {
	return buildRepository(db)
}
