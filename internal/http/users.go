package http

import (
	"encoding/json"
	"net/http"
	"pull-request-service/internal/service"
)

func (h *Handler) handleUserSetIsActive(w http.ResponseWriter, r *http.Request) {
	const handlerName = "user_set_is_active"

	var req setIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, handlerName, service.ErrBadRequest("invalid JSON"))
		return
	}

	if err := ValidateSetIsActiveRequest(req); err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	ctx := r.Context()
	user, err := h.Users.SetIsActive(ctx, req.UserID, req.IsActive)
	if err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := userResponse{User: user}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleUserGetReview(w http.ResponseWriter, r *http.Request) {
	const handlerName = "user_get_review"

	userID := r.URL.Query().Get("user_id")
	if err := ValidateUserIDQuery(userID); err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	ctx := r.Context()
	prs, err := h.PRs.ListAssignedToUser(ctx, userID)
	if err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := getUserReviewResponse{
		UserID:       userID,
		PullRequests: prs,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
