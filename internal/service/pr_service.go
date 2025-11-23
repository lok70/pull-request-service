// Package service содержит бизнес-логику назначения ревьюверов и операций над командами и PR.
package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"pull-request-service/internal/model"
	"pull-request-service/internal/repository"
)

// TransactionManager описывает интерфейс для управления транзакциями (чтобы можно было мокать).
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// PRRepository описывает контракт репозитория для работы с pull request'ами.
type PRRepository interface {
	CreatePRWithReviewers(ctx context.Context, pr model.PullRequest, reviewerIDs []string) (model.PullRequest, error)
	GetPR(ctx context.Context, prID string) (model.PullRequest, error)
	MarkMerged(ctx context.Context, prID string, mergedAt time.Time) (model.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) (model.PullRequest, error)
	ListAssignedToUser(ctx context.Context, userID string) ([]model.PullRequestShort, error)
}

// PRService инкапсулирует бизнес-логику создания PR,
// назначения и переназначения ревьюверов и работы со списком PR пользователя.
type PRService struct {
	prRepo    PRRepository
	userRepo  UserRepository
	txManager TransactionManager
}

// NewPRService создаёт новый сервис для работы с pull request'ами.
func NewPRService(prRepo PRRepository, userRepo UserRepository, txManager TransactionManager) *PRService {
	return &PRService{
		prRepo:    prRepo,
		userRepo:  userRepo,
		txManager: txManager,
	}
}

// CreatePR создаёт новый pull request и автоматически назначает до двух ревьюверов
// из команды автора. Валидирует вход и оборачивает ошибки репозитория в AppError.
func (s *PRService) CreatePR(ctx context.Context, input model.PullRequest) (model.PullRequest, error) {
	// 1. Валидация входных данных
	if input.PullRequestID == "" || input.PullRequestName == "" || input.AuthorID == "" {
		return model.PullRequest{}, ErrBadRequest("pull_request_id, pull_request_name and author_id are required")
	}

	// 2. Проверка существования автора
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

	// 3. Получение списка кандидатов (активные участники команды, кроме автора)
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

	// 4. Выбор ревьюверов (бизнес-логика)
	reviewers := chooseReviewers(members, 2)
	reviewerIDs := make([]string, 0, len(reviewers))
	for _, u := range reviewers {
		reviewerIDs = append(reviewerIDs, u.UserID)
	}

	input.Status = model.StatusOpen
	var pr model.PullRequest

	// 5. СОХРАНЕНИЕ В ТРАНЗАКЦИИ
	// Мы используем RunInTransaction, чтобы создание PR и запись ревьюверов прошли атомарно.
	err = s.txManager.RunInTransaction(ctx, func(ctx context.Context) error {
		var errTx error
		// Репозиторий достанет транзакцию из контекста внутри CreatePRWithReviewers
		pr, errTx = s.prRepo.CreatePRWithReviewers(ctx, input, reviewerIDs)
		return errTx
	})

	// 6. Обработка ошибок транзакции
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

// chooseReviewers выбирает не более max ревьюверов из списка кандидатов.
// При числе кандидатов ≤ max возвращает всех.
func chooseReviewers(candidates []model.User, limit int) []model.User {
	if len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}

// MergePR помечает pull request как MERGED (идемпотентно) и возвращает обновлённое состояние PR.
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

// ReassignReviewer переназначает одного из текущих ревьюверов PR на другого участника той же команды.
// Учитывает статус PR, проверяет, что пользователь был назначен, и выбирает замену случайным образом.
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
		idx := rand.Intn(len(candidates))
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

// ListAssignedToUser возвращает список PR (в кратком виде),
// в которых указанный пользователь назначен ревьювером.
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
