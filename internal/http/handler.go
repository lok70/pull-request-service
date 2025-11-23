package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"pull-request-service/internal/service"
)

type Handler struct {
	Teams *service.TeamService
	Users *service.UserService
	PRs   *service.PRService
	Log   *slog.Logger
}

func NewHandler(teams *service.TeamService, users *service.UserService, prs *service.PRService, log *slog.Logger) *Handler {
	return &Handler{
		Teams: teams,
		Users: users,
		PRs:   prs,
		Log:   log,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

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

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
