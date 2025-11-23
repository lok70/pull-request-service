package service

import (
	"context"
	"errors"

	"pull-request-service/internal/model"
	"pull-request-service/internal/repository"
)

// UserRepository описывает контракт репозитория пользователей для бизнес-слоя.
type UserRepository interface {
	GetByUserID(ctx context.Context, userID string) (model.User, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) (model.User, error)
	ListActiveTeamMembersExcept(ctx context.Context, teamName string, exclude []string) ([]model.User, error)
}

// UserService содержит бизнес-логику, связанную с пользователями,
// в частности управление их активностью.
type UserService struct {
	repo UserRepository
}

// NewUserService создаёт новый сервис для операций над пользователями.
func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

// SetIsActive обновляет признак активности пользователя и возвращает его актуальное состояние.
func (s *UserService) SetIsActive(ctx context.Context, userID string, isActive bool) (model.User, error) {
	if userID == "" {
		return model.User{}, ErrBadRequest("user_id is required")
	}
	user, err := s.repo.SetIsActive(ctx, userID, isActive)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.User{}, ErrNotFound("user not found")
		}
		return model.User{}, &AppError{
			Code:    "INTERNAL",
			Message: "failed to update user",
			Status:  500,
			Err:     err,
		}
	}
	return user, nil
}
