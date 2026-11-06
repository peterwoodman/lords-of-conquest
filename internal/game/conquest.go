package game

import "slices"

// CanAttack checks if a player can attack a territory.
func (g *GameState) CanAttack(attackerID, targetID string) bool {
	attacker := g.Players[attackerID]
	if attacker == nil || attacker.Eliminated || attacker.AttacksRemaining <= 0 {
		return false
	}

	target := g.Territories[targetID]
	if target == nil || target.Owner == attackerID {
		return false // Can't attack own territory
	}

	// Must have adjacent territory
	for _, adjID := range target.Adjacent {
		adj := g.Territories[adjID]
		if adj.Owner == attackerID {
			return true
		}
	}

	return false
}

// GetAttackableTargets returns territories the player can attack.
func (g *GameState) GetAttackableTargets(attackerID string) []string {
	targets := make([]string, 0)

	for id, t := range g.Territories {
		if t.Owner != attackerID && g.CanAttack(attackerID, id) {
			targets = append(targets, id)
		}
	}

	return targets
}

// PlanAttackResult contains information for planning an attack.
type PlanAttackResult struct {
	TargetID        string
	AttackStrength  int
	DefenseStrength int
	CanAttack       bool
	Reinforcements  []ReinforcementOption
}

// ReinforcementOption describes a unit that can be brought into battle.
type ReinforcementOption struct {
	UnitType      string
	FromTerritory string
	WaterBodyID   string // For boats: which water body the boat is in
	Strength      int
	CanCarry      []string // Types of units this can carry
}

// GetAttackPlan returns a preview of an attack.
func (g *GameState) GetAttackPlan(attackerID, targetID string) *PlanAttackResult {
	target := g.Territories[targetID]
	if target == nil {
		return nil
	}

	result := &PlanAttackResult{
		TargetID:        targetID,
		AttackStrength:  g.CalculateAttackStrength(attackerID, target, nil),
		DefenseStrength: g.CalculateDefenseStrength(target),
		CanAttack:       g.CanAttack(attackerID, targetID),
		Reinforcements:  make([]ReinforcementOption, 0),
	}

	// Find available reinforcements
	for id, t := range g.Territories {
		if t.Owner != attackerID {
			continue
		}

		isAdjacent := g.IsAdjacent(id, targetID)

		// Boats (water movement) - one option per water body with boats
		// Boats are ALWAYS listed as reinforcements (even from adjacent territories)
		// because they are never counted in base attack strength
		if t.TotalBoats() > 0 && t.IsCoastal() {
			for waterID, boatCount := range t.Boats {
				if boatCount > 0 && g.canBoatReachTargetViaWater(targetID, waterID) {
					opt := ReinforcementOption{
						UnitType:      "boat",
						FromTerritory: id,
						WaterBodyID:   waterID,
						Strength:      2,
						CanCarry:      []string{},
					}
					if t.HasHorse {
						opt.CanCarry = append(opt.CanCarry, "horse")
					}
					if t.HasWeapon {
						opt.CanCarry = append(opt.CanCarry, "weapon")
					}
					result.Reinforcements = append(result.Reinforcements, opt)
				}
			}
		}

		// Skip adjacent territories for horses - their strength is already counted in base attack
		if isAdjacent {
			continue
		}

		// Horses (2 movement range) - only from non-adjacent territories
		if t.HasHorse && g.canHorseReachTarget(attackerID, id, targetID) {
			opt := ReinforcementOption{
				UnitType:      "horse",
				FromTerritory: id,
				Strength:      1,
				CanCarry:      []string{},
			}
			if t.HasWeapon {
				opt.CanCarry = append(opt.CanCarry, "weapon")
				opt.Strength += 3 // If carrying weapon
			}
			result.Reinforcements = append(result.Reinforcements, opt)
		}
	}

	return result
}

// canHorseReachTarget checks if a horse can reach a target territory.
func (g *GameState) canHorseReachTarget(playerID, fromID, targetID string) bool {
	from := g.Territories[fromID]

	// Direct adjacency to target
	if g.IsAdjacent(fromID, targetID) {
		return true
	}

	// 2 moves through owned territory, ending adjacent to target
	for _, midID := range from.Adjacent {
		mid := g.Territories[midID]
		if mid.Owner == playerID {
			if g.IsAdjacent(midID, targetID) {
				return true
			}
		}
	}

	return false
}

// canBoatReachTargetViaWater checks if a boat in a specific water body can attack the target.
// Boats can only attack territories that border the same water body.
func (g *GameState) canBoatReachTargetViaWater(targetID, waterBodyID string) bool {
	target := g.Territories[targetID]
	if target == nil || g.WaterBodies[waterBodyID] == nil {
		return false
	}
	return slices.Contains(target.WaterBodies, waterBodyID)
}

// Attack executes an attack during the conquest phase.
func (g *GameState) Attack(attackerID, targetID string, brought *BroughtUnit) (*CombatResult, error) {
	return g.AttackWithAllies(attackerID, targetID, brought, nil, nil)
}

// ValidateAttack checks that it is attackerID's turn to attack targetID in the Conquest
// phase, and that any brought unit is one of the reinforcements GetAttackPlan offers.
func (g *GameState) ValidateAttack(attackerID, targetID string, brought *BroughtUnit) error {
	if g.Phase != PhaseConquest {
		return ErrInvalidAction
	}
	if g.CurrentPlayerID != attackerID {
		return ErrNotYourTurn
	}

	attacker := g.Players[attackerID]
	if attacker == nil || attacker.Eliminated {
		return ErrInvalidTarget
	}
	if attacker.AttacksRemaining <= 0 {
		return ErrNoAttacksRemaining
	}

	target := g.Territories[targetID]
	if target == nil || target.Owner == attackerID {
		return ErrInvalidTarget
	}

	if brought != nil {
		if !g.isOfferedReinforcement(attackerID, targetID, brought) {
			return ErrInvalidTarget
		}
		// An offered boat can reach the target even without an adjacent territory
		if brought.UnitType == UnitBoat {
			return nil
		}
	}

	if !g.CanAttack(attackerID, targetID) {
		return ErrInvalidTarget
	}
	return nil
}

// isOfferedReinforcement reports whether brought matches a reinforcement option for this attack,
// with any carried units coming from the same territory as the carrier.
func (g *GameState) isOfferedReinforcement(attackerID, targetID string, brought *BroughtUnit) bool {
	var unitType string
	switch brought.UnitType {
	case UnitHorse:
		unitType = "horse"
	case UnitBoat:
		unitType = "boat"
	default:
		return false
	}

	plan := g.GetAttackPlan(attackerID, targetID)
	if plan == nil {
		return false
	}
	for _, opt := range plan.Reinforcements {
		if opt.UnitType != unitType || opt.FromTerritory != brought.FromTerritory {
			continue
		}
		if unitType == "boat" && opt.WaterBodyID != brought.WaterBodyID {
			continue
		}
		if brought.CarryingWeapon && (brought.WeaponFromTerritory != brought.FromTerritory || !slices.Contains(opt.CanCarry, "weapon")) {
			return false
		}
		if brought.CarryingHorse && (brought.HorseFromTerritory != brought.FromTerritory || !slices.Contains(opt.CanCarry, "horse")) {
			return false
		}
		return true
	}
	return false
}

// AttackWithAllies executes an attack with ally support.
func (g *GameState) AttackWithAllies(attackerID, targetID string, brought *BroughtUnit, attackerAllies, defenderAllies []string) (*CombatResult, error) {
	if err := g.ValidateAttack(attackerID, targetID, brought); err != nil {
		return nil, err
	}
	attacker := g.Players[attackerID]

	// Execute the attack with allies
	plan := &AttackPlan{
		TargetTerritory: targetID,
		BroughtUnit:     brought,
	}

	result := g.ExecuteAttackWithAllies(attackerID, plan, attackerAllies, defenderAllies)

	// Decrease attacks remaining
	attacker.AttacksRemaining--

	// If first attack fails, end conquest for this player
	if !result.AttackerWins && attacker.AttacksRemaining == 1 {
		attacker.AttacksRemaining = 0
	}

	// If no attacks remaining, advance to next player
	if attacker.AttacksRemaining <= 0 && g.CurrentPlayerID == attackerID {
		g.AdvanceConquestTurn()
	}

	return result, nil
}

// EndConquest ends the conquest phase for a player.
func (g *GameState) EndConquest(playerID string) error {
	if g.Phase != PhaseConquest {
		return ErrInvalidAction
	}

	if g.CurrentPlayerID != playerID {
		return ErrNotYourTurn
	}

	player := g.Players[playerID]
	if player != nil {
		player.AttacksRemaining = 0
	}

	g.AdvanceConquestTurn()
	return nil
}

// AdvanceConquestTurn moves to the next player or next phase.
func (g *GameState) AdvanceConquestTurn() {
	currentIdx := -1
	for i, pid := range g.PlayerOrder {
		if pid == g.CurrentPlayerID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 || len(g.PlayerOrder) == 0 {
		return
	}

	// Find next player with attacks remaining
	for i := 1; i <= len(g.PlayerOrder); i++ {
		nextIdx := (currentIdx + i) % len(g.PlayerOrder)
		nextPlayer := g.Players[g.PlayerOrder[nextIdx]]

		// Safety check for nil player
		if nextPlayer == nil || nextPlayer.Eliminated {
			continue
		}

		if nextIdx == 0 {
			// Wrapped around - use NextPhase to properly transition
			// This increments the round and moves to Development
			pm := NewPhaseManager(g)
			pm.NextPhase()
			return
		}

		// Check if this player still has attacks
		if nextPlayer.AttacksRemaining > 0 {
			g.CurrentPlayerID = g.PlayerOrder[nextIdx]
			return
		}
	}

	// No one has attacks remaining - use NextPhase to properly transition
	pm := NewPhaseManager(g)
	pm.NextPhase()
}
