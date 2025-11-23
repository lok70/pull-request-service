package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"pull-request-service/internal/model"
	"pull-request-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// TeamService описывает методы сервиса команд, используемые HTTP-слоем.
type TeamService interface {
	CreateTeam(ctx context.Context, team model.Team) (model.Team, error)
	GetTeam(ctx context.Context, name string) (model.Team, error)
}

// UserService описывает методы сервиса пользователей, используемые HTTP-слоем.
type UserService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (model.User, error)
}

// PRService описывает методы сервиса pr, используемые HTTP-слоем.
type PRService interface {
	CreatePR(ctx context.Context, input model.PullRequest) (model.PullRequest, error)
	MergePR(ctx context.Context, prID string) (model.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (model.PullRequest, string, error)
	ListAssignedToUser(ctx context.Context, userID string) ([]model.PullRequestShort, error)
}

// Handler агрегирует зависимости HTTP-слоя
type Handler struct {
	Teams *service.TeamService
	Users *service.UserService
	PRs   *service.PRService
	Log   *slog.Logger
}

// NewHandler создаёт и возвращает HTTP-обработчик c маршрутизатором и зависимостями сервисного слоя.
func NewHandler(teams *service.TeamService, users *service.UserService, prs *service.PRService, log *slog.Logger) *Handler {
	return &Handler{
		Teams: teams,
		Users: users,
		PRs:   prs,
		Log:   log,
	}
}

// Router настраивает HTTP-маршруты и middleware, включая CORS, и возвращает корневой роутер chi.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// CORS: разрешаем swagger-ui на 7002 ходить к API 8080
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:7002"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", h.handleHealth)

	r.Route("/team", func(r chi.Router) {
		r.Post("/add", h.handleTeamAdd)
		r.Get("/get", h.handleTeamGet)
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", h.handleUserSetIsActive)
		r.Get("/getReview", h.handleUserGetReview)
	})

	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", h.handlePRCreate)
		r.Post("/merge", h.handlePRMerge)
		r.Post("/reassign", h.handlePRReassign)
	})

	return r
}

func (h *Handler) writeError(w http.ResponseWriter, handlerName string, err error) {
	appErr, ok := err.(*service.AppError)
	if !ok {
		appErr = &service.AppError{
			Code:    "INTERNAL",
			Message: "internal error",
			Status:  http.StatusInternalServerError,
			Err:     err,
		}
	}

	h.Log.Error("handler error",
		slog.String("handler", handlerName),
		slog.String("code", appErr.Code),
		slog.String("message", appErr.Message),
		slog.Any("err", appErr.Err),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Status)

	resp := errorResponse{}
	resp.Error.Code = appErr.Code
	resp.Error.Message = appErr.Message
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
