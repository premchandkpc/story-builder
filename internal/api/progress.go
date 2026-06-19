package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/service"
)

type ProgressHub struct {
	mu     sync.RWMutex
	subs   map[string]map[chan service.ProgressEvent]struct{}
}

func NewProgressHub() *ProgressHub {
	return &ProgressHub{
		subs: make(map[string]map[chan service.ProgressEvent]struct{}),
	}
}

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

func (h *ProgressHub) PublishEvent(genID, step, status string) {
	h.Publish(genID, service.ProgressEvent{GenID: genID, Step: step, Status: status})
}

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
		case <-r.Context().Done():
			return
		}
	}
}
