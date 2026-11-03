package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) GetScenePlan(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	nodeID := chi.URLParam(r, "nodeID")
	if storyID == "" || nodeID == "" {
		writeError(w, http.StatusBadRequest, "storyID and nodeID required")
		return
	}

	plan, err := h.plannerSvc.PlanScene(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, plan)
}

type genDiffRequest struct {
	Against string `json:"against"`
}

func (h *Handlers) GetGenerationDiff(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	nodeID := chi.URLParam(r, "nodeID")
	genID := chi.URLParam(r, "genID")
	if storyID == "" || nodeID == "" || genID == "" {
		writeError(w, http.StatusBadRequest, "storyID, nodeID, and genID required")
		return
	}

	against := r.URL.Query().Get("against")
	if against == "" {
		var req genDiffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Against != "" {
			against = req.Against
		}
	}
	if against == "" {
		writeError(w, http.StatusBadRequest, "missing 'against' query param or body field")
		return
	}

	diff, err := h.diffSvc.GenDiff(r.Context(), storyID, nodeID, genID, against)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, diff)
}
