package config_test

import (
	"testing"

	"chatbot-backend/internal/config"
)

func TestLoadDefaultsToOllamaProvider(t *testing.T) {
	env := map[string]string{
		"OLLAMA_BASE_URL": "http://ollama:11434",
		"OLLAMA_MODEL":    "llama3.1",
		"ADMIN_USERNAME":  "admin",
		"ADMIN_PASSWORD":  "12345",
	}

	cfg, err := config.Load(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ActiveProvider != "ollama" {
		t.Errorf("ActiveProvider = %q, want %q", cfg.ActiveProvider, "ollama")
	}
	if cfg.Ollama.BaseURL != "http://ollama:11434" {
		t.Errorf("Ollama.BaseURL = %q, want %q", cfg.Ollama.BaseURL, "http://ollama:11434")
	}
	if cfg.Ollama.Model != "llama3.1" {
		t.Errorf("Ollama.Model = %q, want %q", cfg.Ollama.Model, "llama3.1")
	}
}

func TestLoadFailsFastForUnimplementedProvider(t *testing.T) {
	env := map[string]string{
		"ACTIVE_PROVIDER": "vllm",
		"ADMIN_USERNAME":  "admin",
		"ADMIN_PASSWORD":  "12345",
	}

	_, err := config.Load(func(key string) string { return env[key] })
	if err == nil {
		t.Fatal("Load should have failed for an unimplemented provider, got nil error")
	}
}

func TestLoadFailsFastWhenOllamaConfigIncomplete(t *testing.T) {
	env := map[string]string{
		"OLLAMA_BASE_URL": "http://ollama:11434",
		// OLLAMA_MODEL missing
		"ADMIN_USERNAME": "admin",
		"ADMIN_PASSWORD": "12345",
	}

	_, err := config.Load(func(key string) string { return env[key] })
	if err == nil {
		t.Fatal("Load should have failed when OLLAMA_MODEL is missing, got nil error")
	}
}
