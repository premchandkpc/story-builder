package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/graph"
)

func chiRequest(method, path string, body *bytes.Buffer, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	chiCtx := chi.NewRouteContext()
	for k, v := range params {
		chiCtx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx)
	return req.WithContext(ctx)
}

func TestParseUUID(t *testing.T) {
	id, err := parseUUID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("expected valid UUID to be accepted: %v", err)
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected UUID: %s", id)
	}

	if _, err := parseUUID("not-a-uuid"); err == nil {
		t.Fatal("expected invalid UUID to be rejected")
	}
}

func TestParseOptionalUUID(t *testing.T) {
	nilStr := "foo"

	v, err := parseOptionalUUID(nil)
	if err != nil || v != nil {
		t.Fatal("expected nil input to return nil")
	}

	empty := ""
	v, err = parseOptionalUUID(&empty)
	if err != nil || v != nil {
		t.Fatal("expected empty input to return nil")
	}

	valid := "550e8400-e29b-41d4-a716-446655440000"
	v, err = parseOptionalUUID(&valid)
	if err != nil {
		t.Fatalf("expected valid UUID to be accepted: %v", err)
	}
	if v.String() != valid {
		t.Fatalf("unexpected UUID: %s", v)
	}

	_, err = parseOptionalUUID(&nilStr)
	if err == nil {
		t.Fatal("expected invalid UUID to be rejected")
	}
}

func TestStoryCreateHandler_BlankTitle(t *testing.T) {
	mem := graph.NewMemoryStore()
	h := &StoryHandler{Service: NewGraphStoryService(mem)}

	body := strings.NewReader(`{"title":"  "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stories/", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoryCreateHandler_Success(t *testing.T) {
	mem := graph.NewMemoryStore()
	h := &StoryHandler{Service: NewGraphStoryService(mem)}

	body := strings.NewReader(`{"title":"The Red Path"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stories/", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var story graph.Story
	if err := json.NewDecoder(w.Body).Decode(&story); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if story.Title != "The Red Path" {
		t.Fatalf("expected title 'The Red Path', got %q", story.Title)
	}
}

func TestNodeCreateHandler_InvalidCharRefs(t *testing.T) {
	mem := graph.NewMemoryStore()
	story, _ := mem.CreateStory("Test Story")
	nh := &NodeHandler{Service: NewGraphNodeService(mem)}

	body := bytes.NewBufferString(`{"beat_intent":"test","character_refs":["not-a-uuid"]}`)
	req := chiRequest(http.MethodPost, "/api/v1/stories/{storyID}/nodes/", body, map[string]string{"storyID": story.ID.String()})
	w := httptest.NewRecorder()

	nh.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNodeCreateHandler_Success(t *testing.T) {
	mem := graph.NewMemoryStore()
	story, _ := mem.CreateStory("Test Story")
	nh := &NodeHandler{Service: NewGraphNodeService(mem)}

	body := bytes.NewBufferString(`{"beat_intent":"open with conflict","pov":"hero","tone":"tense","target_words":500}`)
	req := chiRequest(http.MethodPost, "/api/v1/stories/{storyID}/nodes/", body, map[string]string{"storyID": story.ID.String()})
	w := httptest.NewRecorder()

	nh.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var node graph.Node
	if err := json.NewDecoder(w.Body).Decode(&node); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if node.BeatIntent != "open with conflict" {
		t.Fatalf("unexpected beat_intent: %q", node.BeatIntent)
	}
}

func TestStoryCreateEdge_InvalidNodeRefs(t *testing.T) {
	mem := graph.NewMemoryStore()
	story, _ := mem.CreateStory("Test Story")
	sh := &StoryHandler{Service: NewGraphStoryService(mem)}

	body := bytes.NewBufferString(`{"from_node":"bad-uuid","to_node":"also-bad","edge_type":"seq"}`)
	req := chiRequest(http.MethodPost, "/api/v1/stories/{storyID}/edges/", body, map[string]string{"storyID": story.ID.String()})
	w := httptest.NewRecorder()

	sh.CreateEdge(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSceneHandler_SetStructure_InvalidNodeID(t *testing.T) {
	sh := &SceneHandler{SceneService: NewMemorySceneService()}

	body := bytes.NewBufferString(`{"scene_structure":{"flow_type":"monologue","situation_flow":""}}`)
	req := chiRequest(http.MethodPut, "/api/v1/stories/{storyID}/nodes/{id}/scene/structure", body, map[string]string{"id": "not-a-uuid"})
	w := httptest.NewRecorder()

	sh.SetStructure(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
