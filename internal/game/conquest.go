package game

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

		// Skip adjacent territories (their strength is already counted)
		if g.isAdjacent(id, targetID) {
			continue
		}

		// Horses (2 movement range)
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

		// Boats (water movement)
		if t.Boats > 0 && t.IsCoastal() && g.canBoatReachTarget(id, targetID) {
			opt := ReinforcementOption{
				UnitType:      "boat",
				FromTerritory: id,
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

	return result
}

// canHorseReachTarget checks if a horse can reach a target territory.
func (g *GameState) canHorseReachTarget(playerID, fromID, targetID string) bool {
	from := g.Territories[fromID]

	// Direct adjacency to target
	if g.isAdjacent(fromID, targetID) {
		return true
	}

	// 2 moves through owned territory, ending adjacent to target
	for _, midID := range from.Adjacent {
		mid := g.Territories[midID]
		if mid.Owner == playerID {
			if g.isAdjacent(midID, targetID) {
				return true
			}
		}
	}

	return false
}

// canBoatReachTarget checks if a boat can reach a territory adjacent to target via water.
func (g *GameState) canBoatReachTarget(fromID, targetID string) bool {
	from := g.Territories[fromID]
	target := g.Territories[targetID]

	// Check if we can reach a territory adjacent to the target via water
	for _, waterID := range from.WaterBodies {
		water := g.WaterBodies[waterID]
		for _, coastalID := range water.Territories {
			if g.isAdjacent(coastalID, targetID) || coastalID == targetID {
				// Can reach adjacent territory via water
				return true
			}
			// Check if target has coastal tiles in this water body
			for _, tw := range target.WaterBodies {
				if tw == waterID {
					return true
				}
			}
		}
	}

	return false
}

// Attack executes an attack during the conquest phase.
func (g *GameState) Attack(attackerID, targetID string, brought *BroughtUnit) (*CombatResult, error) {
	// Validate phase
	if g.Phase != PhaseConquest {
		return nil, ErrInvalidAction
	}

	// Validate player turn
	if g.CurrentPlayerID != attackerID {
		return nil, ErrNotYourTurn
	}

	attacker := g.Players[attackerID]
	if attacker == nil {
		return nil, ErrInvalidTarget
	}

	if attacker.AttacksRemaining <= 0 {
		return nil, ErrNoAttacksRemaining
	}

	if !g.CanAttack(attackerID, targetID) {
		return nil, ErrInvalidTarget
	}

	// Execute the attack
	plan := &AttackPlan{
		TargetTerritory: targetID,
		BroughtUnit:     brought,
	}

	result := g.ExecuteAttack(attackerID, plan)

	// Decrease attacks remaining
	attacker.AttacksRemaining--

	// If first attack fails, end conquest for this player
	if !result.AttackerWins && attacker.AttacksRemaining == 1 {
		attacker.AttacksRemaining = 0
	}

	// If no attacks remaining, advance to next player
	if attacker.AttacksRemaining <= 0 {
		g.advanceConquestTurn()
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

	g.advanceConquestTurn()
	return nil
}

// advanceConquestTurn moves to the next player or next phase.
func (g *GameState) advanceConquestTurn() {
	currentIdx := -1
	for i, pid := range g.PlayerOrder {
		if pid == g.CurrentPlayerID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return
	}

	// Find next player with attacks remaining
	for i := 1; i <= len(g.PlayerOrder); i++ {
		nextIdx := (currentIdx + i) % len(g.PlayerOrder)
		nextPlayer := g.Players[g.PlayerOrder[nextIdx]]

		if nextPlayer.Eliminated {
			continue
		}

		if nextIdx == 0 {
			// Wrapped around - move to development phase
			g.Phase = PhaseDevelopment
			g.CurrentPlayerID = g.PlayerOrder[0]
			// Find first non-eliminated player
			for _, pid := range g.PlayerOrder {
				if !g.Players[pid].Eliminated {
					g.CurrentPlayerID = pid
					break
				}
			}
			return
		}

		// Check if this player still has attacks
		if nextPlayer.AttacksRemaining > 0 {
			g.CurrentPlayerID = g.PlayerOrder[nextIdx]
			return
		}
	}

	// No one has attacks remaining, move to development phase
	g.Phase = PhaseDevelopment
	for _, pid := range g.PlayerOrder {
		if !g.Players[pid].Eliminated {
			g.CurrentPlayerID = pid
			break
		}
	}
}

