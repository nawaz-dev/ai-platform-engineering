package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Model struct {
	ModelID         string `json:"model_id"`
	Provider        string `json:"provider"`
	BaseURL         string `json:"base_url"`
	ProviderModelID string `json:"provider_model_id"`
	APIKeyEnv       string `json:"api_key_env,omitempty"`
}

type Registry struct {
	Models []Model `json:"models"`
}
type Config struct {
	Port          string
	GatewayAPIKey string
	Models        map[string]Model
}

func Load() (Config, error) {
	key := os.Getenv("GATEWAY_API_KEY")
	// fmt.Printf("GATEWAY_API_KEY: %s\n", key)
	if len(key) < 16 {
		return Config{}, errors.New("GATEWAY_API_KEY must be at least 16 characters")
	}
	path := os.Getenv("MODELS_CONFIG")
	if path == "" {
		path = "configs/models.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read model registry: %w", err)
	}
	var registry Registry
	if err = json.Unmarshal(raw, &registry); err != nil {
		return Config{}, fmt.Errorf("parse model registry: %w", err)
	}
	if len(registry.Models) == 0 {
		return Config{}, errors.New("model registry is empty")
	}
	models := make(map[string]Model, len(registry.Models))

	for _, m := range registry.Models {
		if m.ModelID == "" ||
			m.Provider == "" ||
			m.ProviderModelID == "" {
			return Config{}, errors.New(
				"model_id, provider and provider_model_id are required",
			)
		}

		u, err := url.Parse(m.BaseURL)
		if err != nil ||
			u.Host == "" ||
			(u.Scheme != "http" && u.Scheme != "https") ||
			u.User != nil ||
			u.RawQuery != "" ||
			u.Fragment != "" ||
			u.Path != "" {
			return Config{}, fmt.Errorf(
				"invalid base_url for model %q (use scheme and host only)",
				m.ModelID,
			)
		}

		if _, exists := models[m.ModelID]; exists {
			return Config{}, fmt.Errorf(
				"duplicate model_id %q", m.ModelID,
			)
		}

		if strings.TrimSpace(m.APIKeyEnv) != "" &&
			os.Getenv(m.APIKeyEnv) == "" {
			return Config{}, fmt.Errorf(
				"missing provider credential env %q",
				m.APIKeyEnv,
			)
		}

		models[m.ModelID] = m
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return Config{Port: port, GatewayAPIKey: key, Models: models}, nil
}
