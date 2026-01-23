package game

import (
	"testing"
)

// Helper to create a test game state with players and territories
func createTestGameState(victoryCities int, players map[string]int, eliminated map[string]bool) *GameState {
	g := &GameState{
		Settings: Settings{
			VictoryCities: victoryCities,
		},
		Players:     make(map[string]*Player),
		Territories: make(map[string]*Territory),
	}

	// Create players
	for playerID := range players {
		isEliminated := false
		if eliminated != nil {
			isEliminated = eliminated[playerID]
		}
		g.Players[playerID] = &Player{
			ID:         playerID,
			Name:       playerID,
			Eliminated: isEliminated,
		}
	}

	// Create territories with cities assigned to players
	territoryID := 1
	for playerID, cityCount := range players {
		for i := 0; i < cityCount; i++ {
			tid := "t" + string(rune('0'+territoryID))
			g.Territories[tid] = &Territory{
				ID:      tid,
				Owner:   playerID,
				HasCity: true,
			}
			territoryID++
		}
	}

	return g
}

func TestIsCityVictory_SinglePlayerAtThreshold(t *testing.T) {
	// Player A has 5 cities, threshold is 5 - should win
	g := createTestGameState(5, map[string]int{
		"A": 5,
		"B": 3,
	}, nil)

	if !g.IsCityVictory() {
		t.Error("Expected city victory when single player at threshold with clear lead")
	}
}

func TestIsCityVictory_SinglePlayerAboveThreshold(t *testing.T) {
	// Player A has 6 cities, threshold is 5 - should win
	g := createTestGameState(5, map[string]int{
		"A": 6,
		"B": 4,
	}, nil)

	if !g.IsCityVictory() {
		t.Error("Expected city victory when single player above threshold with clear lead")
	}
}

func TestIsCityVictory_TwoPlayersAtThreshold_Tied(t *testing.T) {
	// Both players have 5 cities, threshold is 5 - NO winner (tied)
	g := createTestGameState(5, map[string]int{
		"A": 5,
		"B": 5,
	}, nil)

	if g.IsCityVictory() {
		t.Error("Expected NO city victory when two players tied at threshold")
	}
}

func TestIsCityVictory_TwoPlayersAboveThreshold_Tied(t *testing.T) {
	// Both players have 6 cities, threshold is 5 - NO winner (tied)
	g := createTestGameState(5, map[string]int{
		"A": 6,
		"B": 6,
	}, nil)

	if g.IsCityVictory() {
		t.Error("Expected NO city victory when two players tied above threshold")
	}
}

func TestIsCityVictory_TwoPlayersAboveThreshold_OneHigher(t *testing.T) {
	// Player A has 6 cities, Player B has 5, threshold is 5 - A wins
	g := createTestGameState(5, map[string]int{
		"A": 6,
		"B": 5,
	}, nil)

	if !g.IsCityVictory() {
		t.Error("Expected city victory when one player has clear lead above threshold")
	}
}

func TestIsCityVictory_NobodyAtThreshold(t *testing.T) {
	// No player at threshold
	g := createTestGameState(5, map[string]int{
		"A": 3,
		"B": 4,
	}, nil)

	if g.IsCityVictory() {
		t.Error("Expected NO city victory when nobody at threshold")
	}
}

func TestIsCityVictory_EliminatedPlayerIgnored(t *testing.T) {
	// Player A has 5 cities, Player B (eliminated) has 5 - A wins
	g := createTestGameState(5, map[string]int{
		"A": 5,
		"B": 5,
	}, map[string]bool{"B": true})

	if !g.IsCityVictory() {
		t.Error("Expected city victory - eliminated player's cities should be ignored")
	}
}

func TestGetWinner_CityVictory_ClearLead(t *testing.T) {
	// Player A has 6 cities, Player B has 5, threshold is 5
	g := createTestGameState(5, map[string]int{
		"A": 6,
		"B": 5,
	}, nil)

	winner := g.GetWinner()
	if winner == nil {
		t.Fatal("Expected a winner")
	}
	if winner.ID != "A" {
		t.Errorf("Expected player A to win, got %s", winner.ID)
	}
}

func TestGetWinner_CityVictory_NoWinnerWhenTied(t *testing.T) {
	// Both players have 5 cities, threshold is 5 - no winner
	g := createTestGameState(5, map[string]int{
		"A": 5,
		"B": 5,
	}, nil)

	winner := g.GetWinner()
	if winner != nil {
		t.Errorf("Expected no winner when tied, got %s", winner.ID)
	}
}

func TestGetWinner_EliminationVictory(t *testing.T) {
	// Only Player A remains (B is eliminated)
	g := createTestGameState(5, map[string]int{
		"A": 2,
		"B": 0,
	}, map[string]bool{"B": true})

	winner := g.GetWinner()
	if winner == nil {
		t.Fatal("Expected a winner by elimination")
	}
	if winner.ID != "A" {
		t.Errorf("Expected player A to win by elimination, got %s", winner.ID)
	}
}

func TestGetWinner_ReturnsHighestCityCount(t *testing.T) {
	// Three players: A has 7, B has 6, C has 5. Threshold is 5. A should win.
	g := createTestGameState(5, map[string]int{
		"A": 7,
		"B": 6,
		"C": 5,
	}, nil)

	winner := g.GetWinner()
	if winner == nil {
		t.Fatal("Expected a winner")
	}
	if winner.ID != "A" {
		t.Errorf("Expected player A to win (highest cities), got %s", winner.ID)
	}
}

func TestCountCities(t *testing.T) {
	g := &GameState{
		Territories: map[string]*Territory{
			"t1": {ID: "t1", Owner: "A", HasCity: true},
			"t2": {ID: "t2", Owner: "A", HasCity: false},
			"t3": {ID: "t3", Owner: "A", HasCity: true},
			"t4": {ID: "t4", Owner: "B", HasCity: true},
		},
	}

	if count := g.CountCities("A"); count != 2 {
		t.Errorf("Expected player A to have 2 cities, got %d", count)
	}
	if count := g.CountCities("B"); count != 1 {
		t.Errorf("Expected player B to have 1 city, got %d", count)
	}
	if count := g.CountCities("C"); count != 0 {
		t.Errorf("Expected player C to have 0 cities, got %d", count)
	}
}
