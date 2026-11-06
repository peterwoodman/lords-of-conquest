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

func TestSortAllies_UsesAllianceSettings(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	target := g.Territories["b1"] // A attacks B; C borders b1
	cases := []struct {
		setting       AllianceSetting
		att, def, ask int
	}{
		{AllianceNeutral, 0, 0, 0},
		{AllianceAsk, 0, 0, 1},
		{AllianceDefender, 0, 1, 0},
		{"A", 1, 0, 0},
		{"B", 0, 1, 0},
	}
	for _, c := range cases {
		g.Players["C"].Alliance = c.setting
		att, def, ask := g.SortAllies("A", target)
		if len(att) != c.att || len(def) != c.def || len(ask) != c.ask {
			t.Errorf("setting %q: got attacker %v, defender %v, ask %v", c.setting, att, def, ask)
		}
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

func TestResolveCards_SabotageRemovesDoubledWeapons(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	g.Territories["a1"].HasWeapon = true // A's only weapon next to b1
	target := g.Territories["b1"]

	attack := []CombatCard{{ID: "x", Effect: EffectDoubleAttack}}
	defense := []CombatCard{{ID: "y", Effect: EffectSabotage}}

	// Base 4 = territory 1 + weapon 3; doubled to 8, minus the doubled weapon = 2
	result := g.ResolveCards("A", target, 4, 1, attack, defense, nil)
	if result.FinalAttack != 2 {
		t.Fatalf("expected attack 2 after sabotage under double attack, got %d", result.FinalAttack)
	}
}

func TestResolveCards_NegatedBlitzReturnsNothing(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	g.Settings.ChanceLevel = ChanceLow
	target := g.Territories["b1"]

	// Shield Wall negates one random card, so with only Blitz played it is always the one negated
	attack := []CombatCard{{ID: "blitz", Effect: EffectBlitz}}
	defense := []CombatCard{{ID: "wall", Effect: EffectShieldWall}}

	result := g.ResolveCards("A", target, 10, 0, attack, defense, nil)
	if !result.AttackerWins {
		t.Fatal("expected attacker to win")
	}
	if len(result.BlitzReturn) != 0 {
		t.Fatalf("negated Blitz should return no cards, got %d", len(result.BlitzReturn))
	}
}

func TestProduction_HorsesDoNotShareDestination(t *testing.T) {
	g := newRulesTestState(PhaseProduction)
	// Two grasslands with horses, both next to the same empty territory a1
	g.Territories["g1"] = &Territory{ID: "g1", Name: "g1", Owner: "A", Resource: ResourceGrassland, HasHorse: true, Adjacent: []string{"a1"}}
	g.Territories["g2"] = &Territory{ID: "g2", Name: "g2", Owner: "A", Resource: ResourceGrassland, HasHorse: true, Adjacent: []string{"a1"}}
	g.Territories["a1"].Adjacent = append(g.Territories["a1"].Adjacent, "g1", "g2")

	results, _ := NewPhaseManager(g).CalculateProductionForPlayer("A")

	horses := 0
	for _, r := range results {
		if r.Resource == ResourceGrassland {
			horses++
			if r.DestinationID != "a1" {
				t.Fatalf("unexpected horse destination %s", r.DestinationID)
			}
		}
	}
	if horses != 1 {
		t.Fatalf("expected exactly one horse sent to a1, got %d", horses)
	}
}

func TestBoatAttack_OnlyTargetsOnSameWater(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	g.WaterBodies = map[string]*WaterBody{"w1": {ID: "w1", Territories: []string{"a1", "c1"}}}
	from := g.Territories["a1"]
	from.WaterBodies = []string{"w1"}
	from.CoastalTiles = 2
	from.AddBoat("w1")
	g.Territories["c1"].WaterBodies = []string{"w1"}

	if !g.canBoatReachTargetViaWater("c1", "w1") {
		t.Fatal("boat should reach a territory on the same water")
	}
	// b1 is landlocked but next to A's coastal a1
	if g.canBoatReachTargetViaWater("b1", "w1") {
		t.Fatal("boat must not attack a territory that doesn't border its water")
	}
}

func TestBoatAttack_ReplacesBoatInFullTerritory(t *testing.T) {
	g := newRulesTestState(PhaseConquest)
	from := g.Territories["a1"]
	from.WaterBodies = []string{"w1"}
	from.CoastalTiles = 2
	from.AddBoat("w1")
	target := g.Territories["b1"]
	target.WaterBodies = []string{"w1"}
	target.CoastalTiles = 1
	target.AddBoat("w1") // defender's boat fills the only slot

	g.moveBroughtUnit(&BroughtUnit{UnitType: UnitBoat, FromTerritory: "a1", WaterBodyID: "w1"}, target)

	if target.Boats["w1"] != 1 {
		t.Fatalf("expected the attacking boat to replace the captured one, got %d boats", target.Boats["w1"])
	}
	if from.TotalBoats() != 0 {
		t.Fatal("attacking boat should have left its origin")
	}
}

func TestInitCommercePhase_FullSkipStartsConquestWithFirstPlayer(t *testing.T) {
	g := newRulesTestState(PhaseProduction)
	g.CurrentPlayerID = "C" // e.g. the last player who had to place a stockpile

	// Retry until both sub-phases are randomly skipped (or give up; the reset applies either way)
	g.Settings.ChanceLevel = ChanceHigh
	for i := 0; i < 1000; i++ {
		g.SkippedPhases = nil
		if g.InitCommercePhase() {
			break
		}
		g.CurrentPlayerID = "C"
	}

	if g.CurrentPlayerID != "A" {
		t.Fatalf("expected first player A to start, got %s", g.CurrentPlayerID)
	}
}
