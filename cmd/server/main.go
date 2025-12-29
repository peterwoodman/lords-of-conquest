package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lords-of-conquest/internal/server"
	"lords-of-conquest/pkg/maps"
)

func main() {
	port := flag.String("port", "8080", "Server port")
	dbPath := flag.String("db", "data/lords.db", "Database path")
	flag.Parse()

	// Load all maps
	if err := maps.LoadAll(); err != nil {
		log.Fatalf("Failed to load maps: %v", err)
	}
	log.Printf("Loaded %d maps", len(maps.Registry))
	for _, info := range maps.List() {
		m := maps.Get(info.ID)
		log.Printf("  - %s: %dx%d, %d territories, %d water bodies",
			info.Name, info.Width, info.Height, info.TerritoryCount, len(m.WaterBodies))

		// Debug output in verbose mode
		if *port == "debug" {
			fmt.Println(m.Debug())
			fmt.Println(m.PrintAdjacencyMatrix())
		}
	}

	cfg := server.Config{
		Addr:   ":" + *port,
		DBPath: *dbPath,
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
