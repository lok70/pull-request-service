package http

import (
	"fmt"
	"pull-request-service/internal/model"
	"pull-request-service/internal/service"
	"regexp"
)

// Регулярки для проверки корректности u_id и pr_id
var (
	reUserID        = regexp.MustCompile(`^u[0-9]+$`)
	rePullRequestID = regexp.MustCompile(`^pr-[0-9]+$`)
)

// Teams

// ValidateTeam Валидация команды для /team/add
func ValidateTeam(team model.Team) error {
	if team.TeamName == "" {
		return service.ErrBadRequest("team_name is required")
	}
	if len(team.Members) == 0 {
		return service.ErrBadRequest("members must not be empty")
	}

	for i, m := range team.Members {
		if m.UserID == "" {
			return service.ErrBadRequest(fmt.Sprintf("members[%d].user_id is required", i))
		}
		if !reUserID.MatchString(m.UserID) {
			return service.ErrBadRequest(fmt.Sprintf("members[%d].user_id must match pattern u<digits>, e.g. u1", i))
		}
		if m.Username == "" {
			return service.ErrBadRequest(fmt.Sprintf("members[%d].username is required", i))
		}
	}

	return nil
}

// ValidateTeamNameQuery Валидация query-параметра team_name для /team/get
func ValidateTeamNameQuery(teamName string) error {
	if teamName == "" {
		return service.ErrBadRequest("team_name is required")
	}
	return nil
}

//Users

// ValidateSetIsActiveRequest /users/setIsActive — тело запроса
func ValidateSetIsActiveRequest(req setIsActiveRequest) error {
	if req.UserID == "" {
		return service.ErrBadRequest("user_id is required")
	}
	if !reUserID.MatchString(req.UserID) {
		return service.ErrBadRequest("user_id must match pattern u<digits>, e.g. u1")
	}
	// is_active — bool, json.Decoder сам отловит неверный тип
	return nil
}

// ValidateUserIDQuery Валидация query-параметра user_id для /users/getReview
func ValidateUserIDQuery(userID string) error {
	if userID == "" {
		return service.ErrBadRequest("user_id is required")
	}
	if !reUserID.MatchString(userID) {
		return service.ErrBadRequest("user_id must match pattern u<digits>, e.g. u1")
	}
	return nil
}

// Pull Requests

// ValidateCreatePRRequest /pullRequest/create — тело запроса
func ValidateCreatePRRequest(req createPRRequest) error {
	if req.PullRequestID == "" {
		return service.ErrBadRequest("pull_request_id is required")
	}
	if !rePullRequestID.MatchString(req.PullRequestID) {
		return service.ErrBadRequest("pull_request_id must match pattern pr-<digits>, e.g. pr-1001")
	}

	if req.PullRequestName == "" {
		return service.ErrBadRequest("pull_request_name is required")
	}

	if req.AuthorID == "" {
		return service.ErrBadRequest("author_id is required")
	}
	if !reUserID.MatchString(req.AuthorID) {
		return service.ErrBadRequest("author_id must match pattern u<digits>, e.g. u1")
	}

	return nil
}

// ValidateMergePRRequest /pullRequest/merge — тело запроса
func ValidateMergePRRequest(req mergePRRequest) error {
	if req.PullRequestID == "" {
		return service.ErrBadRequest("pull_request_id is required")
	}
	if !rePullRequestID.MatchString(req.PullRequestID) {
		return service.ErrBadRequest("pull_request_id must match pattern pr-<digits>, e.g. pr-1001")
	}
	return nil
}

// ValidateReassignRequest /pullRequest/reassign — тело запроса
func ValidateReassignRequest(req reassignRequest) error {
	if req.PullRequestID == "" {
		return service.ErrBadRequest("pull_request_id is required")
	}
	if !rePullRequestID.MatchString(req.PullRequestID) {
		return service.ErrBadRequest("pull_request_id must match pattern pr-<digits>, e.g. pr-1001")
	}

	if req.OldUserID == "" {
		return service.ErrBadRequest("old_user_id is required")
	}
	if !reUserID.MatchString(req.OldUserID) {
		return service.ErrBadRequest("old_user_id must match pattern u<digits>, e.g. u1")
	}

	return nil
}
