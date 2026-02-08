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
	s.applyGameState(state)
}

// applyGameState actually applies the game state (called directly or after animation)
func (s *GameplayScene) applyGameState(state map[string]interface{}) {
	log.Println("GameplayScene.applyGameState called")
	s.gameState = state

	if mapData, ok := state["map"].(map[string]interface{}); ok {
		s.mapData = mapData
		log.Printf("Map data loaded: %dx%d", int(mapData["width"].(float64)), int(mapData["height"].(float64)))
	} else {
		log.Printf("No map data in state, keys: %v", getKeys(state))
	}

	if territories, ok := state["territories"].(map[string]interface{}); ok {
		s.territories = territories
		s.missingTerritories = nil // Reset for new state
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
		// Clear selection and close menus when phase changes
		if s.currentPhase != phase {
			s.selectedTerritory = ""
			s.shipmentMode = ""
			s.shipmentFromTerritory = ""
			s.shipmentCarryHorse = false
			s.shipmentCarryWeapon = false
			s.shipmentWaterBodyID = ""

			// Reset development phase UI state when entering Development phase
			if phase == "Development" {
				s.buildUseGold = false
				s.selectedBuildType = ""
			}
		}
		s.currentPhase = phase
		log.Printf("Phase: %s", phase)
	}

	if turn, ok := state["currentPlayerId"].(string); ok {
		// Reset shipment state when turn changes
		if s.currentTurn != turn {
			s.shipmentMode = ""
			s.shipmentFromTerritory = ""

			// Show turn toast if turn changed TO us (not on initial load)
			if turn == s.game.config.PlayerID && !s.initialTurnLoad {
				s.showTurnToast = true
				s.turnToastTimer = 0
				s.turnToastPhase = "slide-in"
				log.Println("Turn toast: Your turn!")
			}
		}
		s.currentTurn = turn
		log.Printf("Current turn: %s", turn)
	}

	// After first state load, clear the initial load flag
	if s.initialTurnLoad {
		s.initialTurnLoad = false
	}

	if round, ok := state["round"].(float64); ok {
		newRound := int(round)
		// Play bridge sound when round changes (but not on initial load where s.round is 0)
		if s.round > 0 && newRound > s.round {
			PlayBridgeSound()
		}
		s.round = newRound
		log.Printf("Round: %d", s.round)
	}

	// Update our alliance setting and card hand from player data
	if s.players != nil {
		if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
			player := myPlayer.(map[string]interface{})
			if alliance, ok := player["alliance"].(string); ok {
				s.myAllianceSetting = alliance
			}
			// Parse card hands
			s.myAttackCards = parseCardList(player, "attackCards")
			s.myDefenseCards = parseCardList(player, "defenseCards")
		}
	}

	// Parse combat mode from settings
	if settings, ok := state["settings"].(map[string]interface{}); ok {
		if cm, ok := settings["combatMode"].(float64); ok {
			if int(cm) == 1 {
				s.combatMode = "cards"
			} else {
				s.combatMode = "classic"
			}
		}
	}
}

// parseCardList extracts cards from player data.
func parseCardList(player map[string]interface{}, key string) []CardDisplayInfo {
	cards := make([]CardDisplayInfo, 0)
	if cardList, ok := player[key].([]interface{}); ok {
		for _, c := range cardList {
			if cardMap, ok := c.(map[string]interface{}); ok {
				card := CardDisplayInfo{}
				if v, ok := cardMap["id"].(string); ok {
					card.ID = v
				}
				if v, ok := cardMap["name"].(string); ok {
					card.Name = v
				}
				if v, ok := cardMap["description"].(string); ok {
					card.Description = v
				}
				if v, ok := cardMap["cardType"].(string); ok {
					card.CardType = v
				}
				if v, ok := cardMap["rarity"].(string); ok {
					card.Rarity = v
				}
				if v, ok := cardMap["effect"].(string); ok {
					card.Effect = v
				}
				if v, ok := cardMap["value"].(float64); ok {
					card.Value = int(v)
				}
				cards = append(cards, card)
			}
		}
	}
	return cards
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

// UpdateTerritoryDrawing updates the drawing data for a specific territory from a server broadcast.
func (s *GameplayScene) UpdateTerritoryDrawing(territoryID string, drawing map[string]int) {
	if s.territories == nil {
		return
	}
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		if len(drawing) == 0 {
			delete(terr, "drawing")
		} else {
			// Convert map[string]int to map[string]interface{} for the JSON-parsed territory data
			drawingIface := make(map[string]interface{})
			for k, v := range drawing {
				drawingIface[k] = float64(v)
			}
			terr["drawing"] = drawingIface
		}
	}
}
