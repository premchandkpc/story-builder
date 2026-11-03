package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/service"
)

// ProgressHub is an in-memory pub/sub hub for SSE generation progress events.
// Subscribers receive ProgressEvent messages on a buffered channel.
type ProgressHub struct {
	mu     sync.RWMutex
	subs   map[string]map[chan service.ProgressEvent]struct{}
}

// NewProgressHub creates an empty ProgressHub.
func NewProgressHub() *ProgressHub {
	return &ProgressHub{
		subs: make(map[string]map[chan service.ProgressEvent]struct{}),
	}
}

// Subscribe registers a channel for a given genID. Returns the channel.
func (h *ProgressHub) Subscribe(genID string) chan service.ProgressEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan service.ProgressEvent, 16)
	if h.subs[genID] == nil {
		h.subs[genID] = make(map[chan service.ProgressEvent]struct{})
	}
	h.subs[genID][ch] = struct{}{}
	return ch
}

// Unsubscribe removes a channel and closes it. Cleans up the map entry when empty.
func (h *ProgressHub) Unsubscribe(genID string, ch chan service.ProgressEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.subs[genID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(h.subs, genID)
		}
	}
}

// Publish sends an event to all subscribers of a genID. Drops if a subscriber is slow.
func (h *ProgressHub) Publish(genID string, event service.ProgressEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if subs, ok := h.subs[genID]; ok {
		for ch := range subs {
			select {
			case ch <- event:
			default:
				slog.Warn("progress hub: dropping event for slow subscriber", "genId", genID)
			}
		}
	}
}

// PublishEvent is a convenience wrapper that creates a ProgressEvent and publishes it.
func (h *ProgressHub) PublishEvent(genID, step, status string) {
	h.Publish(genID, service.ProgressEvent{GenID: genID, Step: step, Status: status})
}

// SSEGenerationProgress handles GET /api/v1/generations/{genID}/progress.
// Streams Server-Sent Events for generation lifecycle (connected → progress → complete/error).
func (h *Handlers) SSEGenerationProgress(w http.ResponseWriter, r *http.Request) {
	genID := chi.URLParam(r, "genID")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := h.progress.Subscribe(genID)
	defer h.progress.Unsubscribe(genID, ch)

	fmt.Fprintf(w, "event: connected\ndata: {\"genId\":%q}\n\n", genID)
	flusher.Flush()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			errStr := ""
			if evt.Error != "" {
				errStr = ",\"error\":" + evt.Error
			}
			fmt.Fprintf(w, "event: progress\ndata: {\"genId\":%q,\"step\":%q,\"status\":%q%s}\n\n", evt.GenID, evt.Step, evt.Status, errStr)
			flusher.Flush()

			if evt.Status == "complete" || evt.Status == "failed" {
				fmt.Fprintf(w, "event: %s\ndata: {\"genId\":%q}\n\n", evt.Status, evt.GenID)
				flusher.Flush()
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
