package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/log"
)

// Server wraps the chi router and handler dependencies.
type Server struct {
	Router    *chi.Mux
	handler   *Handlers
}

// NewServer creates a chi router, applies middleware, and registers all routes.
func NewServer(h *Handlers, limiter *cache.SlidingWindowRateLimiter) *Server {
	s := &Server{
		handler: h,
	}

	r := chi.NewRouter()

	r.Use(log.StructuredHTTPLogger(slog.Default()))
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
		// Stories CRUD + LLM generation
		r.Route("/stories", func(r chi.Router) {
			r.Post("/", h.CreateStory)
			r.Get("/", h.ListStories)
			r.With(contextTimeout(5*time.Minute)).Post("/generate", h.GenerateStory)
			r.Post("/generate-title", h.GenerateTitle)

			r.Route("/{storyID}", func(r chi.Router) {
				r.Get("/", h.GetStory)
				r.Put("/", h.UpdateStory)
				r.Delete("/", h.DeleteStory)

				r.Get("/topology", h.V2Topology)

				// V2 graph nodes
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
					})
				})

				// Edges
				r.Route("/edges", func(r chi.Router) {
					r.Post("/", h.V2CreateEdge)
					r.Get("/", h.V2ListEdges)
					r.Delete("/", h.DeleteEdge)
					r.Delete("/{edgeID}", h.DeleteEdgeByID)
				})

				// Chapters (deprecated — no new features)
				r.Route("/chapters", func(r chi.Router) {
					r.Get("/", h.ListChapters)
					r.Post("/", h.CreateChapter)
					r.Route("/{chapterID}", func(r chi.Router) {
						r.Get("/", h.GetChapter)
						r.Put("/", h.UpdateChapter)
						r.Delete("/", h.DeleteChapter)
					})
				})

				// Story-scoped character + location + timeline + summary routes
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
					r.Get("/nodes/{nodeID}", h.GetSceneSummary)
				})

				r.Get("/metrics/llm", h.GetLlmMetrics)
				r.Get("/critic-scores", h.ListCriticScores)
				r.Get("/bibles/referencing", h.ListReferencingBibles)
				r.Post("/bibles/link", h.LinkBibleToStory)
				r.Post("/bibles/unlink", h.UnlinkBibleFromStory)
				r.Post("/timeline/cross-story", h.CreateCrossStoryEvent)
				r.Get("/timeline/cross-story", h.ListCrossStoryEvents)
				r.Post("/characters/{charID}/migrate", h.MigrateCharacter)
			})
		})

		// Top-level character CRUD
		r.Get("/characters", h.V2ListCharacters)
		r.Post("/characters", h.V2CreateCharacter)
		r.Route("/characters/{charID}", func(r chi.Router) {
			r.Get("/", h.V2GetCharacter)
			r.Put("/", h.V2UpdateCharacter)
			r.Get("/memories", h.ListMemories)
			r.Post("/memories/search", h.SearchMemories)
		})

		// Story blueprint
		r.Route("/stories/{storyID}/blueprint", func(r chi.Router) {
			r.Get("/", h.GetBlueprint)
			r.Put("/", h.UpdateBlueprint)
		})

		// Generation status and SSE progress
		r.Route("/generations/{genID}", func(r chi.Router) {
			r.Get("/status", h.GetGenerationStatus)
			r.Get("/progress", h.SSEGenerationProgress)
		})

		// Story bible
		r.Route("/stories/{storyID}/bible", func(r chi.Router) {
			r.Get("/", h.GetBible)
			r.Post("/generate", h.GenerateBible)
			r.Put("/", h.UpdateBible)
			r.Delete("/", h.DeleteBible)
		})

		// Top-level location lookup
		r.Route("/locations/{id}", func(r chi.Router) {
			r.Get("/", h.GetLocation)
			r.Put("/", h.UpdateLocation)
		})

		// Experimental / in-development features
		r.Route("/experimental", func(r chi.Router) {
			// Scene turns (interactive turn-by-turn generation)
			r.Route("/stories/{storyID}/nodes/{nodeID}/scene", func(r chi.Router) {
				r.Get("/structure", h.NotImplemented)
				r.Put("/structure", h.NotImplemented)
				r.Post("/start", h.NotImplemented)
				r.Post("/next", h.NotImplemented)
				r.Post("/finish", h.NotImplemented)
				r.Get("/turns", h.ListSceneTurns)
				r.Get("/deltas", h.ListSceneCanonDeltas)
				r.Post("/deltas", h.RecordCanonDelta)
				r.Get("/turns/role", h.ListSceneTurnsByRole)
			})

			// Agent runs
			r.Get("/agent-runs", h.ListAgentRuns)

			// Actors
			r.Get("/actors", h.EmptyArray)
			r.Post("/actors", h.NotImplemented)
			r.Route("/actors/{id}", func(r chi.Router) {
				r.Get("/", h.NotImplemented)
				r.Put("/", h.NotImplemented)
			})

			// Character traits
			r.Get("/character-traits", h.EmptyArray)
			r.Get("/character-traits/{id}", h.NotImplemented)
			r.Post("/character-traits", h.NotImplemented)
			r.Post("/characters/{charID}/traits/assign", h.NotImplemented)
			r.Delete("/characters/{charID}/traits/{traitID}", h.NotImplemented)

			// Lore
			r.Get("/lore", h.EmptyArray)
			r.Post("/lore", h.NotImplemented)
			r.Post("/lore/search", h.NotImplemented)

			// Casting
			r.Post("/stories/{storyID}/casting", h.NotImplemented)
			r.Get("/stories/{storyID}/casting", h.EmptyArray)
			r.Get("/casting/actor/{actorID}", h.NotImplemented)
			r.Get("/casting/character/{characterID}", h.NotImplemented)
		})

		// Agent configs — community features
		r.Route("/agent-configs", func(r chi.Router) {
			r.Get("/", h.ListAgentConfigs)
			r.Post("/", h.CreateAgentConfig)
			r.Get("/marketplace", h.ListMarketplaceAgentConfigs)
			r.Route("/{name}", func(r chi.Router) {
				r.Get("/", h.GetAgentConfig)
				r.Delete("/", h.DeleteAgentConfig)
				r.Get("/export", h.ExportAgentConfig)
				r.Post("/import", h.ImportAgentConfig)
			})
		})

		// Character agents — runtime inspection + control
		r.Route("/agents/characters", func(r chi.Router) {
			r.Post("/broadcast", h.BroadcastCharEvent)
			r.Get("/proposals", h.ListCharAgentProposals)
			r.Route("/{charID}", func(r chi.Router) {
				r.Get("/state", h.GetCharAgentState)
			})
		})

		// Story runs — durable orchestration tracking
		r.Route("/runs/{runID}", func(r chi.Router) {
			r.Get("/", h.GetRun)
			r.Get("/steps", h.GetRunSteps)
			r.Post("/cancel", h.CancelRun)
		})
		r.Route("/stories/{storyID}/runs", func(r chi.Router) {
			r.Get("/", h.ListStoryRuns)
		})

		// Narrative events — append-only state mutation log
		r.Route("/stories/{storyID}/narrative-events", func(r chi.Router) {
			r.Get("/", h.ListNarrativeEvents)
		})
		r.Route("/experimental/stories/{storyID}/nodes/{nodeID}/narrative-events", func(r chi.Router) {
			r.Get("/", h.ListNarrativeEventsByScene)
		})
	})

	s.Router = r
	return s
}

// writeJSON serializes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// middlewareRateLimit returns a middleware that enforces a sliding-window rate limit.
// Keys are derived from the request method + route pattern for finer-grained control.
func middlewareRateLimit(limiter *cache.SlidingWindowRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Derive key from method + first two path segments for route-group awareness
			key := routeRateLimitKey(r.Method, r.URL.Path)
			ok, err := limiter.Allow(r.Context(), key)
			if err != nil || !ok {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// routeRateLimitKey produces a rate-limit key scoped by HTTP method and route group.
// Examples: "POST:/api/v1/stories/generate", "GET:/api/v1/stories/{id}/topology"
func routeRateLimitKey(method, path string) string {
	return method + ":" + path
}

// timeoutWriter wraps http.ResponseWriter to detect handler deadline exceeded.
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

// contextTimeout returns middleware that applies a context deadline to the request.
func contextTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
