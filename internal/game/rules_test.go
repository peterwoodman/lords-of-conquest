package game

import "testing"

// newRulesTestState builds a 3-player game: territories a1(A) - b1(B) - c1(C) - a2(A) in a line,
// plus a far territory a3(A) with a horse that is not within reach of b1.
func newRulesTestState(phase Phase) *GameState {
	g := &GameState{
		Phase:       phase,
		Round:       2,
		Settings:    Settings{VictoryCities: 99},
		Players:     make(map[string]*Player),
		Territories: make(map[string]*Territory),
		PlayerOrder: []string{"A", "B", "C"},
	}
	for _, id := range g.PlayerOrder {
		g.Players[id] = &Player{ID: id, Name: id, Stockpile: NewStockpile(), AttacksRemaining: 2}
	}
	add := func(id, owner string, adjacent ...string) *Territory {
		t := &Territory{ID: id, Name: id, Owner: owner, Adjacent: adjacent, Boats: map[string]int{}}
		g.Territories[id] = t
		return t
	}
	add("a1", "A", "b1")
	add("b1", "B", "a1", "c1")
	add("c1", "C", "b1", "a2")
	add("a2", "A", "c1")
	add("a3", "A").HasHorse = true
	g.CurrentPlayerID = "A"
	return g
}

func TestValidateTrade_RejectsNegativeAmounts(t *testing.T) {
	g := newRulesTestState(PhaseCommerce)
	g.CommerceSubPhase = SubPhaseTrade

	offer := &TradeOffer{FromPlayerID: "A", ToPlayerID: "B", OfferCoal: -10}
	if err := g.ValidateTrade(offer); err == nil {
		t.Fatal("expected negative offer to be rejected")
	}
}

func TestValidateTrade_RejectsDuplicateHorseTerritory(t *testing.T) {
	g := newRulesTestState(PhaseCommerce)
	g.CommerceSubPhase = SubPhaseTrade

	offer := &TradeOffer{FromPlayerID: "A", ToPlayerID: "B", OfferHorses: 2, OfferHorseTerrs: []string{"a3", "a3"}}
	if err := g.ValidateTrade(offer); err == nil {
		t.Fatal("expected the same horse offered twice to be rejected")
	}
}

func TestExecuteTrade_RevalidatesResources(t *testing.T) {
	g := newRulesTestState(PhaseCommerce)
	g.CommerceSubPhase = SubPhaseTrade
	g.Players["A"].Stockpile.Gold = 5

	offer := &TradeOffer{FromPlayerID: "A", ToPlayerID: "B", OfferGold: 5}
	if err := g.ValidateTrade(offer); err != nil {
		t.Fatalf("offer should be valid at proposal time: %v", err)
	}

	// Proposer spends the gold before the target accepts
	g.Players["A"].Stockpile.Gold = 0
	if err := g.ExecuteTrade(offer, nil, nil, nil); err == nil {
		t.Fatal("expected trade to be rejected once the proposer no longer has the gold")
	}
	if g.Players["A"].Stockpile.Gold != 0 || g.Players["B"].Stockpile.Gold != 0 {
		t.Fatal("rejected trade must not move resources")
	}
}

func TestExecuteTrade_MovesHorseToChosenTerritory(t *testing.T) {
	g := newRulesTestState(PhaseCommerce)
	g.CommerceSubPhase = SubPhaseTrade

	offer := &TradeOffer{FromPlayerID: "A", ToPlayerID: "B", OfferHorses: 1, OfferHorseTerrs: []string{"a3"}}
	if err := g.ExecuteTrade(offer, nil, []string{"b1"}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Territories["a3"].HasHorse || !g.Territories["b1"].HasHorse {
		t.Fatal("expected horse to move from a3 to b1")
	}
}

func TestValidateAttack_RejectsOutOfTurn(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	if err := g.ValidateAttack("B", "a1", nil); err != ErrNotYourTurn {
		t.Fatalf("expected ErrNotYourTurn, got %v", err)
	}
}

func TestValidateAttack_RejectsUnreachableBroughtUnit(t *testing.T) {
	g := newRulesTestState(PhaseConquest)

	// a3's horse is two steps away with no owned territory in between
	brought := &BroughtUnit{UnitType: UnitHorse, FromTerritory: "a3"}
	if err := g.ValidateAttack("A", "b1", brought); err == nil {
		t.Fatal("expected unreachable horse to be rejected")
	}

	// A territory that doesn't exist must be rejected rather than panic later
	brought = &BroughtUnit{UnitType: UnitHorse, FromTerritory: "nowhere"}
	if err := g.ValidateAttack("A", "b1", brought); err == nil {
		t.Fatal("expected unknown source territory to be rejected")
	}
}

func TestValidateAttack_AllowsAdjacentAttack(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	if err := g.ValidateAttack("A", "b1", nil); err != nil {
		t.Fatalf("expected valid attack, got %v", err)
	}
}

func TestSurrender_LastPlayerInConquestEndsRound(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	g.CurrentPlayerID = "C"

	g.Surrender("C", "B")

	if g.Phase != PhaseDevelopment || g.Round != 3 {
		t.Fatalf("expected round to end (Development, round 3), got %s round %d", g.Phase, g.Round)
	}
	for _, id := range g.PlayerOrder {
		if id == "C" {
			t.Fatal("surrendered player still in turn order")
		}
	}
}

func TestSurrender_MidOrderPassesTurnToNextPlayer(t *testing.T) {
	g := newRulesTestState(PhaseCommerce)
	g.CommerceSubPhase = SubPhaseTrade
	g.CurrentPlayerID = "B"

	g.Surrender("B", "A")

	if g.Phase != PhaseCommerce || g.CurrentPlayerID != "C" || g.CommerceSubPhase != SubPhaseTrade {
		t.Fatalf("expected C's trade turn, got %s/%s for %s", g.Phase, g.CommerceSubPhase, g.CurrentPlayerID)
	}
}

func TestEliminationOfCurrentPlayerPassesConquestTurn(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	g.CurrentPlayerID = "B"

	// B loses its only territory during its own turn (e.g. to a Counter-Attack card)
	g.Territories["b1"].Owner = "A"
	g.checkElimination("B")

	if g.CurrentPlayerID != "C" {
		t.Fatalf("expected turn to pass to C, got %s", g.CurrentPlayerID)
	}
}

func TestCaptureStockpile_MovesResources(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	g.Players["B"].StockpileTerritory = "b1"
	g.Players["B"].Stockpile.Iron = 4

	captured := g.captureStockpile("A", "B", "b1")

	if captured == nil || captured.Iron != 4 {
		t.Fatal("expected 4 iron captured")
	}
	if g.Players["A"].Stockpile.Iron != 4 || g.Players["B"].StockpileTerritory != "" {
		t.Fatal("expected stockpile to move to captor and be cleared from loser")
	}
}
