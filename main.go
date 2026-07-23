package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/handlers"
	"url-shortener/internal/middleware"
	"url-shortener/internal/store"
)

func main() {
	cfg := config.Load()

	s := store.NewStore()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if cfg.CleanupInterval > 0 {
		s.StartCleanup(ctx, cfg.CleanupInterval)
	}

	if cfg.APIKey == "" {
		log.Println("WARNING: API key not set, /shorten and /stats are open.")
	}

	shortenHandler := &handlers.ShortenHandler{
		Store: 					s,
		BaseURL: 				cfg.BaseURL,
		CodeLength: 			cfg.CodeLength,
		MaxCollisionRetries: 	cfg.MaxCollisionRetries,
		DefaultTTL: 			cfg.DefaultTTL,
		MaxClicksPerURL: 		cfg.MaxClicksPerURL,
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", &handlers.HealthHandler{Store: s})
	mux.Handle("POST /shorten", middleware.RequireAPIKey(cfg.APIKey, shortenHandler))
	mux.Handle("GET /r/{code}", &handlers.RedirectHandler{Store: s})
	mux.Handle("GET /stats/{code}", middleware.RequireAPIKey(cfg.APIKey, &handlers.StatsHandler{Store: s}))
	mux.Handle("DELETE /links/{code}", middleware.RequireAPIKey(cfg.APIKey, &handlers.DeleteHandler{Store: s}))


	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      withCORS(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("URL Shorterner listening on :%s (base_url=%s)", cfg.Port, cfg.BaseURL)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

}


func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}