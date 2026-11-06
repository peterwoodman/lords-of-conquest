package ai

import (
	"slices"
	"testing"

	"lords-of-conquest/internal/game"
)

// allianceMap: A attacks B's target; C borders the target from c1.
// A has cities on a1 and a2, so A leads.
func allianceMap() *testMap {
	m := conquestMap("A", "B", "C")
	m.add("a1", "A", game.ResourceNone).HasCity = true
	m.add("a2", "A", game.ResourceNone).HasCity = true
	m.add("target", "B", game.ResourceNone)
	m.add("c1", "C", game.ResourceNone)
	m.add("c2", "C", game.ResourceNone)
	m.link("a1", "target", "c1")
	m.link("a2", "a1")
	m.link("c1", "c2")
	m.s.Players["C"].StockpileTerritory = "c2"
	for _, p := range m.s.Players {
		p.Alliance = game.AllianceAsk
	}
	return m
}

func TestVoteAlliance(t *testing.T) {
	m := allianceMap()
	if got := VoteAlliance(m.s, "C", "A", "target", game.AIEasy); got != SideNeutral {
		t.Errorf("easy: expected neutral, got %s", got)
	}
	if got := VoteAlliance(m.s, "C", "A", "target", game.AIMedium); got != SideDefender {
		t.Errorf("medium: expected to resist the city leader, got %s", got)
	}
	if got := VoteAlliance(m.s, "C", "A", "target", game.AIHard); got != SideDefender {
		t.Errorf("hard: expected to resist the leader, got %s", got)
	}

	// B now leads: Hard helps A take B's land.
	m.s.Territories["a1"].HasCity = false
	m.s.Territories["a2"].HasCity = false
	m.s.Territories["target"].HasCity = true
	if got := VoteAlliance(m.s, "C", "A", "target", game.AIHard); got != SideAttacker {
		t.Errorf("hard: expected to attack the leader, got %s", got)
	}

	// Land next to C's stockpile is always defended.
	m.s.Players["C"].StockpileTerritory = "c1"
	if got := VoteAlliance(m.s, "C", "A", "target", game.AIHard); got != SideDefender {
		t.Errorf("hard: expected to defend next to its stockpile, got %s", got)
	}
}

func TestResolveAllies_CPUsVoteHumansAreAsked(t *testing.T) {
	m := allianceMap()
	m.s.Players["C"].AIDifficulty = game.AIHard
	m.s.Players["D"] = &game.Player{ID: "D", Name: "D", Stockpile: game.NewStockpile(), Alliance: game.AllianceAsk}
	m.add("d1", "D", game.ResourceNone)
	m.link("d1", "target")

	att, def, ask := ResolveAllies(m.s, "A", m.s.Territories["target"])
	if len(att) != 0 || !slices.Equal(def, []string{"C"}) || !slices.Equal(ask, []string{"D"}) {
		t.Fatalf("expected C defending and D asked, got attacker %v, defender %v, ask %v", att, def, ask)
	}
}
