package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Hoega/gostock/internal/persistence"
	"github.com/Hoega/gostock/internal/server"
)

func main() {
	defaultPort := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			defaultPort = v
		}
	}

	port := flag.Int("port", defaultPort, "HTTP server port")
	flag.Parse()

	// Initialize persistence store
	store, err := persistence.NewSQLiteStore()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	srv := server.Start(*port, store)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := store.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
	log.Println("Server stopped")
}
