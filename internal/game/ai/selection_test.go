package ai

import (
	"testing"

	"lords-of-conquest/internal/game"
)

// selectionMap: A owns home. home borders gold, grass, timber and plain.
// farGold is unclaimed and not adjacent to anything A owns.
func selectionMap() *testMap {
	m := newTestMap("A", "B")
	m.add("home", "A", game.ResourceNone)
	m.add("gold", "", game.ResourceGold)
	m.add("grass", "", game.ResourceGrassland)
	m.add("timber", "", game.ResourceTimber)
	m.add("plain", "", game.ResourceNone)
	m.add("farGold", "", game.ResourceGold)
	m.add("farPlain", "", game.ResourceNone)
	m.link("home", "gold")
	m.link("home", "grass")
	m.link("home", "timber")
	m.link("home", "plain")
	m.link("farGold", "farPlain")
	return m
}

func TestChooseTerritory_PrefersAdjacentGold(t *testing.T) {
	m := selectionMap()
	if got := ChooseTerritory(m.s, "A", game.AIHard, seeded(1)); got != "gold" {
		t.Fatalf("expected adjacent gold, got %s", got)
	}
}

func TestChooseTerritory_MediumStaysAdjacent(t *testing.T) {
	m := selectionMap()
	for seed := int64(0); seed < 100; seed++ {
		got := ChooseTerritory(m.s, "A", game.AIMedium, seeded(seed))
		if ownedNeighbours(m.s, "A", m.s.Territories[got]) == 0 {
			t.Fatalf("seed %d: medium picked non-adjacent %s", seed, got)
		}
	}
}

func TestChooseTerritory_ResourceOrderAmongAdjacent(t *testing.T) {
	m := selectionMap()
	m.s.Territories["gold"].Owner = "B"
	if got := ChooseTerritory(m.s, "A", game.AIHard, seeded(1)); got != "grass" {
		t.Fatalf("expected adjacent grassland after gold, got %s", got)
	}

	m.s.Territories["grass"].Owner = "B"
	if got := ChooseTerritory(m.s, "A", game.AIHard, seeded(1)); got != "timber" {
		t.Fatalf("expected adjacent timber after grassland, got %s", got)
	}
}

func TestChooseTerritory_AdjacencyBeatsDistantGold(t *testing.T) {
	m := selectionMap()
	for _, id := range []string{"gold", "grass", "timber"} {
		m.s.Territories[id].Owner = "B"
	}
	for _, d := range []game.AIDifficulty{game.AIMedium, game.AIHard} {
		for seed := int64(0); seed < 50; seed++ {
			if got := ChooseTerritory(m.s, "A", d, seeded(seed)); got != "plain" {
				t.Fatalf("%s seed %d: expected adjacent plain over distant gold, got %s", d, seed, got)
			}
		}
	}
}

func TestChooseTerritory_FirstPickTakesGold(t *testing.T) {
	m := selectionMap()
	m.s.Territories["home"].Owner = ""
	got := ChooseTerritory(m.s, "A", game.AIHard, seeded(1))
	if m.s.Territories[got].Resource != game.ResourceGold {
		t.Fatalf("expected a gold territory for the first pick, got %s", got)
	}
}

func TestChooseTerritory_PrefersJoiningRegions(t *testing.T) {
	// A owns west and east, which are separate. bridge borders both; side borders only west.
	m := newTestMap("A")
	m.add("west", "A", game.ResourceNone)
	m.add("east", "A", game.ResourceNone)
	m.add("bridge", "", game.ResourceTimber)
	m.add("side", "", game.ResourceTimber)
	m.link("west", "bridge", "east")
	m.link("west", "side")
	if got := ChooseTerritory(m.s, "A", game.AIHard, seeded(1)); got != "bridge" {
		t.Fatalf("expected bridge joining both regions, got %s", got)
	}
}

func TestChooseTerritory_NoneLeft(t *testing.T) {
	m := newTestMap("A")
	m.add("home", "A", game.ResourceGold)
	if got := ChooseTerritory(m.s, "A", game.AIEasy, seeded(1)); got != "" {
		t.Fatalf("expected no choice, got %s", got)
	}
}
