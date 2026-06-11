package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	router           *chi.Mux
	characterHandler *CharacterHandler
	actorHandler     *ActorHandler
	traitHandler     *CharacterTraitHandler
	castingHandler   *CastingHandler
	locationHandler  *LocationHandler
	loreHandler      *LoreHandler
	storyHandler     *StoryHandler
	nodeHandler      *NodeHandler
	genHandler       *GenerationHandler
	sceneHandler     *SceneHandler
	summaryHandler   *SummaryHandler
	storyGenHandler  *StoryGeneratorHandler
}

func NewServer(
	charH *CharacterHandler,
	actorH *ActorHandler,
	traitH *CharacterTraitHandler,
	castingH *CastingHandler,
	locH *LocationHandler,
	loreH *LoreHandler,
	storyH *StoryHandler,
	nodeH *NodeHandler,
	genH *GenerationHandler,
	sceneH *SceneHandler,
	summaryH *SummaryHandler,
	storyGenH *StoryGeneratorHandler,
) *Server {
	s := &Server{
		router:           chi.NewRouter(),
		characterHandler: charH,
		actorHandler:     actorH,
		traitHandler:     traitH,
		castingHandler:   castingH,
		locationHandler:  locH,
		loreHandler:      loreH,
		storyHandler:     storyH,
		nodeHandler:      nodeH,
		genHandler:       genH,
		sceneHandler:     sceneH,
		summaryHandler:   summaryH,
		storyGenHandler:  storyGenH,
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
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Post("/stories/generate", s.storyGenHandler.Generate)

		r.Route("/actors", func(r chi.Router) {
			r.Post("/", s.actorHandler.Create)
			r.Get("/", s.actorHandler.List)
			r.Get("/{id}", s.actorHandler.Get)
			r.Put("/{id}", s.actorHandler.Update)
		})

		r.Route("/characters", func(r chi.Router) {
			r.Post("/", s.characterHandler.Create)
			r.Get("/", s.characterHandler.List)
			r.Get("/{id}", s.characterHandler.Get)
			r.Put("/{id}", s.characterHandler.Update)
			r.Route("/{characterID}/traits", func(r chi.Router) {
				r.Post("/assign", s.traitHandler.Assign)
				r.Delete("/{traitID}", s.traitHandler.Unassign)
				r.Get("/", s.traitHandler.GetAssignments)
			})
		})

		r.Route("/character-traits", func(r chi.Router) {
			r.Post("/", s.traitHandler.Create)
			r.Get("/", s.traitHandler.List)
			r.Get("/{id}", s.traitHandler.Get)
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
			r.Route("/{storyID}/casting", func(r chi.Router) {
				r.Post("/", s.castingHandler.Create)
				r.Get("/", s.castingHandler.ListForStory)
			})
			r.Route("/{storyID}/blueprint", func(r chi.Router) {
				r.Post("/", s.storyHandler.UpsertBlueprint)
				r.Get("/", s.storyHandler.GetBlueprint)
			})
			r.Route("/{storyID}/timeline", func(r chi.Router) {
				r.Post("/", s.storyHandler.UpsertTimelineEvent)
				r.Get("/", s.storyHandler.ListTimelineEvents)
			})
			r.Route("/{storyID}/nodes", func(r chi.Router) {
				r.Post("/", s.nodeHandler.Create)
				r.Get("/", s.nodeHandler.List)
				r.Get("/{id}", s.nodeHandler.Get)
				r.Put("/{id}", s.nodeHandler.Update)
				r.Post("/{id}/generate", s.genHandler.Generate)
				r.Post("/{id}/accept", s.genHandler.AcceptGeneration)
				r.Get("/{id}/generations", s.genHandler.ListGenerations)
				r.Put("/{id}/scene/structure", s.sceneHandler.SetStructure)
				r.Get("/{id}/scene/structure", s.sceneHandler.GetStructure)
				r.Post("/{id}/scene/start", s.sceneHandler.Start)
				r.Post("/{id}/scene/next", s.sceneHandler.Next)
				r.Post("/{id}/scene/finish", s.sceneHandler.Finish)
				r.Get("/{id}/scene/turns", s.sceneHandler.Turns)
			})
			r.Route("/{storyID}/edges", func(r chi.Router) {
				r.Post("/", s.storyHandler.CreateEdge)
				r.Get("/", s.storyHandler.ListEdges)
			})
			r.Get("/{storyID}/topology", s.storyHandler.Topology)
		})

		r.Route("/casting", func(r chi.Router) {
			r.Get("/actor/{actorID}", s.castingHandler.ListForActor)
			r.Get("/character/{characterID}", s.castingHandler.ListForCharacter)
		})

		r.Route("/stories/{storyID}/summaries", func(r chi.Router) {
			r.Get("/level", s.summaryHandler.GetByLevel)
			r.Get("/count", s.summaryHandler.CountByLevel)
			r.Get("/elevate", s.summaryHandler.ShouldElevate)
			r.Get("/nodes/{nodeID}", s.summaryHandler.GetSceneSummary)
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
