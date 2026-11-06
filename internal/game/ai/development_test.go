package ai

import (
	"testing"

	"lords-of-conquest/internal/game"
)

func developmentMap(players ...string) *testMap {
	m := newTestMap(players...)
	m.s.Phase = game.PhaseDevelopment
	return m
}

func TestPlanDevelopment_CityWhereItDoublesMost(t *testing.T) {
	// hub borders gold and coal; edge only borders gold.
	m := developmentMap("A")
	m.add("gold", "A", game.ResourceGold)
	m.add("coal", "A", game.ResourceCoal)
	m.add("hub", "A", game.ResourceNone)
	m.add("edge", "A", game.ResourceNone)
	m.link("gold", "hub", "coal")
	m.link("edge", "gold")
	*m.s.Players["A"].Stockpile = game.Stockpile{Coal: 1, Gold: 1, Iron: 1, Timber: 1}

	got := PlanDevelopment(m.s, "A", game.AIHard, seeded(1))
	if got == nil || got.Type != game.BuildCity || got.Territory != "hub" || got.UseGold {
		t.Fatalf("expected a city on hub paid with resources, got %+v", got)
	}
}

// weaponMap: enemy(B, weapon) - front(A) - mid(A) - back(A, stockpile).
// B attacks front with 4 against 2.
func weaponMap() *testMap {
	m := developmentMap("A", "B")
	m.add("enemy", "B", game.ResourceNone).HasWeapon = true
	m.add("front", "A", game.ResourceNone)
	m.add("mid", "A", game.ResourceNone)
	m.add("back", "A", game.ResourceNone)
	m.link("enemy", "front", "mid", "back")
	m.s.Players["A"].StockpileTerritory = "back"
	return m
}

func TestPlanDevelopment_WeaponOnThreatenedFront(t *testing.T) {
	m := weaponMap()
	*m.s.Players["A"].Stockpile = game.Stockpile{Coal: 1, Iron: 1}
	got := PlanDevelopment(m.s, "A", game.AIHard, seeded(1))
	if got == nil || got.Type != game.BuildWeapon || got.Territory != "front" {
		t.Fatalf("expected a weapon on the front, got %+v", got)
	}
	if m.s.Territories["front"].HasWeapon {
		t.Fatal("planning must not leave the weapon built")
	}
}

func TestPlanDevelopment_NothingUsefulToBuild(t *testing.T) {
	m := developmentMap("A")
	m.add("home", "A", game.ResourceNone)
	m.add("field", "A", game.ResourceNone)
	m.link("home", "field")
	*m.s.Players["A"].Stockpile = game.Stockpile{Coal: 1, Iron: 1}
	for _, d := range []game.AIDifficulty{game.AIMedium, game.AIHard} {
		if got := PlanDevelopment(m.s, "A", d, seeded(1)); got != nil {
			t.Fatalf("%s: expected nothing built, got %+v", d, got)
		}
	}
}

func TestPlanDevelopment_HardSavesForNearlyAffordableCity(t *testing.T) {
	m := weaponMap()
	// One timber short of a city; a weapon would spend the coal and iron it needs.
	*m.s.Players["A"].Stockpile = game.Stockpile{Coal: 1, Gold: 1, Iron: 1}

	if got := PlanDevelopment(m.s, "A", game.AIMedium, seeded(1)); got == nil || got.Type != game.BuildWeapon {
		t.Fatalf("expected medium to build a weapon, got %+v", got)
	}
	if got := PlanDevelopment(m.s, "A", game.AIHard, seeded(1)); got != nil {
		t.Fatalf("expected hard to save for a city, got %+v", got)
	}
}
