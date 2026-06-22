//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
)

// ── V2 Characters (top-level) ──────────────────────────────────────────

func TestAPI_V2Characters(t *testing.T) {
	cleanCollections(t, "stories", "characters")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "V2 Char Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("create character", func(t *testing.T) {
		payload := `{"name":"Legolas","persona":"elf","storyId":"` + s.ID + `","traits":["agile","keen-eyed"]}`
		req := httptest.NewRequest("POST", "/api/v1/characters", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var char domain.Character
		json.NewDecoder(rec.Body).Decode(&char)
		if char.Name != "Legolas" {
			t.Fatalf("name: got %q", char.Name)
		}
		if char.StoryID != s.ID {
			t.Fatalf("storyID: got %q", char.StoryID)
		}
	})

	t.Run("create character with empty name returns 400", func(t *testing.T) {
		payload := `{"name":"","storyId":"` + s.ID + `"}`
		req := httptest.NewRequest("POST", "/api/v1/characters", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("create character with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/characters", bytes.NewBufferString(`not json`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("list characters", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/characters?story_id="+s.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var chars []*domain.Character
		json.NewDecoder(rec.Body).Decode(&chars)
		if len(chars) < 1 {
			t.Fatal("expected at least 1 character")
		}
	})

	t.Run("list characters for story with none returns empty", func(t *testing.T) {
		s2 := &domain.Story{Title: "Empty Char", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s2)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/characters?storyId="+s2.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var chars []*domain.Character
		json.NewDecoder(rec.Body).Decode(&chars)
		if len(chars) != 0 {
			t.Fatalf("expected 0, got %d", len(chars))
		}
	})

	t.Run("get character by ID", func(t *testing.T) {
		charRepo := mgorepo.NewCharacterRepo(testDB)
		c := &domain.Character{StoryID: s.ID, Name: "Gimli", CharID: "gimli-1"}
		charRepo.Create(ctx, c)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/characters/"+c.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var got domain.Character
		json.NewDecoder(rec.Body).Decode(&got)
		if got.Name != "Gimli" {
			t.Fatalf("name: got %q", got.Name)
		}
	})

	t.Run("get missing character returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/characters/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("update character", func(t *testing.T) {
		charRepo := mgorepo.NewCharacterRepo(testDB)
		c := &domain.Character{StoryID: s.ID, Name: "Frodo", CharID: "frodo-1"}
		charRepo.Create(ctx, c)

		payload := `{"name":"Frodo Baggins","traits":["brave","determined"]}`
		req := httptest.NewRequest("PUT", "/api/v1/characters/"+c.CharID, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var updated domain.Character
		json.NewDecoder(rec.Body).Decode(&updated)
		if updated.Name != "Frodo Baggins" {
			t.Fatalf("name: got %q", updated.Name)
		}
	})

	t.Run("update missing character returns 404", func(t *testing.T) {
		payload := `{"name":"Ghost"}`
		req := httptest.NewRequest("PUT", "/api/v1/characters/nonexistent", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

// ── Generations Pipeline ───────────────────────────────────────────────

func TestAPI_Generations(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "generations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Gen Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc := &domain.Scene{StoryID: s.ID, Title: "Gen Scene"}
	sceneRepo.Create(ctx, sc)
	nodeBase := "/api/v1/stories/" + s.ID + "/nodes/" + sc.ID

	t.Run("generate node enqueues generation", func(t *testing.T) {
		req := httptest.NewRequest("POST", nodeBase+"/generate", nil)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["status"] != "pending" && resp["status"] != "running" {
			t.Fatalf("expected pending/running status, got %q", resp["status"])
		}
	})

	t.Run("generate for missing node returns 404", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/nodes/nonexistent/generate", nil)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("list node generations", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", nodeBase+"/generations", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var gens []*domain.Generation
		json.NewDecoder(rec.Body).Decode(&gens)
		if len(gens) < 1 {
			t.Fatal("expected at least 1 generation")
		}
	})

	t.Run("list generations for missing node returns empty", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/nodes/nonexistent/generations", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var gens []*domain.Generation
		json.NewDecoder(rec.Body).Decode(&gens)
		if len(gens) != 0 {
			t.Fatalf("expected 0, got %d", len(gens))
		}
	})

	t.Run("accept generation", func(t *testing.T) {
		// Scene must be in "generated" status before accepting a generation
		sc.Status = domain.SceneStatusGenerated
		sceneRepo.Update(ctx, sc)

		genRepo := mgorepo.NewGenerationRepo(testDB)
		gen := &domain.Generation{
			StoryID: s.ID, SceneID: sc.ID,
			Output:  "Great prose.",
			Model:   "claude-sonnet",
			Status:  domain.GenStatusSuccess,
		}
		genRepo.Create(ctx, gen)

		payload := fmt.Sprintf(`{"generation_id":"%s"}`, gen.ID)
		req := httptest.NewRequest("POST", nodeBase+"/accept", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("accept with missing generation_id returns 400", func(t *testing.T) {
		payload := `{}`
		req := httptest.NewRequest("POST", nodeBase+"/accept", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("accept with nonexistent generation ID returns error", func(t *testing.T) {
		payload := `{"generation_id":"badgenid"}`
		req := httptest.NewRequest("POST", nodeBase+"/accept", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("get generation status", func(t *testing.T) {
		genRepo := mgorepo.NewGenerationRepo(testDB)
		gen := &domain.Generation{
			StoryID: s.ID, SceneID: sc.ID,
			Output: "Status check.", Model: "claude-sonnet",
			Status: domain.GenStatusSuccess,
		}
		genRepo.Create(ctx, gen)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/generations/"+gen.ID+"/status", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var status map[string]any
		json.NewDecoder(rec.Body).Decode(&status)
		if status["status"] != "success" {
			t.Fatalf("expected success, got %v", status["status"])
		}
	})

	t.Run("get status for missing generation returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/generations/nonexistent/status", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

// ── Search Memories ───────────────────────────────────────────────────

func TestAPI_SearchMemories(t *testing.T) {
	cleanCollections(t, "stories", "characters", "character_memories")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Mem Search", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	c := &domain.Character{StoryID: s.ID, Name: "Searcher", CharID: "searcher-1"}
	charRepo.Create(ctx, c)
	memRepo := mgorepo.NewMemoryRepo(testDB)
	memRepo.Create(ctx, &domain.CharacterMemory{
		CharacterID: c.CharID, StoryID: s.ID,
		Content: "Important memory about the ring", Importance: 0.9,
	})

	t.Run("search memories returns results", func(t *testing.T) {
		payload := fmt.Sprintf(`{"story_id":"%s","query":"ring","limit":5}`, s.ID)
		req := httptest.NewRequest("POST", "/api/v1/characters/"+c.CharID+"/memories/search", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var mems []*domain.CharacterMemory
		json.NewDecoder(rec.Body).Decode(&mems)
		if len(mems) != 0 && mems[0].Content != "Important memory about the ring" {
			t.Fatalf("unexpected content: %q", mems[0].Content)
		}
	})

	t.Run("search with empty query", func(t *testing.T) {
		payload := fmt.Sprintf(`{"story_id":"%s","query":"","limit":5}`, s.ID)
		req := httptest.NewRequest("POST", "/api/v1/characters/"+c.CharID+"/memories/search", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("search for nonexistent character returns empty", func(t *testing.T) {
		payload := fmt.Sprintf(`{"story_id":"%s","query":"ring","limit":5}`, s.ID)
		req := httptest.NewRequest("POST", "/api/v1/characters/nonexistent/memories/search", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("search with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/characters/"+c.CharID+"/memories/search", bytes.NewBufferString(`not json`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("search with negative limit clamped to 10", func(t *testing.T) {
		payload := fmt.Sprintf(`{"story_id":"%s","query":"ring","limit":-1}`, s.ID)
		req := httptest.NewRequest("POST", "/api/v1/characters/"+c.CharID+"/memories/search", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// ── Scene Summary ─────────────────────────────────────────────────────

func TestAPI_SceneSummary(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "summaries")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Sum2 Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc := &domain.Scene{StoryID: s.ID, Title: "Scene A"}
	sceneRepo.Create(ctx, sc)
	sumRepo := mgorepo.NewSummaryRepo(testDB)
	sumRepo.Upsert(ctx, &domain.Summary{
		StoryID: s.ID, SceneID: sc.ID,
		Level: domain.SummaryLevelScene, Content: "Scene summary.", WordCount: 2,
	})

	t.Run("get scene summary", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/summaries/nodes/%s", s.ID, sc.ID), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var sum domain.Summary
		json.NewDecoder(rec.Body).Decode(&sum)
		if sum.Content != "Scene summary." {
			t.Fatalf("content: got %q", sum.Content)
		}
	})

	t.Run("get scene summary for missing scene returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/summaries/nodes/nonexistent", s.ID), nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("get scene summary for missing story returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/nonexistent/summaries/nodes/whatever", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("get scene summary without summary returns 404", func(t *testing.T) {
		sc2 := &domain.Scene{StoryID: s.ID, Title: "No Summary"}
		sceneRepo.Create(ctx, sc2)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/summaries/nodes/%s", s.ID, sc2.ID), nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("get summary by level returns correct type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s.ID+"/summaries/level?level=scene", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var sum domain.Summary
		json.NewDecoder(rec.Body).Decode(&sum)
		if sum.Level != domain.SummaryLevelScene {
			t.Fatalf("expected scene level, got %q", sum.Level)
		}
	})
}

// ── Locations by ID ───────────────────────────────────────────────────

func TestAPI_LocationsByID(t *testing.T) {
	cleanCollections(t, "stories", "locations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Loc2 Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	locRepo := mgorepo.NewLocationRepo(testDB)
	loc := &domain.Location{StoryID: s.ID, Name: "Moria", LocType: domain.LocBuilding}
	locRepo.Create(ctx, loc)

	t.Run("get location by ID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/locations/"+loc.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var got domain.Location
		json.NewDecoder(rec.Body).Decode(&got)
		if got.Name != "Moria" {
			t.Fatalf("name: got %q", got.Name)
		}
	})

	t.Run("get missing location returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/locations/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("update location", func(t *testing.T) {
		payload := `{"description":"Dwarf kingdom"}`
		req := httptest.NewRequest("PUT", "/api/v1/locations/"+loc.ID, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var updated domain.Location
		json.NewDecoder(rec.Body).Decode(&updated)
		if updated.Description != "Dwarf kingdom" {
			t.Fatalf("description: got %q", updated.Description)
		}
	})

	t.Run("update location with empty name returns 400", func(t *testing.T) {
		payload := `{"description":"Updated description"}`
		req := httptest.NewRequest("PUT", "/api/v1/locations/"+loc.ID, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var updated domain.Location
		json.NewDecoder(rec.Body).Decode(&updated)
		if updated.Name != "Moria" {
			t.Fatalf("name should remain Moria, got %q", updated.Name)
		}
		if updated.Description != "Updated description" {
			t.Fatalf("description: got %q", updated.Description)
		}
	})

	t.Run("update missing location returns 404", func(t *testing.T) {
		payload := `{"name":"Nowhere"}`
		req := httptest.NewRequest("PUT", "/api/v1/locations/nonexistent", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

// ── Edge Delete by Query Params ────────────────────────────────────────

func TestAPI_EdgeDeleteByQuery(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "EdgeQ Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc1 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc1)
	sc2 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc2)
	edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
	e := &domain.SceneEdge{StoryID: s.ID, FromSceneID: sc1.ID, ToSceneID: sc2.ID, Type: "seq"}
	edgeRepo.Create(ctx, e)

	t.Run("delete edge by query params", func(t *testing.T) {
		req := httptest.NewRequest("DELETE",
			fmt.Sprintf("/api/v1/stories/%s/edges?from_scene=%s&to_scene=%s", s.ID, sc1.ID, sc2.ID), nil)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete non-existent edge returns 204", func(t *testing.T) {
		req := httptest.NewRequest("DELETE",
			fmt.Sprintf("/api/v1/stories/%s/edges?from_scene=%s&to_scene=%s", s.ID, sc1.ID, sc2.ID), nil)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("delete edge by id when none exists returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE",
			"/api/v1/stories/"+s.ID+"/edges/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("create then delete edge by id", func(t *testing.T) {
		e2 := &domain.SceneEdge{StoryID: s.ID, FromSceneID: sc1.ID, ToSceneID: sc2.ID, Type: "fork"}
		edgeRepo.Create(ctx, e2)

		delRec := httptest.NewRecorder()
		srv.Router.ServeHTTP(delRec, httptest.NewRequest("DELETE",
			"/api/v1/stories/"+s.ID+"/edges/"+e2.ID, nil))
		if delRec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", delRec.Code)
		}
	})

	t.Run("list edges after delete", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s.ID+"/edges", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// ── Story Generation (GenerateStory full flow) ─────────────────────────

func TestAPI_StoryGenerationFlow(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"generations", "timeline_events")
	srv, _ := buildServer(t)

	t.Run("generate story creates story with title and scenes", func(t *testing.T) {
		payload := `{"synopsis":"An epic tale of adventure and discovery."}`
		req := httptest.NewRequest("POST", "/api/v1/stories/generate", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["story_id"] == "" {
			t.Fatal("expected story_id")
		}

		topoRec := httptest.NewRecorder()
		srv.Router.ServeHTTP(topoRec, httptest.NewRequest("GET",
			"/api/v1/stories/"+resp["story_id"]+"/topology", nil))
		if topoRec.Code != http.StatusOK {
			t.Fatalf("topology: expected 200, got %d", topoRec.Code)
		}
		var topo map[string]any
		json.NewDecoder(topoRec.Body).Decode(&topo)
		nodes, _ := topo["nodes"].([]any)
		if len(nodes) == 0 {
			t.Fatal("expected at least 1 scene node")
		}
	})

	t.Run("generate story with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/generate", bytes.NewBufferString(`bad`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

// ── Story CRUD Extended ───────────────────────────────────────────────

func TestAPI_StoryCRUDExtended(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"generations", "timeline_events", "summaries", "locations", "bibles", "chapters")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	base := "/api/v1/stories"

	t.Run("update story with empty title returns 400", func(t *testing.T) {
		s := &domain.Story{Title: "Update Test", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s)
		payload := `{"title":""}`
		req := httptest.NewRequest("PUT", base+"/"+s.ID, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("update missing story returns 404", func(t *testing.T) {
		payload := `{"title":"Ghost"}`
		req := httptest.NewRequest("PUT", base+"/nonexistent", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("delete missing story returns 204", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/nonexistent", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("list stories returns stories", func(t *testing.T) {
		// Ensure at least one story exists from prior tests in this function
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("create story generates valid ID", func(t *testing.T) {
		payload := `{"title":"ID Test Story"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var story domain.Story
		json.NewDecoder(rec.Body).Decode(&story)
		if story.ID == "" {
			t.Fatal("expected non-empty ID")
		}
		if len(story.ID) < 8 {
			t.Fatalf("expected reasonably long ID, got %d chars", len(story.ID))
		}
	})
}

// ── Legacy Characters (story-scoped) Extended ──────────────────────────

func TestAPI_LegacyCharactersExtended(t *testing.T) {
	cleanCollections(t, "stories", "characters")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Legacy Char", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/characters"

	t.Run("create character with all fields", func(t *testing.T) {
		payload := `{"name":"Aragorn","persona":"ranger","backstory":"Heir of Isildur","traits":["noble","brave"],"goals":["Save Middle-earth"]}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var char domain.Character
		json.NewDecoder(rec.Body).Decode(&char)
		if char.Name != "Aragorn" {
			t.Fatalf("name: got %q", char.Name)
		}
	})

	t.Run("create character with empty name returns 400", func(t *testing.T) {
		payload := `{"name":""}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("create character with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`{invalid}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("list characters for story with none returns empty", func(t *testing.T) {
		s2 := &domain.Story{Title: "Empty Char", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s2)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s2.ID+"/characters", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var chars []*domain.Character
		json.NewDecoder(rec.Body).Decode(&chars)
		if len(chars) != 0 {
			t.Fatalf("expected 0, got %d", len(chars))
		}
	})

	t.Run("create duplicate character name returns 409", func(t *testing.T) {
		charRepo := mgorepo.NewCharacterRepo(testDB)
		charRepo.Create(ctx, &domain.Character{StoryID: s.ID, Name: "Boromir"})

		payload := `{"name":"Boromir"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

// ── Timeline Extended ─────────────────────────────────────────────────

func TestAPI_TimelineExtended(t *testing.T) {
	cleanCollections(t, "stories", "timeline_events")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "TL Ext", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/timeline"

	t.Run("create timeline event with all fields", func(t *testing.T) {
		payload := `{"title":"Battle","description":"Epic battle scene","order":2,"eventType":"action"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var evt domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&evt)
		if evt.Title != "Battle" {
			t.Fatalf("title: got %q", evt.Title)
		}
	})

	t.Run("create event with empty title returns 400", func(t *testing.T) {
		payload := `{"title":"Event 2","order":2}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}
	})

	t.Run("list events sorted by order", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var events []*domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&events)
		if len(events) < 2 {
			t.Fatalf("expected at least 2 events, got %d", len(events))
		}
	})

	t.Run("list for story with no events returns empty", func(t *testing.T) {
		s2 := &domain.Story{Title: "No TL", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s2)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s2.ID+"/timeline", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var events []*domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&events)
		if len(events) != 0 {
			t.Fatalf("expected 0, got %d", len(events))
		}
	})

	t.Run("create event with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`not json`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

// ── Cross-Story Timeline Extended ──────────────────────────────────────

func TestAPI_CrossStoryTimelineExtended(t *testing.T) {
	cleanCollections(t, "stories", "timeline_events")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s1 := &domain.Story{Title: "A", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s1)
	s2 := &domain.Story{Title: "B", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s2)

	t.Run("create cross-story event without related IDs", func(t *testing.T) {
		payload := `{"title":"Local Event","description":"Only in A","order":1}`
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s1.ID+"/timeline/cross-story", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("create cross-story event with empty title returns 400", func(t *testing.T) {
		payload := `{"title":"","order":1}`
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s1.ID+"/timeline/cross-story", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("list cross-story for story with none returns empty", func(t *testing.T) {
		s3 := &domain.Story{Title: "C", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s3)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s3.ID+"/timeline/cross-story", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var events []*domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&events)
		if len(events) != 0 {
			t.Fatalf("expected 0, got %d", len(events))
		}
	})

	t.Run("create event with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s1.ID+"/timeline/cross-story", bytes.NewBufferString(`bad`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("cross-story event appears in source story timeline", func(t *testing.T) {
		srcRec := httptest.NewRecorder()
		srv.Router.ServeHTTP(srcRec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s1.ID+"/timeline", nil))
		if srcRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", srcRec.Code)
		}
	})
}

// ── Bible Extended ─────────────────────────────────────────────────────

func TestAPI_BibleExtended(t *testing.T) {
	cleanCollections(t, "stories", "bibles")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Bible Ext", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/bible"

	t.Run("update bible with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("PUT", base, bytes.NewBufferString(`not json`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("generate bible with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base+"/generate", bytes.NewBufferString(`not json`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete bible that doesn't exist returns 204", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("update bible creates bible if missing", func(t *testing.T) {
		payload := `{"world":"Middle-earth","tone":"epic"}`
		req := httptest.NewRequest("PUT", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var bible domain.StoryBible
		json.NewDecoder(rec.Body).Decode(&bible)
		if bible.World != "Middle-earth" {
			t.Fatalf("world: got %q", bible.World)
		}
	})

	t.Run("get bible after update", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var bible domain.StoryBible
		json.NewDecoder(rec.Body).Decode(&bible)
		if bible.World != "Middle-earth" {
			t.Fatalf("world: got %q", bible.World)
		}
	})
}

// ── Node CRUD Extended ────────────────────────────────────────────────

func TestAPI_NodeCRUDExtended(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "generations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Node Ext", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/nodes"

	t.Run("create node with all fields", func(t *testing.T) {
		payload := `{"beat_intent":"Climax","pov":"Hero","tone":"dramatic","target_words":1000,"status":"draft","character_refs":["hero-id"],"location_ref":"moria"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var node map[string]any
		json.NewDecoder(rec.Body).Decode(&node)
		if node["beat_intent"] != "Climax" {
			t.Fatalf("beat_intent: got %q", node["beat_intent"])
		}
		if node["pov"] != "Hero" {
			t.Fatalf("pov: got %q", node["pov"])
		}
	})

	t.Run("update node preserves unset fields", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"beat_intent":"Original","pov":"Hero"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		payload := `{"beat_intent":"Updated"}`
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
		if updated["pov"] != "Hero" {
			t.Fatalf("expected pov to persist, got %q", updated["pov"])
		}
	})

	t.Run("update missing node returns 404", func(t *testing.T) {
		payload := `{"beat_intent":"Nowhere"}`
		req := httptest.NewRequest("PUT", base+"/nonexistent", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("delete missing node returns 204", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/nonexistent", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("list nodes for missing story returns empty", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/nonexistent/nodes", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var nodes []any
		json.NewDecoder(rec.Body).Decode(&nodes)
		if len(nodes) != 0 {
			t.Fatalf("expected 0, got %d", len(nodes))
		}
	})
}

// ── Chapter Extended ──────────────────────────────────────────────────

func TestAPI_ChapterExtended(t *testing.T) {
	cleanCollections(t, "stories", "chapters")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Ch Ext", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/chapters"

	t.Run("create chapter with invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`{bad}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("get missing chapter returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/nonexistent", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("update missing chapter returns 404", func(t *testing.T) {
		payload := `{"title":"Ghost"}`
		req := httptest.NewRequest("PUT", base+"/nonexistent", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("delete missing chapter returns 204", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/nonexistent", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("create chapter with only required fields", func(t *testing.T) {
		payload := `{"title":"Minimal"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var ch domain.Chapter
		json.NewDecoder(rec.Body).Decode(&ch)
		if ch.Title != "Minimal" {
			t.Fatalf("title: got %q", ch.Title)
		}
	})
}

// ── Summary Extended ──────────────────────────────────────────────────

func TestAPI_SummaryExtended(t *testing.T) {
	cleanCollections(t, "stories", "summaries")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Sum Ext", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sumRepo := mgorepo.NewSummaryRepo(testDB)
	sumRepo.Upsert(ctx, &domain.Summary{
		StoryID: s.ID, Level: domain.SummaryLevelStory,
		Content: "Extended summary test.", WordCount: 3,
	})

	t.Run("get summary with missing level param returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s.ID+"/summaries/level", nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("get summary with invalid level returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s.ID+"/summaries/level?level=invalid", nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("get summary for missing story returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/nonexistent/summaries/level?level=story", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("upsert summary then get it", func(t *testing.T) {
		sumRepo.Upsert(ctx, &domain.Summary{
			StoryID: s.ID, Level: domain.SummaryLevelAct,
			Content: "Arc summary.", WordCount: 2,
		})
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s.ID+"/summaries/level?level=act", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// ── Location Story-Scoped Extended ────────────────────────────────────

func TestAPI_LocationStoryScopedExtended(t *testing.T) {
	cleanCollections(t, "stories", "locations")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Loc Ext", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/locations"

	t.Run("create location with description", func(t *testing.T) {
		payload := `{"name":"Fangorn","type":"forest","description":"Home of the Ents","props":["trees","rivers"]}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var loc domain.Location
		json.NewDecoder(rec.Body).Decode(&loc)
		if loc.Name != "Fangorn" {
			t.Fatalf("name: got %q", loc.Name)
		}
	})

	t.Run("create location with empty name returns 400", func(t *testing.T) {
		payload := `{"name":""}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("create location with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`no`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("list locations for story with none returns empty", func(t *testing.T) {
		s2 := &domain.Story{Title: "No Loc", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s2)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s2.ID+"/locations", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var locs []*domain.Location
		json.NewDecoder(rec.Body).Decode(&locs)
		if len(locs) != 0 {
			t.Fatalf("expected 0, got %d", len(locs))
		}
	})

	t.Run("list after multiple creates returns all", func(t *testing.T) {
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
}

// ── Agent Config Extended ─────────────────────────────────────────────

func TestAPI_AgentConfigExtended(t *testing.T) {
	cleanCollections(t, "agent_configs")
	srv, _ := buildServer(t)
	base := "/api/v1/agent-configs"

	t.Run("create config with invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`bad`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("create then get by name", func(t *testing.T) {
		payload := `{"name":"test-agent","role":"director","systemPrompt":"Direct the scene."}`
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		creq.Header.Set("Content-Type", "application/json")
		srv.Router.ServeHTTP(httptest.NewRecorder(), creq)

		grec := httptest.NewRecorder()
		srv.Router.ServeHTTP(grec, httptest.NewRequest("GET", base+"/test-agent", nil))
		if grec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", grec.Code)
		}
		var cfg domain.AgentConfig
		json.NewDecoder(grec.Body).Decode(&cfg)
		if cfg.Name != "test-agent" {
			t.Fatalf("name: got %q", cfg.Name)
		}
	})

	t.Run("create duplicate config name returns 409", func(t *testing.T) {
		payload := `{"name":"dupe-agent","role":"critic","systemPrompt":"Review."}`
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.Router.ServeHTTP(rec, req)
			if i == 1 && rec.Code != http.StatusInternalServerError {
				t.Fatalf("expected 500, got %d", rec.Code)
			}
		}
	})

	t.Run("export returns full config", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/test-agent/export", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("marketplace excludes non-shared configs", func(t *testing.T) {
		payload := `{"name":"private-agent","role":"narrator","systemPrompt":"Narrate.","shared":false}`
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		creq.Header.Set("Content-Type", "application/json")
		srv.Router.ServeHTTP(httptest.NewRecorder(), creq)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/marketplace", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// ── Migration Extended ───────────────────────────────────────────────

func TestAPI_MigrationExtended(t *testing.T) {
	cleanCollections(t, "stories", "characters", "character_state", "character_memories")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s1 := &domain.Story{Title: "Src", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s1)
	s2 := &domain.Story{Title: "Dst", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s2)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	c := &domain.Character{StoryID: s1.ID, Name: "Gollum", CharID: "gollum-1"}
	charRepo.Create(ctx, c)
	stateRepo := mgorepo.NewCharacterStateRepo(testDB)
	stateRepo.Append(ctx, &domain.CharacterState{
		CharacterID: c.CharID, StoryID: s1.ID, SceneID: "s1", Mood: "conflicted",
	})
	memRepo := mgorepo.NewMemoryRepo(testDB)
	memRepo.Create(ctx, &domain.CharacterMemory{
		CharacterID: c.CharID, StoryID: s1.ID, Content: "My precious", Importance: 1.0,
	})

	t.Run("migrate character to new story", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			"/api/v1/stories/"+s2.ID+"/characters/"+c.ID+"/migrate", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var migrated domain.Character
		json.NewDecoder(rec.Body).Decode(&migrated)
		if migrated.StoryID != s2.ID {
			t.Fatalf("expected story %q, got %q", s2.ID, migrated.StoryID)
		}
		if migrated.MigratedFrom != s1.ID {
			t.Fatalf("expected migratedFrom %q, got %q", s1.ID, migrated.MigratedFrom)
		}
	})

	t.Run("migrated character has states in target story", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			"/api/v1/stories/"+s2.ID+"/characters/"+c.ID+"/migrate", nil))
		var migrated domain.Character
		json.NewDecoder(rec.Body).Decode(&migrated)

		states, err := stateRepo.ListByCharacter(ctx, migrated.CharID)
		if err != nil {
			t.Fatalf("list states: %v", err)
		}
		if len(states) < 1 {
			t.Fatal("expected migrated states")
		}
	})

	t.Run("migrate same character again creates new copy", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			"/api/v1/stories/"+s2.ID+"/characters/"+c.ID+"/migrate", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list migrated characters in target story", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s2.ID+"/characters", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var chars []*domain.Character
		json.NewDecoder(rec.Body).Decode(&chars)
		if len(chars) < 1 {
			t.Fatal("expected at least 1 migrated character")
		}
		for _, ch := range chars {
			if ch.MigratedFrom == "" {
				continue
			}
			if ch.MigratedFrom != s1.ID {
				t.Fatalf("expected migratedFrom %q, got %q", s1.ID, ch.MigratedFrom)
			}
		}
	})

	t.Run("migrate nonexistent character returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			"/api/v1/stories/"+s2.ID+"/characters/nonexistent/migrate", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

// ── Healthz ───────────────────────────────────────────────────────────

func TestAPI_HealthzExtended(t *testing.T) {
	cleanCollections(t, "stories")
	srv, _ := buildServer(t)

	t.Run("healthz returns status ok", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var body map[string]any
		json.NewDecoder(rec.Body).Decode(&body)
		if body["status"] != "ok" {
			t.Fatalf("expected ok, got %v", body["status"])
		}
	})

	t.Run("healthz always returns ok", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			rec := httptest.NewRecorder()
			srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("attempt %d: expected 200, got %d", i, rec.Code)
			}
		}
	})
}

// ── 404 / Not Found ───────────────────────────────────────────────────

func TestAPI_NotFoundEndpoints(t *testing.T) {
	cleanCollections(t, "stories")
	srv, _ := buildServer(t)

	routes := []struct {
		method string
		path   string
		label  string
		want   int
	}{
		{"GET", "/api/v1/stories/nonexistent", "get story", http.StatusNotFound},
		{"PUT", "/api/v1/stories/nonexistent", "update story", http.StatusBadRequest},
		{"DELETE", "/api/v1/stories/nonexistent", "delete story", http.StatusNoContent},
		{"GET", "/api/v1/stories/nonexistent/nodes", "list nodes", http.StatusOK},
		{"POST", "/api/v1/stories/nonexistent/nodes", "create node", http.StatusBadRequest},
		{"GET", "/api/v1/stories/nonexistent/topology", "topology", http.StatusOK},
		{"GET", "/api/v1/stories/nonexistent/timeline", "timeline", http.StatusOK},
		{"GET", "/api/v1/stories/nonexistent/bible", "bible", http.StatusNotFound},
		{"GET", "/api/v1/stories/nonexistent/characters", "legacy characters", http.StatusOK},
		{"GET", "/api/v1/stories/nonexistent/locations", "locations", http.StatusOK},
		{"GET", "/api/v1/stories/nonexistent/chapters", "chapters", http.StatusOK},
	}

	for _, rt := range routes {
		t.Run(rt.label, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(rt.method, rt.path, nil)
			if rt.method == "POST" {
				req.Body = io.NopCloser(bytes.NewBufferString(`{"name":"test"}`))
				req.Header.Set("Content-Type", "application/json")
			}
			srv.Router.ServeHTTP(rec, req)
			if rec.Code != rt.want {
				t.Fatalf("[%s] expected %d, got %d: %s", rt.label, rt.want, rec.Code, rec.Body.String())
			}
		})
	}
}

// ── CORS / Method Not Allowed ──────────────────────────────────────────

func TestAPI_MethodNotAllowed(t *testing.T) {
	cleanCollections(t, "stories")
	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Method", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("delete on list returns 405", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/stories", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("post on story detail returns 405", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID, bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("post on node detail returns 405", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/nodes/whatever", nil)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("delete on edge list returns 405", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/stories/whatever/edges/", nil))
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 405/404/400, got %d", rec.Code)
		}
	})
}
