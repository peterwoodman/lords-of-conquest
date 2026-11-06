package ai

import (
	"testing"

	"lords-of-conquest/internal/game"
)

// frontMap: enemy(B, weapon) - front(A) - mid(A) - back(A).
// B attacks front with 1 + 3 = 4; front defends with 1 + mid's 1 = 2.
func frontMap() *testMap {
	m := newTestMap("A", "B")
	m.add("enemy", "B", game.ResourceNone).HasWeapon = true
	m.add("front", "A", game.ResourceTimber)
	m.add("mid", "A", game.ResourceNone)
	m.add("back", "A", game.ResourceNone)
	m.link("enemy", "front", "mid", "back")
	return m
}

func TestPlanShipment_MovesThreatenedStockpile(t *testing.T) {
	m := frontMap()
	m.s.Players["A"].StockpileTerritory = "front"
	got := PlanShipment(m.s, "A", game.AIHard, seeded(1))
	if got.Kind != ShipStockpile || got.To == "front" {
		t.Fatalf("expected stockpile moved away from front, got %+v", got)
	}
}

func TestPlanShipment_MovesWeaponToDefendFront(t *testing.T) {
	m := frontMap()
	m.s.Players["A"].StockpileTerritory = "back"
	m.s.Territories["back"].HasWeapon = true
	got := PlanShipment(m.s, "A", game.AIHard, seeded(1))
	want := ShipmentDecision{Kind: ShipUnit, UnitType: "weapon", From: "back", To: "mid"}
	if got != want {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
	if !m.s.Territories["back"].HasWeapon || m.s.Territories["mid"].HasWeapon {
		t.Fatal("planning must leave units where they were")
	}
}

func TestPlanShipment_EasyNeverMovesUnits(t *testing.T) {
	m := frontMap()
	m.s.Players["A"].StockpileTerritory = "back"
	m.s.Territories["back"].HasWeapon = true
	for seed := int64(0); seed < 50; seed++ {
		if got := PlanShipment(m.s, "A", game.AIEasy, seeded(seed)); got.Kind == ShipUnit {
			t.Fatalf("seed %d: easy moved a unit: %+v", seed, got)
		}
	}
}

func TestPlanShipment_SkipsWhenSafe(t *testing.T) {
	m := frontMap()
	m.s.Territories["enemy"].HasWeapon = false
	m.s.Territories["front"].HasWeapon = true
	m.s.Players["A"].StockpileTerritory = "back"
	if got := PlanShipment(m.s, "A", game.AIHard, seeded(1)); got.Kind != ShipSkip {
		t.Fatalf("expected skip, got %+v", got)
	}
}

func TestChooseStockpile_AvoidsFront(t *testing.T) {
	m := frontMap()
	for _, d := range []game.AIDifficulty{game.AIMedium, game.AIHard} {
		for seed := int64(0); seed < 30; seed++ {
			if got := ChooseStockpile(m.s, "A", d, seeded(seed)); got == "front" {
				t.Fatalf("%s seed %d: placed stockpile on the threatened front", d, seed)
			}
		}
	}
}
