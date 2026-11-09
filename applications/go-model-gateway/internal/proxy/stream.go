package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func Forward(
	w http.ResponseWriter,
	r *http.Request,
	upstreamURL string,
	upstreamAPIKey string,
	body []byte,
) {
	req, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		strings.TrimRight(upstreamURL, "/")+"/v1/chat/completions",
		strings.NewReader(string(body)),
	)
	if err != nil {
		http.Error(w, "Invalid upstream request", http.StatusBadGateway)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if upstreamAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+upstreamAPIKey)
	}

	client := &http.Client{
		Timeout: 0, // No overall timeout for long-running streams
	}

	resp, err := client.Do(req)
	if err != nil {
		if r.Context().Err() == context.Canceled {
			return
		}
		http.Error(w, "Upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := w.Write(buf[:n]); err != nil {
				return
			}
			flusher.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF {
				// Connection may have been interrupted.
			}
			return
		}
	}
}
