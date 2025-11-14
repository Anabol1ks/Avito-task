package httpapi

import (
	"net/http"
	"reviewer_pr/internal/service"

	"github.com/gin-gonic/gin"
)

type createPRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

func (h *Handler) PRCreate(c *gin.Context) {
	var req createPRRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		req.PullRequestID == "" || req.PullRequestName == "" || req.AuthorID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{
				Code:    "NOT_FOUND",
				Message: "invalid request body",
			},
		})
		return
	}

	in := service.CreatePRInput{
		ID:       req.PullRequestID,
		Name:     req.PullRequestName,
		AuthorID: req.AuthorID,
	}

	res, err := h.services.PRs.CreateWithAutoAssign(c.Request.Context(), in)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	reviewerIDs := make([]string, 0, len(res.Reviewers))
	for _, u := range res.Reviewers {
		reviewerIDs = append(reviewerIDs, u.ID)
	}

	dto := PullRequestDTO{
		PullRequestID:     res.PR.ID,
		PullRequestName:   res.PR.Name,
		AuthorID:          res.PR.AuthorID,
		Status:            string(res.PR.Status),
		AssignedReviewers: reviewerIDs,
		CreatedAt:         &res.PR.CreatedAt,
		MergedAt:          res.PR.MergedAt,
	}

	c.JSON(http.StatusCreated, gin.H{"pr": dto})
}

type mergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

func (h *Handler) PRMerge(c *gin.Context) {
	var req mergePRRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PullRequestID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{
				Code:    "INVALID_REQUEST",
				Message: "invalid request body",
			},
		})
		return
	}

	pr, err := h.services.PRs.Merge(c.Request.Context(), req.PullRequestID)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	reviewers, err := h.services.PRs.GetReviewersForPR(c.Request.Context(), pr.ID)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	reviewerIDs := make([]string, 0, len(reviewers))
	for _, r := range reviewers {
		reviewerIDs = append(reviewerIDs, r.ReviewerID)
	}

	dto := PullRequestDTO{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: reviewerIDs,
		CreatedAt:         &pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}

	c.JSON(http.StatusOK, gin.H{"pr": dto})
}

type reassignPRRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_reviewer_id"`
}

func (h *Handler) PRReassign(c *gin.Context) {
	var req reassignPRRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PullRequestID == "" || req.OldReviewerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{
				Code:    "INVALID_REQUEST",
				Message: "invalid request body",
			},
		})
		return
	}

	in := service.ReassignInput{
		PRID:          req.PullRequestID,
		OldReviewerID: req.OldReviewerID,
	}

	out, err := h.services.PRs.ReassignReviewer(c.Request.Context(), in)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	reviewers, err := h.services.PRs.GetReviewersForPR(c.Request.Context(), out.PR.ID)
	if err != nil {
		writeSerErr(c, err)
		return
	}

	reviewerIDs := make([]string, 0, len(reviewers))
	for _, r := range reviewers {
		reviewerIDs = append(reviewerIDs, r.ReviewerID)
	}

	dto := PullRequestDTO{
		PullRequestID:     out.PR.ID,
		PullRequestName:   out.PR.Name,
		AuthorID:          out.PR.AuthorID,
		Status:            string(out.PR.Status),
		AssignedReviewers: reviewerIDs,
		CreatedAt:         &out.PR.CreatedAt,
		MergedAt:          out.PR.MergedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"pr":          dto,
		"replaced_by": out.ReplacedByID,
	})
}
