package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.services.Stats.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{
				Code:    "NOT_FOUND",
				Message: "failed to get stats",
			},
		})
		return
	}

	resp := StatsResponseDTO{
		ByUser: make([]UserStatsDTO, 0, len(stats.ByUser)),
		ByPR:   make([]PRStatsDTO, 0, len(stats.ByPR)),
	}

	for _, u := range stats.ByUser {
		resp.ByUser = append(resp.ByUser, UserStatsDTO{
			UserID:      u.UserID,
			Username:    u.Username,
			TeamName:    u.TeamName,
			ReviewCount: u.ReviewCount,
		})
	}

	for _, p := range stats.ByPR {
		resp.ByPR = append(resp.ByPR, PRStatsDTO{
			PullRequestID: p.PullRequestID,
			ReviewerCount: p.ReviewerCount,
		})
	}

	c.JSON(http.StatusOK, resp)
}
