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
	g.StockpilePlacementPending = true // All players need to place stockpiles

	// First production: players place their stockpiles
	// This is handled as a sub-phase of production

	// Randomize player order for the game
	shufflePlayerOrder(g)
	g.CurrentPlayerID = g.PlayerOrder[0]
}

// PlaceStockpile places a player's stockpile during the production phase.
// This is used at the start of round 1 and after a player loses their stockpile.
// Note: Does NOT automatically trigger production - server should call
// AllStockpilesPlaced() to check, then trigger production animation.
func (g *GameState) PlaceStockpile(playerID, territoryID string) error {
	// Must be in production phase with stockpile placement pending
	if g.Phase != PhaseProduction || !g.StockpilePlacementPending {
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

	return nil
}

// AllStockpilesPlaced checks if all players have placed their stockpiles.
func (g *GameState) AllStockpilesPlaced() bool {
	if g.Phase != PhaseProduction || !g.StockpilePlacementPending {
		return false
	}
	for _, p := range g.Players {
		if !p.Eliminated && p.StockpileTerritory == "" {
			return false
		}
	}
	return true
}

// NeedsStockpilePlacement checks if any player needs to place a stockpile.
func (g *GameState) NeedsStockpilePlacement() bool {
	for _, p := range g.Players {
		if !p.Eliminated && p.StockpileTerritory == "" {
			return true
		}
	}
	return false
}

// GetPlayersNeedingStockpile returns IDs of players who need to place stockpiles.
func (g *GameState) GetPlayersNeedingStockpile() []string {
	players := make([]string, 0)
	for id, p := range g.Players {
		if !p.Eliminated && p.StockpileTerritory == "" {
			players = append(players, id)
		}
	}
	return players
}

// AdvanceFromStockpilePlacement advances from stockpile placement to the next phase.
// Called by server AFTER production animation is complete.
func (g *GameState) AdvanceFromStockpilePlacement() {
	pm := NewPhaseManager(g)
	// Move to next phase (will be Trade or Shipment)
	g.Phase, _ = pm.NextPhase()
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

