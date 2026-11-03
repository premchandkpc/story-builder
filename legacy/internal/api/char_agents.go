package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/agents"
)

func (h *Handlers) GetCharAgentState(w http.ResponseWriter, r *http.Request) {
	charID := chi.URLParam(r, "charID")
	if charID == "" {
		writeError(w, http.StatusBadRequest, "charID required")
		return
	}

	state, err := h.charAgentSvc.GetAgentState(r.Context(), charID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

type broadcastRequest struct {
	StoryID   string         `json:"story_id"`
	EventType string         `json:"event_type"`
	Data      map[string]any `json:"data,omitempty"`
}

func (h *Handlers) BroadcastCharEvent(w http.ResponseWriter, r *http.Request) {
	var req broadcastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.StoryID == "" || req.EventType == "" {
		writeError(w, http.StatusBadRequest, "story_id and event_type required")
		return
	}

	if err := h.charAgentSvc.BroadcastEvent(r.Context(), req.StoryID, req.EventType, req.Data); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "broadcasted"})
}

func (h *Handlers) ListCharAgentProposals(w http.ResponseWriter, r *http.Request) {
	sceneID := r.URL.Query().Get("scene_id")
	if sceneID == "" {
		writeError(w, http.StatusBadRequest, "scene_id query param required")
		return
	}

	proposals, err := h.charAgentSvc.GetProposals(r.Context(), sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if proposals == nil {
		proposals = []agents.ProposalSnapshot{}
	}
	writeJSON(w, http.StatusOK, proposals)
}
