package main

import (
	"log"

	"lords-of-conquest/internal/client"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	game, err := client.NewGame()
	if err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}

	ebiten.SetWindowSize(client.ScreenWidth, client.ScreenHeight)
	ebiten.SetWindowTitle("Lords of Conquest")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
