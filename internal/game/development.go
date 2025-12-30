package game

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
func (g *GameState) Build(playerID string, buildType BuildType, territoryID string, useGold bool) error {
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
		territory.Boats++
	}

	return nil
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
	currentIdx := -1
	for i, pid := range g.PlayerOrder {
		if pid == g.CurrentPlayerID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 || len(g.PlayerOrder) == 0 {
		return
	}

	nextIdx := (currentIdx + 1) % len(g.PlayerOrder)

	// Skip eliminated players (with safety counter to prevent infinite loop)
	iterations := 0
	for g.Players[g.PlayerOrder[nextIdx]] != nil && g.Players[g.PlayerOrder[nextIdx]].Eliminated {
		nextIdx = (nextIdx + 1) % len(g.PlayerOrder)
		iterations++
		if nextIdx == currentIdx || iterations >= len(g.PlayerOrder) {
			// All other players eliminated - game should be over
			return
		}
	}

	// Check if we've completed all players (wrapped around to start)
	if nextIdx <= currentIdx {
		// End of round - start new round
		g.startNewRound()
	} else {
		g.CurrentPlayerID = g.PlayerOrder[nextIdx]
	}
}

// startNewRound begins a new round with production phase.
func (g *GameState) startNewRound() {
	g.Round++

	// Shuffle player order
	shufflePlayerOrder(g)

	// Reset player turns
	for _, p := range g.Players {
		p.ResetTurn()
	}

	// Check for phase skip
	pm := NewPhaseManager(g)

	if ShouldSkipPhase(PhaseProduction, g.Settings.ChanceLevel) {
		if len(g.Players) >= 3 {
			g.Phase = PhaseTrade
		} else {
			g.Phase = PhaseShipment
			if ShouldSkipPhase(PhaseShipment, g.Settings.ChanceLevel) {
				g.Phase = PhaseConquest
			}
		}
	} else {
		g.Phase = PhaseProduction
		pm.ProcessProduction()

		// After production, advance to next phase
		if len(g.Players) >= 3 {
			g.Phase = PhaseTrade
		} else {
			g.Phase = PhaseShipment
			if ShouldSkipPhase(PhaseShipment, g.Settings.ChanceLevel) {
				g.Phase = PhaseConquest
			}
		}
	}

	// Set first player
	for _, pid := range g.PlayerOrder {
		if !g.Players[pid].Eliminated {
			g.CurrentPlayerID = pid
			break
		}
	}
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

