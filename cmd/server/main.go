package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lords-of-conquest/internal/server"
)

func main() {
	port := flag.String("port", "30000", "Server port")
	dbPath := flag.String("db", "data/lords.db", "Database path")
	flag.Parse()

	// Use PORT env var if set (required for Render.com and similar platforms)
	actualPort := *port
	if envPort := os.Getenv("PORT"); envPort != "" {
		actualPort = envPort
		log.Printf("Using PORT from environment: %s", actualPort)
	}

	// Use DB_PATH env var if set, for cloud deployments with persistent disks
	actualDBPath := *dbPath
	if envDBPath := os.Getenv("DB_PATH"); envDBPath != "" {
		actualDBPath = envDBPath
		log.Printf("Using DB_PATH from environment: %s", actualDBPath)
	}

	cfg := server.Config{
		Addr:   ":" + actualPort,
		DBPath: actualDBPath,
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle shutdown gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Printf("Lords of Conquest Server running on %s", cfg.Addr)
	log.Printf("Database: %s", cfg.DBPath)

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
