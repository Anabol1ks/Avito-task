package httpapi

import (
	"net/http"
	"reviewer_pr/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) TeamAdd(c *gin.Context) {
	var req TeamDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{
				Code:    "NOT_FOUND",
				Message: "invalid request body",
			},
		})
		return
	}

	in := service.CreateTeamInput{
		TeamName: req.TeamName,
		Members:  make([]service.CreateTeamMemberInput, 0, len(req.Members)),
	}

	for _, m := range req.Members {
		in.Members = append(in.Members, service.CreateTeamMemberInput{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	res, err := h.services.Teams.AddTeam(c.Request.Context(), in)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	members := make([]TeamMemberDTO, 0, len(res.Members))
	for _, u := range res.Members {
		members = append(members, TeamMemberDTO{
			UserID:   u.ID,
			Username: u.Username,
			IsActive: u.IsActive,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"team": TeamDTO{
			TeamName: res.Team.Name,
			Members:  members,
		},
	})
}

func (h *Handler) TeamGet(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{
				Code:    "NOT_FOUND",
				Message: "team_name is required",
			},
		})
		return
	}

	res, err := h.services.Teams.GetTeam(c.Request.Context(), teamName)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	members := make([]TeamMemberDTO, 0, len(res.Members))
	for _, u := range res.Members {
		members = append(members, TeamMemberDTO{
			UserID:   u.ID,
			Username: u.Username,
			IsActive: u.IsActive,
		})
	}

	c.JSON(http.StatusOK, TeamDTO{
		TeamName: res.Team.Name,
		Members:  members,
	})
}
