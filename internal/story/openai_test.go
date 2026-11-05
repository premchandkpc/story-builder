package story

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sendOpenAIEvent(w http.ResponseWriter, kind, delta string) {
	data, _ := json.Marshal(map[string]string{"type": kind, "delta": delta})
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, data)
}
func TestOpenAILongScene(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("missing endpoint or server-side authentication")
		}
		var request struct {
			Model           string
			Input           []map[string]string
			Instructions    string
			Stream          bool
			Store           bool
			MaxOutputTokens int `json:"max_output_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Model != "fiction" || request.MaxOutputTokens < 1000 || !request.Stream || request.Store {
			t.Error("missing request controls")
		}
		if calls == 1 && (!strings.Contains(request.Input[0]["content"], "Mira") || !strings.Contains(request.Input[0]["content"], "accepted parent")) {
			t.Error("missing narrative context")
		}
		sendOpenAIEvent(w, "response.created", "")
		sendOpenAIEvent(w, "response.output_text.delta", strings.Repeat("story ", 600))
		sendOpenAIEvent(w, "response.completed", "")
	}))
	defer server.Close()
	g := OpenAIGenerator{BaseURL: server.URL, Model: "fiction", APIKey: "test-secret", TargetWords: 1000}
	var prose strings.Builder
	err := g.GenerateWithContext(context.Background(), Scene{Title: "Opening"}, GenerationContext{Characters: []Character{{Name: "Mira"}}, Parents: []Scene{{Prose: "accepted parent"}}}, func(s string) error { prose.WriteString(s); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(strings.Fields(prose.String())) < 1000 {
		t.Fatalf("incomplete scene: calls=%d words=%d", calls, len(strings.Fields(prose.String())))
	}
	if g.CacheKey() == (OpenAIGenerator{Model: "fiction", TargetWords: 500}).CacheKey() {
		t.Fatal("word count missing from cache key")
	}
}
func TestOpenAIFailures(t *testing.T) {
	for _, kind := range []string{"http_error", "unauthorized", "rate_limit", "stream_error", "interrupted", "incomplete", "refused", "short", "cancelled", "send_error", "missing_key", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch kind {
				case "http_error":
					w.WriteHeader(503)
				case "unauthorized":
					w.WriteHeader(401)
				case "rate_limit":
					w.WriteHeader(429)
				case "stream_error":
					sendOpenAIEvent(w, "error", "")
				case "interrupted":
					sendOpenAIEvent(w, "response.output_text.delta", "partial")
				case "incomplete":
					sendOpenAIEvent(w, "response.incomplete", "")
				case "refused":
					sendOpenAIEvent(w, "response.refusal.delta", "")
				case "malformed":
					fmt.Fprint(w, "data: invalid JSON\n\n")
				default:
					sendOpenAIEvent(w, "response.output_text.delta", "too short")
					sendOpenAIEvent(w, "response.completed", "")
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancelled" {
				cancel()
			}
			key := "test-secret"
			if kind == "missing_key" {
				key = ""
			}
			g := OpenAIGenerator{BaseURL: server.URL, Model: "fiction", APIKey: key, TargetWords: 1000}
			err := g.Generate(ctx, Scene{}, func(string) error {
				if kind == "send_error" {
					return fmt.Errorf("disconnected")
				}
				return nil
			})
			if err == nil {
				t.Fatal("incomplete generation succeeded")
			}
			if strings.Contains(err.Error(), "test-secret") {
				t.Fatal("credential leaked in error")
			}
			if kind == "short" && calls != 3 {
				t.Fatalf("want bounded continuations, got %d", calls)
			}
			if kind == "missing_key" && calls != 0 {
				t.Fatal("unconfigured request sent")
			}
		})
	}
}

func TestOpenAIStreamErrorCodes(t *testing.T) {
	for _, tc := range []struct{ code, want string }{{"insufficient_quota", "quota or credits"}, {"rate_limit_exceeded", "rate limit"}, {"server_error", "server_error"}, {"secret with spaces", "generation failed"}} {
		got := openAIStreamError(tc.code).Error()
		if !strings.Contains(got, tc.want) {
			t.Fatalf("unexpected error: %s", got)
		}
		if tc.code == "secret with spaces" && strings.Contains(got, tc.code) {
			t.Fatal("unsafe error exposed")
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"code\":\"insufficient_quota\",\"message\":\"private details\"}}}\n\n")
	}))
	defer server.Close()
	err := (OpenAIGenerator{BaseURL: server.URL, Model: "fiction", APIKey: "test-secret"}).Generate(context.Background(), Scene{}, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "quota or credits") || strings.Contains(err.Error(), "private details") {
		t.Fatalf("unsafe or unhelpful error: %v", err)
	}
}

func TestOpenAINestedCreditError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"type\":\"error\",\"error\":{\"code\":\"credit_balance_exhausted\",\"message\":\"private details\"}}\n\n")
	}))
	defer server.Close()
	err := (OpenAIGenerator{BaseURL: server.URL, Model: "fiction", APIKey: "test-secret"}).Generate(context.Background(), Scene{}, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "quota or credits") || strings.Contains(err.Error(), "private details") {
		t.Fatalf("unexpected provider diagnostic: %v", err)
	}
}
