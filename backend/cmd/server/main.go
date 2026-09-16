// Command server runs the chatbot backend: the Conversation/Message HTTP
// API, plus (when present) the built React frontend as static files.
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"chatbot-backend/internal/config"
	"chatbot-backend/internal/httpapi"
	"chatbot-backend/internal/provider"
	"chatbot-backend/internal/store"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	// ACTIVE_PROVIDER only ever resolves to "ollama" today (config.Load
	// fails fast otherwise) — this switch grows a case per ADR-0002 when a
	// vLLM Provider implementation is added.
	var p provider.Provider
	switch cfg.ActiveProvider {
	case "ollama":
		p = provider.NewOllama(cfg.Ollama.BaseURL, cfg.Ollama.Model)
	default:
		log.Fatalf("no Provider implementation for %q", cfg.ActiveProvider)
	}

	api := httpapi.New(st, p, cfg.AdminUsername, cfg.AdminPassword)

	mux := http.NewServeMux()
	mux.Handle("/api/", api)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./web"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		mux.Handle("/", spaHandler(staticDir))
		log.Printf("serving frontend from %s", staticDir)
	} else {
		log.Printf("no frontend found at %s, serving API only", staticDir)
	}

	log.Printf("listening on %s (provider=%s)", cfg.Addr, cfg.ActiveProvider)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// spaHandler serves static files, falling back to index.html for unknown
// paths so client-side routing works on a hard refresh.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}
