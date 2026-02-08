package game

import (
	"fmt"
	"math/rand"
)

// CombatMode determines which combat system is used.
type CombatMode int

const (
	CombatModeClassic CombatMode = iota // Default: original strength comparison
	CombatModeCards                     // Card-based combat with bluffing
)

// String returns the combat mode name.
func (m CombatMode) String() string {
	switch m {
	case CombatModeCards:
		return "cards"
	default:
		return "classic"
	}
}

// ParseCombatMode converts a string to CombatMode.
func ParseCombatMode(s string) CombatMode {
	switch s {
	case "cards":
		return CombatModeCards
	default:
		return CombatModeClassic
	}
}

// CardType is either attack or defense.
type CardType string

const (
	CardTypeAttack  CardType = "attack"
	CardTypeDefense CardType = "defense"
)

// CardRarity represents the rarity tier of a card.
type CardRarity string

const (
	RarityCommon    CardRarity = "common"
	RarityUncommon  CardRarity = "uncommon"
	RarityRare      CardRarity = "rare"
	RarityUltraRare CardRarity = "ultra_rare"
)

// CardEffect identifies the specific effect of a card.
type CardEffect string

// Attack card effects
const (
	EffectSkirmish          CardEffect = "skirmish"           // +1 attack
	EffectAdvance           CardEffect = "advance"            // +2 attack
	EffectCharge            CardEffect = "charge"             // +3 attack
	EffectRallyCavalry      CardEffect = "rally_cavalry"      // +1 per adjacent horse
	EffectArsenal           CardEffect = "arsenal"            // +1 per adjacent weapon
	EffectAssault           CardEffect = "assault"            // +5 attack
	EffectNavalBombardment  CardEffect = "naval_bombardment"  // +2 per adjacent boat
	EffectSafeRetreat       CardEffect = "safe_retreat"       // Brought unit returns home on loss
	EffectDoubleAttack      CardEffect = "double_attack"      // 2x base attack
	EffectBlitz             CardEffect = "blitz"              // Win = return played attack cards
)

// Defense card effects
const (
	EffectFortify       CardEffect = "fortify"        // +1 defense
	EffectBarricade     CardEffect = "barricade"      // +2 defense
	EffectEntrench      CardEffect = "entrench"       // +3 defense
	EffectShieldWall    CardEffect = "shield_wall"    // Negate one random attack card
	EffectSabotage      CardEffect = "sabotage"       // Negate weapon contributions to attack
	EffectBunker        CardEffect = "bunker"         // +5 defense
	EffectDoubleDefense CardEffect = "double_defense" // 2x base defense
	EffectCounterAttack CardEffect = "counter_attack" // Win = capture random attacker territory
	EffectBribe         CardEffect = "bribe"          // Pay 3 gold to auto-win defense
)

// MaxAttackCards is the maximum attack cards a player can hold.
const MaxAttackCards = 5

// MaxDefenseCards is the maximum defense cards a player can hold.
const MaxDefenseCards = 5

// CardBuyCost is the resource cost to purchase a card (2 of any single resource).
const CardBuyCost = 2

// BribeCost is the gold cost to activate the Bribe card effect.
const BribeCost = 3

// CombatCard represents a card in a player's hand.
type CombatCard struct {
	ID          string     `json:"id"`          // Unique instance ID (e.g., "atk_skirmish_1")
	Name        string     `json:"name"`        // Display name
	Description string     `json:"description"` // Effect description
	CardType    CardType   `json:"cardType"`    // attack or defense
	Rarity      CardRarity `json:"rarity"`      // Rarity tier
	Effect      CardEffect `json:"effect"`      // Effect identifier
	Value       int        `json:"value"`       // Flat bonus value (0 for special effects)
}

// cardTemplate defines a card type in the catalog.
type cardTemplate struct {
	Name        string
	Description string
	CardType    CardType
	Rarity      CardRarity
	Effect      CardEffect
	Value       int
	Weight      int // Relative draw weight within catalog (out of 1000)
}

// attackCardTemplates defines all attack card types with their draw weights.
var attackCardTemplates = []cardTemplate{
	// Common (50% total) -- 25% each
	{Name: "Skirmish", Description: "+1 attack", CardType: CardTypeAttack, Rarity: RarityCommon, Effect: EffectSkirmish, Value: 1, Weight: 250},
	{Name: "Advance", Description: "+2 attack", CardType: CardTypeAttack, Rarity: RarityCommon, Effect: EffectAdvance, Value: 2, Weight: 250},
	// Uncommon (30% total) -- 10% each
	{Name: "Charge", Description: "+3 attack", CardType: CardTypeAttack, Rarity: RarityUncommon, Effect: EffectCharge, Value: 3, Weight: 100},
	{Name: "Rally Cavalry", Description: "+1 attack per adjacent horse", CardType: CardTypeAttack, Rarity: RarityUncommon, Effect: EffectRallyCavalry, Value: 0, Weight: 100},
	{Name: "Arsenal", Description: "+1 attack per adjacent weapon", CardType: CardTypeAttack, Rarity: RarityUncommon, Effect: EffectArsenal, Value: 0, Weight: 100},
	// Rare (15% total) -- 5% each
	{Name: "Assault", Description: "+5 attack", CardType: CardTypeAttack, Rarity: RarityRare, Effect: EffectAssault, Value: 5, Weight: 50},
	{Name: "Naval Bombardment", Description: "Adjacent boats join attack (+2 each)", CardType: CardTypeAttack, Rarity: RarityRare, Effect: EffectNavalBombardment, Value: 0, Weight: 50},
	{Name: "Safe Retreat", Description: "Brought unit returns home on loss", CardType: CardTypeAttack, Rarity: RarityRare, Effect: EffectSafeRetreat, Value: 0, Weight: 50},
	// Ultra-Rare (5% total) -- 2.5% each
	{Name: "Double Attack", Description: "Double base attack strength", CardType: CardTypeAttack, Rarity: RarityUltraRare, Effect: EffectDoubleAttack, Value: 0, Weight: 25},
	{Name: "Blitz", Description: "Win: return all other played attack cards", CardType: CardTypeAttack, Rarity: RarityUltraRare, Effect: EffectBlitz, Value: 0, Weight: 25},
}

// defenseCardTemplates defines all defense card types with their draw weights.
var defenseCardTemplates = []cardTemplate{
	// Common (50% total) -- 25% each
	{Name: "Fortify", Description: "+1 defense", CardType: CardTypeDefense, Rarity: RarityCommon, Effect: EffectFortify, Value: 1, Weight: 250},
	{Name: "Barricade", Description: "+2 defense", CardType: CardTypeDefense, Rarity: RarityCommon, Effect: EffectBarricade, Value: 2, Weight: 250},
	// Uncommon (30% total) -- 10% each
	{Name: "Entrench", Description: "+3 defense", CardType: CardTypeDefense, Rarity: RarityUncommon, Effect: EffectEntrench, Value: 3, Weight: 100},
	{Name: "Shield Wall", Description: "Negate one random attack card", CardType: CardTypeDefense, Rarity: RarityUncommon, Effect: EffectShieldWall, Value: 0, Weight: 100},
	{Name: "Sabotage", Description: "Negate weapon contributions to attack", CardType: CardTypeDefense, Rarity: RarityUncommon, Effect: EffectSabotage, Value: 0, Weight: 100},
	// Rare (15% total) -- 7.5% each
	{Name: "Bunker", Description: "+5 defense", CardType: CardTypeDefense, Rarity: RarityRare, Effect: EffectBunker, Value: 5, Weight: 75},
	{Name: "Double Defense", Description: "Double base defense strength", CardType: CardTypeDefense, Rarity: RarityRare, Effect: EffectDoubleDefense, Value: 0, Weight: 75},
	// Ultra-Rare (5% total) -- 2.5% each
	{Name: "Counter-Attack", Description: "Win: capture random attacker territory", CardType: CardTypeDefense, Rarity: RarityUltraRare, Effect: EffectCounterAttack, Value: 0, Weight: 25},
	{Name: "Bribe", Description: "Pay 3 gold to auto-win defense", CardType: CardTypeDefense, Rarity: RarityUltraRare, Effect: EffectBribe, Value: 0, Weight: 25},
}

// cardIDCounter is used to generate unique card instance IDs.
var cardIDCounter int

// DrawCard randomly draws a card of the given type based on rarity weights.
func DrawCard(cardType CardType) CombatCard {
	var templates []cardTemplate
	if cardType == CardTypeAttack {
		templates = attackCardTemplates
	} else {
		templates = defenseCardTemplates
	}

	// Sum total weight
	totalWeight := 0
	for _, t := range templates {
		totalWeight += t.Weight
	}

	// Roll random number
	roll := rand.Intn(totalWeight)

	// Find the card
	cumulative := 0
	for _, t := range templates {
		cumulative += t.Weight
		if roll < cumulative {
			return newCardFromTemplate(t)
		}
	}

	// Fallback (shouldn't happen)
	return newCardFromTemplate(templates[0])
}

// newCardFromTemplate creates a card instance from a template.
func newCardFromTemplate(t cardTemplate) CombatCard {
	cardIDCounter++
	return CombatCard{
		ID:          fmt.Sprintf("%s_%s_%d", t.CardType, t.Effect, cardIDCounter),
		Name:        t.Name,
		Description: t.Description,
		CardType:    t.CardType,
		Rarity:      t.Rarity,
		Effect:      t.Effect,
		Value:       t.Value,
	}
}

// CardResolutionResult contains the output of card-based combat resolution.
type CardResolutionResult struct {
	FinalAttack       int          // Attack strength after all card effects
	FinalDefense      int          // Defense strength after all card effects
	AttackerWins      bool         // Result of the combat
	BribeActivated    bool         // True if Bribe auto-won defense
	CounterAttackTerr string       // Territory captured by counter-attack (empty if none)
	SafeRetreat       bool         // True if brought unit should be protected on loss
	BlitzReturn       []CombatCard // Cards returned to attacker by Blitz
	NegatedCards      []CombatCard // Attack cards negated by Shield Wall
	SabotageCount     int          // Number of weapons negated by Sabotage
}

// ResolveCards applies card effects and determines combat outcome.
// baseAttack/baseDefense are the pre-card strength values from CalculateAttack/DefenseStrength.
// The final comparison uses the game's existing ResolveCombat (ChanceLevel) formula.
func (g *GameState) ResolveCards(
	attackerID string,
	target *Territory,
	baseAttack, baseDefense int,
	attackCards, defenseCards []CombatCard,
	brought *BroughtUnit,
) *CardResolutionResult {
	result := &CardResolutionResult{
		FinalAttack:  baseAttack,
		FinalDefense: baseDefense,
	}

	// Build working copies of card lists (so we can remove negated cards)
	activeAttackCards := make([]CombatCard, len(attackCards))
	copy(activeAttackCards, attackCards)
	activeDefenseCards := make([]CombatCard, len(defenseCards))
	copy(activeDefenseCards, defenseCards)

	// --- Step 2: Auto-win check (Bribe) ---
	bribeIdx := -1
	for i, c := range activeDefenseCards {
		if c.Effect == EffectBribe {
			bribeIdx = i
			break
		}
	}
	if bribeIdx >= 0 {
		defender := g.Players[target.Owner]
		if defender != nil && defender.Stockpile.Gold >= BribeCost {
			defender.Stockpile.Gold -= BribeCost
			result.BribeActivated = true
			result.AttackerWins = false

			// Check for Safe Retreat even on bribe
			for _, c := range activeAttackCards {
				if c.Effect == EffectSafeRetreat {
					result.SafeRetreat = true
					break
				}
			}
			return result
		}
		// Bribe fizzles if can't afford -- card is still consumed (already removed from hand)
	}

	// --- Step 3: Negation effects ---

	// Shield Wall: negate one random attack card
	for _, c := range activeDefenseCards {
		if c.Effect == EffectShieldWall && len(activeAttackCards) > 0 {
			negIdx := rand.Intn(len(activeAttackCards))
			result.NegatedCards = append(result.NegatedCards, activeAttackCards[negIdx])
			activeAttackCards = append(activeAttackCards[:negIdx], activeAttackCards[negIdx+1:]...)
		}
	}

	// Sabotage: negate weapon contributions to attack strength
	for _, c := range activeDefenseCards {
		if c.Effect == EffectSabotage {
			// Count weapons that were contributing to attack (adjacent territories)
			weaponCount := 0
			for _, adjID := range target.Adjacent {
				adj := g.Territories[adjID]
				if adj.Owner == attackerID && adj.HasWeapon {
					weaponCount++
				}
			}
			// Also count brought weapon if applicable
			if brought != nil {
				if brought.UnitType == UnitWeapon && !g.IsAdjacent(brought.FromTerritory, target.ID) {
					weaponCount++
				}
				if brought.CarryingWeapon && !g.IsAdjacent(brought.WeaponFromTerritory, target.ID) {
					weaponCount++
				}
			}
			reduction := weaponCount * 3
			result.FinalAttack -= reduction
			result.SabotageCount = weaponCount
		}
	}

	// --- Step 4: Unit synergy ---
	for _, c := range activeAttackCards {
		switch c.Effect {
		case EffectRallyCavalry:
			// +1 per adjacent horse owned by attacker
			for _, adjID := range target.Adjacent {
				adj := g.Territories[adjID]
				if adj.Owner == attackerID && adj.HasHorse {
					result.FinalAttack++
				}
			}
		case EffectArsenal:
			// +1 per adjacent weapon owned by attacker
			for _, adjID := range target.Adjacent {
				adj := g.Territories[adjID]
				if adj.Owner == attackerID && adj.HasWeapon {
					result.FinalAttack++
				}
			}
		case EffectNavalBombardment:
			// +2 per boat on adjacent territories owned by attacker
			for _, adjID := range target.Adjacent {
				adj := g.Territories[adjID]
				if adj.Owner == attackerID {
					result.FinalAttack += adj.TotalBoats() * 2
				}
			}
		}
	}

	// --- Step 5: Multipliers (applied to base strength, before flat bonuses) ---
	attackMultiplier := 1
	defenseMultiplier := 1

	for _, c := range activeAttackCards {
		if c.Effect == EffectDoubleAttack {
			attackMultiplier = 2
		}
	}
	for _, c := range activeDefenseCards {
		if c.Effect == EffectDoubleDefense {
			defenseMultiplier = 2
		}
	}

	// Apply multipliers to the current strength (which is base + sabotage reduction + synergy)
	// We want to multiply only the base, so recalculate:
	// finalAttack = baseAttack * multiplier + (synergy bonuses) - (sabotage reduction)
	// Actually, per the plan: multipliers apply to base strength only, before flat bonuses.
	// So we need to track separately.

	// Recalculate: start from base, apply multiplier, then re-add synergy and sabotage
	synergyBonus := result.FinalAttack - baseAttack + (result.SabotageCount * 3) // undo sabotage to get synergy
	sabotageReduction := result.SabotageCount * 3

	result.FinalAttack = (baseAttack * attackMultiplier) + synergyBonus - sabotageReduction
	result.FinalDefense = baseDefense * defenseMultiplier

	// --- Step 6: Flat bonuses ---
	for _, c := range activeAttackCards {
		if c.Value > 0 {
			result.FinalAttack += c.Value
		}
	}
	for _, c := range activeDefenseCards {
		if c.Value > 0 {
			result.FinalDefense += c.Value
		}
	}

	// Ensure minimum of 0
	if result.FinalAttack < 0 {
		result.FinalAttack = 0
	}
	if result.FinalDefense < 0 {
		result.FinalDefense = 0
	}

	// --- Step 7: Final comparison using existing ChanceLevel formula ---
	result.AttackerWins = g.ResolveCombat(result.FinalAttack, result.FinalDefense)

	// --- Step 8: Post-combat effects ---

	// Safe Retreat: if attacker loses, brought unit is not destroyed
	if !result.AttackerWins {
		for _, c := range activeAttackCards {
			if c.Effect == EffectSafeRetreat {
				result.SafeRetreat = true
				break
			}
		}
	}

	// Blitz: if attacker wins, return all other played attack cards to hand
	if result.AttackerWins {
		for _, c := range attackCards {
			if c.Effect == EffectBlitz {
				// Return all attack cards EXCEPT Blitz itself
				for _, rc := range attackCards {
					if rc.Effect != EffectBlitz {
						result.BlitzReturn = append(result.BlitzReturn, rc)
					}
				}
				break
			}
		}
	}

	// Counter-Attack: if defender wins, capture a random attacker adjacent territory
	if !result.AttackerWins {
		for _, c := range defenseCards {
			if c.Effect == EffectCounterAttack {
				// Find attacker's territories adjacent to the target
				adjTerritories := make([]string, 0)
				for _, adjID := range target.Adjacent {
					adj := g.Territories[adjID]
					if adj.Owner == attackerID {
						adjTerritories = append(adjTerritories, adjID)
					}
				}
				if len(adjTerritories) > 0 {
					result.CounterAttackTerr = adjTerritories[rand.Intn(len(adjTerritories))]
				}
				break
			}
		}
	}

	return result
}

// CanBuyCard checks if a player can buy a card of the given type.
func (g *GameState) CanBuyCard(playerID string, cardType CardType, resource ResourceType) error {
	if g.Phase != PhaseDevelopment {
		return ErrInvalidAction
	}

	player := g.Players[playerID]
	if player == nil {
		return ErrInvalidTarget
	}

	if g.Settings.CombatMode != CombatModeCards {
		return ErrInvalidAction
	}

	// Check hand limit
	if cardType == CardTypeAttack && len(player.AttackCards) >= MaxAttackCards {
		return ErrHandFull
	}
	if cardType == CardTypeDefense && len(player.DefenseCards) >= MaxDefenseCards {
		return ErrHandFull
	}

	// Check resource (must be a stockpilable resource)
	if !resource.IsStockpilable() {
		return ErrInvalidTarget
	}

	// Check player has enough (2 of the chosen resource)
	if player.Stockpile.Get(resource) < CardBuyCost {
		return ErrInsufficientResources
	}

	return nil
}

// BuyCard purchases a random card during the Development phase.
// Returns the drawn card.
func (g *GameState) BuyCard(playerID string, cardType CardType, resource ResourceType) (*CombatCard, error) {
	if g.Phase != PhaseDevelopment {
		return nil, ErrInvalidAction
	}

	if g.CurrentPlayerID != playerID {
		return nil, ErrNotYourTurn
	}

	if err := g.CanBuyCard(playerID, cardType, resource); err != nil {
		return nil, err
	}

	player := g.Players[playerID]

	// Deduct resources
	player.Stockpile.Remove(resource, CardBuyCost)

	// Draw a random card
	card := DrawCard(cardType)

	// Add to hand
	if cardType == CardTypeAttack {
		player.AttackCards = append(player.AttackCards, card)
	} else {
		player.DefenseCards = append(player.DefenseCards, card)
	}

	return &card, nil
}

// RemoveCardFromHand removes a card by ID from the appropriate hand.
// Returns the removed card, or nil if not found.
func (p *Player) RemoveCardFromHand(cardID string) *CombatCard {
	for i, c := range p.AttackCards {
		if c.ID == cardID {
			removed := p.AttackCards[i]
			p.AttackCards = append(p.AttackCards[:i], p.AttackCards[i+1:]...)
			return &removed
		}
	}
	for i, c := range p.DefenseCards {
		if c.ID == cardID {
			removed := p.DefenseCards[i]
			p.DefenseCards = append(p.DefenseCards[:i], p.DefenseCards[i+1:]...)
			return &removed
		}
	}
	return nil
}

// RemoveCardsFromHand removes multiple cards by ID from the player's hand.
// Returns the removed cards.
func (p *Player) RemoveCardsFromHand(cardIDs []string) []CombatCard {
	removed := make([]CombatCard, 0, len(cardIDs))
	idSet := make(map[string]bool, len(cardIDs))
	for _, id := range cardIDs {
		idSet[id] = true
	}

	// Remove from attack cards
	newAttack := make([]CombatCard, 0, len(p.AttackCards))
	for _, c := range p.AttackCards {
		if idSet[c.ID] {
			removed = append(removed, c)
			delete(idSet, c.ID)
		} else {
			newAttack = append(newAttack, c)
		}
	}
	p.AttackCards = newAttack

	// Remove from defense cards
	newDefense := make([]CombatCard, 0, len(p.DefenseCards))
	for _, c := range p.DefenseCards {
		if idSet[c.ID] {
			removed = append(removed, c)
			delete(idSet, c.ID)
		} else {
			newDefense = append(newDefense, c)
		}
	}
	p.DefenseCards = newDefense

	return removed
}

// ReturnCardsToHand adds cards back to the player's hand (for Blitz effect).
func (p *Player) ReturnCardsToHand(cards []CombatCard) {
	for _, c := range cards {
		if c.CardType == CardTypeAttack && len(p.AttackCards) < MaxAttackCards {
			p.AttackCards = append(p.AttackCards, c)
		} else if c.CardType == CardTypeDefense && len(p.DefenseCards) < MaxDefenseCards {
			p.DefenseCards = append(p.DefenseCards, c)
		}
	}
}

// GetCardByID finds a card in the player's hand by ID.
func (p *Player) GetCardByID(cardID string) *CombatCard {
	for i := range p.AttackCards {
		if p.AttackCards[i].ID == cardID {
			return &p.AttackCards[i]
		}
	}
	for i := range p.DefenseCards {
		if p.DefenseCards[i].ID == cardID {
			return &p.DefenseCards[i]
		}
	}
	return nil
}
