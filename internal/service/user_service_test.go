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

func TestUserService_SetIsActive(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		isActive   bool
		setupMocks func(ur *mocks.UserRepository)
		wantErr    bool
	}{
		{
			name:     "Success",
			userID:   "u1",
			isActive: false,
			setupMocks: func(ur *mocks.UserRepository) {
				ur.On("SetIsActive", mock.Anything, "u1", false).
					Return(model.User{UserID: "u1", IsActive: false}, nil)
			},
			wantErr: false,
		},
		{
			name:     "Fail: Empty ID",
			userID:   "",
			isActive: true,
			setupMocks: func(ur *mocks.UserRepository) {
				// Repo не должен вызываться
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ur := new(mocks.UserRepository)
			tt.setupMocks(ur)

			svc := service.NewUserService(ur)
			_, err := svc.SetIsActive(context.Background(), tt.userID, tt.isActive)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			ur.AssertExpectations(t)
		})
	}
}
