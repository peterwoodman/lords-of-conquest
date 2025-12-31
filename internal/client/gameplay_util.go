package client

import "log"

// gridToScreen converts grid coordinates to screen coordinates
func (s *GameplayScene) gridToScreen(gridX, gridY int) (int, int) {
	return s.offsetX + gridX*s.cellSize, s.offsetY + gridY*s.cellSize
}

// screenToGrid converts screen coordinates to grid coordinates
func (s *GameplayScene) screenToGrid(screenX, screenY int) [2]int {
	if s.mapData == nil {
		return [2]int{-1, -1}
	}

	gridX := (screenX - s.offsetX) / s.cellSize
	gridY := (screenY - s.offsetY) / s.cellSize

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	if gridX < 0 || gridX >= width || gridY < 0 || gridY >= height {
		return [2]int{-1, -1}
	}

	return [2]int{gridX, gridY}
}

// SetGameState updates the game state from the server.
func (s *GameplayScene) SetGameState(state map[string]interface{}) {
	log.Println("GameplayScene.SetGameState called")
	s.gameState = state

	if mapData, ok := state["map"].(map[string]interface{}); ok {
		s.mapData = mapData
		log.Printf("Map data loaded: %dx%d", int(mapData["width"].(float64)), int(mapData["height"].(float64)))
	} else {
		log.Printf("No map data in state, keys: %v", getKeys(state))
	}

	if territories, ok := state["territories"].(map[string]interface{}); ok {
		s.territories = territories
		log.Printf("Loaded %d territories", len(territories))
	} else {
		log.Println("No territories in state")
	}

	if players, ok := state["players"].(map[string]interface{}); ok {
		s.players = players
		log.Printf("Loaded %d players", len(players))
	} else {
		log.Println("No players in state")
	}

	if playerOrder, ok := state["playerOrder"].([]interface{}); ok {
		s.playerOrder = playerOrder
		log.Printf("Loaded player order: %d players", len(playerOrder))
	} else {
		log.Println("No player order in state")
	}

	if phase, ok := state["phase"].(string); ok {
		// Clear selection when phase changes
		if s.currentPhase != phase {
			s.selectedTerritory = ""
		}
		s.currentPhase = phase
		log.Printf("Phase: %s", phase)
	}

	if turn, ok := state["currentPlayerId"].(string); ok {
		s.currentTurn = turn
		log.Printf("Current turn: %s", turn)
	}

	if round, ok := state["round"].(float64); ok {
		s.round = int(round)
		log.Printf("Round: %d", s.round)
	}

	// Update our alliance setting from player data
	if s.players != nil {
		if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
			player := myPlayer.(map[string]interface{})
			if alliance, ok := player["alliance"].(string); ok {
				s.myAllianceSetting = alliance
			}
		}
	}
}

// getKeys returns the keys of a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// min returns the smaller of two uint8 values
func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}
