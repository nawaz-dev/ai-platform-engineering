package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-platform-fde/applications/model-gateway/internal/config"
)

func TestChatProxiesToBackend(t *testing.T) {
	t.Setenv("TEST_UPSTREAM_KEY", "provider-secret")
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer provider-secret" {
			t.Errorf("unexpected upstream authorization: %q", got)
		}
		if r.Header.Get("Cookie") != "" {
			t.Error("client cookie forwarded to backend")
		}
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode upstream request: %v", err)
		}
		if body.Model != "provider-model" {
			t.Errorf("unexpected upstream model: %q", body.Model)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat-test"}`))
	}))
	defer backend.Close()
	rt := New(map[string]config.Model{
		"alias": {BaseURL: backend.URL, ProviderModelID: "provider-model", APIKeyEnv: "TEST_UPSTREAM_KEY"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"alias","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer gateway-secret")
	req.Header.Set("Cookie", "session=private")
	response := httptest.NewRecorder()
	rt.Chat(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("chat returned %d: %s", response.Code, response.Body.String())
	}
	if response.Body.String() != `{"id":"chat-test"}` {
		t.Errorf("unexpected response: %s", response.Body.String())
	}
}
