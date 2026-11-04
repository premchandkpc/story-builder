package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/premchand/story-builder/internal/httpapi"
	"github.com/premchand/story-builder/internal/memory"
	"github.com/premchand/story-builder/internal/story"
)

func TestAuthenticatedCharacterAndDraftFlow(t *testing.T) {
	ctx := context.Background()
	svc := story.NewService(memory.NewRepository())
	st, err := svc.CreateStory(ctx, "Map")
	if err != nil {
		t.Fatal(err)
	}
	scene, err := svc.CreateScene(ctx, st.ID, "Opening", "", story.Position{})
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(svc, "secret").Handler()
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request(http.MethodGet, "/api/v1/stories", "", "wrong"); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	if w := request(http.MethodGet, "/api/v1/healthz", "", ""); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	w := request(http.MethodPost, "/api/v1/stories/"+st.ID+"/characters", `{"name":"Mira","motivation":"Find the truth","initialKnowledge":["A door is locked"]}`, "secret")
	if w.Code != http.StatusCreated {
		t.Fatalf("character: %d %s", w.Code, w.Body.String())
	}
	var c story.Character
	if err := json.Unmarshal(w.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	w = request(http.MethodPut, "/api/v1/stories/"+st.ID+"/characters/"+c.CharacterID, `{"name":"Mira","motivation":"Protect the truth","expectedVersion":1}`, "secret")
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	w = request(http.MethodPatch, "/api/v1/scenes/"+scene.ID, `{"brief":"New brief","expectedRevision":99}`, "secret")
	if w.Code != http.StatusConflict {
		t.Fatalf("scene conflict: %d %s", w.Code, w.Body.String())
	}
	d, err := svc.GenerateDraft(ctx, scene.ID, story.TemplateGenerator{}, func(string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	w = request(http.MethodPost, "/api/v1/drafts/"+d.ID+"/reject", "", "secret")
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("reject: %d %q", w.Code, w.Body.String())
	}
}
