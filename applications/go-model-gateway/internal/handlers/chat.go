package handlers

import (
	"ai-platform-fde/applications/model-gateway/internal/vllm"
	"encoding/json"
	"io"
	"net/http"
)

type ChatHandler struct {
	Client *vllm.Client
	Model  string
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var payload map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload == nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}

	var messages []json.RawMessage
	if err := json.Unmarshal(payload["messages"], &messages); err != nil || len(messages) == 0 {
		http.Error(w, "messages must be a non-empty array", http.StatusBadRequest)
		return
	}

	if _, exists := payload["model"]; !exists {
		model, _ := json.Marshal(h.Model)
		payload["model"] = model
	}

	var stream bool
	if raw, exists := payload["stream"]; exists {
		if err := json.Unmarshal(raw, &stream); err != nil {
			http.Error(w, "stream must be a boolean", http.StatusBadRequest)
			return
		}
	}
	if stream {
		http.Error(w, "streaming is not supported yet", http.StatusBadRequest)
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "unable to encode request", http.StatusBadRequest)
		return
	}

	resp, err := h.Client.Chat(r.Context(), body)
	if err != nil {
		http.Error(w, "upstream vLLM unavailable", http.StatusBadGateway)
		return
	}

	result, err := vllm.ReadResponse(resp)
	if err != nil {
		http.Error(w, "unable to read upstream response", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(result)
}

var _ = io.EOF
