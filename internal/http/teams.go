package http

import (
	"encoding/json"
	"net/http"
	"pull-request-service/internal/model"
	"pull-request-service/internal/service"
)

func (h *Handler) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	const handlerName = "team_add"

	var req model.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, handlerName, service.ErrBadRequest("invalid JSON"))
		return
	}

	if err := ValidateTeam(req); err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	ctx := r.Context()
	team, err := h.Teams.CreateTeam(ctx, req)
	if err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := createTeamResponse{Team: team}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	const handlerName = "team_get"

	teamName := r.URL.Query().Get("team_name")
	if err := ValidateTeamNameQuery(teamName); err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	ctx := r.Context()
	team, err := h.Teams.GetTeam(ctx, teamName)
	if err != nil {
		h.writeError(w, handlerName, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(team)
}
