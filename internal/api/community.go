package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) ListAgentConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.agentCfgSvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

func (h *Handlers) GetAgentConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	cfg, err := h.agentCfgSvc.Get(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg == nil {
		writeError(w, http.StatusNotFound, "agent config not found")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handlers) CreateAgentConfig(w http.ResponseWriter, r *http.Request) {
	var cfg domain.AgentConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if cfg.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := h.agentCfgSvc.Create(r.Context(), &cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cfg)
}

func (h *Handlers) DeleteAgentConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.agentCfgSvc.Delete(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) ExportAgentConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	cfg, err := h.agentCfgSvc.Export(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handlers) ImportAgentConfig(w http.ResponseWriter, r *http.Request) {
	var cfg domain.AgentConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if cfg.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := h.agentCfgSvc.Import(r.Context(), &cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cfg)
}

func (h *Handlers) ListMarketplaceAgentConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.agentCfgSvc.ListShared(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configs)
}
