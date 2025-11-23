package service

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func ErrBadRequest(msg string) *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: msg,
		Status:  http.StatusBadRequest,
	}
}

func ErrNotFound(msg string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: msg,
		Status:  http.StatusNotFound,
	}
}

func ErrDomain(code, msg string) *AppError {
	status := http.StatusConflict
	if code == "TEAM_EXISTS" {
		status = http.StatusBadRequest
	}
	if code == "PR_EXISTS" {
		status = http.StatusConflict
	}
	return &AppError{
		Code:    code,
		Message: msg,
		Status:  status,
	}
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if app, ok := err.(*AppError); ok {
		return app.Status == http.StatusNotFound
	}
	return false
}
