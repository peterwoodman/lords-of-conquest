package client

import (
	"fmt"
	"log"
)

func (s *GameplayScene) handleCellClick(x, y int) {
	if s.mapData == nil {
		return
	}

	grid := s.mapData["grid"].([]interface{})
	row := grid[y].([]interface{})
	territoryID := int(row[x].(float64))

	if territoryID == 0 {
		return // Water
	}

	tid := fmt.Sprintf("t%d", territoryID)

	// Handle based on current phase
	switch s.currentPhase {
	case "Territory Selection":
		s.handleTerritorySelection(tid)
	case "Production":
		s.handleStockpilePlacement(tid)
	case "Trade":
		s.handleTrade(tid)
	case "Shipment":
		s.handleShipment(tid)
	case "Conquest":
		s.handleConquest(tid)
	case "Development":
		s.handleDevelopment(tid)
	default:
		log.Printf("No handler for phase: %s", s.currentPhase)
	}
}

func (s *GameplayScene) handleTerritorySelection(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == "" {
			// Unclaimed, we can select it
			s.game.SelectTerritory(territoryID)
		}
	}
}

func (s *GameplayScene) handleStockpilePlacement(territoryID string) {
	// Can only place on your own territories
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == s.game.config.PlayerID {
			// This is our territory, place stockpile here
			s.game.PlaceStockpile(territoryID)
			log.Printf("Placing stockpile at %s", territoryID)
		}
	}
}

func (s *GameplayScene) handleTrade(territoryID string) {
	// Trade phase - clicking on territories doesn't do anything special
	// Trade would require a separate UI dialog to propose/accept trades
	// For now, players can just click "End Turn" to skip
	log.Printf("Trade phase - trade UI not yet implemented. Click 'End Turn' to skip.")
}

func (s *GameplayScene) handleShipment(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	// If no mode selected yet, ignore clicks
	if s.shipmentMode == "" {
		return
	}

	// Check if we own this territory
	terr, ok := s.territories[territoryID].(map[string]interface{})
	if !ok {
		return
	}
	owner := terr["owner"].(string)
	if owner != s.game.config.PlayerID {
		return
	}

	// Handle based on current mode
	switch s.shipmentMode {
	case "stockpile":
		s.handleStockpileMove(territoryID)
	case "horse":
		s.handleHorseMove(territoryID, terr)
	case "boat":
		s.handleBoatMove(territoryID, terr)
	}
}

// startShipmentMode begins a specific type of shipment move.
func (s *GameplayScene) startShipmentMode(mode string) {
	s.shipmentMode = mode
	s.shipmentFromTerritory = ""
	s.selectedTerritory = ""
	s.shipmentCarryHorse = false
	s.shipmentCarryWeapon = false
	s.shipmentWaterBodyID = ""
	log.Printf("Started shipment mode: %s", mode)
}

// cancelShipmentMode cancels the current shipment selection.
func (s *GameplayScene) cancelShipmentMode() {
	s.shipmentMode = ""
	s.shipmentFromTerritory = ""
	s.selectedTerritory = ""
	s.shipmentCarryHorse = false
	s.shipmentCarryWeapon = false
	s.shipmentWaterBodyID = ""
}

// handleStockpileMove handles stockpile movement selection.
func (s *GameplayScene) handleStockpileMove(tid string) {
	// Get player's stockpile location
	myPlayer, ok := s.players[s.game.config.PlayerID]
	if !ok {
		return
	}
	player := myPlayer.(map[string]interface{})
	stockpileTerr, hasStockpile := player["stockpileTerritory"]
	if !hasStockpile || stockpileTerr == nil || stockpileTerr == "" {
		log.Printf("No stockpile to move")
		return
	}

	// Set destination directly (stockpile can move to any connected territory)
	s.selectedTerritory = tid
}

// handleHorseMove handles horse movement selection.
func (s *GameplayScene) handleHorseMove(tid string, terr map[string]interface{}) {
	if s.shipmentFromTerritory == "" {
		// First click - select source territory with horse
		hasHorse, _ := terr["hasHorse"].(bool)
		if !hasHorse {
			log.Printf("No horse in %s", tid)
			return
		}
		s.shipmentFromTerritory = tid
		// Check if we can carry weapon
		hasWeapon, _ := terr["hasWeapon"].(bool)
		if hasWeapon {
			s.shipmentCarryWeapon = true // Default to carrying if available
		}
		log.Printf("Selected horse from %s", tid)
	} else {
		// Second click - select destination
		s.selectedTerritory = tid
	}
}

// handleBoatMove handles boat movement selection.
func (s *GameplayScene) handleBoatMove(tid string, terr map[string]interface{}) {
	if s.shipmentFromTerritory == "" {
		// First click - select source territory with boat
		totalBoats, _ := terr["totalBoats"].(float64)
		if totalBoats == 0 {
			log.Printf("No boats in %s", tid)
			return
		}
		s.shipmentFromTerritory = tid

		// Get the water body ID for the boat
		boats, ok := terr["boats"].(map[string]interface{})
		if ok {
			for waterID, count := range boats {
				if c, _ := count.(float64); c > 0 {
					s.shipmentWaterBodyID = waterID
					break
				}
			}
		}

		// Check cargo options
		hasHorse, _ := terr["hasHorse"].(bool)
		hasWeapon, _ := terr["hasWeapon"].(bool)
		s.shipmentCarryHorse = hasHorse
		s.shipmentCarryWeapon = hasWeapon
		log.Printf("Selected boat from %s (water body: %s)", tid, s.shipmentWaterBodyID)
	} else {
		// Second click - select destination
		s.selectedTerritory = tid
	}
}

// confirmShipment executes the selected shipment move.
func (s *GameplayScene) confirmShipment() {
	if s.selectedTerritory == "" {
		log.Printf("No destination selected")
		return
	}

	switch s.shipmentMode {
	case "stockpile":
		log.Printf("Moving stockpile to %s", s.selectedTerritory)
		s.game.MoveStockpile(s.selectedTerritory)

	case "horse":
		if s.shipmentFromTerritory == "" {
			log.Printf("No source territory selected")
			return
		}
		log.Printf("Moving horse from %s to %s (carry weapon: %v)",
			s.shipmentFromTerritory, s.selectedTerritory, s.shipmentCarryWeapon)
		s.game.MoveUnit("horse", s.shipmentFromTerritory, s.selectedTerritory,
			"", false, s.shipmentCarryWeapon)

	case "boat":
		if s.shipmentFromTerritory == "" {
			log.Printf("No source territory selected")
			return
		}
		log.Printf("Moving boat from %s to %s (water: %s, carry horse: %v, weapon: %v)",
			s.shipmentFromTerritory, s.selectedTerritory, s.shipmentWaterBodyID,
			s.shipmentCarryHorse, s.shipmentCarryWeapon)
		s.game.MoveUnit("boat", s.shipmentFromTerritory, s.selectedTerritory,
			s.shipmentWaterBodyID, s.shipmentCarryHorse, s.shipmentCarryWeapon)
	}

	// Reset shipment state
	s.shipmentMode = ""
	s.shipmentFromTerritory = ""
	s.selectedTerritory = ""
	s.shipmentCarryHorse = false
	s.shipmentCarryWeapon = false
	s.shipmentWaterBodyID = ""
}

func (s *GameplayScene) handleConquest(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	// Check if the territory can be attacked (enemy or unclaimed)
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == s.game.config.PlayerID {
			log.Printf("Cannot attack your own territory")
		} else {
			// Enemy or unclaimed territory - request attack preview
			log.Printf("Planning attack on territory %s", territoryID)
			s.game.PlanAttack(territoryID)
		}
	}
}

func (s *GameplayScene) handleDevelopment(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	// Check if the territory belongs to us
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == s.game.config.PlayerID {
			// Our territory - show build menu
			s.buildMenuTerritory = territoryID
			s.showBuildMenu = true
			log.Printf("Opened build menu for %s", territoryID)
		} else {
			log.Printf("Cannot build on enemy territory")
		}
	}
}

// doBuild executes a build action
func (s *GameplayScene) doBuild(buildType string) {
	if s.buildMenuTerritory == "" {
		return
	}

	// For boats, check if we need to select a water body
	if buildType == "boat" {
		if terr, ok := s.territories[s.buildMenuTerritory].(map[string]interface{}); ok {
			if waterBodies, ok := terr["waterBodies"].([]interface{}); ok && len(waterBodies) > 1 {
				// Multiple water bodies - show selection UI
				s.waterBodyOptions = make([]string, len(waterBodies))
				for i, wb := range waterBodies {
					s.waterBodyOptions[i] = wb.(string)
				}
				s.showWaterBodySelect = true
				s.showBuildMenu = false
				return
			}
		}
	}

	// Determine if we should use gold (if we can't afford regular cost)
	useGold := s.shouldUseGold(buildType)

	log.Printf("Building %s at %s (useGold: %v)", buildType, s.buildMenuTerritory, useGold)
	s.game.Build(buildType, s.buildMenuTerritory, useGold)
	s.showBuildMenu = false
	s.buildMenuTerritory = ""
}

// shouldUseGold determines if gold should be used for a build
func (s *GameplayScene) shouldUseGold(buildType string) bool {
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if stockpile, ok := player["stockpile"].(map[string]interface{}); ok {
			coal, iron, timber := 0, 0, 0
			if v, ok := stockpile["coal"].(float64); ok {
				coal = int(v)
			}
			if v, ok := stockpile["iron"].(float64); ok {
				iron = int(v)
			}
			if v, ok := stockpile["timber"].(float64); ok {
				timber = int(v)
			}

			switch buildType {
			case "city":
				gold := 0
				if v, ok := stockpile["gold"].(float64); ok {
					gold = int(v)
				}
				if !(coal >= 1 && gold >= 1 && iron >= 1 && timber >= 1) {
					return true
				}
			case "weapon":
				if !(coal >= 1 && iron >= 1) {
					return true
				}
			case "boat":
				if timber < 3 {
					return true
				}
			}
		}
	}
	return false
}

// doBuildBoatInWater builds a boat in a specific water body
func (s *GameplayScene) doBuildBoatInWater(waterBodyID string) {
	if s.buildMenuTerritory == "" {
		return
	}

	useGold := s.shouldUseGold("boat")
	log.Printf("Building boat at %s in water body %s (useGold: %v)", s.buildMenuTerritory, waterBodyID, useGold)
	s.game.BuildBoatInWater(s.buildMenuTerritory, waterBodyID, useGold)
	s.showWaterBodySelect = false
	s.waterBodyOptions = nil
	s.buildMenuTerritory = ""
}

// handleWaterBodyClick handles clicking on a water cell during water body selection
func (s *GameplayScene) handleWaterBodyClick(cellX, cellY int) {
	if s.buildMenuTerritory == "" || len(s.waterBodyOptions) == 0 {
		return
	}

	grid := s.mapData["grid"].([]interface{})
	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// Check if clicked cell is water
	if cellX < 0 || cellX >= width || cellY < 0 || cellY >= height {
		return
	}
	row := grid[cellY].([]interface{})
	if int(row[cellX].(float64)) != 0 {
		return // Not water
	}

	waterBodies, hasWaterBodies := s.mapData["waterBodies"].(map[string]interface{})
	if !hasWaterBodies {
		return
	}

	// Extract numeric territory ID
	var numTerritoryID int
	if len(s.buildMenuTerritory) > 1 && s.buildMenuTerritory[0] == 't' {
		fmt.Sscanf(s.buildMenuTerritory[1:], "%d", &numTerritoryID)
	}

	// Check if this cell is in one of our water body options and adjacent to territory
	for _, waterBodyID := range s.waterBodyOptions {
		wbData, ok := waterBodies[waterBodyID].(map[string]interface{})
		if !ok {
			continue
		}

		wbCells, ok := wbData["cells"].([]interface{})
		if !ok {
			continue
		}

		// Check if clicked cell is in this water body
		for _, cellData := range wbCells {
			cell, ok := cellData.([]interface{})
			if !ok || len(cell) < 2 {
				continue
			}
			wx := int(cell[0].(float64))
			wy := int(cell[1].(float64))

			if wx != cellX || wy != cellY {
				continue
			}

			// Check if adjacent to territory
			dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
			for _, d := range dirs {
				nx, ny := wx+d[0], wy+d[1]
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					nrow := grid[ny].([]interface{})
					if int(nrow[nx].(float64)) == numTerritoryID {
						// Found it! Build boat in this water body
						s.doBuildBoatInWater(waterBodyID)
						return
					}
				}
			}
		}
	}
}

// calculateCombatStrength estimates attack and defense strength for a territory
func (s *GameplayScene) calculateCombatStrength(targetTID string) (attack, defense int) {
	target, ok := s.territories[targetTID].(map[string]interface{})
	if !ok {
		return 0, 0
	}

	targetOwner := target["owner"].(string)
	myID := s.game.config.PlayerID

	// Defense: 1 for the territory itself
	defense = 1

	// Add target's buildings and units
	defense += s.getTerritoryStrength(target)

	// Attack: count our adjacent territories
	for _, terrData := range s.territories {
		terr := terrData.(map[string]interface{})
		if terr["owner"].(string) != myID {
			continue
		}

		// Check if this territory is adjacent to target
		terrID := terr["id"].(string)
		if s.isAdjacent(terrID, targetTID) {
			attack++ // Territory contribution
			attack += s.getTerritoryStrength(terr)
		}
	}

	// Count defender's adjacent territories (only if territory has an owner)
	// Unclaimed territories don't get reinforcements from other unclaimed territories
	if targetOwner == "" {
		return attack, defense
	}
	for _, terrData := range s.territories {
		terr := terrData.(map[string]interface{})
		if terr["owner"].(string) != targetOwner {
			continue
		}
		terrID := terr["id"].(string)
		if terrID != targetTID && s.isAdjacent(terrID, targetTID) {
			defense++ // Adjacent territory contribution
			defense += s.getTerritoryStrength(terr)
		}
	}

	return attack, defense
}

// getTerritoryStrength returns the combat strength bonus from a territory's buildings and units
func (s *GameplayScene) getTerritoryStrength(terr map[string]interface{}) int {
	strength := 0

	// City: +2
	if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
		strength += 2
	}

	// Weapon: +3
	if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
		strength += 3
	}

	// Horse: +1
	if hasHorse, ok := terr["hasHorse"].(bool); ok && hasHorse {
		strength += 1
	}

	// Boats: +2 each (using totalBoats)
	if totalBoats, ok := terr["totalBoats"].(float64); ok && int(totalBoats) > 0 {
		strength += int(totalBoats) * 2
	}

	return strength
}

// isAdjacent checks if two territories are adjacent (simplified check based on grid proximity)
func (s *GameplayScene) isAdjacent(tid1, tid2 string) bool {
	// Get centers of both territories to verify they exist
	grid := s.mapData["grid"].([]interface{})
	c1x, _ := s.findTerritoryCenter(tid1, grid)
	c2x, _ := s.findTerritoryCenter(tid2, grid)

	if c1x < 0 || c2x < 0 {
		return false
	}

	// Check if any cell of territory 1 is adjacent to any cell of territory 2
	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// Extract numeric IDs
	var num1, num2 int
	fmt.Sscanf(tid1[1:], "%d", &num1)
	fmt.Sscanf(tid2[1:], "%d", &num2)

	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			if int(row[x].(float64)) != num1 {
				continue
			}
			// Check neighbors
			neighbors := [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}}
			for _, n := range neighbors {
				nx, ny := n[0], n[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				neighborRow := grid[ny].([]interface{})
				if int(neighborRow[nx].(float64)) == num2 {
					return true
				}
			}
		}
	}
	return false
}
