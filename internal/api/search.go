package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/premchand/story-builder/internal/repository"
)

type SearchService interface {
	Search(ctx context.Context, query, entityType, storyID string, limit, offset int) (*repository.SearchResult, error)
}

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "query parameter q is required")
		return
	}
	entityType := r.URL.Query().Get("entity")
	storyID := r.URL.Query().Get("story_id")
	limit := queryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := queryInt(r, "offset", 0)

	slog.Debug("search request", "query", q, "entity", entityType, "story_id", storyID, "limit", limit, "offset", offset)

	result, err := h.searchSvc.Search(r.Context(), q, entityType, storyID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if result == nil {
		result = &repository.SearchResult{Hits: []repository.SearchHit{}}
	}
	writeJSON(w, http.StatusOK, result)
}
