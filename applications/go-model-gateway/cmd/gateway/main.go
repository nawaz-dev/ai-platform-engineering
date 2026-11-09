package main

import (
	"ai-platform-fde/applications/model-gateway/internal/config"
	"ai-platform-fde/applications/model-gateway/internal/middleware"
	"ai-platform-fde/applications/model-gateway/internal/router"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	rt := router.New(cfg.Models)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	mux.Handle("/v1/chat/completions", middleware.APIKeyAuth(cfg.GatewayAPIKey, http.HandlerFunc(rt.Chat)))
	mux.Handle("/v1/models", middleware.APIKeyAuth(cfg.GatewayAPIKey, http.HandlerFunc(rt.Models)))
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 120 * time.Second}
	log.Printf("model gateway listening on :%s with %d models", cfg.Port, len(cfg.Models))
	log.Fatal(srv.ListenAndServe())
}
