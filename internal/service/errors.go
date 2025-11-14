package service

import "fmt"

type ErrorCode string

const (
	ErrorCodeTeamExists  ErrorCode = "TEAM_EXISTS"
	ErrorCodePRExists    ErrorCode = "PR_EXISTS"
	ErrorCodePRMerged    ErrorCode = "PR_MERGED"
	ErrorCodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	ErrorCodeNoCandidate ErrorCode = "NO_CANDIDATE"
	ErrorCodeNotFound    ErrorCode = "NOT_FOUND"
)

type Error struct {
	Code ErrorCode
	Msg  string
}

func (e *Error) Error() string {
	if e.Msg == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

func NewErr(code ErrorCode, msg string) *Error {
	return &Error{
		Code: code,
		Msg:  msg,
	}
}
