package api

import (
	"net/http"
)

func (h *Handlers) EmptyArray(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (h *Handlers) NotImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not yet implemented")
}
