// Package http реализует HTTP-обработчики и DTO поверх доменных сервисов.
package http

import "pull-request-service/internal/model"

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type createTeamResponse struct {
	Team model.Team `json:"team"`
}

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type userResponse struct {
	User model.User `json:"user"`
}

type getUserReviewResponse struct {
	UserID       string                   `json:"user_id"`
	PullRequests []model.PullRequestShort `json:"pull_requests"`
}

type createPRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type mergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type reassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type prResponse struct {
	PR model.PullRequest `json:"pr"`
}

type reassignResponse struct {
	PR         model.PullRequest `json:"pr"`
	ReplacedBy string            `json:"replaced_by"`
}

type massDeactivateRequest struct {
	UserIDs []string `json:"user_ids"`
}
