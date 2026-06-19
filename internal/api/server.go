package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/premchand/story-builder/internal/cache"
)

type Server struct {
	Router    *chi.Mux
	handler   *Handlers
}

func NewServer(h *Handlers, limiter *cache.SlidingWindowRateLimiter) *Server {
	s := &Server{
		handler: h,
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:           300,
	}))

	if limiter != nil {
		r.Use(middlewareRateLimit(limiter))
	}

	r.Get("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// ─── Stories ──────────────────────────────────────────
		r.Route("/stories", func(r chi.Router) {
			r.Post("/", h.CreateStory)
			r.Get("/", h.ListStories)
			r.With(contextTimeout(5*time.Minute)).Post("/generate", h.GenerateStory)
			r.Post("/generate-title", h.GenerateTitle)

			r.Route("/{storyID}", func(r chi.Router) {
				r.Get("/", h.GetStory)
				r.Put("/", h.UpdateStory)
				r.Delete("/", h.DeleteStory)

				// V2-compat topology (includes topological_order)
				r.Get("/topology", h.V2Topology)

				// V2-compat nodes
				r.Route("/nodes", func(r chi.Router) {
					r.Get("/", h.ListNodes)
					r.Post("/", h.CreateNode)
					r.Route("/{nodeID}", func(r chi.Router) {
						r.Get("/", h.GetNode)
						r.Put("/", h.UpdateNode)
						r.Delete("/", h.DeleteNode)
						r.Post("/generate", h.V2GenerateNode)
						r.Get("/generations", h.V2ListNodeGenerations)
						r.Post("/accept", h.V2AcceptGeneration)
						// Interactive scene (stub — not yet implemented)
						r.Route("/scene", func(r chi.Router) {
							r.Put("/structure", h.NotImplemented)
							r.Get("/structure", h.NotImplemented)
							r.Post("/start", h.NotImplemented)
							r.Post("/next", h.NotImplemented)
							r.Post("/finish", h.NotImplemented)
							r.Get("/turns", h.NotImplemented)
						})
					})
				})

				// Edges
				r.Route("/edges", func(r chi.Router) {
					r.Post("/", h.V2CreateEdge)
					r.Get("/", h.V2ListEdges)
					r.Delete("/", h.DeleteEdge)
				})

				// Chapters
				r.Route("/chapters", func(r chi.Router) {
					r.Get("/", h.ListChapters)
					r.Post("/", h.CreateChapter)
					r.Route("/{chapterID}", func(r chi.Router) {
						r.Get("/", h.GetChapter)
						r.Put("/", h.UpdateChapter)
						r.Delete("/", h.NotImplemented)
						r.Get("/scenes", h.NotImplemented)
					})
				})

				r.Route("/characters", func(r chi.Router) {
					r.Post("/", h.CreateCharacter)
					r.Get("/", h.ListCharacters)
				})

				r.Route("/locations", func(r chi.Router) {
					r.Post("/", h.CreateLocation)
					r.Get("/", h.ListStoryLocations)
				})

				r.Route("/timeline", func(r chi.Router) {
					r.Post("/", h.CreateTimelineEvent)
					r.Get("/", h.ListTimelineEvents)
				})

				r.Route("/summaries", func(r chi.Router) {
					r.Get("/level", h.GetSummaryByLevel)
					r.Get("/count", h.NotImplemented)
					r.Get("/elevate", h.NotImplemented)
					r.Get("/scenes/{sceneID}", h.GetSceneSummary)
					r.Get("/nodes/{nodeID}", h.GetSceneSummary)
				})
			})
		})

		// ─── Characters (top-level) ──────────────────────────
		r.Get("/characters", h.V2ListCharacters)
		r.Post("/characters", h.V2CreateCharacter)
		r.Route("/characters/{charID}", func(r chi.Router) {
			r.Get("/", h.V2GetCharacter)
			r.Put("/", h.V2UpdateCharacter)
			r.Get("/memories", h.ListMemories)
			r.Post("/memories/search", h.SearchMemories)
			r.Get("/traits", h.EmptyArray)
			r.Post("/traits/assign", h.NotImplemented)
			r.Delete("/traits/{traitID}", h.NotImplemented)
		})

		// ─── Actors (stub) ───────────────────────────────────
		r.Get("/actors", h.EmptyArray)
		r.Post("/actors", h.NotImplemented)
		r.Route("/actors/{id}", func(r chi.Router) {
			r.Get("/", h.NotImplemented)
			r.Put("/", h.NotImplemented)
		})

		// ─── Character Traits (stub) ─────────────────────────
		r.Get("/character-traits", h.EmptyArray)
		r.Get("/character-traits/{id}", h.NotImplemented)
		r.Post("/character-traits", h.NotImplemented)

		// ─── Story Blueprint ──────────────────────────────────
		r.Route("/stories/{storyID}/blueprint", func(r chi.Router) {
			r.Get("/", h.GetBlueprint)
			r.Put("/", h.UpdateBlueprint)
		})

		// ─── Generations ─────────────────────────────────────
		r.Route("/generations/{genID}", func(r chi.Router) {
			r.Get("/status", h.GetGenerationStatus)
			r.Get("/progress", h.SSEGenerationProgress)
		})

		// ─── Bible ────────────────────────────────────────────
		r.Route("/stories/{storyID}/bible", func(r chi.Router) {
			r.Get("/", h.GetBible)
			r.Post("/generate", h.GenerateBible)
			r.Put("/", h.UpdateBible)
			r.Delete("/", h.DeleteBible)
		})

		// ─── Locations ───────────────────────────────────────
		r.Route("/locations/{id}", func(r chi.Router) {
			r.Get("/", h.GetLocation)
			r.Put("/", h.UpdateLocation)
		})

		// ─── Lore (stub) ─────────────────────────────────────
		r.Get("/lore", h.EmptyArray)
		r.Post("/lore", h.NotImplemented)
		r.Post("/lore/search", h.NotImplemented)

		// ─── Casting (stub) ──────────────────────────────────
		r.Post("/stories/{storyID}/casting", h.NotImplemented)
		r.Get("/stories/{storyID}/casting", h.EmptyArray)
		r.Get("/casting/actor/{actorID}", h.NotImplemented)
		r.Get("/casting/character/{characterID}", h.NotImplemented)
	})

	s.Router = r
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func middlewareRateLimit(limiter *cache.SlidingWindowRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, err := limiter.Allow(r.Context(), "http:api")
			if err != nil || !ok {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type timeoutWriter struct {
	http.ResponseWriter
	done chan struct{}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	select {
	case <-tw.done:
		return 0, http.ErrHandlerTimeout
	default:
		return tw.ResponseWriter.Write(b)
	}
}

func contextTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
