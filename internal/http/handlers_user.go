package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

func (h *Handler) UserSetIsActive(c *gin.Context) {
	var req setIsActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{
				Code:    "NOT_FOUND",
				Message: "invalid request body",
			},
		})
		return
	}

	u, err := h.services.Users.SetIsActive(c.Request.Context(), req.UserID, req.IsActive)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": UserDTO{
			UserID:   u.ID,
			Username: u.Username,
			TeamName: u.TeamName,
			IsActive: u.IsActive,
		},
	})
}

func (h *Handler) UserGetReview(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{
				Code:    "NOT_FOUND",
				Message: "user_id is required",
			},
		})
		return
	}

	prs, err := h.services.PRs.GetReviewsByUser(c.Request.Context(), userID)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	out := make([]PullRequestShortDTO, 0, len(prs))
	for _, pr := range prs {
		out = append(out, PullRequestShortDTO{
			PullRequestID:   pr.ID,
			PullRequestName: pr.Name,
			AuthorID:        pr.AuthorID,
			Status:          string(pr.Status),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"pull_requests": out,
	})
}
