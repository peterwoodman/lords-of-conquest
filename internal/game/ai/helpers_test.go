package ai

import (
	"math/rand"
	"testing"

	"lords-of-conquest/internal/game"
)

// testMap builds game states from a list of territories. Adjacency is made symmetric.
type testMap struct {
	s *game.GameState
}

func newTestMap(players ...string) *testMap {
	s := &game.GameState{
		Round:       1,
		Settings:    game.Settings{VictoryCities: 3, ChanceLevel: game.ChanceMedium},
		Players:     make(map[string]*game.Player),
		Territories: make(map[string]*game.Territory),
		WaterBodies: make(map[string]*game.WaterBody),
		PlayerOrder: players,
	}
	for _, id := range players {
		s.Players[id] = &game.Player{ID: id, Name: id, Stockpile: game.NewStockpile(), AttacksRemaining: 2, IsAI: true}
	}
	if len(players) > 0 {
		s.CurrentPlayerID = players[0]
	}
	return &testMap{s: s}
}

// add creates a territory owned by owner ("" for unclaimed) with the given resource.
func (m *testMap) add(id, owner string, resource game.ResourceType) *game.Territory {
	t := &game.Territory{ID: id, Name: id, Owner: owner, Resource: resource, Boats: map[string]int{}}
	m.s.Territories[id] = t
	return t
}

// link makes territories adjacent to each other in a chain.
func (m *testMap) link(ids ...string) {
	for i := 1; i < len(ids); i++ {
		a, b := m.s.Territories[ids[i-1]], m.s.Territories[ids[i]]
		a.Adjacent = append(a.Adjacent, b.ID)
		b.Adjacent = append(b.Adjacent, a.ID)
	}
}

func seeded(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

func TestChoose_HardPicksBest(t *testing.T) {
	opts := []option{{"a", 1}, {"b", 5}, {"c", 3}}
	for seed := int64(0); seed < 20; seed++ {
		if got := choose(opts, game.AIHard, seeded(seed)); got != "b" {
			t.Fatalf("seed %d: expected b, got %s", seed, got)
		}
	}
}

func TestChoose_MediumIgnoresOptionsFarBelowBest(t *testing.T) {
	opts := []option{{"best", 10}, {"close", 9}, {"far", 2}}
	seen := map[string]bool{}
	for seed := int64(0); seed < 200; seed++ {
		seen[choose(opts, game.AIMedium, seeded(seed))] = true
	}
	if seen["far"] {
		t.Fatal("medium picked an option far below the best")
	}
	if !seen["best"] || !seen["close"] {
		t.Fatalf("expected medium to vary between best and close, saw %v", seen)
	}
}

func TestChoose_EasySometimesPicksWorst(t *testing.T) {
	opts := []option{{"a", 1}, {"b", 5}, {"c", 3}, {"d", 4}}
	seen := map[string]bool{}
	for seed := int64(0); seed < 200; seed++ {
		seen[choose(opts, game.AIEasy, seeded(seed))] = true
	}
	if !seen["a"] {
		t.Fatal("expected easy to occasionally pick the worst option")
	}
}

func TestChoose_EmptyReturnsNothing(t *testing.T) {
	if got := choose(nil, game.AIHard, seeded(1)); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
