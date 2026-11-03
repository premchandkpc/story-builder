package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/premchand/story-builder/internal/story"
)

type Server struct {
	router  chi.Router
	stories *story.Service
}

func New(stories *story.Service) *Server {
	s := &Server{stories: stories}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, contentType)
	r.Get("/api/v1/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/stories", s.listStories)
		r.Post("/stories", s.createStory)
		r.Get("/stories/{storyID}/graph", s.graph)
		r.Post("/stories/{storyID}/scenes", s.createScene)
		r.Post("/stories/{storyID}/edges", s.createEdge)
		r.Post("/stories/{storyID}/characters", s.createCharacter)
		r.Get("/stories/{storyID}/characters", s.listCharacters)
		r.Get("/stories/{storyID}/character-states", s.characterStates)
		r.Post("/scenes/{sceneID}/generate", s.generate)
		r.Post("/drafts/{draftID}/accept", s.acceptDraft)
		r.Post("/drafts/{draftID}/reject", s.rejectDraft)
		r.Patch("/scenes/{sceneID}", s.updateScene)
	})
	s.router = r
	return s
}
func (s *Server) createCharacter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string   `json:"name"`
		Traits []string `json:"traits"`
	}
	if !decode(w, r, &body) {
		return
	}
	value, err := s.stories.CreateCharacter(r.Context(), chi.URLParam(r, "storyID"), body.Name, body.Traits)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (s *Server) listCharacters(w http.ResponseWriter, r *http.Request) {
	values, err := s.stories.ListCharacters(r.Context(), chi.URLParam(r, "storyID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, values)
}
func (s *Server) characterStates(w http.ResponseWriter, r *http.Request) {
	values, err := s.stories.ReplayCharacterStates(r.Context(), chi.URLParam(r, "storyID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, values)
}
func (s *Server) generate(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unavailable", "Streaming is not supported.")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	draft, err := s.stories.GenerateDraft(r.Context(), chi.URLParam(r, "sceneID"), story.TemplateGenerator{}, func(chunk string) error {
		payload, _ := json.Marshal(map[string]string{"text": chunk})
		_, err := w.Write([]byte("event: chunk\ndata: " + string(payload) + "\n\n"))
		flusher.Flush()
		return err
	})
	if err != nil {
		payload, _ := json.Marshal(map[string]string{"message": err.Error()})
		_, _ = w.Write([]byte("event: error\ndata: " + string(payload) + "\n\n"))
		flusher.Flush()
		return
	}
	payload, _ := json.Marshal(draft)
	_, _ = w.Write([]byte("event: complete\ndata: " + string(payload) + "\n\n"))
	flusher.Flush()
}
func (s *Server) acceptDraft(w http.ResponseWriter, r *http.Request) {
	value, err := s.stories.AcceptDraft(r.Context(), chi.URLParam(r, "draftID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func (s *Server) rejectDraft(w http.ResponseWriter, r *http.Request) {
	if err := s.stories.RejectDraft(r.Context(), chi.URLParam(r, "draftID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) Handler() http.Handler { return s.router }
func (s *Server) listStories(w http.ResponseWriter, r *http.Request) {
	values, err := s.stories.ListStories(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, values)
}
func (s *Server) createStory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if !decode(w, r, &body) {
		return
	}
	value, err := s.stories.CreateStory(r.Context(), body.Title)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (s *Server) graph(w http.ResponseWriter, r *http.Request) {
	value, err := s.stories.Graph(r.Context(), chi.URLParam(r, "storyID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func (s *Server) createScene(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title    string         `json:"title"`
		Brief    string         `json:"brief"`
		Position story.Position `json:"position"`
	}
	if !decode(w, r, &body) {
		return
	}
	value, err := s.stories.CreateScene(r.Context(), chi.URLParam(r, "storyID"), body.Title, body.Brief, body.Position)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (s *Server) updateScene(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title    *string         `json:"title"`
		Brief    *string         `json:"brief"`
		Position *story.Position `json:"position"`
	}
	if !decode(w, r, &body) {
		return
	}
	value, err := s.stories.UpdateScene(r.Context(), chi.URLParam(r, "sceneID"), body.Title, body.Brief, body.Position)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func (s *Server) createEdge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FromSceneID string `json:"from_scene_id"`
		ToSceneID   string `json:"to_scene_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	value, err := s.stories.CreateEdge(r.Context(), chi.URLParam(r, "storyID"), body.FromSceneID, body.ToSceneID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func contentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, story.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "The requested story item does not exist.")
	case errors.Is(err, story.ErrInvalidTitle), errors.Is(err, story.ErrInvalidEdge):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, story.ErrCycle):
		writeError(w, http.StatusUnprocessableEntity, "cycle_detected", "That connection would create a cycle.")
	case errors.Is(err, story.ErrCrossStoryEdge), errors.Is(err, story.ErrDuplicateEdge):
		writeError(w, http.StatusConflict, "invalid_edge", err.Error())
	case errors.Is(err, story.ErrStaleDraft):
		writeError(w, http.StatusConflict, "stale_draft", "Canon changed. Regenerate this draft.")
	case errors.Is(err, story.ErrDraftNotReady):
		writeError(w, http.StatusConflict, "draft_not_ready", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "The graph could not be saved.")
	}
}
