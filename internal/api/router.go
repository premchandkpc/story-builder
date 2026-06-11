package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	router      *chi.Mux
	characterHandler *CharacterHandler
	locationHandler  *LocationHandler
	loreHandler      *LoreHandler
	storyHandler     *StoryHandler
	nodeHandler      *NodeHandler
	genHandler       *GenerationHandler
}

func NewServer(
	charH *CharacterHandler,
	locH *LocationHandler,
	loreH *LoreHandler,
	storyH *StoryHandler,
	nodeH *NodeHandler,
	genH *GenerationHandler,
) *Server {
	s := &Server{
		router:      chi.NewRouter(),
		characterHandler: charH,
		locationHandler:  locH,
		loreHandler:      loreH,
		storyHandler:     storyH,
		nodeHandler:      nodeH,
		genHandler:       genH,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	r := s.router

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/characters", func(r chi.Router) {
			r.Post("/", s.characterHandler.Create)
			r.Get("/", s.characterHandler.List)
			r.Get("/{id}", s.characterHandler.Get)
			r.Put("/{id}", s.characterHandler.Update)
		})

		r.Route("/locations", func(r chi.Router) {
			r.Post("/", s.locationHandler.Create)
			r.Get("/", s.locationHandler.List)
			r.Get("/{id}", s.locationHandler.Get)
			r.Put("/{id}", s.locationHandler.Update)
		})

		r.Route("/lore", func(r chi.Router) {
			r.Post("/", s.loreHandler.Create)
			r.Get("/", s.loreHandler.List)
			r.Post("/search", s.loreHandler.Search)
		})

		r.Route("/stories", func(r chi.Router) {
			r.Post("/", s.storyHandler.Create)
			r.Get("/", s.storyHandler.List)
			r.Get("/{id}", s.storyHandler.Get)
			r.Route("/{storyID}/nodes", func(r chi.Router) {
				r.Post("/", s.nodeHandler.Create)
				r.Get("/", s.nodeHandler.List)
				r.Get("/{id}", s.nodeHandler.Get)
				r.Put("/{id}", s.nodeHandler.Update)
				r.Post("/{id}/generate", s.genHandler.Generate)
				r.Post("/{id}/accept", s.genHandler.AcceptGeneration)
				r.Get("/{id}/generations", s.genHandler.ListGenerations)
			})
			r.Route("/{storyID}/edges", func(r chi.Router) {
				r.Post("/", s.storyHandler.CreateEdge)
				r.Get("/", s.storyHandler.ListEdges)
			})
			r.Get("/{storyID}/topology", s.storyHandler.Topology)
		})
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
