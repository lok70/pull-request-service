package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"pull-request-service/internal/model"
	"pull-request-service/internal/repository"
	"pull-request-service/internal/service"
	"pull-request-service/internal/service/mocks"
)

func TestPRService_CreatePR(t *testing.T) {
	// Подготовка тестовых данных
	author := model.User{UserID: "u1", Username: "Author", TeamName: "backend", IsActive: true}
	u2 := model.User{UserID: "u2", Username: "Rev1", TeamName: "backend", IsActive: true}
	u3 := model.User{UserID: "u3", Username: "Rev2", TeamName: "backend", IsActive: true}

	tests := []struct {
		name          string
		input         model.PullRequest
		setupMocks    func(userRepo *mocks.UserRepository, prRepo *mocks.PRRepository, txManager *mocks.TransactionManager)
		wantReviewers int
		wantErr       bool
	}{
		{
			name: "Success: 2 reviewers available",
			input: model.PullRequest{
				PullRequestID:   "pr-1",
				PullRequestName: "Fix",
				AuthorID:        "u1",
			},
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PRRepository, txManager *mocks.TransactionManager) {
				userRepo.On("GetByUserID", mock.Anything, "u1").Return(author, nil)

				userRepo.On("ListActiveTeamMembersExcept", mock.Anything, "backend", []string{"u1"}).
					Return([]model.User{u2, u3}, nil)

				txManager.On("RunInTransaction", mock.Anything, mock.Anything).
					Return(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})

				prRepo.On("CreatePRWithReviewers", mock.Anything, mock.AnythingOfType("model.PullRequest"), mock.Anything).
					Return(func(ctx context.Context, pr model.PullRequest, rIDs []string) model.PullRequest {
						pr.AssignedReviewers = rIDs
						return pr
					}, nil)
			},
			wantReviewers: 2,
			wantErr:       false,
		},
		{
			name: "Fail: Author not found",
			input: model.PullRequest{
				PullRequestID:   "pr-3",
				PullRequestName: "Fix",
				AuthorID:        "u999",
			},
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PRRepository, txManager *mocks.TransactionManager) {
				userRepo.On("GetByUserID", mock.Anything, "u999").
					Return(model.User{}, repository.ErrUserNotFound)
			},
			wantReviewers: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := new(mocks.UserRepository)
			prRepo := new(mocks.PRRepository)
			txManager := new(mocks.TransactionManager)

			tt.setupMocks(userRepo, prRepo, txManager)

			svc := service.NewPRService(prRepo, userRepo, txManager)

			got, err := svc.CreatePR(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got.AssignedReviewers, tt.wantReviewers)
				assert.NotContains(t, got.AssignedReviewers, tt.input.AuthorID, "Author should not be a reviewer")
			}

			userRepo.AssertExpectations(t)
			prRepo.AssertExpectations(t)
			txManager.AssertExpectations(t)
		})
	}
}
