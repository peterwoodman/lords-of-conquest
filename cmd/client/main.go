package main

import (
	"flag"
	"log"

	"lords-of-conquest/internal/client"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	profile := flag.String("profile", "", "Profile name for separate config (e.g., player1, player2)")
	flag.Parse()

	client.SetProfile(*profile)

	game, err := client.NewGame()
	if err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}

	// Restore window size from config, or use default
	cfg := game.GetConfig()
	if cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		ebiten.SetWindowSize(cfg.WindowWidth, cfg.WindowHeight)
	} else {
		ebiten.SetWindowSize(client.ScreenWidth, client.ScreenHeight)
	}

	// Restore window position from config
	if cfg.WindowWidth > 0 {
		ebiten.SetWindowPosition(cfg.WindowX, cfg.WindowY)
	}

	ebiten.SetWindowTitle("Lords of Conquest")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	// Save window geometry before exiting
	game.Cleanup()
}
