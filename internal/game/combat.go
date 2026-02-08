package game

import (
	"math/rand"
)

// CombatResult represents the outcome of a battle.
type CombatResult struct {
	AttackerWins      bool
	AttackStrength    int
	DefenseStrength   int
	UnitsDestroyed    []UnitInfo // Attacker's brought-in units if attack fails
	UnitsCaptured     []UnitInfo // Defender's units if attack succeeds
	StockpileCaptured *Stockpile // Defender's stockpile if captured
}

// UnitInfo describes a unit involved in combat.
type UnitInfo struct {
	Type        UnitType
	TerritoryID string
}

// UnitType represents a type of unit.
type UnitType int

const (
	UnitHorse UnitType = iota
	UnitWeapon
	UnitBoat
)

// AttackPlan represents a planned attack with optional reinforcements.
type AttackPlan struct {
	TargetTerritory string
	BroughtUnit     *BroughtUnit
}

// BroughtUnit represents a unit being brought into battle from elsewhere.
type BroughtUnit struct {
	UnitType            UnitType
	FromTerritory       string
	WaterBodyID         string // For boats: which water body the boat is in
	CarryingWeapon      bool
	WeaponFromTerritory string
	CarryingHorse       bool
	HorseFromTerritory  string
}

// CalculateAttackStrength calculates the attacker's combat strength.
func (g *GameState) CalculateAttackStrength(attackerID string, target *Territory, brought *BroughtUnit) int {
	strength := 0

	// Count adjacent territories owned by attacker
	for _, adjID := range target.Adjacent {
		adj := g.Territories[adjID]
		if adj.Owner == attackerID {
			strength++ // Territory contribution

			// Units in adjacent territories (except boats, which must be "brought")
			if adj.HasCity {
				strength += 2
			}
			if adj.HasWeapon {
				strength += 3
			}
			if adj.HasHorse {
				strength += 1
			}
		}
	}

	// Add strength from brought unit
	if brought != nil {
		fromAdj := g.IsAdjacent(brought.FromTerritory, target.ID)

		switch brought.UnitType {
		case UnitHorse:
			if !fromAdj {
				strength += 1
			}
			if brought.CarryingWeapon && !g.IsAdjacent(brought.WeaponFromTerritory, target.ID) {
				strength += 3
			}
		case UnitBoat:
			strength += 2 // Boats always add (they're not counted in adjacent)
			if brought.CarryingHorse {
				if !g.IsAdjacent(brought.HorseFromTerritory, target.ID) {
					strength += 1
				}
			}
			if brought.CarryingWeapon {
				if !g.IsAdjacent(brought.WeaponFromTerritory, target.ID) {
					strength += 3
				}
			}
		case UnitWeapon:
			if !fromAdj {
				strength += 3
			}
		}
	}

	return strength
}

// CalculateDefenseStrength calculates the defender's combat strength.
func (g *GameState) CalculateDefenseStrength(target *Territory) int {
	strength := 1 // The territory itself

	// Units in the territory
	if target.HasCity {
		strength += 2
	}
	if target.HasWeapon {
		strength += 3
	}
	if target.HasHorse {
		strength += 1
	}
	strength += target.TotalBoats() * 2

	// Adjacent territories owned by defender (only if territory has an owner)
	// Unclaimed territories don't get reinforcements from other unclaimed territories
	if target.Owner != "" {
		for _, adjID := range target.Adjacent {
			adj := g.Territories[adjID]
			if adj.Owner == target.Owner {
				strength++ // Territory contribution

				if adj.HasCity {
					strength += 2
				}
				if adj.HasWeapon {
					strength += 3
				}
				if adj.HasHorse {
					strength += 1
				}
			}
		}
	}

	return strength
}

// CalculateDefenseWithAllies adds allied player strength.
func (g *GameState) CalculateDefenseWithAllies(target *Territory, allyIDs []string) int {
	strength := g.CalculateDefenseStrength(target)

	// Add strength from allied territories
	for _, adjID := range target.Adjacent {
		adj := g.Territories[adjID]
		for _, allyID := range allyIDs {
			if adj.Owner == allyID {
				strength++ // Territory contribution
				if adj.HasCity {
					strength += 2
				}
				if adj.HasWeapon {
					strength += 3
				}
				if adj.HasHorse {
					strength += 1
				}
				break
			}
		}
	}

	return strength
}

// CalculateAttackWithAllies adds allied player strength to attack.
func (g *GameState) CalculateAttackWithAllies(attackerID string, target *Territory, brought *BroughtUnit, allyIDs []string) int {
	strength := g.CalculateAttackStrength(attackerID, target, brought)

	// Add strength from allied territories
	for _, adjID := range target.Adjacent {
		adj := g.Territories[adjID]
		for _, allyID := range allyIDs {
			if adj.Owner == allyID {
				strength++ // Territory contribution
				if adj.HasCity {
					strength += 2
				}
				if adj.HasWeapon {
					strength += 3
				}
				if adj.HasHorse {
					strength += 1
				}
				break
			}
		}
	}

	return strength
}

// GetThirdPartyPlayers returns players who are adjacent to a territory but not attacker/defender.
func (g *GameState) GetThirdPartyPlayers(attackerID string, target *Territory) []string {
	seen := make(map[string]bool)
	thirdParties := make([]string, 0)

	for _, adjID := range target.Adjacent {
		adj := g.Territories[adjID]
		ownerID := adj.Owner
		
		// Skip if no owner, or if it's attacker or defender
		if ownerID == "" || ownerID == attackerID || ownerID == target.Owner {
			continue
		}
		
		// Skip if already seen
		if seen[ownerID] {
			continue
		}
		seen[ownerID] = true
		thirdParties = append(thirdParties, ownerID)
	}

	return thirdParties
}

// CalculatePlayerStrengthAtTerritory calculates how much strength a player contributes adjacent to a territory.
func (g *GameState) CalculatePlayerStrengthAtTerritory(playerID string, target *Territory) int {
	strength := 0

	for _, adjID := range target.Adjacent {
		adj := g.Territories[adjID]
		if adj.Owner == playerID {
			strength++ // Territory contribution
			if adj.HasCity {
				strength += 2
			}
			if adj.HasWeapon {
				strength += 3
			}
			if adj.HasHorse {
				strength += 1
			}
		}
	}

	return strength
}

// ResolveCombat determines the outcome of a battle.
func (g *GameState) ResolveCombat(attack, defense int) bool {
	switch g.Settings.ChanceLevel {
	case ChanceLow:
		// Attacker wins ties
		return attack >= defense
	case ChanceMedium:
		// Defender wins ties, random if equal
		if attack > defense {
			return true
		}
		if attack == defense {
			return rand.Float32() < 0.5
		}
		return false
	case ChanceHigh:
		// Probability based on strength ratio
		if attack == 0 && defense == 0 {
			return rand.Float32() < 0.5
		}
		ratio := float32(attack) / float32(attack+defense)
		return rand.Float32() < ratio
	default:
		return attack >= defense
	}
}

// ExecuteAttack performs an attack and returns the result.
func (g *GameState) ExecuteAttack(attackerID string, plan *AttackPlan) *CombatResult {
	return g.ExecuteAttackWithAllies(attackerID, plan, nil, nil)
}

// ExecuteAttackWithAllies performs an attack with ally support.
func (g *GameState) ExecuteAttackWithAllies(attackerID string, plan *AttackPlan, attackerAllies, defenderAllies []string) *CombatResult {
	target := g.Territories[plan.TargetTerritory]
	defenderID := target.Owner

	// Calculate strength with allies
	attackStrength := g.CalculateAttackWithAllies(attackerID, target, plan.BroughtUnit, attackerAllies)
	defenseStrength := g.CalculateDefenseWithAllies(target, defenderAllies)

	result := &CombatResult{
		AttackStrength:  attackStrength,
		DefenseStrength: defenseStrength,
		UnitsDestroyed:  []UnitInfo{},
		UnitsCaptured:   []UnitInfo{},
	}

	if g.ResolveCombat(attackStrength, defenseStrength) {
		result.AttackerWins = true

		// Transfer territory
		target.Owner = attackerID

		// Capture units (if attacker didn't bring their own)
		if plan.BroughtUnit == nil || plan.BroughtUnit.UnitType != UnitHorse {
			if target.HasHorse {
				result.UnitsCaptured = append(result.UnitsCaptured, UnitInfo{UnitHorse, target.ID})
			}
		}
		if plan.BroughtUnit == nil || plan.BroughtUnit.UnitType != UnitWeapon {
			if target.HasWeapon {
				result.UnitsCaptured = append(result.UnitsCaptured, UnitInfo{UnitWeapon, target.ID})
			}
		}

		// Move brought unit into territory
		if plan.BroughtUnit != nil {
			g.moveBroughtUnit(plan.BroughtUnit, target)
		}

		// Check for stockpile capture
		defender := g.Players[defenderID]
		if defender != nil && defender.StockpileTerritory == target.ID {
			result.StockpileCaptured = defender.Stockpile.Clone()
			attacker := g.Players[attackerID]
			// Transfer resources
			attacker.Stockpile.Coal += defender.Stockpile.Coal
			attacker.Stockpile.Gold += defender.Stockpile.Gold
			attacker.Stockpile.Iron += defender.Stockpile.Iron
			attacker.Stockpile.Timber += defender.Stockpile.Timber
			// Clear defender's stockpile
			defender.Stockpile = NewStockpile()
			defender.StockpileTerritory = ""
		}

		// Check if defender is eliminated
		g.checkElimination(defenderID)

	} else {
		result.AttackerWins = false

		// Destroy brought units
		if plan.BroughtUnit != nil {
			result.UnitsDestroyed = append(result.UnitsDestroyed,
				UnitInfo{plan.BroughtUnit.UnitType, plan.BroughtUnit.FromTerritory})

			// Remove the unit
			from := g.Territories[plan.BroughtUnit.FromTerritory]
			switch plan.BroughtUnit.UnitType {
			case UnitHorse:
				from.HasHorse = false
			case UnitWeapon:
				from.HasWeapon = false
			case UnitBoat:
				from.RemoveBoat(plan.BroughtUnit.WaterBodyID)
			}

			// Destroy carried units too
			if plan.BroughtUnit.CarryingWeapon {
				weaponFrom := g.Territories[plan.BroughtUnit.WeaponFromTerritory]
				weaponFrom.HasWeapon = false
				result.UnitsDestroyed = append(result.UnitsDestroyed,
					UnitInfo{UnitWeapon, plan.BroughtUnit.WeaponFromTerritory})
			}
			if plan.BroughtUnit.CarryingHorse {
				horseFrom := g.Territories[plan.BroughtUnit.HorseFromTerritory]
				horseFrom.HasHorse = false
				result.UnitsDestroyed = append(result.UnitsDestroyed,
					UnitInfo{UnitHorse, plan.BroughtUnit.HorseFromTerritory})
			}
		}
	}

	return result
}

// ExecuteCardAttackWithAllies performs an attack using card combat resolution.
// attackCards and defenseCards are the cards each player committed (already removed from hands).
func (g *GameState) ExecuteCardAttackWithAllies(attackerID string, plan *AttackPlan, attackerAllies, defenderAllies []string, attackCards, defenseCards []CombatCard) (*CombatResult, *CardResolutionResult) {
	target := g.Territories[plan.TargetTerritory]
	defenderID := target.Owner

	// Calculate base strength with allies (same as classic)
	baseAttack := g.CalculateAttackWithAllies(attackerID, target, plan.BroughtUnit, attackerAllies)
	baseDefense := g.CalculateDefenseWithAllies(target, defenderAllies)

	// Resolve cards
	cardResult := g.ResolveCards(attackerID, target, baseAttack, baseDefense, attackCards, defenseCards, plan.BroughtUnit)

	result := &CombatResult{
		AttackStrength:  cardResult.FinalAttack,
		DefenseStrength: cardResult.FinalDefense,
		AttackerWins:    cardResult.AttackerWins,
		UnitsDestroyed:  []UnitInfo{},
		UnitsCaptured:   []UnitInfo{},
	}

	if cardResult.AttackerWins {
		// Transfer territory
		target.Owner = attackerID

		// Capture units (same as classic)
		if plan.BroughtUnit == nil || plan.BroughtUnit.UnitType != UnitHorse {
			if target.HasHorse {
				result.UnitsCaptured = append(result.UnitsCaptured, UnitInfo{UnitHorse, target.ID})
			}
		}
		if plan.BroughtUnit == nil || plan.BroughtUnit.UnitType != UnitWeapon {
			if target.HasWeapon {
				result.UnitsCaptured = append(result.UnitsCaptured, UnitInfo{UnitWeapon, target.ID})
			}
		}

		// Move brought unit into territory
		if plan.BroughtUnit != nil {
			g.moveBroughtUnit(plan.BroughtUnit, target)
		}

		// Check for stockpile capture
		defender := g.Players[defenderID]
		if defender != nil && defender.StockpileTerritory == target.ID {
			result.StockpileCaptured = defender.Stockpile.Clone()
			attacker := g.Players[attackerID]
			attacker.Stockpile.Coal += defender.Stockpile.Coal
			attacker.Stockpile.Gold += defender.Stockpile.Gold
			attacker.Stockpile.Iron += defender.Stockpile.Iron
			attacker.Stockpile.Timber += defender.Stockpile.Timber
			defender.Stockpile = NewStockpile()
			defender.StockpileTerritory = ""
		}

		// Blitz: return attack cards to attacker's hand
		attacker := g.Players[attackerID]
		if attacker != nil && len(cardResult.BlitzReturn) > 0 {
			attacker.ReturnCardsToHand(cardResult.BlitzReturn)
		}

		// Check if defender is eliminated
		g.checkElimination(defenderID)

	} else {
		// Attack failed
		if plan.BroughtUnit != nil {
			if cardResult.SafeRetreat {
				// Safe Retreat: brought unit is NOT destroyed, stays in origin territory
				// (no action needed - unit is still where it was)
			} else {
				// Destroy brought units (same as classic)
				result.UnitsDestroyed = append(result.UnitsDestroyed,
					UnitInfo{plan.BroughtUnit.UnitType, plan.BroughtUnit.FromTerritory})

				from := g.Territories[plan.BroughtUnit.FromTerritory]
				switch plan.BroughtUnit.UnitType {
				case UnitHorse:
					from.HasHorse = false
				case UnitWeapon:
					from.HasWeapon = false
				case UnitBoat:
					from.RemoveBoat(plan.BroughtUnit.WaterBodyID)
				}

				if plan.BroughtUnit.CarryingWeapon {
					weaponFrom := g.Territories[plan.BroughtUnit.WeaponFromTerritory]
					weaponFrom.HasWeapon = false
					result.UnitsDestroyed = append(result.UnitsDestroyed,
						UnitInfo{UnitWeapon, plan.BroughtUnit.WeaponFromTerritory})
				}
				if plan.BroughtUnit.CarryingHorse {
					horseFrom := g.Territories[plan.BroughtUnit.HorseFromTerritory]
					horseFrom.HasHorse = false
					result.UnitsDestroyed = append(result.UnitsDestroyed,
						UnitInfo{UnitHorse, plan.BroughtUnit.HorseFromTerritory})
				}
			}
		}

		// Counter-Attack: defender captures a random attacker adjacent territory
		if cardResult.CounterAttackTerr != "" {
			counterTerr := g.Territories[cardResult.CounterAttackTerr]
			if counterTerr != nil && counterTerr.Owner == attackerID {
				counterTerr.Owner = defenderID
				// Check if attacker is now eliminated
				g.checkElimination(attackerID)
			}
		}
	}

	return result, cardResult
}

// IsAdjacent checks if two territories are adjacent.
func (g *GameState) IsAdjacent(id1, id2 string) bool {
	t1 := g.Territories[id1]
	if t1 == nil {
		return false
	}
	for _, adj := range t1.Adjacent {
		if adj == id2 {
			return true
		}
	}
	return false
}

// moveBroughtUnit moves a brought unit into the captured territory.
func (g *GameState) moveBroughtUnit(brought *BroughtUnit, target *Territory) {
	from := g.Territories[brought.FromTerritory]

	switch brought.UnitType {
	case UnitHorse:
		from.HasHorse = false
		target.HasHorse = true
		if brought.CarryingWeapon {
			weaponFrom := g.Territories[brought.WeaponFromTerritory]
			weaponFrom.HasWeapon = false
			target.HasWeapon = true
		}
	case UnitWeapon:
		from.HasWeapon = false
		target.HasWeapon = true
	case UnitBoat:
		// Boat stays in the same water body
		from.RemoveBoat(brought.WaterBodyID)
		target.AddBoat(brought.WaterBodyID)
		if brought.CarryingHorse {
			horseFrom := g.Territories[brought.HorseFromTerritory]
			horseFrom.HasHorse = false
			target.HasHorse = true
		}
		if brought.CarryingWeapon {
			weaponFrom := g.Territories[brought.WeaponFromTerritory]
			weaponFrom.HasWeapon = false
			target.HasWeapon = true
		}
	}
}

// checkElimination checks if a player has been eliminated.
func (g *GameState) checkElimination(playerID string) {
	for _, t := range g.Territories {
		if t.Owner == playerID {
			return // Still has territories
		}
	}
	if player := g.Players[playerID]; player != nil {
		player.Eliminated = true
	}
}
