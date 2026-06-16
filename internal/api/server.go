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
		r.Route("/stories", func(r chi.Router) {
			r.Post("/", h.CreateStory)
			r.Get("/", h.ListStories)

			r.Route("/{storyID}", func(r chi.Router) {
				r.Get("/", h.GetStory)
				r.Put("/", h.UpdateStory)
				r.Delete("/", h.DeleteStory)
				r.Get("/topology", h.Topology)

				r.Route("/scenes", func(r chi.Router) {
					r.Post("/", h.CreateScene)
					r.Get("/", h.ListScenes)
					r.Route("/{sceneID}", func(r chi.Router) {
						r.Get("/", h.GetScene)
						r.Put("/", h.UpdateScene)
						r.Delete("/", h.DeleteScene)
						r.Post("/generate", h.GenerateScene)
						r.Get("/generations", h.ListGenerations)
						r.Post("/accept", h.AcceptGeneration)
					})
				})

				r.Route("/edges", func(r chi.Router) {
					r.Post("/", h.CreateEdge)
					r.Get("/", h.ListEdges)
					r.Delete("/", h.DeleteEdge)
				})

				r.Route("/characters", func(r chi.Router) {
					r.Post("/", h.CreateCharacter)
					r.Get("/", h.ListCharacters)
				})

				r.Route("/timeline", func(r chi.Router) {
					r.Post("/", h.CreateTimelineEvent)
					r.Get("/", h.ListTimelineEvents)
				})

				r.Route("/summaries", func(r chi.Router) {
					r.Get("/level", h.GetSummaryByLevel)
					r.Get("/scenes/{sceneID}", h.GetSceneSummary)
				})
			})
		})

		r.Route("/characters/{charID}", func(r chi.Router) {
			r.Get("/", h.GetCharacter)
			r.Get("/memories", h.ListMemories)
			r.Post("/memories/search", h.SearchMemories)
		})
	})

	s.Router = r
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func middlewareRateLimit(limiter *cache.SlidingWindowRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, err := limiter.Allow(r.Context(), "api")
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
