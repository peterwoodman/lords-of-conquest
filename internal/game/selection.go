package game

// SelectTerritory handles a player claiming a territory during territory selection.
func (g *GameState) SelectTerritory(playerID, territoryID string) error {
	// Validate phase
	if g.Phase != PhaseTerritorySelection {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	// Get territory
	territory, ok := g.Territories[territoryID]
	if !ok {
		return ErrInvalidTarget
	}

	// Check if already claimed
	if territory.Owner != "" {
		return ErrTerritoryOccupied
	}

	// Claim territory
	territory.Owner = playerID

	// Move to next player
	g.advancePlayerTurn()

	// Check if territory selection is complete
	if g.isTerritorySelectionComplete() {
		g.startFirstRound()
	}

	return nil
}

// advancePlayerTurn moves to the next player in the order.
func (g *GameState) advancePlayerTurn() {
	for i, pid := range g.PlayerOrder {
		if pid == g.CurrentPlayerID {
			next := (i + 1) % len(g.PlayerOrder)
			g.CurrentPlayerID = g.PlayerOrder[next]
			return
		}
	}
}

// isTerritorySelectionComplete checks if all territories that should be claimed are claimed.
func (g *GameState) isTerritorySelectionComplete() bool {
	unclaimed := 0
	for _, t := range g.Territories {
		if t.Owner == "" {
			unclaimed++
		}
	}

	// Leave territories unclaimed if fewer than player count
	// (following original game rules)
	return unclaimed < len(g.Players)
}

// startFirstRound transitions from territory selection to first game round.
func (g *GameState) startFirstRound() {
	g.Round = 1
	g.Phase = PhaseProduction

	// First production: players place their stockpiles
	// This is handled as a sub-phase of production

	// Randomize player order for the game
	shufflePlayerOrder(g)
	g.CurrentPlayerID = g.PlayerOrder[0]
}

// PlaceStockpile places a player's stockpile during the first production phase.
func (g *GameState) PlaceStockpile(playerID, territoryID string) error {
	if g.Round != 1 || g.Phase != PhaseProduction {
		return ErrInvalidAction
	}

	player := g.Players[playerID]
	if player == nil {
		return ErrInvalidTarget
	}

	// Check if player already placed stockpile
	if player.StockpileTerritory != "" {
		return ErrInvalidAction
	}

	// Check if player owns the territory
	territory, ok := g.Territories[territoryID]
	if !ok || territory.Owner != playerID {
		return ErrInvalidTarget
	}

	// Place stockpile
	player.StockpileTerritory = territoryID

	// Check if all players have placed stockpiles
	allPlaced := true
	for _, p := range g.Players {
		if !p.Eliminated && p.StockpileTerritory == "" {
			allPlaced = false
			break
		}
	}

	// If all placed, start the actual production phase
	if allPlaced {
		pm := NewPhaseManager(g)
		pm.ProcessProduction()
		// Move to next phase (will be Trade or Shipment)
		g.Phase, _ = pm.NextPhase()
	}

	return nil
}

// GetClaimableTerritories returns territories that can be claimed.
func (g *GameState) GetClaimableTerritories() []string {
	if g.Phase != PhaseTerritorySelection {
		return nil
	}

	claimable := make([]string, 0)
	for id, t := range g.Territories {
		if t.Owner == "" {
			claimable = append(claimable, id)
		}
	}
	return claimable
}

// GetPlayerTerritories returns all territories owned by a player.
func (g *GameState) GetPlayerTerritories(playerID string) []string {
	territories := make([]string, 0)
	for id, t := range g.Territories {
		if t.Owner == playerID {
			territories = append(territories, id)
		}
	}
	return territories
}

