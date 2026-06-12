package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/graph"
	blueprintsvc "github.com/premchand/story-builder/internal/service/blueprint"
	canonsvc "github.com/premchand/story-builder/internal/service/canon"
	"github.com/premchand/story-builder/internal/service/edge"
	gentsvc "github.com/premchand/story-builder/internal/service/generation"
	nodesvc "github.com/premchand/story-builder/internal/service/node"
	scenesvc "github.com/premchand/story-builder/internal/service/scene"
	storysvc "github.com/premchand/story-builder/internal/service/story"
	summarysvc "github.com/premchand/story-builder/internal/service/summary"
	timelinesvc "github.com/premchand/story-builder/internal/service/timeline"
)

func TestSmoke_CriticalFlows(t *testing.T) {
	mem := graph.NewMemoryStore()

	charSvc := canonsvc.NewMemoryCharacterService()
	actorSvc := canonsvc.NewMemoryActorService()
	traitSvc := canonsvc.NewMemoryTraitService()
	locSvc := canonsvc.NewMemoryLocationService()
	loreSvc := canonsvc.NewMemoryLoreService()
	storySvc := storysvc.NewMemoryService(mem)
	nodeSvc := nodesvc.NewMemoryService(mem)
	sceneSvc := scenesvc.NewMemoryService()
	genSvc := gentsvc.NewMemoryGenerationService()
	summarySvc := summarysvc.NewMemoryService()
	storyGenSvc := gentsvc.NewMemoryStoryGeneratorService()
	castingSvc := canonsvc.NewMemoryCastingService()
	blueprintSvc := blueprintsvc.NewMemoryService()

	srv := NewServer(
		&CharacterHandler{Service: charSvc},
		&ActorHandler{Service: actorSvc},
		&CharacterTraitHandler{Service: traitSvc},
		&CastingHandler{Service: castingSvc},
		&LocationHandler{Service: locSvc},
		&LoreHandler{Service: loreSvc},
		&StoryHandler{StorySvc: storySvc, EdgeSvc: edge.NewMemoryService(mem), NodeSvc: nodeSvc, BlueprintService: blueprintSvc, TimelineService: timelinesvc.NewMemoryService()},
		&NodeHandler{Service: nodeSvc},
		&GenerationHandler{Service: genSvc},
		&SceneHandler{SceneService: sceneSvc},
		&SummaryHandler{Service: summarySvc},
		&StoryGeneratorHandler{Service: storyGenSvc},
		nil,
	)

	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := ts.Client()

	t.Run("health check", func(t *testing.T) {
		res, err := client.Get(ts.URL + "/api/v1/healthz")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = res.Body.Close()
		}()
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var body map[string]string
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "ok" {
			t.Fatalf("expected status ok, got %s", body["status"])
		}
	})

	t.Run("create story", func(t *testing.T) {
		res, err := client.Post(ts.URL+"/api/v1/stories/", "application/json", strings.NewReader(`{"title":"The Red Path"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d", res.StatusCode)
		}
	})

	t.Run("create story blank title rejected", func(t *testing.T) {
		res, err := client.Post(ts.URL+"/api/v1/stories/", "application/json", strings.NewReader(`{"title":"  "}`))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 400 {
			t.Fatalf("expected 400, got %d", res.StatusCode)
		}
	})

	t.Run("list stories", func(t *testing.T) {
		res, err := client.Get(ts.URL + "/api/v1/stories/")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var stories []graph.Story
		if err := json.NewDecoder(res.Body).Decode(&stories); err != nil {
			t.Fatal(err)
		}
		if len(stories) == 0 {
			t.Fatal("expected at least 1 story")
		}
	})

	t.Run("create story blueprint", func(t *testing.T) {
		body := `{"premise":"A thief steals the moon","theme":"sacrifice","conflict":"the city hunts the thief","acts":[{"title":"Act I","goal":"introduce the thief"}]}`
		res, err := client.Post(ts.URL+"/api/v1/stories/"+uuid.NewString()+"/blueprint/", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 404 {
			t.Fatalf("expected 404 for unknown story id, got %d", res.StatusCode)
		}
	})

	t.Run("create timeline event", func(t *testing.T) {
		body := `{"title":"Opening","description":"The story begins","order":1}`
		res, err := client.Post(ts.URL+"/api/v1/stories/"+uuid.NewString()+"/timeline/", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 404 {
			t.Fatalf("expected 404 for unknown story id, got %d", res.StatusCode)
		}
	})

	t.Run("create character", func(t *testing.T) {
		body := `{"name":"Aria","persona":"brave warrior","backstory":"orphan","moral_alignment":"good","personality":["brave"],"flaws":["stubborn"],"goals":["find truth"],"traits":["fighter"],"voice_samples":["hello"]}`
		res, err := client.Post(ts.URL+"/api/v1/characters/", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
		}
	})

	t.Run("list characters", func(t *testing.T) {
		res, err := client.Get(ts.URL + "/api/v1/characters/")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
	})

	t.Run("create location", func(t *testing.T) {
		body := `{"name":"Dark Forest","description":"a shadowy woods"}`
		res, err := client.Post(ts.URL+"/api/v1/locations/", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
		}
	})

	t.Run("create actor", func(t *testing.T) {
		body := `{"name":"Alice","gender":"female","age_range":"25-35","notes":"versatile"}`
		res, err := client.Post(ts.URL+"/api/v1/actors/", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d", res.StatusCode)
		}
	})

	t.Run("create character trait", func(t *testing.T) {
		body := `{"name":"brave","category":"personality","description":"courageous"}`
		res, err := client.Post(ts.URL+"/api/v1/character-traits/", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d", res.StatusCode)
		}
	})

	t.Run("full story graph CRUD", func(t *testing.T) {
		// create story
		res, err := client.Post(ts.URL+"/api/v1/stories/", "application/json", strings.NewReader(`{"title":"Graph Test"}`))
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d", res.StatusCode)
		}
		var story graph.Story
		if err := json.NewDecoder(res.Body).Decode(&story); err != nil {
			t.Fatal(err)
		}
		if story.ID == uuid.Nil {
			t.Fatal("expected non-zero story ID")
		}

		storyURL := ts.URL + "/api/v1/stories/" + story.ID.String()

		// create node 1
		node1Body := `{"beat_intent":"introduction","pov":"hero","tone":"light","target_words":300}`
		res, err = client.Post(storyURL+"/nodes/", "application/json", strings.NewReader(node1Body))
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
		}
		var node1 graph.Node
		if err := json.NewDecoder(res.Body).Decode(&node1); err != nil {
			t.Fatal(err)
		}

		// create node 2
		node2Body := `{"beat_intent":"rising action","pov":"hero","tone":"tense","target_words":500}`
		res, err = client.Post(storyURL+"/nodes/", "application/json", strings.NewReader(node2Body))
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d", res.StatusCode)
		}
		var node2 graph.Node
		if err := json.NewDecoder(res.Body).Decode(&node2); err != nil {
			t.Fatal(err)
		}

		// create edge
		edgeBody := `{"from_node":"` + node1.ID.String() + `","to_node":"` + node2.ID.String() + `","edge_type":"seq"}`
		res, err = client.Post(storyURL+"/edges/", "application/json", strings.NewReader(edgeBody))
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 201 {
			t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
		}

		// list edges
		res, err = client.Get(storyURL + "/edges/")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var edges []graph.Edge
		if err := json.NewDecoder(res.Body).Decode(&edges); err != nil {
			t.Fatal(err)
		}
		if len(edges) != 1 {
			t.Fatalf("expected 1 edge, got %d", len(edges))
		}

		// topology
		res, err = client.Get(storyURL + "/topology")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var topo struct {
			Nodes []graph.Node `json:"nodes"`
			Edges []graph.Edge `json:"edges"`
		}
		if err := json.NewDecoder(res.Body).Decode(&topo); err != nil {
			t.Fatal(err)
		}
		if len(topo.Nodes) != 2 {
			t.Fatalf("expected 2 nodes in topology, got %d", len(topo.Nodes))
		}
		if len(topo.Edges) != 1 {
			t.Fatalf("expected 1 edge in topology, got %d", len(topo.Edges))
		}

		// set scene structure
		ssBody := `{"scene_structure":{"flow_type":"monologue","situation_flow":"hero enters forest","beat_flow":"","emotional_flow":"curious","conflict_flow":"","shift_flow":"","revelation_flow":"","thematic_flow":""}}`
		ssReq, _ := http.NewRequest(http.MethodPut, storyURL+"/nodes/"+node1.ID.String()+"/scene/structure", strings.NewReader(ssBody))
		ssReq.Header.Set("Content-Type", "application/json")
		res, err = client.Do(ssReq)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d: %s", res.StatusCode, res.Body)
		}

		// get scene structure
		res, err = client.Get(storyURL + "/nodes/" + node1.ID.String() + "/scene/structure")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}

		// update node
		updateBody := `{"beat_intent":"updated intro","pov":"hero","tone":"dark","target_words":400,"character_refs":[],"scene_structure":null}`
		upReq, _ := http.NewRequest(http.MethodPut, storyURL+"/nodes/"+node1.ID.String(), strings.NewReader(updateBody))
		upReq.Header.Set("Content-Type", "application/json")
		res, err = client.Do(upReq)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d: %s", res.StatusCode, res.Body)
		}

		// create invalid edge (bad UUID) -> 400
		badEdgeBody := `{"from_node":"bad-id","to_node":"` + node2.ID.String() + `","edge_type":"seq"}`
		res, err = client.Post(storyURL+"/edges/", "application/json", strings.NewReader(badEdgeBody))
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 400 {
			t.Fatalf("expected 400 for bad edge UUID, got %d", res.StatusCode)
		}
	})

	t.Run("404 on unknown story", func(t *testing.T) {
		res, err := client.Get(ts.URL + "/api/v1/stories/00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 404 {
			t.Fatalf("expected 404, got %d", res.StatusCode)
		}
	})

	t.Run("404 on unknown node", func(t *testing.T) {
		res, err := client.Get(ts.URL + "/api/v1/stories/00000000-0000-0000-0000-000000000000/nodes/00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 404 {
			t.Fatalf("expected 404, got %d", res.StatusCode)
		}
	})

	t.Run("scene structure and summaries", func(t *testing.T) {
		// create story + node
		res, err := client.Post(ts.URL+"/api/v1/stories/", "application/json", strings.NewReader(`{"title":"Scene Test"}`))
		if err != nil {
			t.Fatal(err)
		}
		var story graph.Story
		if err := json.NewDecoder(res.Body).Decode(&story); err != nil {
			t.Fatal(err)
		}
		storyURL := ts.URL + "/api/v1/stories/" + story.ID.String()

		res, err = client.Post(storyURL+"/nodes/", "application/json", strings.NewReader(`{"beat_intent":"scene test","pov":"hero","tone":"neutral","target_words":200}`))
		if err != nil {
			t.Fatal(err)
		}
		var node graph.Node
		if err := json.NewDecoder(res.Body).Decode(&node); err != nil {
			t.Fatal(err)
		}
		nodeURL := storyURL + "/nodes/" + node.ID.String()

		// set scene structure
		ssBody := `{"scene_structure":{"flow_type":"monologue","situation_flow":"hero enters building","beat_flow":"","emotional_flow":"curious","conflict_flow":"","shift_flow":"","revelation_flow":"","thematic_flow":""}}`
		ssReq, _ := http.NewRequest(http.MethodPut, nodeURL+"/scene/structure", strings.NewReader(ssBody))
		ssReq.Header.Set("Content-Type", "application/json")
		res, err = client.Do(ssReq)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d: %s", res.StatusCode, res.Body)
		}

		// get scene structure
		res, err = client.Get(nodeURL + "/scene/structure")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}

		// get turns (empty, no scene started)
		res, err = client.Get(nodeURL + "/scene/turns")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}

		// summaries endpoints (may 404 if no summaries exist yet)
		for _, path := range []string{
			"/summaries/level",
			"/summaries/count",
			"/summaries/elevate",
			"/summaries/nodes/" + node.ID.String(),
		} {
			res, err := client.Get(storyURL + path)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != 200 && res.StatusCode != 404 {
				t.Fatalf("expected 200 or 404 for %s, got %d", path, res.StatusCode)
			}
		}
	})
}
