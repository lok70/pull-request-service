package http_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	httpapi "pull-request-service/internal/http"
	"pull-request-service/internal/http/mocks"
	"pull-request-service/internal/service"
)

func TestHandler_MassDeactivate(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name           string
		body           string
		mockBehavior   func(ts *mocks.TeamService)
		expectedStatus int
	}{
		{
			name: "Success",
			body: `{"user_ids": ["u1", "u2"]}`,
			mockBehavior: func(ts *mocks.TeamService) {
				ts.On("MassDeactivate", mock.Anything, []string{"u1", "u2"}).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Bad Request: Invalid JSON",
			body: `{"user_ids": "broken`,
			mockBehavior: func(ts *mocks.TeamService) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Bad Request: Empty list",
			body: `{"user_ids": []}`, // Или nil
			mockBehavior: func(ts *mocks.TeamService) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Internal Error",
			body: `{"user_ids": ["u1"]}`,
			mockBehavior: func(ts *mocks.TeamService) {
				ts.On("MassDeactivate", mock.Anything, []string{"u1"}).
					Return(service.ErrDomain("INTERNAL", "db error"))
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamSvc := new(mocks.TeamService)
			userSvc := new(mocks.UserService)
			prSvc := new(mocks.PRService)
			tt.mockBehavior(teamSvc)

			h := httpapi.NewHandler(teamSvc, userSvc, prSvc, logger)

			req := httptest.NewRequest("POST", "/team/deactivate", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.Router().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			teamSvc.AssertExpectations(t)
		})
	}
}
