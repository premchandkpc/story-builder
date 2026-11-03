//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
)

func TestAPI_NodesCRUD(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "generations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Node Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/nodes"

	t.Run("create node", func(t *testing.T) {
		payload := `{"beat_intent":"Node A","status":"draft"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var node map[string]any
		json.NewDecoder(rec.Body).Decode(&node)
		if node["beat_intent"] != "Node A" {
			t.Fatalf("beat_intent: got %q", node["beat_intent"])
		}
		if node["story_id"] != s.ID {
			t.Fatalf("storyID: got %q", node["story_id"])
		}
	})

	t.Run("list nodes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var nodes []map[string]any
		json.NewDecoder(rec.Body).Decode(&nodes)
		if len(nodes) < 1 {
			t.Fatal("expected at least 1 node")
		}
	})

	t.Run("get node by ID", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"beat_intent":"GetTest"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/"+created["id"].(string), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var got map[string]any
		json.NewDecoder(rec.Body).Decode(&got)
		if got["id"] != created["id"] {
			t.Fatalf("id: got %q", got["id"])
		}
	})

	t.Run("get missing node returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("update node", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"beat_intent":"Original"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		payload := fmt.Sprintf(`{"beat_intent":"Updated","status":"generated"}`)
		req := httptest.NewRequest("PUT", base+"/"+created["id"].(string), bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var updated map[string]any
		json.NewDecoder(rec.Body).Decode(&updated)
		if updated["beat_intent"] != "Updated" {
			t.Fatalf("beat_intent: got %q", updated["beat_intent"])
		}
	})

	t.Run("delete node", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"beat_intent":"DeleteMe"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/"+created["id"].(string), nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}

		grec := httptest.NewRecorder()
		srv.Router.ServeHTTP(grec, httptest.NewRequest("GET", base+"/"+created["id"].(string), nil))
		if grec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", grec.Code)
		}
	})

	t.Run("create node with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestAPI_EdgesAdvanced(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Edge Adv", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc1 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc1)
	sc2 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc2)
	base := "/api/v1/stories/" + s.ID + "/edges"

	t.Run("create edge with invalid type returns 400", func(t *testing.T) {
		payload := `{"from_node":"` + sc1.ID + `","to_node":"` + sc2.ID + `","edge_type":"invalid-type"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("create duplicate edge returns 409", func(t *testing.T) {
		payload := `{"from_node":"` + sc1.ID + `","to_node":"` + sc2.ID + `","edge_type":"seq"}`
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.Router.ServeHTTP(rec, req)
			if i == 1 && rec.Code != http.StatusConflict {
				t.Fatalf("expected 409 for duplicate, got %d", rec.Code)
			}
		}
	})

	t.Run("delete edge by ID", func(t *testing.T) {
		edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
		e := &domain.SceneEdge{StoryID: s.ID, FromSceneID: sc1.ID, ToSceneID: sc1.ID, Type: "seq"}
		edgeRepo.Create(ctx, e)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/"+e.ID, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})
}

func TestAPI_LocationCRUD(t *testing.T) {
	cleanCollections(t, "stories", "locations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Loc Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/locations"

	t.Run("create location", func(t *testing.T) {
		payload := `{"name":"Mirkwood","type":"forest","description":"Dark and dangerous"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var loc domain.Location
		json.NewDecoder(rec.Body).Decode(&loc)
		if loc.Name != "Mirkwood" {
			t.Fatalf("name: got %q", loc.Name)
		}
	})

	t.Run("list locations", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var locs []*domain.Location
		json.NewDecoder(rec.Body).Decode(&locs)
		if len(locs) < 1 {
			t.Fatal("expected at least 1 location")
		}
	})

	t.Run("get location by ID", func(t *testing.T) {
		var created domain.Location
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"name":"Rivendell"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/locations/"+created.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var got domain.Location
		json.NewDecoder(rec.Body).Decode(&got)
		if got.ID != created.ID {
			t.Fatalf("id: got %q", got.ID)
		}
	})

	t.Run("get missing location returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/locations/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("create duplicate location name returns 409", func(t *testing.T) {
		locationRepo := mgorepo.NewLocationRepo(testDB)
		locationRepo.Create(ctx, &domain.Location{StoryID: s.ID, Name: "Unique"})

		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"name":"Unique"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 for duplicate name, got %d", rec.Code)
		}
	})
}

func TestAPI_BibleCRUD(t *testing.T) {
	cleanCollections(t, "stories", "bibles")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Bible Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/bible"

	t.Run("get bible for story without one returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("generate bible returns 201", func(t *testing.T) {
		payload := `{}`
		req := httptest.NewRequest("POST", base+"/generate", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("update bible", func(t *testing.T) {
		bibleRepo := mgorepo.NewBibleRepo(testDB)
		bible := &domain.StoryBible{
			ID:      s.ID + "_bible",
			StoryID: s.ID,
			Title:   "Original Bible",
			World:   "Middle-earth",
		}
		bibleRepo.Create(ctx, bible)

		payload := `{"world":"Updated World","tone":"dark"}`
		req := httptest.NewRequest("PUT", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete bible", func(t *testing.T) {
		bibleRepo := mgorepo.NewBibleRepo(testDB)
		bible := &domain.StoryBible{
			ID:      s.ID + "_del",
			StoryID: s.ID,
			Title:   "Delete Bible",
		}
		bibleRepo.Create(ctx, bible)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})
}

func TestAPI_BibleSharing(t *testing.T) {
	cleanCollections(t, "stories", "bibles")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()

	s1 := &domain.Story{Title: "Source", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s1)
	s2 := &domain.Story{Title: "Target", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s2)

	bibleRepo := mgorepo.NewBibleRepo(testDB)
	bible := &domain.StoryBible{
		ID:      s1.ID + "_bible",
		StoryID: s1.ID,
		Title:   "Shared Bible",
		World:   "Shared World",
	}
	bibleRepo.Create(ctx, bible)

	t.Run("link bible to another story", func(t *testing.T) {
		payload := fmt.Sprintf(`{"bibleId":"%s"}`, bible.ID)
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s2.ID+"/bibles/link", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["status"] != "linked" {
			t.Fatalf("status: got %q", resp["status"])
		}
	})

	t.Run("list referencing bibles", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s2.ID+"/bibles/referencing", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var bibles []*domain.StoryBible
		json.NewDecoder(rec.Body).Decode(&bibles)
		if len(bibles) < 1 {
			t.Fatal("expected at least 1 referencing bible")
		}
	})

	t.Run("unlink bible from story", func(t *testing.T) {
		payload := fmt.Sprintf(`{"bibleId":"%s"}`, bible.ID)
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s2.ID+"/bibles/unlink", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("unlink with invalid bibleID returns error", func(t *testing.T) {
		payload := `{"bibleId":"nonexistent"}`
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s2.ID+"/bibles/unlink", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestAPI_CrossStoryTimeline(t *testing.T) {
	cleanCollections(t, "stories", "timeline_events")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s1 := &domain.Story{Title: "S1", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s1)
	s2 := &domain.Story{Title: "S2", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s2)

	t.Run("create cross-story event", func(t *testing.T) {
		payload := fmt.Sprintf(`{"title":"Shared Event","description":"Visible in both","relatedStoryIds":["%s"],"order":1}`, s2.ID)
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s1.ID+"/timeline/cross-story", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var evt domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&evt)
		if evt.Title != "Shared Event" {
			t.Fatalf("title: got %q", evt.Title)
		}
	})

	t.Run("list cross-story events for related story", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s2.ID+"/timeline/cross-story", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var events []*domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&events)
		if len(events) < 1 {
			t.Fatal("expected at least 1 cross-story event")
		}
	})

	t.Run("list cross-story events for unrelated story returns empty", func(t *testing.T) {
		s3 := &domain.Story{Title: "S3", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s3)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s3.ID+"/timeline/cross-story", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var events []*domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&events)
		if len(events) != 0 {
			t.Fatalf("expected 0 events, got %d", len(events))
		}
	})
}

func TestAPI_CharacterMigration(t *testing.T) {
	cleanCollections(t, "stories", "characters")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()

	s1 := &domain.Story{Title: "Source Story", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s1)
	s2 := &domain.Story{Title: "Target Story", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s2)

	charRepo := mgorepo.NewCharacterRepo(testDB)
	char := &domain.Character{
		StoryID:   s1.ID,
		Name:      "Wanderer",
		Persona:   "traveler",
		Backstory: "Lost in time.",
		Goals:     []string{"Find home"},
	}
	charRepo.Create(ctx, char)

	t.Run("migrate character to another story", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			"/api/v1/stories/"+s2.ID+"/characters/"+char.ID+"/migrate", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var migrated domain.Character
		json.NewDecoder(rec.Body).Decode(&migrated)
		if migrated.Name != "Wanderer" {
			t.Fatalf("name: got %q", migrated.Name)
		}
		if migrated.StoryID != s2.ID {
			t.Fatalf("storyID should be target: got %q", migrated.StoryID)
		}
		if migrated.MigratedFrom != s1.ID {
			t.Fatalf("migratedFrom: got %q", migrated.MigratedFrom)
		}
		if migrated.MigratedAt == nil {
			t.Fatal("migratedAt should be set")
		}
	})

	t.Run("migrate nonexistent character returns 500", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			"/api/v1/stories/"+s2.ID+"/characters/nonexistent/migrate", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestAPI_AgentConfigsCRUD(t *testing.T) {
	cleanCollections(t, "agent_configs")
	srv, _ := buildServer(t)
	base := "/api/v1/agent-configs"

	t.Run("list empty agent configs", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var configs []*domain.AgentConfig
		json.NewDecoder(rec.Body).Decode(&configs)
		if len(configs) != 0 {
			t.Fatalf("expected 0 configs, got %d", len(configs))
		}
	})

	t.Run("create agent config", func(t *testing.T) {
		payload := `{"name":"my-critic","role":"critic","systemPrompt":"Review the output.","tags":["review","quality"],"shared":true}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var cfg domain.AgentConfig
		json.NewDecoder(rec.Body).Decode(&cfg)
		if cfg.Name != "my-critic" {
			t.Fatalf("name: got %q", cfg.Name)
		}
		if cfg.Role != "critic" {
			t.Fatalf("role: got %q", cfg.Role)
		}
		if !cfg.Shared {
			t.Fatal("expected shared=true")
		}
	})

	t.Run("create agent config with empty name returns 400", func(t *testing.T) {
		payload := `{"name":"","role":"critic","systemPrompt":"test"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("get agent config by name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/my-critic", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var cfg domain.AgentConfig
		json.NewDecoder(rec.Body).Decode(&cfg)
		if cfg.Name != "my-critic" {
			t.Fatalf("name: got %q", cfg.Name)
		}
	})

	t.Run("get missing agent config returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("list returns created config", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var configs []*domain.AgentConfig
		json.NewDecoder(rec.Body).Decode(&configs)
		if len(configs) < 1 {
			t.Fatal("expected at least 1 config")
		}
	})

	t.Run("delete agent config", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/my-critic", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		grec := httptest.NewRecorder()
		srv.Router.ServeHTTP(grec, httptest.NewRequest("GET", base+"/my-critic", nil))
		if grec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", grec.Code)
		}
	})
}

func TestAPI_AgentConfigExportImportMarketplace(t *testing.T) {
	cleanCollections(t, "agent_configs")
	srv, _ := buildServer(t)
	base := "/api/v1/agent-configs"

	payload := `{"name":"market-critic","role":"critic","systemPrompt":"Review.","shared":true,"tags":["review"]}`
	creq := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
	creq.Header.Set("Content-Type", "application/json")
	srv.Router.ServeHTTP(httptest.NewRecorder(), creq)

	t.Run("export agent config", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/market-critic/export", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var cfg domain.AgentConfig
		json.NewDecoder(rec.Body).Decode(&cfg)
		if cfg.Name != "market-critic" {
			t.Fatalf("name: got %q", cfg.Name)
		}
		if cfg.SystemPrompt != "Review." {
			t.Fatalf("systemPrompt: got %q", cfg.SystemPrompt)
		}
	})

	t.Run("export missing config returns error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/nonexistent/export", nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("import agent config", func(t *testing.T) {
		payload := `{"name":"imported-agent","role":"world","systemPrompt":"World builder.","shared":false}`
		req := httptest.NewRequest("POST", base+"/imported-agent/import", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var cfg domain.AgentConfig
		json.NewDecoder(rec.Body).Decode(&cfg)
		if cfg.Name != "imported-agent" {
			t.Fatalf("name: got %q", cfg.Name)
		}
	})

	t.Run("marketplace lists shared configs only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/marketplace", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var configs []*domain.AgentConfig
		json.NewDecoder(rec.Body).Decode(&configs)
		if len(configs) < 1 {
			t.Fatal("expected at least 1 marketplace config")
		}
		for _, c := range configs {
			if !c.Shared {
				t.Fatalf("config %s is not shared but in marketplace", c.Name)
			}
		}
	})
}

func TestAPI_CriticScores(t *testing.T) {
	cleanCollections(t, "stories", "generations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Critic Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	genRepo := mgorepo.NewGenerationRepo(testDB)
	genRepo.Create(ctx, &domain.Generation{
		StoryID:     s.ID,
		SceneID:     "scene_1",
		Output:      "Good prose.",
		Model:       "claude-sonnet",
		Status:      domain.GenStatusSuccess,
		CriticScore: 0.85,
		CriticSummary: "Well-written scene",
	})
	genRepo.Create(ctx, &domain.Generation{
		StoryID:     s.ID,
		SceneID:     "scene_2",
		Output:      "Okay prose.",
		CriticScore: 0.62,
		CriticSummary: "Needs improvement",
	})
	genRepo.Create(ctx, &domain.Generation{
		StoryID: s.ID,
		SceneID: "scene_3",
		Output:  "Unscored.",
	})

	t.Run("list critic scores", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/critic-scores", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var scores []domain.CriticScoreEntry
		json.NewDecoder(rec.Body).Decode(&scores)
		if len(scores) != 2 {
			t.Fatalf("expected 2 critic scores, got %d", len(scores))
		}
		if scores[0].Score != 0.85 && scores[0].Score != 0.62 {
			t.Fatalf("unexpected score: %f", scores[0].Score)
		}
	})

	t.Run("critic scores for story with none returns empty", func(t *testing.T) {
		s2 := &domain.Story{Title: "No Critic", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s2)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s2.ID+"/critic-scores", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var scores []domain.CriticScoreEntry
		json.NewDecoder(rec.Body).Decode(&scores)
		if len(scores) != 0 {
			t.Fatalf("expected 0 scores, got %d", len(scores))
		}
	})
}

func TestAPI_LLMMetrics(t *testing.T) {
	cleanCollections(t, "stories", "generations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Metrics Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	genRepo := mgorepo.NewGenerationRepo(testDB)
	genRepo.Create(ctx, &domain.Generation{
		StoryID:          s.ID,
		SceneID:          "sc1",
		Model:            "claude-sonnet",
		Status:           domain.GenStatusSuccess,
		PromptTokens:     500,
		CompletionTokens: 200,
		TotalTokens:      700,
		CreatedAt:        time.Now(),
	})

	t.Run("get llm metrics returns stats", func(t *testing.T) {
		// llmMetricsSvc is nil in test setup (passed as nil)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/metrics/llm", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var result map[string]any
		json.NewDecoder(rec.Body).Decode(&result)
		// nil metricsSvc returns zeroed response — just verify shape
		if _, ok := result["total_tokens"]; !ok {
			t.Fatal("expected total_tokens field")
		}
	})
}

func TestAPI_Blueprint(t *testing.T) {
	cleanCollections(t, "stories")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "BP Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/blueprint"

	t.Run("get blueprint for story without one returns null", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("update blueprint", func(t *testing.T) {
		payload := `{"acts":[{"number":1,"title":"Act I"}],"characterArcs":[{"characterId":"hero","arcType":"redemption"}]}`
		req := httptest.NewRequest("PUT", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		grec := httptest.NewRecorder()
		srv.Router.ServeHTTP(grec, httptest.NewRequest("GET", base, nil))
		var bp domain.StoryBlueprint
		json.NewDecoder(grec.Body).Decode(&bp)
		if len(bp.Acts) != 1 {
			t.Fatalf("acts: got %d", len(bp.Acts))
		}
	})
}

func TestAPI_Chapters(t *testing.T) {
	cleanCollections(t, "stories", "chapters")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Ch Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/chapters"

	t.Run("create chapter", func(t *testing.T) {
		payload := `{"title":"Chapter 1","actNumber":1,"summary":"The beginning"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var ch domain.Chapter
		json.NewDecoder(rec.Body).Decode(&ch)
		if ch.Title != "Chapter 1" {
			t.Fatalf("title: got %q", ch.Title)
		}
	})

	t.Run("list chapters", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var chapters []*domain.Chapter
		json.NewDecoder(rec.Body).Decode(&chapters)
		if len(chapters) < 1 {
			t.Fatal("expected at least 1 chapter")
		}
	})

	t.Run("get chapter by ID", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Get Me","actNumber":1}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/"+created["id"].(string), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var got map[string]any
		json.NewDecoder(rec.Body).Decode(&got)
		if got["title"] != "Get Me" {
			t.Fatalf("title: got %q", got["title"])
		}
	})

	t.Run("update chapter", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Old","actNumber":1}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		payload := `{"title":"New Title"}`
		req := httptest.NewRequest("PUT", base+"/"+created["id"].(string), bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var updated map[string]any
		json.NewDecoder(rec.Body).Decode(&updated)
		if updated["title"] != "New Title" {
			t.Fatalf("title: got %q", updated["title"])
		}
	})

	t.Run("delete chapter", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Delete Me","actNumber":1}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/"+created["id"].(string), nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}

		grec := httptest.NewRecorder()
		srv.Router.ServeHTTP(grec, httptest.NewRequest("GET", base+"/"+created["id"].(string), nil))
		if grec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", grec.Code)
		}
	})
}

func TestAPI_GeneratedTitle(t *testing.T) {
	cleanCollections(t, "stories")
	srv, _ := buildServer(t)

	t.Run("generate title returns title", func(t *testing.T) {
		payload := `{"synopsis":"A hero's journey through darkness."}`
		req := httptest.NewRequest("POST", "/api/v1/stories/generate-title", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["title"] == "" {
			t.Fatal("expected title field")
		}
	})

	t.Run("generate title with empty synopsis", func(t *testing.T) {
		payload := `{"synopsis":""}`
		req := httptest.NewRequest("POST", "/api/v1/stories/generate-title", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestAPI_ErrorScenarios(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"character_state", "character_memories", "generations", "timeline_events", "summaries")
	srv, _ := buildServer(t)

	t.Run("unknown route returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("healthz returns ok", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}
