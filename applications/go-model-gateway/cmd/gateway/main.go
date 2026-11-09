package main

import (
	"ai-platform-fde/applications/model-gateway/internal/config"
	"ai-platform-fde/applications/model-gateway/internal/middleware"
	"ai-platform-fde/applications/model-gateway/internal/router"
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	rt := router.New(cfg.Models)

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rpm := int64(60)
	if value := os.Getenv("RATE_LIMIT_RPM"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			log.Fatal("RATE_LIMIT_RPM must be a positive integer")
		}
		rpm = parsed
	}

	redisClient := middleware.NewRedisClient(redisAddr)
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := middleware.CheckRedis(ctx, redisClient); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	routes := http.NewServeMux()
	routes.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	routes.Handle("/metrics", middleware.MetricsHandler())

	chatHandler := middleware.RateLimit(
		redisClient,
		cfg.GatewayAPIKey,
		rpm,
		http.HandlerFunc(rt.Chat),
	)
	modelsHandler := middleware.RateLimit(
		redisClient,
		cfg.GatewayAPIKey,
		rpm,
		http.HandlerFunc(rt.Models),
	)

	routes.Handle(
		"/v1/chat/completions",
		middleware.APIKeyAuth(cfg.GatewayAPIKey, chatHandler),
	)
	routes.Handle(
		"/v1/models",
		middleware.APIKeyAuth(cfg.GatewayAPIKey, modelsHandler),
	)
	mux := http.NewServeMux()
	mux.Handle("/", middleware.Metrics(routes))
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 120 * time.Second}
	log.Printf("model gateway listening on :%s with %d models", cfg.Port, len(cfg.Models))
	log.Fatal(srv.ListenAndServe())
}
