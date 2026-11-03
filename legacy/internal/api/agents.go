package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) ListSceneTurns(w http.ResponseWriter, r *http.Request) {
	sceneID := chi.URLParam(r, "nodeID")
	if h.agentSvc == nil {
		writeJSON(w, http.StatusOK, []*domain.SceneTurn{})
		return
	}
	turns, err := h.agentSvc.GetTurns(r.Context(), sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turns)
}

func (h *Handlers) ListSceneTurnsByRole(w http.ResponseWriter, r *http.Request) {
	sceneID := chi.URLParam(r, "nodeID")
	role := r.URL.Query().Get("role")
	if h.agentSvc == nil {
		writeJSON(w, http.StatusOK, []*domain.SceneTurn{})
		return
	}
	turns, err := h.agentSvc.GetTurnsByRole(r.Context(), sceneID, role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turns)
}

func (h *Handlers) ListSceneCanonDeltas(w http.ResponseWriter, r *http.Request) {
	sceneID := chi.URLParam(r, "nodeID")
	if h.agentSvc == nil {
		writeJSON(w, http.StatusOK, []*domain.CanonDelta{})
		return
	}
	deltas, err := h.agentSvc.GetCanonDeltas(r.Context(), sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, deltas)
}

func (h *Handlers) RecordCanonDelta(w http.ResponseWriter, r *http.Request) {
	sceneID := chi.URLParam(r, "nodeID")
	if h.agentSvc == nil {
		writeError(w, http.StatusNotImplemented, "agent service not available")
		return
	}
	var d domain.CanonDelta
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	d.SceneID = sceneID
	if err := h.agentSvc.RecordStateDelta(r.Context(), &d); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handlers) ListAgentRuns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}
