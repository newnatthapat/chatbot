// Package config loads and validates the app's environment-variable
// configuration (ADR-0002: the active Provider is fixed at startup, not
// toggled at runtime).
package config

import "fmt"

type OllamaConfig struct {
	BaseURL string
	Model   string
}

type Config struct {
	ActiveProvider string
	Ollama         OllamaConfig
	AdminUsername  string
	AdminPassword  string
	DBPath         string
	Addr           string
}

// Getenv matches os.Getenv's signature so callers can pass a fake in tests.
type Getenv func(key string) string

func Load(getenv Getenv) (Config, error) {
	cfg := Config{
		ActiveProvider: getenv("ACTIVE_PROVIDER"),
		Ollama: OllamaConfig{
			BaseURL: getenv("OLLAMA_BASE_URL"),
			Model:   getenv("OLLAMA_MODEL"),
		},
		AdminUsername: getenv("ADMIN_USERNAME"),
		AdminPassword: getenv("ADMIN_PASSWORD"),
		DBPath:        getenv("DB_PATH"),
		Addr:          getenv("ADDR"),
	}

	if cfg.ActiveProvider == "" {
		cfg.ActiveProvider = "ollama"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "chatbot.db"
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}

	if cfg.ActiveProvider != "ollama" {
		return Config{}, fmt.Errorf("config: no Provider implementation for ACTIVE_PROVIDER=%q", cfg.ActiveProvider)
	}
	if cfg.Ollama.BaseURL == "" {
		return Config{}, fmt.Errorf("config: OLLAMA_BASE_URL is required when ACTIVE_PROVIDER=ollama")
	}
	if cfg.Ollama.Model == "" {
		return Config{}, fmt.Errorf("config: OLLAMA_MODEL is required when ACTIVE_PROVIDER=ollama")
	}
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		return Config{}, fmt.Errorf("config: ADMIN_USERNAME and ADMIN_PASSWORD are required")
	}

	return cfg, nil
}
