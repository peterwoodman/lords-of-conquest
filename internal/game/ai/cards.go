package ai

import (
	"math/rand"
	"sort"

	"lords-of-conquest/internal/game"
)

// expectedCardStrength is the assumed bonus from each card an opponent holds or plays,
// for up to maxCountedCards cards. Hands are public by count only.
const (
	expectedCardStrength = 1
	maxCountedCards      = 3
)

// cardSpendCost is how much playing one card counts against an attack.
const cardSpendCost = 0.5

// estimatedDefenseCards is the defence bonus an attacker should expect from the cards
// the target's owner holds. Easy ignores defence cards.
func estimatedDefenseCards(s *game.GameState, target *game.Territory, d game.AIDifficulty) int {
	owner := s.Players[target.Owner]
	if owner == nil || game.ParseAIDifficulty(string(d)) == game.AIEasy {
		return 0
	}
	return expectedCardStrength * min(len(owner.DefenseCards), maxCountedCards)
}

// attackCardBonus estimates the strength an attack card adds to this attack.
func attackCardBonus(s *game.GameState, attackerID string, target *game.Territory, baseAttack int, c game.CombatCard) int {
	bonus := c.Value
	for _, n := range neighbours(s, target) {
		if n.Owner != attackerID {
			continue
		}
		switch c.Effect {
		case game.EffectRallyCavalry:
			if n.HasHorse {
				bonus++
			}
		case game.EffectArsenal:
			if n.HasWeapon {
				bonus++
			}
		case game.EffectNavalBombardment:
			bonus += 2 * n.TotalBoats()
		}
	}
	if c.Effect == game.EffectDoubleAttack {
		bonus += baseAttack
	}
	return bonus
}

// chooseAttackCards picks the attack cards to play and returns them with the resulting
// attack strength. Easy plays its whole hand; Medium and Hard play the strongest cards
// until the attack reaches minChance, plus Safe Retreat when bringing a unit and Blitz
// when playing several cards.
func chooseAttackCards(s *game.GameState, player *game.Player, target *game.Territory, brought *game.BroughtUnit,
	attack, defense int, minChance float64, d game.AIDifficulty) ([]string, int) {

	hand := player.AttackCards
	if game.ParseAIDifficulty(string(d)) == game.AIEasy {
		ids := make([]string, 0, len(hand))
		for _, c := range hand {
			ids = append(ids, c.ID)
			attack += attackCardBonus(s, player.ID, target, attack, c)
		}
		return ids, attack
	}

	base := attack
	type scored struct {
		card  game.CombatCard
		bonus int
	}
	var strength []scored
	var safeRetreat, blitz *game.CombatCard
	for i, c := range hand {
		switch c.Effect {
		case game.EffectSafeRetreat:
			safeRetreat = &hand[i]
		case game.EffectBlitz:
			blitz = &hand[i]
		default:
			if b := attackCardBonus(s, player.ID, target, base, c); b > 0 {
				strength = append(strength, scored{c, b})
			}
		}
	}
	sort.SliceStable(strength, func(i, j int) bool { return strength[i].bonus > strength[j].bonus })

	var ids []string
	for _, sc := range strength {
		if winProbability(s, attack, defense) >= minChance {
			break
		}
		ids = append(ids, sc.card.ID)
		attack += sc.bonus
	}
	chance := winProbability(s, attack, defense)
	if safeRetreat != nil && brought != nil && chance < 1 {
		ids = append(ids, safeRetreat.ID)
	}
	if blitz != nil && len(ids) >= 2 {
		ids = append(ids, blitz.ID)
	}
	return ids, attack
}

// ChooseDefenseCards picks the defence cards a CPU plays when targetID is attacked with
// baseAttack against baseDefense and attackCardCount attack cards.
func ChooseDefenseCards(s *game.GameState, defenderID, targetID string, baseAttack, baseDefense, attackCardCount int,
	d game.AIDifficulty, rng *rand.Rand) []string {

	defender := s.Players[defenderID]
	target := s.Territories[targetID]
	if defender == nil || target == nil || len(defender.DefenseCards) == 0 {
		return nil
	}
	hand := defender.DefenseCards
	difficulty := game.ParseAIDifficulty(string(d))
	if difficulty == game.AIEasy {
		return []string{hand[rng.Intn(len(hand))].ID}
	}

	value := territoryValue(target)
	if defender.StockpileTerritory == targetID {
		value += stockpileValue
	}
	minValue := 6.0
	if difficulty == game.AIHard {
		minValue = 3
	}
	if value < minValue {
		return nil
	}

	attack := baseAttack + expectedCardStrength*min(attackCardCount, maxCountedCards)
	lossChance := func(defense int) float64 { return winProbability(s, attack, defense) }
	if lossChance(baseDefense) == 0 {
		return nil
	}

	for _, c := range hand {
		if c.Effect == game.EffectBribe && defender.Stockpile.Gold >= game.BribeCost {
			return []string{c.ID}
		}
	}

	type scored struct {
		card  game.CombatCard
		bonus int
	}
	var strength []scored
	var counter *game.CombatCard
	for i, c := range hand {
		bonus := c.Value
		switch c.Effect {
		case game.EffectDoubleDefense:
			bonus = baseDefense
		case game.EffectShieldWall:
			if attackCardCount > 0 {
				bonus = expectedCardStrength
			}
		case game.EffectSabotage:
			bonus = 3 * attackingWeapons(s, target, defenderID)
		case game.EffectCounterAttack:
			counter = &hand[i]
		}
		if bonus > 0 {
			strength = append(strength, scored{c, bonus})
		}
	}
	sort.SliceStable(strength, func(i, j int) bool { return strength[i].bonus > strength[j].bonus })

	defense := baseDefense
	var ids []string
	for _, sc := range strength {
		if lossChance(defense) <= 0.25 {
			break
		}
		ids = append(ids, sc.card.ID)
		defense += sc.bonus
	}
	// Cards that can't change the outcome are better kept
	if lossChance(defense) >= lossChance(baseDefense) {
		return nil
	}
	if counter != nil {
		ids = append(ids, counter.ID)
	}
	return ids
}

// attackingWeapons counts weapons next to target owned by anyone but the defender,
// which Sabotage would cancel.
func attackingWeapons(s *game.GameState, target *game.Territory, defenderID string) int {
	count := 0
	for _, n := range neighbours(s, target) {
		if n.Owner != "" && n.Owner != defenderID && n.HasWeapon {
			count++
		}
	}
	return count
}

// CardPurchase is one card a CPU wants to buy.
type CardPurchase struct {
	Type     game.CardType
	Resource game.ResourceType
}

// maxCardsPerTurn caps how many cards a CPU buys in one development phase.
const maxCardsPerTurn = 2

// PlanCardPurchases decides which cards to buy with the CPU's remaining resources.
// Easy spends its most plentiful resource on attack cards. Medium and Hard keep one of
// each city resource (and Bribe gold when holding a Bribe) and buy the type their
// position needs.
func PlanCardPurchases(s *game.GameState, playerID string, d game.AIDifficulty) []CardPurchase {
	player := s.Players[playerID]
	if player == nil || player.Stockpile == nil || s.Settings.CombatMode != game.CombatModeCards {
		return nil
	}
	difficulty := game.ParseAIDifficulty(string(d))
	stock := *player.Stockpile
	attackHand, defenseHand := len(player.AttackCards), len(player.DefenseCards)

	reserve := map[game.ResourceType]int{}
	if difficulty != game.AIEasy {
		for _, r := range []game.ResourceType{game.ResourceCoal, game.ResourceGold, game.ResourceIron, game.ResourceTimber} {
			reserve[r] = 1
		}
		for _, c := range player.DefenseCards {
			if c.Effect == game.EffectBribe {
				reserve[game.ResourceGold] = game.BribeCost
			}
		}
	}

	preferDefense := false
	if difficulty != game.AIEasy {
		exposure := newDefenceModel(s, playerID).expectedLoss(player.StockpileTerritory)
		preferDefense = defenseHand < attackHand || (exposure >= 3 && defenseHand <= attackHand)
	}

	var purchases []CardPurchase
	for len(purchases) < maxCardsPerTurn {
		resource, spare := game.ResourceNone, 0
		// Gold last: it is the most useful resource for building
		for _, r := range []game.ResourceType{game.ResourceCoal, game.ResourceIron, game.ResourceTimber, game.ResourceGold} {
			if have := stock.Get(r) - reserve[r]; have >= game.CardBuyCost && have > spare {
				resource, spare = r, have
			}
		}
		if resource == game.ResourceNone {
			break
		}

		cardType := game.CardTypeAttack
		if preferDefense {
			cardType = game.CardTypeDefense
		}
		if cardType == game.CardTypeAttack && attackHand >= game.MaxAttackCards {
			cardType = game.CardTypeDefense
		}
		if cardType == game.CardTypeDefense && defenseHand >= game.MaxDefenseCards {
			cardType = game.CardTypeAttack
			if attackHand >= game.MaxAttackCards {
				break
			}
		}

		purchases = append(purchases, CardPurchase{Type: cardType, Resource: resource})
		stock.Remove(resource, game.CardBuyCost)
		if cardType == game.CardTypeAttack {
			attackHand++
		} else {
			defenseHand++
		}
		if difficulty != game.AIEasy {
			preferDefense = defenseHand < attackHand
		}
	}
	return purchases
}
