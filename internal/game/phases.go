package game

import (
	"log"
	"math/rand"
)

// PhaseManager handles phase transitions and validations.
type PhaseManager struct {
	State *GameState
}

// NewPhaseManager creates a new phase manager.
func NewPhaseManager(state *GameState) *PhaseManager {
	return &PhaseManager{State: state}
}

// CanSkipPhase returns true if the given phase might be skipped.
func CanSkipPhase(phase Phase, chanceLevel ChanceLevel) bool {
	if chanceLevel == ChanceLow {
		return false
	}
	return phase == PhaseProduction || phase == PhaseShipment || phase == PhaseTrade
}

// ShouldSkipPhase randomly determines if a phase should be skipped.
func ShouldSkipPhase(phase Phase, chanceLevel ChanceLevel) bool {
	if !CanSkipPhase(phase, chanceLevel) {
		return false
	}
	// 25% chance to skip production, shipment, or trade
	return rand.Float32() < 0.25
}

// PhaseSkipReasons contains funny/weird reasons for skipping each phase type.
var PhaseSkipReasons = map[Phase][]string{
	PhaseProduction: {
		"A series of snafus prevents production",
		"Widespread insanity prevents production",
		"Growth of bizarre cult prevents production",
		"Mass confusion halts all factories",
		"Workers distracted by unusual weather",
		"Mysterious illness sweeps the land",
		"Everyone forgot how to work",
		"Tools have gone missing mysteriously",
		"Collective daydreaming stops work",
		"A plague of laziness spreads",
		"Machines refuse to cooperate",
		"Raw materials vanish overnight",
		"Workers protest working conditions",
		"Superstition halts production",
		"Fear of the unknown stops work",
	},
	PhaseShipment: {
		"Glorification of ignorance prevents shipment",
		"All roads mysteriously blocked",
		"Horses refuse to move",
		"Ships lost in dense fog",
		"Bridges collapse simultaneously",
		"Bandits control all trade routes",
		"Carts have square wheels today",
		"Navigation charts are upside down",
		"Teamsters on unexpected holiday",
		"Rivers flowing backwards",
		"Mountains appeared overnight",
		"Cargo spontaneously combusts",
		"Everyone forgot where everything goes",
		"Maps have been eaten by goats",
		"Wheels invented in wrong shape",
	},
	PhaseTrade: {
		"Merchants refuse to negotiate",
		"Currency has become worthless",
		"Everyone forgot how to count",
		"Trust issues prevent all trades",
		"Markets closed for odd festival",
		"Traders speaking unknown language",
		"Prices too confusing to calculate",
		"Bartering skills mysteriously lost",
		"Trade goods turned to dust",
		"Economy collapses temporarily",
		"Merchants distracted by shiny objects",
		"No one can agree on anything",
		"All scales are broken",
		"Coins have vanished into thin air",
		"Trade secrets leaked everywhere",
	},
}

// GetSkipReason returns a random reason for skipping the given phase.
func GetSkipReason(phase Phase) string {
	reasons, ok := PhaseSkipReasons[phase]
	if !ok || len(reasons) == 0 {
		return "Unknown circumstances prevent progress"
	}
	return reasons[rand.Intn(len(reasons))]
}

// PhaseSkipInfo contains information about a skipped phase.
type PhaseSkipInfo struct {
	Phase  Phase
	Reason string
}

// CheckPhaseSkips checks which phases should be skipped from the current phase.
// Returns a list of phases that will be skipped and the resulting phase.
func CheckPhaseSkips(currentPhase Phase, nextPhase Phase, numPlayers int, chanceLevel ChanceLevel) ([]PhaseSkipInfo, Phase) {
	skipped := []PhaseSkipInfo{}
	resultPhase := nextPhase

	// Check if the next phase should be skipped
	for {
		if !ShouldSkipPhase(resultPhase, chanceLevel) {
			break
		}

		// Record the skip
		skipped = append(skipped, PhaseSkipInfo{
			Phase:  resultPhase,
			Reason: GetSkipReason(resultPhase),
		})

		// Move to the next phase
		switch resultPhase {
		case PhaseProduction:
			if numPlayers >= 3 {
				resultPhase = PhaseTrade
			} else {
				resultPhase = PhaseShipment
			}
		case PhaseTrade:
			resultPhase = PhaseShipment
		case PhaseShipment:
			resultPhase = PhaseConquest
		default:
			// Can't skip further
			break
		}
	}

	return skipped, resultPhase
}

// NextPhase advances to the next phase.
// Phase order: Development (skipped Year 1) → Production → Trade → Shipment → Conquest
// End-game is checked after Conquest, then round increments.
func (pm *PhaseManager) NextPhase() (Phase, bool) {
	s := pm.State

	switch s.Phase {
	case PhaseTerritorySelection:
		// Check if all territories are claimed
		if pm.allTerritoriesClaimed() {
			// Year 1 starts - skip Development, go straight to Production
			s.Phase = PhaseProduction
			s.Round = 1
			pm.shufflePlayerOrder()
			return s.Phase, false
		}
		// Advance to next player for selection
		pm.advancePlayerOrder()
		return s.Phase, false

	case PhaseDevelopment:
		// Development → Production transition is handled by transitionToProduction()
		// This case should not be called directly; advanceDevelopmentTurn handles it
		log.Printf("WARNING: NextPhase called for Development - use advanceDevelopmentTurn instead")
		s.Phase = PhaseProduction
		return s.Phase, false

	case PhaseProduction:
		// After Production → Trade (if 3+ players) or Shipment
		if len(s.Players) >= 3 {
			s.Phase = PhaseTrade
		} else {
			s.Phase = PhaseShipment
			// Check for skip
			if ShouldSkipPhase(PhaseShipment, s.Settings.ChanceLevel) {
				s.Phase = PhaseConquest
				return s.Phase, true // skipped
			}
		}
		return s.Phase, false

	case PhaseTrade:
		s.Phase = PhaseShipment
		// Check for skip
		if ShouldSkipPhase(PhaseShipment, s.Settings.ChanceLevel) {
			s.Phase = PhaseConquest
			return s.Phase, true // skipped
		}
		return s.Phase, false

	case PhaseShipment:
		s.Phase = PhaseConquest
		return s.Phase, false

	case PhaseConquest:
		// End of round - check for victory BEFORE advancing
		// Victory conditions are checked at end of round so all players have equal chances
		if s.IsGameOver() {
			log.Printf("NextPhase: Game is over at end of round %d - not advancing to next round", s.Round)
			// Don't change phase - keep in Conquest, game is over
			return s.Phase, false
		}

		// Game continues - increment round and go to Development
		s.Round++
		log.Printf("NextPhase: Advancing to round %d, Development phase", s.Round)
		pm.rotatePlayerOrder() // Rotate instead of shuffle for Year 2+
		pm.resetPlayerTurns()

		// Always go to Development first (Year 2+)
		// Stockpile placement will be handled when Production phase starts
		s.Phase = PhaseDevelopment
		return s.Phase, false
	}

	return s.Phase, false
}

// allTerritoriesClaimed checks if territory selection is complete.
func (pm *PhaseManager) allTerritoriesClaimed() bool {
	unclaimed := 0
	for _, t := range pm.State.Territories {
		if t.Owner == "" {
			unclaimed++
		}
	}
	// Leave territories unclaimed if fewer than player count
	return unclaimed < len(pm.State.Players)
}

// shufflePlayerOrder randomizes player order for the new round.
// Used only for Year 1 initial random order.
func (pm *PhaseManager) shufflePlayerOrder() {
	order := make([]string, 0, len(pm.State.Players))
	for id, p := range pm.State.Players {
		if !p.Eliminated {
			order = append(order, id)
		}
	}
	rand.Shuffle(len(order), func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})
	pm.State.PlayerOrder = order
	if len(order) > 0 {
		pm.State.CurrentPlayerID = order[0]
	}
}

// rotatePlayerOrder moves the first player to the end of the order.
// Used for Year 2+ to give each player a fair chance to go first.
func (pm *PhaseManager) rotatePlayerOrder() {
	if len(pm.State.PlayerOrder) <= 1 {
		return
	}

	// Remove eliminated players from consideration
	activeOrder := make([]string, 0, len(pm.State.PlayerOrder))
	for _, pid := range pm.State.PlayerOrder {
		if p := pm.State.Players[pid]; p != nil && !p.Eliminated {
			activeOrder = append(activeOrder, pid)
		}
	}

	if len(activeOrder) <= 1 {
		pm.State.PlayerOrder = activeOrder
		if len(activeOrder) > 0 {
			pm.State.CurrentPlayerID = activeOrder[0]
		}
		return
	}

	// Rotate: move first player to end
	rotated := make([]string, len(activeOrder))
	copy(rotated, activeOrder[1:])
	rotated[len(rotated)-1] = activeOrder[0]

	pm.State.PlayerOrder = rotated
	pm.State.CurrentPlayerID = rotated[0]
	log.Printf("rotatePlayerOrder: New order %v, first player is %s", rotated, rotated[0])
}

// advancePlayerOrder moves to the next player.
func (pm *PhaseManager) advancePlayerOrder() {
	order := pm.State.PlayerOrder
	current := pm.State.CurrentPlayerID

	for i, id := range order {
		if id == current {
			next := (i + 1) % len(order)
			pm.State.CurrentPlayerID = order[next]
			return
		}
	}
}

// resetPlayerTurns resets all players for the new round.
func (pm *PhaseManager) resetPlayerTurns() {
	for _, p := range pm.State.Players {
		p.ResetTurn()
	}
}

// ProcessProduction generates resources for all players.
func (pm *PhaseManager) ProcessProduction() {
	log.Printf("ProcessProduction: Starting production for round %d", pm.State.Round)

	// Debug: Log territory resources
	resourceCount := 0
	for id, t := range pm.State.Territories {
		if t.Resource != ResourceNone {
			resourceCount++
			log.Printf("ProcessProduction: Territory %s (%s) has resource %s, owner=%s",
				id, t.Name, t.Resource.String(), t.Owner)
		}
	}
	log.Printf("ProcessProduction: Found %d territories with resources", resourceCount)

	for _, player := range pm.State.Players {
		if player.Eliminated {
			continue
		}

		produced := 0
		for _, territory := range pm.State.Territories {
			if territory.Owner != player.ID {
				continue
			}
			if territory.Resource == ResourceNone {
				continue
			}

			// Grassland produces horses that spread on the map
			if territory.Resource == ResourceGrassland {
				pm.spreadHorses(player.ID, territory)
				continue
			}

			// Calculate production amount
			amount := 1
			if pm.hasAdjacentCity(territory, player.ID) {
				amount = 2
			}

			player.Stockpile.Add(territory.Resource, amount)
			produced += amount
			log.Printf("ProcessProduction: Player %s produced %d %s from %s",
				player.Name, amount, territory.Resource.String(), territory.Name)
		}

		log.Printf("ProcessProduction: Player %s total stockpile - Coal:%d Gold:%d Iron:%d Timber:%d",
			player.Name, player.Stockpile.Coal, player.Stockpile.Gold,
			player.Stockpile.Iron, player.Stockpile.Timber)
	}
}

// hasAdjacentCity checks if a territory has an adjacent city.
func (pm *PhaseManager) hasAdjacentCity(t *Territory, playerID string) bool {
	if t.HasCity {
		return true
	}
	for _, adjID := range t.Adjacent {
		adj := pm.State.Territories[adjID]
		if adj.Owner == playerID && adj.HasCity {
			return true
		}
	}
	return false
}

// spreadHorses places a horse in an adjacent territory if possible.
func (pm *PhaseManager) spreadHorses(playerID string, source *Territory) {
	// If source doesn't have a horse yet, place one there
	if !source.HasHorse {
		source.HasHorse = true
		return
	}

	// Find an adjacent territory without a horse
	candidates := []string{}
	for _, adjID := range source.Adjacent {
		adj := pm.State.Territories[adjID]
		if adj.Owner == playerID && !adj.HasHorse {
			candidates = append(candidates, adjID)
		}
	}

	if len(candidates) > 0 {
		// Randomly choose one
		chosen := candidates[rand.Intn(len(candidates))]
		pm.State.Territories[chosen].HasHorse = true
	}
}

// ProductionResult describes a single production event for animation.
type ProductionResult struct {
	TerritoryID     string
	TerritoryName   string
	Resource        ResourceType
	Amount          int
	DestinationID   string // For horses, where they go
	DestinationName string
}

// CalculateProductionForPlayer calculates what a player will produce (without applying it).
// Returns the list of productions and the stockpile territory ID.
func (pm *PhaseManager) CalculateProductionForPlayer(playerID string) ([]ProductionResult, string) {
	player := pm.State.Players[playerID]
	if player == nil || player.Eliminated {
		return nil, ""
	}

	results := []ProductionResult{}

	for terrID, territory := range pm.State.Territories {
		if territory.Owner != playerID || territory.Resource == ResourceNone {
			continue
		}

		// Grassland produces horses that spread on the map
		if territory.Resource == ResourceGrassland {
			// Calculate where horse will go
			destID, destName := pm.calculateHorseDestination(playerID, territory)
			if destID != "" {
				results = append(results, ProductionResult{
					TerritoryID:     terrID,
					TerritoryName:   territory.Name,
					Resource:        ResourceGrassland,
					Amount:          1,
					DestinationID:   destID,
					DestinationName: destName,
				})
			}
			continue
		}

		// Calculate production amount
		amount := 1
		if pm.hasAdjacentCity(territory, playerID) {
			amount = 2
		}

		results = append(results, ProductionResult{
			TerritoryID:   terrID,
			TerritoryName: territory.Name,
			Resource:      territory.Resource,
			Amount:        amount,
		})
	}

	return results, player.StockpileTerritory
}

// calculateHorseDestination determines where a horse will be placed.
func (pm *PhaseManager) calculateHorseDestination(playerID string, source *Territory) (string, string) {
	// If source doesn't have a horse yet, horse goes there
	if !source.HasHorse {
		// Find the territory ID for this source
		for id, t := range pm.State.Territories {
			if t == source {
				return id, source.Name
			}
		}
		return "", ""
	}

	// Find an adjacent territory without a horse
	candidates := []string{}
	for _, adjID := range source.Adjacent {
		adj := pm.State.Territories[adjID]
		if adj.Owner == playerID && !adj.HasHorse {
			candidates = append(candidates, adjID)
		}
	}

	if len(candidates) > 0 {
		// Randomly choose one (use consistent seed for determinism)
		chosen := candidates[rand.Intn(len(candidates))]
		return chosen, pm.State.Territories[chosen].Name
	}

	return "", "" // No valid destination
}

// ApplyProductionResults applies previously calculated production results.
func (pm *PhaseManager) ApplyProductionResults(playerID string, results []ProductionResult) {
	player := pm.State.Players[playerID]
	if player == nil {
		return
	}

	for _, result := range results {
		if result.Resource == ResourceGrassland {
			// Place horse at destination
			if dest := pm.State.Territories[result.DestinationID]; dest != nil {
				dest.HasHorse = true
				log.Printf("ApplyProduction: Placed horse at %s for player %s",
					result.DestinationName, player.Name)
			}
		} else {
			// Add to stockpile
			player.Stockpile.Add(result.Resource, result.Amount)
			log.Printf("ApplyProduction: Player %s gained %d %s from %s",
				player.Name, result.Amount, result.Resource.String(), result.TerritoryName)
		}
	}
}

// ValidateAction checks if an action is valid for the current phase.
func (pm *PhaseManager) ValidateAction(playerID string, action interface{}) error {
	if pm.State.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	switch pm.State.Phase {
	case PhaseTerritorySelection:
		return pm.validateTerritorySelection(action)
	case PhaseProduction:
		return pm.validateProduction(action)
	case PhaseTrade:
		return pm.validateTrade(action)
	case PhaseShipment:
		return pm.validateShipment(action)
	case PhaseConquest:
		return pm.validateConquest(action)
	case PhaseDevelopment:
		return pm.validateDevelopment(action)
	}

	return ErrInvalidAction
}

// Validation stubs - to be implemented
func (pm *PhaseManager) validateTerritorySelection(action interface{}) error {
	// TODO: Implement
	return nil
}

func (pm *PhaseManager) validateProduction(action interface{}) error {
	// TODO: Implement
	return nil
}

func (pm *PhaseManager) validateTrade(action interface{}) error {
	// TODO: Implement
	return nil
}

func (pm *PhaseManager) validateShipment(action interface{}) error {
	// TODO: Implement
	return nil
}

func (pm *PhaseManager) validateConquest(action interface{}) error {
	// TODO: Implement
	return nil
}

func (pm *PhaseManager) validateDevelopment(action interface{}) error {
	// TODO: Implement
	return nil
}

// SkipTrade allows a player to pass the trade phase without trading.
func (g *GameState) SkipTrade(playerID string) error {
	if g.Phase != PhaseTrade {
		return ErrInvalidAction
	}

	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	g.advanceTradeTurn()
	return nil
}

// advanceTradeTurn moves to the next player or next phase.
func (g *GameState) advanceTradeTurn() {
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

	// If we've completed all players, move to shipment phase
	if nextIdx == 0 {
		// Check for shipment skip
		if ShouldSkipPhase(PhaseShipment, g.Settings.ChanceLevel) {
			g.SkippedPhases = append(g.SkippedPhases, PhaseSkipInfo{
				Phase:  PhaseShipment,
				Reason: GetSkipReason(PhaseShipment),
			})
			log.Printf("Trade phase complete, skipping Shipment - %s", g.SkippedPhases[len(g.SkippedPhases)-1].Reason)
			g.Phase = PhaseConquest
		} else {
			g.Phase = PhaseShipment
			log.Printf("Trade phase complete, moving to Shipment phase")
		}
		g.CurrentPlayerID = g.PlayerOrder[0]
	} else {
		g.CurrentPlayerID = g.PlayerOrder[nextIdx]
		log.Printf("Trade turn advancing to player %s", g.CurrentPlayerID)
	}
}
