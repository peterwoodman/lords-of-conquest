package client

import (
	"fmt"
	"log"
	"sort"

	"lords-of-conquest/internal/protocol"
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

	// Must have a build type selected first
	if s.selectedBuildType == "" {
		log.Printf("Select what to build first (City, Weapon, or Boat)")
		return
	}

	// Check if the territory belongs to us
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner != s.game.config.PlayerID {
			log.Printf("Cannot build on enemy territory")
			return
		}

		// For boats, check if we need to select a water body
		if s.selectedBuildType == "boat" {
			if waterBodies, ok := terr["waterBodies"].([]interface{}); ok && len(waterBodies) > 1 {
				// Multiple water bodies - show selection UI
				s.waterBodyOptions = make([]string, len(waterBodies))
				for i, wb := range waterBodies {
					s.waterBodyOptions[i] = wb.(string)
				}
				s.showWaterBodySelect = true
				s.buildMenuTerritory = territoryID
				return
			}
		}

		// Build immediately
		log.Printf("Building %s at %s (useGold: %v)", s.selectedBuildType, territoryID, s.buildUseGold)
		s.game.Build(s.selectedBuildType, territoryID, s.buildUseGold)
		// Deselect after building
		s.selectedBuildType = ""
	} else {
		log.Printf("Territory not found: %s", territoryID)
	}
}

// doBuildBoatInWater builds a boat in a specific water body
func (s *GameplayScene) doBuildBoatInWater(waterBodyID string) {
	if s.buildMenuTerritory == "" {
		return
	}

	// Use the gold toggle setting
	useGold := s.buildUseGold
	log.Printf("Building boat at %s in water body %s (useGold: %v)", s.buildMenuTerritory, waterBodyID, useGold)
	s.game.BuildBoatInWater(s.buildMenuTerritory, waterBodyID, useGold)
	s.showWaterBodySelect = false
	s.waterBodyOptions = nil
	s.buildMenuTerritory = ""
	// Deselect after building
	s.selectedBuildType = ""
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
// Note: Boats are NOT counted here - they must be "brought" as reinforcements
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

	// Boats: NOT counted in base attack strength
	// Boats must be selected as reinforcements to participate in combat

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

// ==================== Trade Functions ====================

// resetTradeForm resets the trade form to default values.
func (s *GameplayScene) resetTradeForm() {
	s.tradeTargetPlayer = ""
	s.tradeOfferCoal = 0
	s.tradeOfferGold = 0
	s.tradeOfferIron = 0
	s.tradeOfferTimber = 0
	s.tradeOfferHorses = 0
	s.tradeOfferHorseTerrs = nil
	s.tradeRequestCoal = 0
	s.tradeRequestGold = 0
	s.tradeRequestIron = 0
	s.tradeRequestTimber = 0
	s.tradeRequestHorses = 0
	s.tradeRequestHorseDestTerrs = nil
	s.tradeHorseDestTerrs = nil
	s.tradeHorseSourceTerrs = nil
}

// sendTradeOffer sends the trade offer to the server.
func (s *GameplayScene) sendTradeOffer() {
	if s.tradeTargetPlayer == "" {
		log.Println("No target player selected")
		return
	}

	// If offering horses and not enough selected, enter map selection mode
	if s.tradeOfferHorses > 0 && len(s.tradeOfferHorseTerrs) < s.tradeOfferHorses {
		s.showTradePropose = false
		s.pendingHorseSelection = "offer"
		s.pendingHorseCount = s.tradeOfferHorses
		s.tradeOfferHorseTerrs = nil // Reset selection
		return
	}

	// If requesting horses and not enough destinations selected, enter map selection mode
	if s.tradeRequestHorses > 0 && len(s.tradeRequestHorseDestTerrs) < s.tradeRequestHorses {
		s.showTradePropose = false
		s.pendingHorseSelection = "request"
		s.pendingHorseCount = s.tradeRequestHorses
		s.tradeRequestHorseDestTerrs = nil // Reset selection
		return
	}

	s.completeSendTradeOffer()
}

// completeSendTradeOffer actually sends the trade after horse selection.
func (s *GameplayScene) completeSendTradeOffer() {
	// Build offer horse territory list from selection
	offerHorseTerrs := make([]string, 0)
	for i := 0; i < s.tradeOfferHorses && i < len(s.tradeOfferHorseTerrs); i++ {
		offerHorseTerrs = append(offerHorseTerrs, s.tradeOfferHorseTerrs[i])
	}

	// Build request horse destination list from selection
	requestHorseDestTerrs := make([]string, 0)
	for i := 0; i < s.tradeRequestHorses && i < len(s.tradeRequestHorseDestTerrs); i++ {
		requestHorseDestTerrs = append(requestHorseDestTerrs, s.tradeRequestHorseDestTerrs[i])
	}

	log.Printf("Sending trade offer to %s", s.tradeTargetPlayer)
	s.game.ProposeTrade(
		s.tradeTargetPlayer,
		s.tradeOfferCoal, s.tradeOfferGold, s.tradeOfferIron, s.tradeOfferTimber,
		s.tradeOfferHorses, offerHorseTerrs,
		s.tradeRequestCoal, s.tradeRequestGold, s.tradeRequestIron, s.tradeRequestTimber,
		s.tradeRequestHorses, requestHorseDestTerrs,
	)
	s.showTradePropose = false
	s.pendingHorseSelection = ""
	s.pendingHorseCount = 0
	s.waitingForTrade = true
}

// acceptTrade accepts an incoming trade proposal.
func (s *GameplayScene) acceptTrade() {
	if s.tradeProposal == nil {
		return
	}

	// If receiving horses (OfferHorses) and not enough destinations selected, enter map selection mode
	if s.tradeProposal.OfferHorses > 0 && len(s.tradeHorseDestTerrs) < s.tradeProposal.OfferHorses {
		s.showTradeIncoming = false
		s.pendingHorseSelection = "receive"
		s.pendingHorseCount = s.tradeProposal.OfferHorses
		s.tradeHorseDestTerrs = nil // Reset selection
		return
	}

	// If giving horses (RequestHorses) and not enough sources selected, enter map selection mode
	if s.tradeProposal.RequestHorses > 0 && len(s.tradeHorseSourceTerrs) < s.tradeProposal.RequestHorses {
		s.showTradeIncoming = false
		s.pendingHorseSelection = "give"
		s.pendingHorseCount = s.tradeProposal.RequestHorses
		s.tradeHorseSourceTerrs = nil // Reset selection
		return
	}

	s.completeAcceptTrade()
}

// completeAcceptTrade actually accepts the trade after horse selection.
func (s *GameplayScene) completeAcceptTrade() {
	if s.tradeProposal == nil {
		return
	}

	// Build horse destination list (for OfferHorses - horses accepter receives)
	horseDests := make([]string, 0)
	for i := 0; i < s.tradeProposal.OfferHorses && i < len(s.tradeHorseDestTerrs); i++ {
		horseDests = append(horseDests, s.tradeHorseDestTerrs[i])
	}

	// Build horse source list (for RequestHorses - horses accepter gives)
	horseSources := make([]string, 0)
	for i := 0; i < s.tradeProposal.RequestHorses && i < len(s.tradeHorseSourceTerrs); i++ {
		horseSources = append(horseSources, s.tradeHorseSourceTerrs[i])
	}

	log.Printf("Accepting trade %s", s.tradeProposal.TradeID)
	s.game.RespondTrade(s.tradeProposal.TradeID, true, horseDests, horseSources)
	s.showTradeIncoming = false
	s.pendingHorseSelection = ""
	s.pendingHorseCount = 0
	s.tradeProposal = nil
}

// rejectTrade rejects an incoming trade proposal.
func (s *GameplayScene) rejectTrade() {
	if s.tradeProposal == nil {
		return
	}

	log.Printf("Rejecting trade %s", s.tradeProposal.TradeID)
	s.game.RespondTrade(s.tradeProposal.TradeID, false, nil, nil)
	s.showTradeIncoming = false
	s.tradeProposal = nil
}

// ShowTradeProposal shows an incoming trade proposal popup.
func (s *GameplayScene) ShowTradeProposal(payload *protocol.TradeProposalPayload) {
	s.tradeProposal = &TradeProposalData{
		TradeID:        payload.TradeID,
		FromPlayerID:   payload.FromPlayerID,
		FromPlayerName: payload.FromPlayerName,
		OfferCoal:      payload.OfferCoal,
		OfferGold:      payload.OfferGold,
		OfferIron:      payload.OfferIron,
		OfferTimber:    payload.OfferTimber,
		OfferHorses:    payload.OfferHorses,
		RequestCoal:    payload.RequestCoal,
		RequestGold:    payload.RequestGold,
		RequestIron:    payload.RequestIron,
		RequestTimber:  payload.RequestTimber,
		RequestHorses:  payload.RequestHorses,
	}
	s.showTradeIncoming = true
	s.tradeHorseDestTerrs = nil // Reset horse destinations
}

// ShowTradeResult shows the result of a trade proposal.
func (s *GameplayScene) ShowTradeResult(payload *protocol.TradeResultPayload) {
	s.tradeResultAccepted = payload.Accepted
	s.tradeResultMessage = payload.Message
	s.showTradeResult = true
	s.waitingForTrade = false // Clear waiting indicator
}

// getOnlinePlayers returns a list of online player IDs (excluding self and AI).
func (s *GameplayScene) getOnlinePlayers() []string {
	players := make([]string, 0)
	for id, pData := range s.players {
		if id == s.game.config.PlayerID {
			continue // Skip self
		}
		player := pData.(map[string]interface{})
		isAI, _ := player["isAI"].(bool)
		if isAI {
			continue // Skip AI
		}
		isOnline, _ := player["isOnline"].(bool)
		if !isOnline {
			continue // Skip offline
		}
		players = append(players, id)
	}
	sort.Strings(players) // Ensure consistent order for UI rendering
	return players
}

// getPlayerHorseTerritories returns territories where the player has horses.
func (s *GameplayScene) getPlayerHorseTerritories() []string {
	terrs := make([]string, 0)
	for id, tData := range s.territories {
		terr := tData.(map[string]interface{})
		owner, _ := terr["owner"].(string)
		if owner != s.game.config.PlayerID {
			continue
		}
		hasHorse, _ := terr["hasHorse"].(bool)
		if hasHorse {
			terrs = append(terrs, id)
		}
	}
	return terrs
}

// getMyStockpile returns the current player's stockpile resources.
func (s *GameplayScene) getMyStockpile() (coal, gold, iron, timber int) {
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if stockpile, ok := player["stockpile"].(map[string]interface{}); ok {
			if v, ok := stockpile["coal"].(float64); ok {
				coal = int(v)
			}
			if v, ok := stockpile["gold"].(float64); ok {
				gold = int(v)
			}
			if v, ok := stockpile["iron"].(float64); ok {
				iron = int(v)
			}
			if v, ok := stockpile["timber"].(float64); ok {
				timber = int(v)
			}
		}
	}
	return
}

// getPlayerStockpile returns a player's stockpile resources.
func (s *GameplayScene) getPlayerStockpile(playerID string) (coal, gold, iron, timber int) {
	if pData, ok := s.players[playerID]; ok {
		player := pData.(map[string]interface{})
		if stockpile, ok := player["stockpile"].(map[string]interface{}); ok {
			if v, ok := stockpile["coal"].(float64); ok {
				coal = int(v)
			}
			if v, ok := stockpile["gold"].(float64); ok {
				gold = int(v)
			}
			if v, ok := stockpile["iron"].(float64); ok {
				iron = int(v)
			}
			if v, ok := stockpile["timber"].(float64); ok {
				timber = int(v)
			}
		}
	}
	return
}

// countPlayerHorses returns the number of horses a player has.
func (s *GameplayScene) countPlayerHorses(playerID string) int {
	count := 0
	for _, tData := range s.territories {
		terr := tData.(map[string]interface{})
		owner, _ := terr["owner"].(string)
		if owner != playerID {
			continue
		}
		hasHorse, _ := terr["hasHorse"].(bool)
		if hasHorse {
			count++
		}
	}
	return count
}

// getTerritoriesWithoutHorses returns territories owned by the player that don't have horses.
func (s *GameplayScene) getTerritoriesWithoutHorses() []string {
	terrs := make([]string, 0)
	for id, tData := range s.territories {
		terr := tData.(map[string]interface{})
		owner, _ := terr["owner"].(string)
		if owner != s.game.config.PlayerID {
			continue
		}
		hasHorse, _ := terr["hasHorse"].(bool)
		if !hasHorse {
			terrs = append(terrs, id)
		}
	}
	return terrs
}

// getTerritoryAt returns the territory ID at the given grid coordinates, or empty string if none.
func (s *GameplayScene) getTerritoryAt(gx, gy int) string {
	if s.mapData == nil {
		return ""
	}
	grid, ok := s.mapData["grid"].([]interface{})
	if !ok {
		return ""
	}
	if gy < 0 || gy >= len(grid) {
		return ""
	}
	row, ok := grid[gy].([]interface{})
	if !ok {
		return ""
	}
	if gx < 0 || gx >= len(row) {
		return ""
	}
	terrNum, ok := row[gx].(float64)
	if !ok || terrNum <= 0 {
		return "" // Water or invalid
	}
	// Territory IDs in the game state have "t" prefix (e.g., "t77")
	return fmt.Sprintf("t%d", int(terrNum))
}

// handleOfferHorseClick handles clicking a territory when selecting horses to offer.
func (s *GameplayScene) handleOfferHorseClick(terrID string) {
	// Check if territory is owned by player and has a horse
	tData, ok := s.territories[terrID]
	if !ok {
		return
	}
	terr := tData.(map[string]interface{})
	owner, _ := terr["owner"].(string)
	if owner != s.game.config.PlayerID {
		return // Not our territory
	}
	hasHorse, _ := terr["hasHorse"].(bool)
	if !hasHorse {
		return // No horse here
	}

	// Toggle selection
	for i, t := range s.tradeOfferHorseTerrs {
		if t == terrID {
			// Remove from selection
			s.tradeOfferHorseTerrs = append(s.tradeOfferHorseTerrs[:i], s.tradeOfferHorseTerrs[i+1:]...)
			return
		}
	}

	// Add to selection if not at limit
	if len(s.tradeOfferHorseTerrs) < s.pendingHorseCount {
		s.tradeOfferHorseTerrs = append(s.tradeOfferHorseTerrs, terrID)
	}
}

// handleRequestHorseClick handles clicking a territory when selecting where to place requested horses.
// This is for the PROPOSER who is requesting horses from the accepter.
func (s *GameplayScene) handleRequestHorseClick(terrID string) {
	// Check if territory is owned by player and doesn't have a horse
	tData, ok := s.territories[terrID]
	if !ok {
		return
	}
	terr := tData.(map[string]interface{})
	owner, _ := terr["owner"].(string)
	if owner != s.game.config.PlayerID {
		return // Not our territory
	}
	hasHorse, _ := terr["hasHorse"].(bool)
	if hasHorse {
		return // Already has a horse
	}

	// Toggle selection
	for i, t := range s.tradeRequestHorseDestTerrs {
		if t == terrID {
			// Remove from selection
			s.tradeRequestHorseDestTerrs = append(s.tradeRequestHorseDestTerrs[:i], s.tradeRequestHorseDestTerrs[i+1:]...)
			return
		}
	}

	// Add to selection if not at limit
	if len(s.tradeRequestHorseDestTerrs) < s.pendingHorseCount {
		s.tradeRequestHorseDestTerrs = append(s.tradeRequestHorseDestTerrs, terrID)
	}
}

// handleReceiveHorseClick handles clicking a territory when selecting where to place offered horses.
// This is for the ACCEPTER who is receiving horses offered by the proposer.
func (s *GameplayScene) handleReceiveHorseClick(terrID string) {
	// Check if territory is owned by player and doesn't have a horse
	tData, ok := s.territories[terrID]
	if !ok {
		return
	}
	terr := tData.(map[string]interface{})
	owner, _ := terr["owner"].(string)
	if owner != s.game.config.PlayerID {
		return // Not our territory
	}
	hasHorse, _ := terr["hasHorse"].(bool)
	if hasHorse {
		return // Already has a horse
	}

	// Toggle selection
	for i, t := range s.tradeHorseDestTerrs {
		if t == terrID {
			// Remove from selection
			s.tradeHorseDestTerrs = append(s.tradeHorseDestTerrs[:i], s.tradeHorseDestTerrs[i+1:]...)
			return
		}
	}

	// Add to selection if not at limit
	if len(s.tradeHorseDestTerrs) < s.pendingHorseCount {
		s.tradeHorseDestTerrs = append(s.tradeHorseDestTerrs, terrID)
	}
}

// handleGiveHorseClick handles clicking a territory when selecting which horses to give.
// This is for the ACCEPTER who is giving horses to the proposer (RequestHorses).
func (s *GameplayScene) handleGiveHorseClick(terrID string) {
	// Check if territory is owned by player and HAS a horse
	tData, ok := s.territories[terrID]
	if !ok {
		return
	}
	terr := tData.(map[string]interface{})
	owner, _ := terr["owner"].(string)
	if owner != s.game.config.PlayerID {
		return // Not our territory
	}
	hasHorse, _ := terr["hasHorse"].(bool)
	if !hasHorse {
		return // No horse to give
	}

	// Toggle selection
	for i, t := range s.tradeHorseSourceTerrs {
		if t == terrID {
			// Remove from selection
			s.tradeHorseSourceTerrs = append(s.tradeHorseSourceTerrs[:i], s.tradeHorseSourceTerrs[i+1:]...)
			return
		}
	}

	// Add to selection if not at limit
	if len(s.tradeHorseSourceTerrs) < s.pendingHorseCount {
		s.tradeHorseSourceTerrs = append(s.tradeHorseSourceTerrs, terrID)
	}
}
