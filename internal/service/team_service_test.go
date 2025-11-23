package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"pull-request-service/internal/model"
	"pull-request-service/internal/service"
	"pull-request-service/internal/service/mocks"
)

func TestTeamService_MassDeactivate(t *testing.T) {
	// Тестовые данные
	u1 := model.User{UserID: "u1", TeamName: "backend", IsActive: true}
	u2 := model.User{UserID: "u2", TeamName: "backend", IsActive: true} // Кандидат на замену

	tests := []struct {
		name       string
		userIDs    []string
		setupMocks func(ur *mocks.UserRepository, pr *mocks.PRRepository, tm *mocks.TransactionManager)
		wantErr    bool
	}{
		{
			name:    "Success: No impacted PRs",
			userIDs: []string{"u1"},
			setupMocks: func(ur *mocks.UserRepository, pr *mocks.PRRepository, tm *mocks.TransactionManager) {
				// 1. Транзакция
				tm.On("RunInTransaction", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				})
				// 2. Деактивация
				ur.On("DeactivateUsers", mock.Anything, []string{"u1"}).Return(nil)
				// 3. Поиск PR (пусто)
				pr.On("GetOpenPRsByReviewers", mock.Anything, []string{"u1"}).Return(map[string][]string{}, nil)
			},
			wantErr: false,
		},
		{
			name:    "Success: Reassign to available candidate",
			userIDs: []string{"u1"},
			setupMocks: func(ur *mocks.UserRepository, pr *mocks.PRRepository, tm *mocks.TransactionManager) {
				tm.On("RunInTransaction", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				})
				ur.On("DeactivateUsers", mock.Anything, []string{"u1"}).Return(nil)

				// Найдена 1 PR, где u1 ревьювер
				pr.On("GetOpenPRsByReviewers", mock.Anything, []string{"u1"}).
					Return(map[string][]string{"u1": {"pr-1"}}, nil)

				// Получаем инфо о пользователе
				ur.On("GetByUserID", mock.Anything, "u1").Return(u1, nil)

				// Получаем PR
				pr.On("GetPR", mock.Anything, "pr-1").Return(model.PullRequest{
					PullRequestID: "pr-1", AuthorID: "author", AssignedReviewers: []string{"u1"},
				}, nil)

				// Ищем кандидатов. Ожидаем, что u1 исключен. Возвращаем u2.
				ur.On("ListActiveTeamMembersExcept", mock.Anything, "backend", mock.MatchedBy(func(exclude []string) bool {
					// Проверяем, что u1 есть в списке исключений
					for _, id := range exclude {
						if id == "u1" {
							return true
						}
					}
					return false
				})).Return([]model.User{u2}, nil)

				// Ожидаем переназначение на u2
				pr.On("ReassignReviewer", mock.Anything, "pr-1", "u1", "u2").
					Return(model.PullRequest{}, nil)
			},
			wantErr: false,
		},
		{
			name:    "Success: Remove reviewer (No candidates)",
			userIDs: []string{"u1"},
			setupMocks: func(ur *mocks.UserRepository, pr *mocks.PRRepository, tm *mocks.TransactionManager) {
				tm.On("RunInTransaction", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				})
				ur.On("DeactivateUsers", mock.Anything, []string{"u1"}).Return(nil)
				pr.On("GetOpenPRsByReviewers", mock.Anything, []string{"u1"}).
					Return(map[string][]string{"u1": {"pr-1"}}, nil)
				ur.On("GetByUserID", mock.Anything, "u1").Return(u1, nil)
				pr.On("GetPR", mock.Anything, "pr-1").Return(model.PullRequest{
					PullRequestID: "pr-1", AuthorID: "author", AssignedReviewers: []string{"u1"},
				}, nil)

				// Кандидатов нет (пустой слайс)
				ur.On("ListActiveTeamMembersExcept", mock.Anything, "backend", mock.Anything).
					Return([]model.User{}, nil)

				// Ожидаем УДАЛЕНИЕ
				pr.On("RemoveReviewer", mock.Anything, "pr-1", "u1").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mocks
			ur := new(mocks.UserRepository)
			tr := new(mocks.TeamRepository) // Не используется в MassDeactivate, но нужен конструктору
			pr := new(mocks.PRRepository)
			tm := new(mocks.TransactionManager)

			tt.setupMocks(ur, pr, tm)

			svc := service.NewTeamService(tr, ur, pr, tm)
			err := svc.MassDeactivate(context.Background(), tt.userIDs)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			ur.AssertExpectations(t)
			pr.AssertExpectations(t)
		})
	}
}
