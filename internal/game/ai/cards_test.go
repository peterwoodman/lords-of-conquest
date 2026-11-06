package ai

import (
	"slices"
	"testing"

	"lords-of-conquest/internal/game"
)

func attackCard(id string, effect game.CardEffect, value int) game.CombatCard {
	return game.CombatCard{ID: id, CardType: game.CardTypeAttack, Effect: effect, Value: value}
}

func defenseCard(id string, effect game.CardEffect, value int) game.CombatCard {
	return game.CombatCard{ID: id, CardType: game.CardTypeDefense, Effect: effect, Value: value}
}

func cardMap() *testMap {
	m := conquestMap("A", "B")
	m.s.Settings.CombatMode = game.CombatModeCards
	m.add("home", "A", game.ResourceNone)
	m.add("target", "B", game.ResourceTimber)
	m.link("home", "target")
	return m
}

func TestChooseAttackCards_PlaysFewestNeeded(t *testing.T) {
	m := cardMap()
	a := m.s.Players["A"]
	a.AttackCards = []game.CombatCard{
		attackCard("skirmish", game.EffectSkirmish, 1),
		attackCard("assault", game.EffectAssault, 5),
		attackCard("advance", game.EffectAdvance, 2),
	}
	target := m.s.Territories["target"]

	ids, attack := chooseAttackCards(m.s, a, target, nil, 2, 3, 0.6, game.AIHard)
	if !slices.Equal(ids, []string{"assault"}) || attack != 7 {
		t.Fatalf("expected only assault for 7 attack, got %v (%d)", ids, attack)
	}

	ids, _ = chooseAttackCards(m.s, a, target, nil, 2, 3, 0.6, game.AIEasy)
	if len(ids) != 3 {
		t.Fatalf("expected easy to play its whole hand, got %v", ids)
	}
}

func TestPlanAttack_UsesCardToWin(t *testing.T) {
	m := cardMap()
	m.s.Players["A"].AttackCards = []game.CombatCard{attackCard("skirmish", game.EffectSkirmish, 1)}
	got := PlanAttack(m.s, "A", game.AIHard, seeded(1))
	if got == nil || got.Target != "target" || !slices.Equal(got.AttackCards, []string{"skirmish"}) {
		t.Fatalf("expected an attack on target with skirmish, got %+v", got)
	}

	m.s.Players["A"].AttackCards = nil
	if got := PlanAttack(m.s, "A", game.AIMedium, seeded(1)); got != nil {
		t.Fatalf("expected medium to pass on a coin flip without cards, got %+v", got)
	}
}

func TestPlanAttack_ExpectsDefenseCards(t *testing.T) {
	m := cardMap()
	m.s.Territories["home"].HasHorse = true // 2 attack against 1 defence
	if got := PlanAttack(m.s, "A", game.AIMedium, seeded(1)); got == nil {
		t.Fatal("expected attack against a defender with no cards")
	}
	m.s.Players["B"].DefenseCards = []game.CombatCard{defenseCard("d1", game.EffectFortify, 1)}
	if got := PlanAttack(m.s, "A", game.AIMedium, seeded(1)); got != nil {
		t.Fatalf("expected no attack once the defender holds a card, got %+v", got)
	}
}

func TestChooseDefenseCards(t *testing.T) {
	m := cardMap()
	b := m.s.Players["B"]
	b.StockpileTerritory = "target"
	b.DefenseCards = []game.CombatCard{
		defenseCard("fortify", game.EffectFortify, 1),
		defenseCard("barricade", game.EffectBarricade, 2),
	}

	if got := ChooseDefenseCards(m.s, "B", "target", 4, 3, 0, game.AIMedium, seeded(1)); !slices.Equal(got, []string{"barricade"}) {
		t.Fatalf("expected barricade to hold the stockpile, got %v", got)
	}
	if got := ChooseDefenseCards(m.s, "B", "target", 10, 3, 0, game.AIMedium, seeded(1)); got != nil {
		t.Fatalf("expected no cards wasted on a hopeless defence, got %v", got)
	}

	b.StockpileTerritory = ""
	if got := ChooseDefenseCards(m.s, "B", "target", 4, 3, 0, game.AIMedium, seeded(1)); got != nil {
		t.Fatalf("expected medium to keep cards for valuable territory, got %v", got)
	}

	b.StockpileTerritory = "target"
	b.Stockpile.Gold = game.BribeCost
	b.DefenseCards = append(b.DefenseCards, defenseCard("bribe", game.EffectBribe, 0))
	if got := ChooseDefenseCards(m.s, "B", "target", 4, 3, 0, game.AIHard, seeded(1)); !slices.Equal(got, []string{"bribe"}) {
		t.Fatalf("expected bribe, got %v", got)
	}
}

func TestPlanCardPurchases_KeepsCityResources(t *testing.T) {
	m := developmentMap("A")
	m.s.Settings.CombatMode = game.CombatModeCards
	m.add("home", "A", game.ResourceNone)
	*m.s.Players["A"].Stockpile = game.Stockpile{Coal: 3, Gold: 2}

	got := PlanCardPurchases(m.s, "A", game.AIMedium)
	if len(got) != 1 || got[0].Resource != game.ResourceCoal {
		t.Fatalf("expected one card bought with spare coal, got %+v", got)
	}
	if got := PlanCardPurchases(m.s, "A", game.AIEasy); len(got) != 2 {
		t.Fatalf("expected easy to spend freely, got %+v", got)
	}
}
