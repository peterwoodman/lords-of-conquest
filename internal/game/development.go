package game

import "log"

// BuildType represents what can be built.
type BuildType string

const (
	BuildCity   BuildType = "city"
	BuildWeapon BuildType = "weapon"
	BuildBoat   BuildType = "boat"
)

// GetBuildCost returns the resources needed to build something.
func GetBuildCost(buildType BuildType) *Stockpile {
	switch buildType {
	case BuildCity:
		// City costs: 1 Coal + 1 Gold + 1 Iron + 1 Timber
		return &Stockpile{Coal: 1, Gold: 1, Iron: 1, Timber: 1}
	case BuildWeapon:
		// Weapon costs: 1 Coal + 1 Iron
		return &Stockpile{Coal: 1, Iron: 1}
	case BuildBoat:
		// Boat costs: 3 Timber
		return &Stockpile{Timber: 3}
	default:
		return nil
	}
}

// GoldCost returns the gold-only cost for building.
func GoldCost(buildType BuildType) int {
	switch buildType {
	case BuildCity:
		return 4
	case BuildWeapon:
		return 2
	case BuildBoat:
		return 3
	default:
		return 0
	}
}

// CanBuild checks if a player can build something at a territory.
func (g *GameState) CanBuild(playerID string, buildType BuildType, territoryID string, useGold bool) error {
	if g.Phase != PhaseDevelopment {
		return ErrInvalidAction
	}

	player := g.Players[playerID]
	if player == nil {
		return ErrInvalidTarget
	}

	territory := g.Territories[territoryID]
	if territory == nil || territory.Owner != playerID {
		return ErrInvalidTarget
	}

	// Check game level restrictions
	switch buildType {
	case BuildBoat:
		if g.Settings.GameLevel < LevelAdvanced {
			return ErrInvalidAction
		}
		if !territory.IsCoastal() || !territory.CanAddBoat() {
			return ErrInvalidTarget
		}
	case BuildWeapon:
		if territory.HasWeapon {
			return ErrAlreadyHasUnit
		}
	case BuildCity:
		if territory.HasCity {
			return ErrAlreadyHasUnit
		}
	}

	// Check resources
	if useGold {
		goldNeeded := GoldCost(buildType)
		if player.Stockpile.Gold < goldNeeded {
			return ErrInsufficientResources
		}
	} else {
		cost := GetBuildCost(buildType)
		if !player.Stockpile.CanAffordStockpile(cost) {
			return ErrInsufficientResources
		}
	}

	return nil
}

// Build constructs a unit or city.
// For boats, use BuildBoatInWater instead to specify the water body.
func (g *GameState) Build(playerID string, buildType BuildType, territoryID string, useGold bool) error {
	// For boats, require water body specification if multiple options exist
	if buildType == BuildBoat {
		territory := g.Territories[territoryID]
		if territory != nil && len(territory.WaterBodies) > 1 {
			return ErrInvalidAction // Must use BuildBoatInWater
		}
		// If only one water body, auto-select it
		if territory != nil && len(territory.WaterBodies) == 1 {
			return g.BuildBoatInWater(playerID, territoryID, territory.WaterBodies[0], useGold)
		}
	}

	// Validate phase
	if g.Phase != PhaseDevelopment {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	// Check if can build
	if err := g.CanBuild(playerID, buildType, territoryID, useGold); err != nil {
		return err
	}

	player := g.Players[playerID]
	territory := g.Territories[territoryID]

	// Deduct resources
	if useGold {
		player.Stockpile.Gold -= GoldCost(buildType)
	} else {
		cost := GetBuildCost(buildType)
		player.Stockpile.Subtract(cost)
	}

	// Build the item
	switch buildType {
	case BuildCity:
		territory.HasCity = true
	case BuildWeapon:
		territory.HasWeapon = true
	case BuildBoat:
		// Should have been handled above, but fallback to first water body
		if len(territory.WaterBodies) > 0 {
			territory.AddBoat(territory.WaterBodies[0])
		}
	}

	return nil
}

// BuildBoatInWater builds a boat in a specific water body.
func (g *GameState) BuildBoatInWater(playerID string, territoryID string, waterBodyID string, useGold bool) error {
	// Validate phase
	if g.Phase != PhaseDevelopment {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	// Check if can build
	if err := g.CanBuild(playerID, BuildBoat, territoryID, useGold); err != nil {
		return err
	}

	player := g.Players[playerID]
	territory := g.Territories[territoryID]

	// Validate water body is adjacent to territory
	if !territory.CanAddBoatToWater(waterBodyID) {
		return ErrInvalidTarget
	}

	// Deduct resources
	if useGold {
		player.Stockpile.Gold -= GoldCost(BuildBoat)
	} else {
		cost := GetBuildCost(BuildBoat)
		player.Stockpile.Subtract(cost)
	}

	// Build the boat
	territory.AddBoat(waterBodyID)

	return nil
}

// GetWaterBodiesForBoat returns water bodies where a boat can be built at a territory.
func (g *GameState) GetWaterBodiesForBoat(territoryID string) []string {
	territory := g.Territories[territoryID]
	if territory == nil || !territory.IsCoastal() || !territory.CanAddBoat() {
		return nil
	}

	return territory.WaterBodies
}

// EndDevelopment ends the development phase for a player.
func (g *GameState) EndDevelopment(playerID string) error {
	if g.Phase != PhaseDevelopment {
		return ErrInvalidAction
	}

	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	g.advanceDevelopmentTurn()
	return nil
}

// advanceDevelopmentTurn moves to the next player or next round.
func (g *GameState) advanceDevelopmentTurn() {
	log.Printf("advanceDevelopmentTurn: Current player %s in round %d", g.CurrentPlayerID, g.Round)
	
	currentIdx := -1
	for i, pid := range g.PlayerOrder {
		if pid == g.CurrentPlayerID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 || len(g.PlayerOrder) == 0 {
		log.Printf("advanceDevelopmentTurn: Invalid state - currentIdx=%d, playerOrder len=%d", currentIdx, len(g.PlayerOrder))
		return
	}

	nextIdx := (currentIdx + 1) % len(g.PlayerOrder)
	log.Printf("advanceDevelopmentTurn: currentIdx=%d, initial nextIdx=%d, playerOrder=%v", currentIdx, nextIdx, g.PlayerOrder)

	// Skip eliminated players (with safety counter to prevent infinite loop)
	iterations := 0
	for g.Players[g.PlayerOrder[nextIdx]] != nil && g.Players[g.PlayerOrder[nextIdx]].Eliminated {
		nextIdx = (nextIdx + 1) % len(g.PlayerOrder)
		iterations++
		if nextIdx == currentIdx || iterations >= len(g.PlayerOrder) {
			// All other players eliminated - game should be over
			log.Printf("advanceDevelopmentTurn: All other players eliminated")
			return
		}
	}

	log.Printf("advanceDevelopmentTurn: After skipping eliminated, nextIdx=%d", nextIdx)

	// Check if we've completed all players (wrapped around to start)
	if nextIdx <= currentIdx {
		// End of round - start new round
		log.Printf("advanceDevelopmentTurn: Wrapped around (nextIdx %d <= currentIdx %d), starting new round", nextIdx, currentIdx)
		g.startNewRound()
	} else {
		log.Printf("advanceDevelopmentTurn: Moving to next player %s", g.PlayerOrder[nextIdx])
		g.CurrentPlayerID = g.PlayerOrder[nextIdx]
	}
}

// startNewRound begins a new round with production phase.
func (g *GameState) startNewRound() {
	g.Round++
	log.Printf("startNewRound: Beginning round %d", g.Round)

	// Clear any previous skipped phases
	g.SkippedPhases = nil

	// Shuffle player order
	shufflePlayerOrder(g)

	// Reset player turns
	for _, p := range g.Players {
		p.ResetTurn()
	}

	// Check for phase skip
	if ShouldSkipPhase(PhaseProduction, g.Settings.ChanceLevel) {
		// Record the skip
		g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
			Phase:  PhaseProduction,
			Reason: GetSkipReason(PhaseProduction),
		})
		log.Printf("startNewRound: Skipping production phase - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)

		if len(g.Players) >= 3 {
			// Check if trade should also be skipped
			if ShouldSkipPhase(PhaseTrade, g.Settings.ChanceLevel) {
				g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
					Phase:  PhaseTrade,
					Reason: GetSkipReason(PhaseTrade),
				})
				log.Printf("startNewRound: Skipping trade phase - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)
				g.Phase = PhaseShipment
				// Check shipment skip too
				if ShouldSkipPhase(PhaseShipment, g.Settings.ChanceLevel) {
					g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
						Phase:  PhaseShipment,
						Reason: GetSkipReason(PhaseShipment),
					})
					log.Printf("startNewRound: Skipping shipment phase - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)
					g.Phase = PhaseConquest
				}
			} else {
				g.Phase = PhaseTrade
			}
		} else {
			g.Phase = PhaseShipment
			if ShouldSkipPhase(PhaseShipment, g.Settings.ChanceLevel) {
				g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
					Phase:  PhaseShipment,
					Reason: GetSkipReason(PhaseShipment),
				})
				log.Printf("startNewRound: Skipping shipment phase - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)
				g.Phase = PhaseConquest
			}
		}
	} else {
		// Production phase - set pending flag for server to trigger animation
		g.Phase = PhaseProduction
		g.ProductionPending = true
		log.Printf("startNewRound: Production pending, waiting for animation")
	}

	// Set first player
	for _, pid := range g.PlayerOrder {
		if !g.Players[pid].Eliminated {
			g.CurrentPlayerID = pid
			break
		}
	}
}

// CompleteProduction is called after production animation is done.
// It clears the pending flag and advances to the next phase.
func (g *GameState) CompleteProduction() {
	g.ProductionPending = false

	// Advance to next phase after production
	if len(g.Players) >= 3 {
		// Check if trade should be skipped
		if ShouldSkipPhase(PhaseTrade, g.Settings.ChanceLevel) {
			g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
				Phase:  PhaseTrade,
				Reason: GetSkipReason(PhaseTrade),
			})
			log.Printf("CompleteProduction: Skipping trade phase - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)
			g.Phase = PhaseShipment
			// Check shipment skip too
			if ShouldSkipPhase(PhaseShipment, g.Settings.ChanceLevel) {
				g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
					Phase:  PhaseShipment,
					Reason: GetSkipReason(PhaseShipment),
				})
				log.Printf("CompleteProduction: Skipping shipment phase - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)
				g.Phase = PhaseConquest
			}
		} else {
			g.Phase = PhaseTrade
		}
	} else {
		g.Phase = PhaseShipment
		if ShouldSkipPhase(PhaseShipment, g.Settings.ChanceLevel) {
			g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
				Phase:  PhaseShipment,
				Reason: GetSkipReason(PhaseShipment),
			})
			log.Printf("CompleteProduction: Skipping shipment phase - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)
			g.Phase = PhaseConquest
		}
	}

	log.Printf("CompleteProduction: Advanced to phase %s", g.Phase.String())
}

// GetBuildOptions returns what a player can build.
func (g *GameState) GetBuildOptions(playerID string) []map[string]interface{} {
	options := make([]map[string]interface{}, 0)

	player := g.Players[playerID]
	if player == nil {
		return options
	}

	// Check what can be afforded
	canAffordCity := player.Stockpile.CanAffordStockpile(GetBuildCost(BuildCity)) || player.Stockpile.Gold >= GoldCost(BuildCity)
	canAffordWeapon := player.Stockpile.CanAffordStockpile(GetBuildCost(BuildWeapon)) || player.Stockpile.Gold >= GoldCost(BuildWeapon)
	canAffordBoat := (g.Settings.GameLevel >= LevelAdvanced) &&
		(player.Stockpile.CanAffordStockpile(GetBuildCost(BuildBoat)) || player.Stockpile.Gold >= GoldCost(BuildBoat))

	// Find valid territories for each build type
	for id, t := range g.Territories {
		if t.Owner != playerID {
			continue
		}

		if canAffordCity && !t.HasCity {
			options = append(options, map[string]interface{}{
				"type":        "city",
				"territory":   id,
				"cost":        GetBuildCost(BuildCity),
				"gold_cost":   GoldCost(BuildCity),
			})
		}

		if canAffordWeapon && !t.HasWeapon {
			options = append(options, map[string]interface{}{
				"type":        "weapon",
				"territory":   id,
				"cost":        GetBuildCost(BuildWeapon),
				"gold_cost":   GoldCost(BuildWeapon),
			})
		}

		if canAffordBoat && t.IsCoastal() && t.CanAddBoat() {
			options = append(options, map[string]interface{}{
				"type":        "boat",
				"territory":   id,
				"cost":        GetBuildCost(BuildBoat),
				"gold_cost":   GoldCost(BuildBoat),
			})
		}
	}

	return options
}

