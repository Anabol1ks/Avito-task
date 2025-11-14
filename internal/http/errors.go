package httpapi

import (
	"net/http"
	"reviewer_pr/internal/service"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func writeSerErr(c *gin.Context, err error) {
	if serr, ok := err.(*service.Error); ok {
		status := mapSerErrToStatus(serr.Code)
		c.JSON(status, ErrorResponse{
			Error: ErrorBody{
				Code:    string(serr.Code),
				Message: serr.Msg,
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{
			Code:    "INTERNAL",
			Message: "internal server error",
		},
	})
}

func mapSerErrToStatus(code service.ErrorCode) int {
	switch code {
	case service.ErrorCodeTeamExists:
		return http.StatusBadRequest // /team/add -> 400
	case service.ErrorCodePRExists:
		return http.StatusConflict // /pullRequest/create -> 409
	case service.ErrorCodePRMerged,
		service.ErrorCodeNotAssigned,
		service.ErrorCodeNoCandidate:
		return http.StatusConflict // /pullRequest/reassign -> 409
	case service.ErrorCodeNotFound:
		return http.StatusNotFound // 404
	default:
		return http.StatusInternalServerError
	}
}
