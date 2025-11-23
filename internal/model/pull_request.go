// Package model содержит доменные структуры для команд, пользователей и pr
package model

import "time"

// PullRequestStatus представляет статус pull request'а в доменной модели.
type PullRequestStatus string

const (
	// StatusOpen означает, что pull request находится в открытом состоянии.
	StatusOpen PullRequestStatus = "OPEN"
	// StatusMerged означает, что pull request был влит (merged).
	StatusMerged PullRequestStatus = "MERGED"
)

// PullRequest описывает полный объект pr с авторами, статусом, ревьюверами и временными метками.
type PullRequest struct {
	PullRequestID     string            `json:"pull_request_id"`
	PullRequestName   string            `json:"pull_request_name"`
	AuthorID          string            `json:"author_id"`
	Status            PullRequestStatus `json:"status"`
	AssignedReviewers []string          `json:"assigned_reviewers"`
	CreatedAt         *time.Time        `json:"createdAt,omitempty"`
	MergedAt          *time.Time        `json:"mergedAt,omitempty"`
}

// PullRequestShort описывает укороченное представление pr, которое используется в списках (без ревьюверов и временных полей).
type PullRequestShort struct {
	PullRequestID   string            `json:"pull_request_id"`
	PullRequestName string            `json:"pull_request_name"`
	AuthorID        string            `json:"author_id"`
	Status          PullRequestStatus `json:"status"`
}

// StatsDTO используется для возврата статистики по ревьюверам.
type StatsDTO struct {
	ReviewerID  string `json:"reviewer_id"`
	ReviewCount int    `json:"review_count"`
}
