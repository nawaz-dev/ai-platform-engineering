package router

import (
	"ai-platform-fde/applications/model-gateway/internal/config"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type Router struct {
	models  map[string]config.Model
	proxies map[string]*httputil.ReverseProxy
}

func New(models map[string]config.Model) *Router {
	r := &Router{models: models, proxies: make(map[string]*httputil.ReverseProxy)}
	for name, m := range models {
		target, _ := url.Parse(m.BaseURL)
		p := &httputil.ReverseProxy{}
		p.FlushInterval = -1
		upstreamKey := ""
		if m.APIKeyEnv != "" {
			upstreamKey = getenv(m.APIKeyEnv)
		}
		p.Rewrite = func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
			pr.Out.Header.Del("Authorization")
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Del("Proxy-Authorization")
			if upstreamKey != "" {
				pr.Out.Header.Set("Authorization", "Bearer "+upstreamKey)
			}
		}
		p.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
			log.Printf("upstream error: %v", err)
			http.Error(w, "model backend unavailable", http.StatusBadGateway)
		}
		r.proxies[name] = p
	}
	return r
}
func getenv(k string) string { return os.Getenv(k) }

func (rt *Router) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", 405)
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		http.Error(w, "Content-Type must be application/json", 415)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "request too large or unreadable", 400)
		return
	}
	var body map[string]json.RawMessage
	if err = json.Unmarshal(raw, &body); err != nil || body == nil {
		http.Error(w, "invalid JSON", 400)
		return
	}
	var name string
	if err = json.Unmarshal(body["model"], &name); err != nil || name == "" {
		http.Error(w, "model is required", 400)
		return
	}
	m, ok := rt.models[name]
	if !ok {
		http.Error(w, "unknown model", 404)
		return
	}
	var msgs []json.RawMessage
	if err = json.Unmarshal(body["messages"], &msgs); err != nil || len(msgs) == 0 {
		http.Error(w, "messages must be a non-empty array", 400)
		return
	}
	modelBytes, _ := json.Marshal(m.ProviderModelID)
	body["model"] = modelBytes
	payload, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "invalid payload", 400)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(payload))
	r.ContentLength = int64(len(payload))
	r.Header.Del("Content-Encoding")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Del("Content-Length")
	r.Header.Del("Transfer-Encoding")
	rt.proxies[name].ServeHTTP(w, r)
}
func (rt *Router) Models(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", 405)
		return
	}
	type item struct {
		ID     string `json:"id"`
		Object string `json:"object"`
	}
	out := struct {
		Object string `json:"object"`
		Data   []item `json:"data"`
	}{Object: "list", Data: make([]item, 0, len(rt.models))}
	for name := range rt.models {
		out.Data = append(out.Data, item{ID: name, Object: "model"})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
