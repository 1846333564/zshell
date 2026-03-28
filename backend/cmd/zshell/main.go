package main

import (
	"log"
	"net/http"
	"time"

	"zshell/backend/internal/httpapi"
	"zshell/backend/internal/store"
)

func main() {
	connectionStore := store.NewMemoryStore()
	apiServer := httpapi.NewServer(connectionStore, 10*time.Second)

	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      httpapi.WithCORS(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("zShell backend listening on http://127.0.0.1:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
