package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/httpapi"
	"github.com/premchand/story-builder/internal/memory"
	storymongo "github.com/premchand/story-builder/internal/mongo"
	"github.com/premchand/story-builder/internal/story"
)

type client struct {
	t       *testing.T
	handler http.Handler
}

func (c client) call(method, path string, body any, want int) []byte {
	c.t.Helper()
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	if rec.Code != want {
		c.t.Fatalf("%s %s: want %d, got %d: %s", method, path, want, rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}
func decode[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func (c client) generate(id string) story.Draft {
	c.t.Helper()
	data := c.call("POST", "/scenes/"+id+"/generate", nil, 200)
	var draft story.Draft
	var chunks strings.Builder
	for _, event := range strings.Split(string(data), "\n\n") {
		lines := strings.SplitN(event, "\n", 2)
		if len(lines) != 2 {
			continue
		}
		payload := []byte(strings.TrimPrefix(lines[1], "data: "))
		switch lines[0] {
		case "event: error":
			c.t.Fatalf("generation: %s", payload)
		case "event: chunk":
			v := decode[map[string]string](c.t, payload)
			chunks.WriteString(v["text"])
		case "event: complete":
			draft = decode[story.Draft](c.t, payload)
		}
	}
	if draft.ID == "" || draft.Prose != chunks.String() || draft.Status != story.DraftReady {
		c.t.Fatalf("invalid stream: %s", data)
	}
	return draft
}
func TestAPIScenarios(t *testing.T) {
	factories := map[string]func(*testing.T) story.Repository{"memory": func(*testing.T) story.Repository { return memory.NewRepository() }}
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		factories["mongo"] = func(t *testing.T) story.Repository {
			ctx := context.Background()
			db := "story_builder_http_test_" + uuid.NewString()
			conn, repo, err := storymongo.Connect(ctx, uri, db)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Database(db).Drop(ctx); _ = conn.Disconnect(ctx) })
			return repo
		}
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			c := client{t, httpapi.New(story.NewService(factory(t)), "test-token").Handler()}

			t.Run("empty_and_invalid", func(t *testing.T) {
				c := client{t, c.handler}
				if string(bytes.TrimSpace(c.call("GET", "/stories", nil, 200))) != "[]" {
					t.Error("empty stories must be an array")
				}
				c.call("POST", "/stories", map[string]string{"title": "  "}, 400)
				c.call("GET", "/stories/missing/graph", nil, 404)
				c.call("GET", "/stories/missing/characters", nil, 404)
				c.call("GET", "/stories/missing/character-states", nil, 404)
				c.call("POST", "/stories/missing/scenes", map[string]string{"title": "X"}, 404)
				c.call("POST", "/drafts/missing/accept", nil, 404)
				c.call("POST", "/drafts/missing/reject", nil, 404)
			})
			t.Run("graph_and_draft_errors", func(t *testing.T) { testGraphErrors(client{t, c.handler}) })
			for _, tc := range []struct {
				name, brief string
				scenes      int
			}{
				{"short", "A traveler makes a choice.", 2},
				{"unicode", "மீரா opens a letter. 🌙\n她选择回家。", 3},
				{"long", strings.Repeat("Mira follows the river and remembers the promise. ", 2500), 20},
			} {
				t.Run(tc.name, func(t *testing.T) {
					c := client{t, c.handler}
					st := decode[story.Story](t, c.call("POST", "/stories", map[string]string{"title": tc.name}, 201))
					prefix := "/stories/" + st.ID
					graph := decode[map[string]json.RawMessage](t, c.call("GET", prefix+"/graph", nil, 200))
					if string(graph["scenes"]) != "[]" || string(graph["edges"]) != "[]" {
						t.Fatal("empty graph must have arrays", string(graph["scenes"]), string(graph["edges"]))
					}
					chars := []story.Character{}
					for i := 0; i < 3; i++ {
						ch := decode[story.Character](t, c.call("POST", prefix+"/characters", map[string]any{"name": fmt.Sprintf("Mira %d", i), "traits": []string{"curious", "勇敢"}}, 201))
						chars = append(chars, ch)
					}
					c.call("POST", prefix+"/characters", map[string]string{"name": " "}, 400)
					if len(decode[[]story.Character](t, c.call("GET", prefix+"/characters", nil, 200))) != 3 {
						t.Fatal("missing characters")
					}
					var previous string
					for i := 0; i < tc.scenes; i++ {
						scene := decode[story.Scene](t, c.call("POST", prefix+"/scenes", map[string]any{"title": fmt.Sprintf("Chapter %d", i), "brief": tc.brief}, 201))
						if previous != "" {
							c.call("POST", prefix+"/edges", map[string]string{"from_scene_id": previous, "to_scene_id": scene.ID}, 201)
						}
						d := c.generate(scene.ID)
						if !strings.Contains(d.Prose, strings.TrimSpace(tc.brief)) {
							t.Fatal("prose truncated")
						}
						reused := c.generate(scene.ID)
						if reused.ID != d.ID {
							t.Fatal("generation not deduplicated")
						}
						if i == 0 {
							if len(decode[[]story.CharacterState](t, c.call("GET", prefix+"/character-states", nil, 200))) != 3 {
								t.Fatal("generation changed canon")
							}
							c.call("POST", "/drafts/"+d.ID+"/reject", nil, 204)
							if len(decode[[]story.CharacterState](t, c.call("GET", prefix+"/character-states", nil, 200))) != 3 {
								t.Fatal("rejection changed canon")
							}
							d = c.generate(scene.ID)
						}
						accepted := decode[story.Scene](t, c.call("POST", "/drafts/"+d.ID+"/accept", nil, 200))
						if accepted.Prose != d.Prose {
							t.Fatal("accepted prose mismatch")
						}
						c.call("POST", "/drafts/"+d.ID+"/accept", nil, 200)
						previous = scene.ID
					}
					g := decode[story.Graph](t, c.call("GET", prefix+"/graph", nil, 200))
					if len(g.Scenes) != tc.scenes || len(g.Edges) != tc.scenes-1 || g.Story.CanonVersion != int64(tc.scenes) {
						t.Fatalf("invalid final graph: scenes=%d edges=%d canon=%d", len(g.Scenes), len(g.Edges), g.Story.CanonVersion)
					}
					states := decode[[]story.CharacterState](t, c.call("GET", prefix+"/character-states", nil, 200))
					if len(states) != 3 {
						t.Fatal("missing character states")
					}
					for _, state := range states {
						if len(state.Knowledge) != 0 {
							t.Fatalf("bad state: %+v", state)
						}
					}
				})
			}
		})
	}
}

func TestHTTPValidationAndAuthentication(t *testing.T) {
	handler := httpapi.New(story.NewService(memory.NewRepository()), "secret").Handler()
	for _, tc := range []struct {
		path, token string
		status      int
	}{{"/healthz", "", 200}, {"/stories", "", 401}, {"/stories", "Bearer wrong", 401}, {"/stories", "Bearer secret", 200}} {
		r := httptest.NewRequest("GET", "/api/v1"+tc.path, nil)
		r.Header.Set("Authorization", tc.token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("auth got %d", w.Code)
		}
	}
	c := client{t, handler}
	for _, body := range []string{`{"title":"ok","unknown":1}`, `{"title":"ok"} {"title":"extra"}`, `{"title":"` + strings.Repeat("x", 1<<20) + `"}`} {
		r := httptest.NewRequest("POST", "/api/v1/stories", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer secret")
		w := httptest.NewRecorder()
		c.handler.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Errorf("invalid body accepted: status %d (length %d)", w.Code, len(body))
		}
	}
}

func testGraphErrors(c client) {
	t := c.t
	st := decode[story.Story](t, c.call("POST", "/stories", map[string]string{"title": "Graph errors"}, 201))
	p := "/stories/" + st.ID
	other := decode[story.Story](t, c.call("POST", "/stories", map[string]string{"title": "Other"}, 201))
	a := decode[story.Scene](t, c.call("POST", p+"/scenes", map[string]string{"title": "A"}, 201))
	b := decode[story.Scene](t, c.call("POST", p+"/scenes", map[string]string{"title": "B"}, 201))
	foreign := decode[story.Scene](t, c.call("POST", "/stories/"+other.ID+"/scenes", map[string]string{"title": "Foreign"}, 201))
	edge := func(from, to string, status int) {
		c.call("POST", p+"/edges", map[string]string{"from_scene_id": from, "to_scene_id": to}, status)
	}
	edge(a.ID, a.ID, 400)
	edge(a.ID, foreign.ID, 409)
	edge(a.ID, "missing", 404)
	edge(a.ID, b.ID, 201)
	edge(a.ID, b.ID, 409)
	edge(b.ID, a.ID, 422)
	child := c.generate(b.ID)
	c.call("POST", "/drafts/"+child.ID+"/accept", nil, 409)
	parent := c.generate(a.ID)
	c.call("POST", "/drafts/"+parent.ID+"/accept", nil, 200)
	c.call("POST", "/drafts/"+parent.ID+"/reject", nil, 409)
	c.call("POST", "/drafts/"+parent.ID+"/accept", nil, 200)
	c.call("POST", "/drafts/"+child.ID+"/accept", nil, 409)
	stream := c.call("POST", "/scenes/"+a.ID+"/generate", nil, 200)
	if !strings.Contains(string(stream), "event: error") {
		t.Fatal("accepted scene regenerated")
	}
	c.call("PATCH", "/scenes/"+b.ID, map[string]string{"title": " "}, 400)
	c.call("PATCH", "/scenes/missing", map[string]string{"title": "X"}, 404)
}

func TestFullLengthGenerationHTTP(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := json.Marshal(map[string]string{"type": "response.output_text.delta", "delta": strings.Repeat("novel ", 1000)})
		fmt.Fprintf(w, "data: %s\n\ndata: {\"type\":\"response.completed\"}\n\n", data)
	}))
	defer model.Close()
	generator := story.OpenAIGenerator{BaseURL: model.URL, Model: "fiction", APIKey: "test-secret", TargetWords: 1000}
	c := client{t, httpapi.New(story.NewService(memory.NewRepository()), "test-token", generator).Handler()}
	config := decode[struct {
		Mode         string
		MinimumWords int
	}](t, c.call("GET", "/generation", nil, 200))
	if config.Mode != "openai" || config.MinimumWords != 1000 {
		t.Fatal("incorrect generation configuration", config)
	}
	st := decode[story.Story](t, c.call("POST", "/stories", map[string]string{"title": "Novel"}, 201))
	scene := decode[story.Scene](t, c.call("POST", "/stories/"+st.ID+"/scenes", map[string]string{"title": "Opening"}, 201))
	draft := c.generate(scene.ID)
	if len(strings.Fields(draft.Prose)) != 1000 {
		t.Fatal("long scene truncated")
	}
	accepted := decode[story.Scene](t, c.call("POST", "/drafts/"+draft.ID+"/accept", nil, 200))
	if accepted.Prose != draft.Prose {
		t.Fatal("long scene was not accepted intact")
	}
}
