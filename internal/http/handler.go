package httpapi

import (
	"reviewer_pr/internal/service"

	"go.uber.org/zap"
)

type Handler struct {
	services *service.Services
	log      *zap.Logger
}

func New(services *service.Services, log *zap.Logger) *Handler {
	return &Handler{
		services: services,
		log:      log,
	}
}
