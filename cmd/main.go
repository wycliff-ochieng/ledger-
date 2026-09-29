package main

import (
	"Wycliff-Ochieng/internal/handlers"
	"Wycliff-Ochieng/internal/repository"
	"Wycliff-Ochieng/internal/service"
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/double_ledger?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	repo := repository.NewPostgresRepo(pool)
	svc := service.NewLedgerService(repo)
	handler := handlers.NewLedgerHandler(svc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Serve the static frontend
	staticDir := filepath.Join(".", "static")
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	serverHandler := corsMiddleware(mux)

	log.Println("Starting Ledger Service on :8080")
	if err := http.ListenAndServe(":8080", serverHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
