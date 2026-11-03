package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) ListStoryRuns(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	limit := queryInt(r, "limit", 50)
	runs, err := h.runSvc.ListByStory(r.Context(), storyID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if runs == nil {
		runs = []*domain.StoryRun{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (h *Handlers) GetRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	run, err := h.runSvc.Get(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handlers) GetRunSteps(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	steps, err := h.runSvc.ListSteps(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if steps == nil {
		steps = []*domain.RunStep{}
	}
	writeJSON(w, http.StatusOK, steps)
}

func (h *Handlers) CancelRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	if err := h.runSvc.Cancel(r.Context(), runID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *Handlers) GetRunPromptSections(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	sections, err := h.runSvc.GetPromptSections(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sections == nil {
		writeJSON(w, http.StatusOK, map[string]any{"sections": []any{}, "system": "", "tokenCount": 0})
		return
	}
	writeJSON(w, http.StatusOK, sections)
}

func (h *Handlers) GetRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	limit := queryInt(r, "limit", 100)
	events, err := h.narrativeSvc.ListByRun(r.Context(), runID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if events == nil {
		events = []*domain.NarrativeEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handlers) GetRunCost(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	cost, err := h.runSvc.GetRunCost(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cost == nil {
		cost = &domain.CostSummary{ByModel: map[string]domain.ModelCost{}}
	}
	writeJSON(w, http.StatusOK, cost)
}

func (h *Handlers) GetStoryRunStats(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	stats, err := h.runSvc.GetStoryRunStats(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
