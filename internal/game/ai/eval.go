package ai

import (
	"slices"
	"sort"

	"lords-of-conquest/internal/game"
)

// sortedTerritoryIDs returns all territory IDs in a stable order.
func sortedTerritoryIDs(s *game.GameState) []string {
	ids := make([]string, 0, len(s.Territories))
	for id := range s.Territories {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// resourceValue ranks resources for CPU decisions: gold first, grassland (horses)
// second, everything else third.
func resourceValue(r game.ResourceType) float64 {
	switch r {
	case game.ResourceGold:
		return 5
	case game.ResourceGrassland:
		return 3
	case game.ResourceCoal, game.ResourceIron, game.ResourceTimber:
		return 2
	default:
		return 0
	}
}

// neighbours returns the territories adjacent to t that exist in the state.
func neighbours(s *game.GameState, t *game.Territory) []*game.Territory {
	result := make([]*game.Territory, 0, len(t.Adjacent))
	for _, id := range t.Adjacent {
		if n := s.Territories[id]; n != nil {
			result = append(result, n)
		}
	}
	return result
}

// ownedNeighbours counts neighbours of t owned by playerID.
func ownedNeighbours(s *game.GameState, playerID string, t *game.Territory) int {
	count := 0
	for _, n := range neighbours(s, t) {
		if n.Owner == playerID {
			count++
		}
	}
	return count
}

// enemyNeighbours counts neighbours of t owned by a player other than playerID.
func enemyNeighbours(s *game.GameState, playerID string, t *game.Territory) int {
	count := 0
	for _, n := range neighbours(s, t) {
		if n.Owner != "" && n.Owner != playerID {
			count++
		}
	}
	return count
}

// unclaimedNeighbours counts neighbours of t that nobody owns.
func unclaimedNeighbours(s *game.GameState, t *game.Territory) int {
	count := 0
	for _, n := range neighbours(s, t) {
		if n.Owner == "" {
			count++
		}
	}
	return count
}

// clusters labels each territory owned by playerID with the index of its connected
// region of owned territory, and returns the size of each region.
func clusters(s *game.GameState, playerID string) (map[string]int, []int) {
	label := make(map[string]int)
	var sizes []int
	for _, id := range sortedTerritoryIDs(s) {
		if s.Territories[id].Owner != playerID {
			continue
		}
		if _, seen := label[id]; seen {
			continue
		}
		idx := len(sizes)
		size := 0
		stack := []string{id}
		label[id] = idx
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++
			for _, n := range neighbours(s, s.Territories[cur]) {
				if _, seen := label[n.ID]; !seen && n.Owner == playerID {
					label[n.ID] = idx
					stack = append(stack, n.ID)
				}
			}
		}
		sizes = append(sizes, size)
	}
	return label, sizes
}

// winProbability estimates the chance an attack succeeds under the game's chance level,
// mirroring GameState.ResolveCombat.
func winProbability(s *game.GameState, attack, defense int) float64 {
	switch s.Settings.ChanceLevel {
	case game.ChanceMedium:
		switch {
		case attack > defense:
			return 1
		case attack == defense:
			return 0.5
		default:
			return 0
		}
	case game.ChanceHigh:
		if attack+defense == 0 {
			return 0.5
		}
		return float64(attack) / float64(attack+defense)
	default:
		if attack >= defense {
			return 1
		}
		return 0
	}
}

// bestReinforcement returns the most strength attackerID could add to an attack on
// target by bringing one unit, using the same options the attack planner offers.
func bestReinforcement(s *game.GameState, attackerID string, target *game.Territory) int {
	plan := s.GetAttackPlan(attackerID, target.ID)
	if plan == nil {
		return 0
	}
	best := 0
	for _, r := range plan.Reinforcements {
		strength := r.Strength
		if r.UnitType == "boat" && !s.IsAdjacent(r.FromTerritory, target.ID) {
			if slices.Contains(r.CanCarry, "weapon") {
				strength += 3
			} else if slices.Contains(r.CanCarry, "horse") {
				strength++
			}
		}
		best = max(best, strength)
	}
	return best
}

// threatTo returns the strongest attack any single opponent of playerID could make on t,
// including one brought unit. Opponents can only attack from an adjacent territory.
func threatTo(s *game.GameState, playerID string, t *game.Territory) int {
	best := 0
	for _, oppID := range sortedPlayerIDs(s) {
		opp := s.Players[oppID]
		if oppID == playerID || opp.Eliminated || ownedNeighbours(s, oppID, t) == 0 {
			continue
		}
		attack := s.CalculateAttackStrength(oppID, t, nil) + bestReinforcement(s, oppID, t)
		best = max(best, attack)
	}
	return best
}

// sortedPlayerIDs returns all player IDs in a stable order.
func sortedPlayerIDs(s *game.GameState) []string {
	ids := make([]string, 0, len(s.Players))
	for id := range s.Players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ownsCoast reports whether playerID owns any coastal territory.
func ownsCoast(s *game.GameState, playerID string) bool {
	for _, t := range s.Territories {
		if t.Owner == playerID && t.IsCoastal() {
			return true
		}
	}
	return false
}
