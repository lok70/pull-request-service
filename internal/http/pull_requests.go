package http

import (
	"encoding/json"
	"net/http"

	"pull-request-service/internal/model"
	"pull-request-service/internal/service"
)

func (h *Handler) handlePRCreate(w http.ResponseWriter, r *http.Request) {
	const handlerName = "pr_create"

	var req createPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, handlerName, service.ErrBadRequest("invalid JSON"))
		return
	}

	prInput := model.PullRequest{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
		Status:          model.StatusOpen,
	}

	ctx := r.Context()
	pr, err := h.PRs.CreatePR(ctx, prInput)
	if err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := prResponse{PR: pr}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handlePRMerge(w http.ResponseWriter, r *http.Request) {
	const handlerName = "pr_merge"

	var req mergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, handlerName, service.ErrBadRequest("invalid JSON"))
		return
	}

	ctx := r.Context()
	pr, err := h.PRs.MergePR(ctx, req.PullRequestID)
	if err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := prResponse{PR: pr}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handlePRReassign(w http.ResponseWriter, r *http.Request) {
	const handlerName = "pr_reassign"

	var req reassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, handlerName, service.ErrBadRequest("invalid JSON"))
		return
	}

	ctx := r.Context()
	pr, replacedBy, err := h.PRs.ReassignReviewer(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := reassignResponse{
		PR:         pr,
		ReplacedBy: replacedBy,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
