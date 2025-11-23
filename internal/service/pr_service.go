package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"pull-request-service/internal/model"
	"pull-request-service/internal/repository"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

type PRRepository interface {
	CreatePRWithReviewers(ctx context.Context, pr model.PullRequest, reviewerIDs []string) (model.PullRequest, error)
	GetPR(ctx context.Context, prID string) (model.PullRequest, error)
	MarkMerged(ctx context.Context, prID string, mergedAt time.Time) (model.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) (model.PullRequest, error)
	ListAssignedToUser(ctx context.Context, userID string) ([]model.PullRequestShort, error)
}

type PRService struct {
	prRepo   PRRepository
	userRepo UserRepository
}

func NewPRService(prRepo PRRepository, userRepo UserRepository) *PRService {
	return &PRService{
		prRepo:   prRepo,
		userRepo: userRepo,
	}
}

func (s *PRService) CreatePR(ctx context.Context, input model.PullRequest) (model.PullRequest, error) {
	if input.PullRequestID == "" || input.PullRequestName == "" || input.AuthorID == "" {
		return model.PullRequest{}, ErrBadRequest("pull_request_id, pull_request_name and author_id are required")
	}

	author, err := s.userRepo.GetByUserID(ctx, input.AuthorID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.PullRequest{}, ErrNotFound("author not found")
		}
		return model.PullRequest{}, &AppError{
			Code:    "INTERNAL",
			Message: "failed to get author",
			Status:  500,
			Err:     err,
		}
	}

	exclude := []string{author.UserID}
	members, err := s.userRepo.ListActiveTeamMembersExcept(ctx, author.TeamName, exclude)
	if err != nil {
		return model.PullRequest{}, &AppError{
			Code:    "INTERNAL",
			Message: "failed to list team members",
			Status:  500,
			Err:     err,
		}
	}

	reviewers := chooseReviewers(members, 2)
	reviewerIDs := make([]string, 0, len(reviewers))
	for _, u := range reviewers {
		reviewerIDs = append(reviewerIDs, u.UserID)
	}

	input.Status = model.StatusOpen

	pr, err := s.prRepo.CreatePRWithReviewers(ctx, input, reviewerIDs)
	if err != nil {
		if errors.Is(err, repository.ErrPRExists) {
			return model.PullRequest{}, ErrDomain("PR_EXISTS", "PR id already exists")
		}
		return model.PullRequest{}, &AppError{
			Code:    "INTERNAL",
			Message: "failed to create PR",
			Status:  500,
			Err:     err,
		}
	}
	return pr, nil
}

func chooseReviewers(candidates []model.User, max int) []model.User {
	if len(candidates) <= max {
		return candidates
	}
	return candidates[:max]
}

func (s *PRService) MergePR(ctx context.Context, prID string) (model.PullRequest, error) {
	if prID == "" {
		return model.PullRequest{}, ErrBadRequest("pull_request_id is required")
	}
	pr, err := s.prRepo.MarkMerged(ctx, prID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrPRNotFound) {
			return model.PullRequest{}, ErrNotFound("pull request not found")
		}
		return model.PullRequest{}, &AppError{
			Code:    "INTERNAL",
			Message: "failed to merge PR",
			Status:  500,
			Err:     err,
		}
	}
	return pr, nil
}

func (s *PRService) ReassignReviewer(ctx context.Context, prID, oldUserID string) (model.PullRequest, string, error) {
	if prID == "" || oldUserID == "" {
		return model.PullRequest{}, "", ErrBadRequest("pull_request_id and old_user_id are required")
	}

	pr, err := s.prRepo.GetPR(ctx, prID)
	if err != nil {
		if errors.Is(err, repository.ErrPRNotFound) {
			return model.PullRequest{}, "", ErrNotFound("pull request not found")
		}
		return model.PullRequest{}, "", &AppError{
			Code:    "INTERNAL",
			Message: "failed to get PR",
			Status:  500,
			Err:     err,
		}
	}

	if pr.Status == model.StatusMerged {
		return model.PullRequest{}, "", ErrDomain("PR_MERGED", "cannot reassign on merged PR")
	}

	assigned := false
	for _, rid := range pr.AssignedReviewers {
		if rid == oldUserID {
			assigned = true
			break
		}
	}
	if !assigned {
		return model.PullRequest{}, "", ErrDomain("NOT_ASSIGNED", "reviewer is not assigned to this PR")
	}

	oldUser, err := s.userRepo.GetByUserID(ctx, oldUserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.PullRequest{}, "", ErrNotFound("user not found")
		}
		return model.PullRequest{}, "", &AppError{
			Code:    "INTERNAL",
			Message: "failed to get user",
			Status:  500,
			Err:     err,
		}
	}

	exclude := []string{oldUser.UserID, pr.AuthorID}
	for _, rid := range pr.AssignedReviewers {
		if rid != oldUserID {
			exclude = append(exclude, rid)
		}
	}

	candidates, err := s.userRepo.ListActiveTeamMembersExcept(ctx, oldUser.TeamName, exclude)
	if err != nil {
		return model.PullRequest{}, "", &AppError{
			Code:    "INTERNAL",
			Message: "failed to list replacement candidates",
			Status:  500,
			Err:     err,
		}
	}
	if len(candidates) == 0 {
		return model.PullRequest{}, "", ErrDomain("NO_CANDIDATE", "no active replacement candidate in team")
	}

	var newReviewer model.User
	if len(candidates) == 1 {
		newReviewer = candidates[0]
	} else {
		idx := rnd.Intn(len(candidates))
		newReviewer = candidates[idx]
	}

	updated, err := s.prRepo.ReassignReviewer(ctx, prID, oldUserID, newReviewer.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrPRNotFound) {
			return model.PullRequest{}, "", ErrNotFound("pull request not found")
		}
		return model.PullRequest{}, "", &AppError{
			Code:    "INTERNAL",
			Message: "failed to reassign reviewer",
			Status:  500,
			Err:     err,
		}
	}

	return updated, newReviewer.UserID, nil
}

func (s *PRService) ListAssignedToUser(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	if userID == "" {
		return nil, ErrBadRequest("user_id is required")
	}
	prs, err := s.prRepo.ListAssignedToUser(ctx, userID)
	if err != nil {
		return nil, &AppError{
			Code:    "INTERNAL",
			Message: "failed to list PRs for user",
			Status:  500,
			Err:     err,
		}
	}
	return prs, nil
}
