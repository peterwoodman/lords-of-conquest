package ai

import (
	"testing"

	"lords-of-conquest/internal/game"
)

func conquestMap(players ...string) *testMap {
	m := newTestMap(players...)
	m.s.Phase = game.PhaseConquest
	return m
}

func TestPlanAttack_SkipsUnwinnableAttack(t *testing.T) {
	m := conquestMap("A", "B")
	m.add("home", "A", game.ResourceNone)
	m.add("fort", "B", game.ResourceGold).HasWeapon = true
	m.link("home", "fort")
	for _, d := range []game.AIDifficulty{game.AIEasy, game.AIMedium, game.AIHard} {
		if got := PlanAttack(m.s, "A", d, seeded(1)); got != nil {
			t.Fatalf("%s: expected no attack, got %+v", d, got)
		}
	}
}

func TestPlanAttack_PrefersStockpile(t *testing.T) {
	m := conquestMap("A", "B")
	m.add("home", "A", game.ResourceNone).HasWeapon = true
	m.add("plain", "B", game.ResourceTimber)
	m.add("stock", "B", game.ResourceNone)
	m.link("plain", "home", "stock")
	m.s.Players["B"].StockpileTerritory = "stock"
	got := PlanAttack(m.s, "A", game.AIHard, seeded(1))
	if got == nil || got.Target != "stock" {
		t.Fatalf("expected attack on stockpile, got %+v", got)
	}
}

// reinforcementMap: far(A, horse) - home(A) - target(B). Without the horse the attack is
// a 1 v 1 coin flip; with it, a sure win.
func reinforcementMap() *testMap {
	m := conquestMap("A", "B")
	m.add("far", "A", game.ResourceNone).HasHorse = true
	m.add("home", "A", game.ResourceNone)
	m.add("target", "B", game.ResourceTimber)
	m.link("far", "home", "target")
	m.s.Players["A"].StockpileTerritory = "far"
	return m
}

func TestPlanAttack_BringsHorseToWin(t *testing.T) {
	m := reinforcementMap()
	got := PlanAttack(m.s, "A", game.AIHard, seeded(1))
	if got == nil || got.Target != "target" || got.Brought == nil ||
		got.Brought.UnitType != game.UnitHorse || got.Brought.FromTerritory != "far" {
		t.Fatalf("expected attack bringing the horse from far, got %+v", got)
	}
	if got.WinChance != 1 {
		t.Fatalf("expected a certain win, got %v", got.WinChance)
	}
}

func TestPlanAttack_EasyDoesNotBringUnits(t *testing.T) {
	m := reinforcementMap()
	for seed := int64(0); seed < 20; seed++ {
		if got := PlanAttack(m.s, "A", game.AIEasy, seeded(seed)); got != nil {
			t.Fatalf("seed %d: easy should not take a coin-flip attack, got %+v", seed, got)
		}
	}
}

func TestPlanAttack_CountsDefenderAllies(t *testing.T) {
	m := conquestMap("A", "B", "C")
	m.add("home", "A", game.ResourceNone).HasWeapon = true
	m.add("target", "B", game.ResourceTimber)
	m.add("ally", "C", game.ResourceNone).HasWeapon = true
	m.link("home", "target", "ally")

	m.s.Players["C"].Alliance = game.AllianceNeutral
	if got := PlanAttack(m.s, "A", game.AIHard, seeded(1)); got == nil {
		t.Fatal("expected attack when the neighbour is neutral")
	}

	m.s.Players["C"].Alliance = game.AllianceDefender
	got := PlanAttack(m.s, "A", game.AIHard, seeded(1))
	if got != nil {
		t.Fatalf("expected no attack against a defended territory, got %+v", got)
	}
}

func TestPlanAttack_NoAttacksRemaining(t *testing.T) {
	m := reinforcementMap()
	m.s.Players["A"].AttacksRemaining = 0
	if got := PlanAttack(m.s, "A", game.AIHard, seeded(1)); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}
