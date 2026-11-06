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
	if g.Phase != PhaseCommerce || g.CommerceSubPhase != SubPhaseTrade {
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
	if toPlayer == nil || toPlayer.Eliminated || fromPlayer.Eliminated {
		return ErrInvalidTarget
	}

	// Can't trade with yourself
	if offer.FromPlayerID == offer.ToPlayerID {
		return ErrInvalidTarget
	}

	for _, amount := range []int{
		offer.OfferCoal, offer.OfferGold, offer.OfferIron, offer.OfferTimber, offer.OfferHorses,
		offer.RequestCoal, offer.RequestGold, offer.RequestIron, offer.RequestTimber, offer.RequestHorses,
	} {
		if amount < 0 {
			return ErrInvalidAction
		}
	}

	// Check proposer has the resources to offer
	if fromPlayer.Stockpile.Coal < offer.OfferCoal ||
		fromPlayer.Stockpile.Gold < offer.OfferGold ||
		fromPlayer.Stockpile.Iron < offer.OfferIron ||
		fromPlayer.Stockpile.Timber < offer.OfferTimber {
		return ErrInsufficientResources
	}

	// Check proposer has horses to offer on specified territories
	if err := g.validateHorseTerritories(offer.FromPlayerID, offer.OfferHorseTerrs, offer.OfferHorses, true); err != nil {
		return err
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
// horseSourceTerrs are the territories from the target player's horses (for RequestHorses).
// offerHorseDestTerrs are where the target wants received horses placed (for OfferHorses).
// requestHorseDestTerrs are where the proposer wants received horses placed (for RequestHorses).
// The offer is re-validated against the current state, since it may have changed since proposal.
func (g *GameState) ExecuteTrade(offer *TradeOffer, horseSourceTerrs, offerHorseDestTerrs, requestHorseDestTerrs []string) error {
	if err := g.ValidateTrade(offer); err != nil {
		return err
	}
	if err := g.validateHorseTerritories(offer.ToPlayerID, horseSourceTerrs, offer.RequestHorses, true); err != nil {
		return err
	}
	if err := g.validateHorseTerritories(offer.ToPlayerID, offerHorseDestTerrs, offer.OfferHorses, false); err != nil {
		return err
	}
	if err := g.validateHorseTerritories(offer.FromPlayerID, requestHorseDestTerrs, offer.RequestHorses, false); err != nil {
		return err
	}

	fromPlayer := g.Players[offer.FromPlayerID]
	toPlayer := g.Players[offer.ToPlayerID]

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

	// Transfer horses from proposer to target (OfferHorses)
	for i, terrID := range offer.OfferHorseTerrs {
		g.Territories[terrID].HasHorse = false
		g.Territories[offerHorseDestTerrs[i]].HasHorse = true
	}

	// Transfer horses from target to proposer (RequestHorses)
	for i, terrID := range horseSourceTerrs {
		g.Territories[terrID].HasHorse = false
		g.Territories[requestHorseDestTerrs[i]].HasHorse = true
	}

	return nil
}

// validateHorseTerritories checks that terrIDs lists exactly count distinct territories owned by
// ownerID, each holding a horse (wantHorse) or each free to receive one (!wantHorse).
func (g *GameState) validateHorseTerritories(ownerID string, terrIDs []string, count int, wantHorse bool) error {
	if len(terrIDs) != count {
		return ErrInvalidTarget
	}
	seen := make(map[string]bool, len(terrIDs))
	for _, id := range terrIDs {
		terr := g.Territories[id]
		if terr == nil || terr.Owner != ownerID || terr.HasHorse != wantHorse || seen[id] {
			return ErrInvalidTarget
		}
		seen[id] = true
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
