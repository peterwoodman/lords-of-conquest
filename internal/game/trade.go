package game

// TradeOffer represents a trade proposal between two players.
type TradeOffer struct {
	FromPlayerID    string
	ToPlayerID      string
	OfferCoal       int
	OfferGold       int
	OfferIron       int
	OfferTimber     int
	OfferHorses     int
	OfferHorseTerrs []string // Territory IDs where offered horses come from
	RequestCoal     int
	RequestGold     int
	RequestIron     int
	RequestTimber   int
	RequestHorses   int
}

// ValidateTrade checks if a trade offer is valid (proposer has resources).
func (g *GameState) ValidateTrade(offer *TradeOffer) error {
	// Validate phase
	if g.Phase != PhaseTrade {
		return ErrInvalidAction
	}

	// Validate it's the proposer's turn
	if g.CurrentPlayerID != offer.FromPlayerID {
		return ErrNotYourTurn
	}

	fromPlayer := g.Players[offer.FromPlayerID]
	if fromPlayer == nil {
		return ErrInvalidTarget
	}

	toPlayer := g.Players[offer.ToPlayerID]
	if toPlayer == nil {
		return ErrInvalidTarget
	}

	// Can't trade with yourself
	if offer.FromPlayerID == offer.ToPlayerID {
		return ErrInvalidTarget
	}

	// Check proposer has the resources to offer
	if fromPlayer.Stockpile.Coal < offer.OfferCoal ||
		fromPlayer.Stockpile.Gold < offer.OfferGold ||
		fromPlayer.Stockpile.Iron < offer.OfferIron ||
		fromPlayer.Stockpile.Timber < offer.OfferTimber {
		return ErrInsufficientResources
	}

	// Check proposer has horses to offer on specified territories
	if offer.OfferHorses > 0 {
		if len(offer.OfferHorseTerrs) != offer.OfferHorses {
			return ErrInvalidTarget
		}
		for _, terrID := range offer.OfferHorseTerrs {
			terr := g.Territories[terrID]
			if terr == nil || terr.Owner != offer.FromPlayerID || !terr.HasHorse {
				return ErrInvalidTarget
			}
		}
	}

	// Check target player has the requested resources
	if toPlayer.Stockpile.Coal < offer.RequestCoal ||
		toPlayer.Stockpile.Gold < offer.RequestGold ||
		toPlayer.Stockpile.Iron < offer.RequestIron ||
		toPlayer.Stockpile.Timber < offer.RequestTimber {
		return ErrInsufficientResources
	}

	// Check target player has enough horses
	if offer.RequestHorses > 0 {
		horseCount := 0
		for _, terr := range g.Territories {
			if terr.Owner == offer.ToPlayerID && terr.HasHorse {
				horseCount++
			}
		}
		if horseCount < offer.RequestHorses {
			return ErrInsufficientResources
		}
	}

	return nil
}

// ExecuteTrade performs the trade between two players.
// horseSourceTerrs are the territories from the target player's horses.
// horseDestTerrs are where the proposer wants received horses placed.
func (g *GameState) ExecuteTrade(offer *TradeOffer, horseSourceTerrs, horseDestTerrs []string) error {
	fromPlayer := g.Players[offer.FromPlayerID]
	toPlayer := g.Players[offer.ToPlayerID]

	if fromPlayer == nil || toPlayer == nil {
		return ErrInvalidTarget
	}

	// Transfer stockpile resources from proposer to target
	fromPlayer.Stockpile.Coal -= offer.OfferCoal
	fromPlayer.Stockpile.Gold -= offer.OfferGold
	fromPlayer.Stockpile.Iron -= offer.OfferIron
	fromPlayer.Stockpile.Timber -= offer.OfferTimber

	toPlayer.Stockpile.Coal += offer.OfferCoal
	toPlayer.Stockpile.Gold += offer.OfferGold
	toPlayer.Stockpile.Iron += offer.OfferIron
	toPlayer.Stockpile.Timber += offer.OfferTimber

	// Transfer stockpile resources from target to proposer
	toPlayer.Stockpile.Coal -= offer.RequestCoal
	toPlayer.Stockpile.Gold -= offer.RequestGold
	toPlayer.Stockpile.Iron -= offer.RequestIron
	toPlayer.Stockpile.Timber -= offer.RequestTimber

	fromPlayer.Stockpile.Coal += offer.RequestCoal
	fromPlayer.Stockpile.Gold += offer.RequestGold
	fromPlayer.Stockpile.Iron += offer.RequestIron
	fromPlayer.Stockpile.Timber += offer.RequestTimber

	// Transfer horses from proposer to target
	for i, terrID := range offer.OfferHorseTerrs {
		// Remove horse from source
		if terr := g.Territories[terrID]; terr != nil {
			terr.HasHorse = false
		}
		// Add horse to destination (where target wants it)
		if i < len(horseDestTerrs) {
			if destTerr := g.Territories[horseDestTerrs[i]]; destTerr != nil && destTerr.Owner == offer.ToPlayerID {
				destTerr.HasHorse = true
			}
		}
	}

	// Transfer horses from target to proposer
	for i, terrID := range horseSourceTerrs {
		// Remove horse from source
		if terr := g.Territories[terrID]; terr != nil {
			terr.HasHorse = false
		}
		// For now, proposer needs to specify destination later
		// Actually, the proposer should specify destinations when proposing
		// But this is handled differently - the proposer picks destinations when trade is accepted
		// Let's handle this via a separate message
		_ = i // We'll handle proposer horse destinations separately
	}

	return nil
}

// GetPlayerHorses returns a list of territory IDs where a player has horses.
func (g *GameState) GetPlayerHorses(playerID string) []string {
	horses := make([]string, 0)
	for id, terr := range g.Territories {
		if terr.Owner == playerID && terr.HasHorse {
			horses = append(horses, id)
		}
	}
	return horses
}

// CountPlayerHorses returns the number of horses a player has.
func (g *GameState) CountPlayerHorses(playerID string) int {
	count := 0
	for _, terr := range g.Territories {
		if terr.Owner == playerID && terr.HasHorse {
			count++
		}
	}
	return count
}
