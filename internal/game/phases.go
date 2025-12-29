package game

import (
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
	return phase == PhaseProduction || phase == PhaseShipment
}

// ShouldSkipPhase randomly determines if a phase should be skipped.
func ShouldSkipPhase(phase Phase, chanceLevel ChanceLevel) bool {
	if !CanSkipPhase(phase, chanceLevel) {
		return false
	}
	// 25% chance to skip production or shipment
	return rand.Float32() < 0.25
}

// NextPhase advances to the next phase.
func (pm *PhaseManager) NextPhase() (Phase, bool) {
	s := pm.State

	switch s.Phase {
	case PhaseTerritorySelection:
		// Check if all territories are claimed
		if pm.allTerritoriesClaimed() {
			s.Phase = PhaseProduction
			s.Round = 1
			pm.shufflePlayerOrder()
			return s.Phase, false
		}
		// Advance to next player for selection
		pm.advancePlayerOrder()
		return s.Phase, false

	case PhaseProduction:
		// Trade phase only with 3+ players
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
		s.Phase = PhaseDevelopment
		return s.Phase, false

	case PhaseDevelopment:
		// End of round
		s.Round++
		pm.shufflePlayerOrder()
		pm.resetPlayerTurns()
		
		s.Phase = PhaseProduction
		// Check for skip
		if ShouldSkipPhase(PhaseProduction, s.Settings.ChanceLevel) {
			if len(s.Players) >= 3 {
				s.Phase = PhaseTrade
			} else {
				s.Phase = PhaseShipment
				if ShouldSkipPhase(PhaseShipment, s.Settings.ChanceLevel) {
					s.Phase = PhaseConquest
				}
			}
			return s.Phase, true // skipped
		}
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
	for _, player := range pm.State.Players {
		if player.Eliminated {
			continue
		}

		for _, territory := range pm.State.Territories {
			if territory.Owner != player.ID {
				continue
			}
			if territory.Resource == ResourceNone {
				continue
			}

			// Horses don't go to stockpile
			if territory.Resource == ResourceHorses {
				pm.spreadHorses(player.ID, territory)
				continue
			}

			// Calculate production amount
			amount := 1
			if pm.hasAdjacentCity(territory, player.ID) {
				amount = 2
			}

			player.Stockpile.Add(territory.Resource, amount)
		}
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

