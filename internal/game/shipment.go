package game

// MoveStockpile moves a player's stockpile to an adjacent owned territory.
func (g *GameState) MoveStockpile(playerID, destinationID string) error {
	// Validate phase
	if g.Phase != PhaseShipment {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	player := g.Players[playerID]
	if player == nil {
		return ErrInvalidTarget
	}

	// Check if player has a stockpile placed
	if player.StockpileTerritory == "" {
		return ErrInvalidAction
	}

	// Get destination territory
	dest, ok := g.Territories[destinationID]
	if !ok {
		return ErrInvalidTarget
	}

	// Must own the destination
	if dest.Owner != playerID {
		return ErrInvalidTarget
	}

	// Can stay in same territory (effectively a pass)
	if player.StockpileTerritory == destinationID {
		g.advanceShipmentTurn()
		return nil
	}

	// Check if destination is reachable (adjacent via land or connected water)
	if !g.canReachTerritory(playerID, player.StockpileTerritory, destinationID) {
		return ErrCannotReach
	}

	// Move stockpile
	player.StockpileTerritory = destinationID

	// Advance to next player
	g.advanceShipmentTurn()

	return nil
}

// MoveUnit moves a unit from one territory to another (Expert level only).
func (g *GameState) MoveUnit(playerID, unitType, fromID, toID string, carryWeapon bool) error {
	// Validate phase
	if g.Phase != PhaseShipment {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	// Expert level only for unit movement
	if g.Settings.GameLevel != LevelExpert {
		return ErrInvalidAction
	}

	player := g.Players[playerID]
	if player == nil {
		return ErrInvalidTarget
	}

	// Get territories
	from, ok := g.Territories[fromID]
	if !ok {
		return ErrInvalidTarget
	}

	to, ok := g.Territories[toID]
	if !ok {
		return ErrInvalidTarget
	}

	// Must own both territories
	if from.Owner != playerID || to.Owner != playerID {
		return ErrInvalidTarget
	}

	switch unitType {
	case "horse":
		return g.moveHorse(player, from, to, carryWeapon)
	case "weapon":
		return g.moveWeapon(player, from, to)
	case "boat":
		return g.moveBoat(player, from, to, carryWeapon)
	default:
		return ErrInvalidTarget
	}
}

// moveHorse moves a horse to an adjacent territory.
func (g *GameState) moveHorse(player *Player, from, to *Territory, carryWeapon bool) error {
	if !from.HasHorse {
		return ErrInvalidTarget
	}

	// Horse can move up to 2 territories
	if !g.canHorseReach(player.ID, from.ID, to.ID) {
		return ErrCannotReach
	}

	// Can't move to territory that already has a horse
	if to.HasHorse {
		return ErrTerritoryOccupied
	}

	// Move horse
	from.HasHorse = false
	to.HasHorse = true

	// Optionally carry weapon
	if carryWeapon && from.HasWeapon && !to.HasWeapon {
		from.HasWeapon = false
		to.HasWeapon = true
	}

	g.advanceShipmentTurn()
	return nil
}

// moveWeapon moves a weapon to an adjacent territory.
func (g *GameState) moveWeapon(player *Player, from, to *Territory) error {
	if !from.HasWeapon {
		return ErrInvalidTarget
	}

	// Weapon can only move to adjacent territory
	if !isAdjacent(from, to.ID) {
		return ErrCannotReach
	}

	// Can't move to territory that already has a weapon
	if to.HasWeapon {
		return ErrTerritoryOccupied
	}

	// Move weapon
	from.HasWeapon = false
	to.HasWeapon = true

	g.advanceShipmentTurn()
	return nil
}

// moveBoat moves a boat via water to another coastal territory.
func (g *GameState) moveBoat(player *Player, from, to *Territory, carryWeapon bool) error {
	if from.Boats == 0 {
		return ErrInvalidTarget
	}

	// Both territories must be coastal
	if !from.IsCoastal() || !to.IsCoastal() {
		return ErrCannotReach
	}

	// Must share a water body
	if !g.shareWaterBody(from, to) {
		return ErrCannotReach
	}

	// Destination must have room for another boat
	if !to.CanAddBoat() {
		return ErrTerritoryOccupied
	}

	// Move boat
	from.Boats--
	to.Boats++

	// Optionally carry weapon (boat can also carry horse with weapon)
	if carryWeapon && from.HasWeapon && !to.HasWeapon {
		from.HasWeapon = false
		to.HasWeapon = true
	}

	g.advanceShipmentTurn()
	return nil
}

// SkipShipment allows a player to pass without moving anything.
func (g *GameState) SkipShipment(playerID string) error {
	// Validate phase
	if g.Phase != PhaseShipment {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	g.advanceShipmentTurn()
	return nil
}

// advanceShipmentTurn moves to the next player or next phase.
func (g *GameState) advanceShipmentTurn() {
	// Find next player
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

	// Check if all players have gone (we're back at the start)
	nextIdx := (currentIdx + 1) % len(g.PlayerOrder)

	// If we've completed all players, move to conquest phase
	if nextIdx == 0 {
		g.Phase = PhaseConquest
		g.CurrentPlayerID = g.PlayerOrder[0]
		// Reset attacks for all players
		for _, p := range g.Players {
			p.AttacksRemaining = 2
		}
	} else {
		// Skip eliminated players (with safety counter to prevent infinite loop)
		iterations := 0
		for g.Players[g.PlayerOrder[nextIdx]] != nil && g.Players[g.PlayerOrder[nextIdx]].Eliminated {
			nextIdx = (nextIdx + 1) % len(g.PlayerOrder)
			iterations++
			if nextIdx == 0 || iterations >= len(g.PlayerOrder) {
				g.Phase = PhaseConquest
				g.CurrentPlayerID = g.PlayerOrder[0]
				for _, p := range g.Players {
					p.AttacksRemaining = 2
				}
				return
			}
		}
		g.CurrentPlayerID = g.PlayerOrder[nextIdx]
	}
}

// canReachTerritory checks if a territory is reachable from another for stockpile movement.
// Stockpile can move to any owned connected territory via land or sea routes.
func (g *GameState) canReachTerritory(playerID, fromID, toID string) bool {
	// BFS to find if we can reach the destination
	visited := make(map[string]bool)
	queue := []string{fromID}
	visited[fromID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == toID {
			return true
		}

		territory := g.Territories[current]

		// Check adjacent territories via land
		for _, adjID := range territory.Adjacent {
			if !visited[adjID] {
				adj := g.Territories[adjID]
				if adj.Owner == playerID {
					visited[adjID] = true
					queue = append(queue, adjID)
				}
			}
		}

		// Check territories via shared water bodies (if we have boats)
		if territory.IsCoastal() && territory.Boats > 0 {
			for _, waterID := range territory.WaterBodies {
				water := g.WaterBodies[waterID]
				for _, coastalID := range water.Territories {
					if !visited[coastalID] {
						coastal := g.Territories[coastalID]
						if coastal.Owner == playerID {
							visited[coastalID] = true
							queue = append(queue, coastalID)
						}
					}
				}
			}
		}
	}

	return false
}

// canHorseReach checks if a horse can reach a territory (up to 2 moves).
func (g *GameState) canHorseReach(playerID, fromID, toID string) bool {
	from := g.Territories[fromID]

	// Direct adjacency (1 move)
	if isAdjacent(from, toID) {
		return true
	}

	// 2 moves through owned territory
	for _, midID := range from.Adjacent {
		mid := g.Territories[midID]
		if mid.Owner == playerID {
			if isAdjacent(mid, toID) {
				return true
			}
		}
	}

	return false
}

// isAdjacent checks if a territory is adjacent to another.
func isAdjacent(t *Territory, targetID string) bool {
	for _, adjID := range t.Adjacent {
		if adjID == targetID {
			return true
		}
	}
	return false
}

// shareWaterBody checks if two territories share a water body.
func (g *GameState) shareWaterBody(t1, t2 *Territory) bool {
	for _, w1 := range t1.WaterBodies {
		for _, w2 := range t2.WaterBodies {
			if w1 == w2 {
				return true
			}
		}
	}
	return false
}

// GetValidStockpileDestinations returns territories the stockpile can move to.
func (g *GameState) GetValidStockpileDestinations(playerID string) []string {
	player := g.Players[playerID]
	if player == nil || player.StockpileTerritory == "" {
		return nil
	}

	destinations := make([]string, 0)
	for id, t := range g.Territories {
		if t.Owner == playerID {
			if g.canReachTerritory(playerID, player.StockpileTerritory, id) {
				destinations = append(destinations, id)
			}
		}
	}
	return destinations
}

// GetMovableUnits returns units that can be moved by a player (Expert level only).
func (g *GameState) GetMovableUnits(playerID string) []map[string]interface{} {
	if g.Settings.GameLevel != LevelExpert {
		return nil
	}

	units := make([]map[string]interface{}, 0)

	for id, t := range g.Territories {
		if t.Owner != playerID {
			continue
		}

		if t.HasHorse {
			units = append(units, map[string]interface{}{
				"type":        "horse",
				"territory":   id,
				"can_carry":   t.HasWeapon,
			})
		}

		if t.HasWeapon {
			units = append(units, map[string]interface{}{
				"type":      "weapon",
				"territory": id,
			})
		}

		if t.Boats > 0 {
			units = append(units, map[string]interface{}{
				"type":        "boat",
				"territory":   id,
				"count":       t.Boats,
				"can_carry":   t.HasWeapon || t.HasHorse,
			})
		}
	}

	return units
}

