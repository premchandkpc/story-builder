package api

import (
	"net/http"
)

// EmptyArray is a stub handler that returns HTTP 200 with an empty JSON array.
// Used for not-yet-implemented list endpoints so the frontend receives [] instead of null.
func (h *Handlers) EmptyArray(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

// NotImplemented is a stub handler that returns HTTP 501.
func (h *Handlers) NotImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not yet implemented")
}
