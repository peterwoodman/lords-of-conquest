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

// MoveUnit moves a unit from one territory to another.
func (g *GameState) MoveUnit(playerID, unitType, fromID, toID, waterBodyID string, carryHorse, carryWeapon bool) error {
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
		return g.moveBoat(player, from, to, waterBodyID, carryHorse, carryWeapon)
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
// The boat stays in the same water body it was in.
func (g *GameState) moveBoat(player *Player, from, to *Territory, waterBodyID string, carryHorse, carryWeapon bool) error {
	if from.TotalBoats() == 0 {
		return ErrInvalidTarget
	}

	// Both territories must be coastal
	if !from.IsCoastal() || !to.IsCoastal() {
		return ErrCannotReach
	}

	// If waterBodyID specified, use that; otherwise find a shared one
	sharedWaterBody := waterBodyID
	if sharedWaterBody == "" {
		// Find a shared water body that has a boat
		for wID, count := range from.Boats {
			if count > 0 {
				// Check if destination also borders this water body
				for _, destWater := range to.WaterBodies {
					if destWater == wID {
						sharedWaterBody = wID
						break
					}
				}
			}
			if sharedWaterBody != "" {
				break
			}
		}
	}

	if sharedWaterBody == "" {
		return ErrCannotReach
	}

	// Verify the boat exists in the specified water body
	if from.BoatsInWater(sharedWaterBody) == 0 {
		return ErrInvalidTarget
	}

	// Verify destination borders this water body
	destBordersWater := false
	for _, wb := range to.WaterBodies {
		if wb == sharedWaterBody {
			destBordersWater = true
			break
		}
	}
	if !destBordersWater {
		return ErrCannotReach
	}

	// Destination must have room for another boat
	if !to.CanAddBoat() {
		return ErrTerritoryOccupied
	}

	// Move boat (stays in same water body)
	from.RemoveBoat(sharedWaterBody)
	to.AddBoat(sharedWaterBody)

	// Optionally carry horse (lost if destination already has one)
	if carryHorse && from.HasHorse {
		from.HasHorse = false
		if !to.HasHorse {
			to.HasHorse = true
		}
		// If destination already has horse, the carried one is lost
	}

	// Optionally carry weapon (lost if destination already has one)
	if carryWeapon && from.HasWeapon {
		from.HasWeapon = false
		if !to.HasWeapon {
			to.HasWeapon = true
		}
		// If destination already has weapon, the carried one is lost
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

		// Check territories via shared water bodies (if we have boats in that water body)
		if territory.IsCoastal() {
			for _, waterID := range territory.WaterBodies {
				// Only use water routes if we have a boat in this water body
				if territory.BoatsInWater(waterID) > 0 {
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

// GetMovableUnits returns units that can be moved by a player.
func (g *GameState) GetMovableUnits(playerID string) []map[string]interface{} {
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

		if t.TotalBoats() > 0 {
			units = append(units, map[string]interface{}{
				"type":        "boat",
				"territory":   id,
				"count":       t.TotalBoats(),
				"boats":       t.Boats, // Map of water body -> count
				"can_carry":   t.HasWeapon || t.HasHorse,
			})
		}
	}

	return units
}

