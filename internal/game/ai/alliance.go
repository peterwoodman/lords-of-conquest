package ai

import "lords-of-conquest/internal/game"

// Side is a third party's choice in a battle. Values match the server's alliance votes.
type Side string

const (
	SideNeutral  Side = "neutral"
	SideAttacker Side = "attacker"
	SideDefender Side = "defender"
)

// VoteAlliance decides which side CPU voterID joins when attackerID attacks targetID.
//   - Easy stays neutral.
//   - Medium backs the defender against an attacker who leads in cities.
//   - Hard backs whoever keeps the leader from gaining, and always defends land next to
//     its own stockpile.
func VoteAlliance(s *game.GameState, voterID, attackerID, targetID string, d game.AIDifficulty) Side {
	target := s.Territories[targetID]
	voter := s.Players[voterID]
	if target == nil || voter == nil {
		return SideNeutral
	}
	defenderID := target.Owner

	switch game.ParseAIDifficulty(string(d)) {
	case game.AIEasy:
		return SideNeutral
	case game.AIMedium:
		if leadsInCities(s, attackerID) && s.CountCities(attackerID) > s.CountCities(defenderID) {
			return SideDefender
		}
		return SideNeutral
	default:
		if home := s.Territories[voter.StockpileTerritory]; home != nil && s.IsAdjacent(home.ID, targetID) {
			return SideDefender
		}
		switch leader(s, voterID) {
		case "":
			return SideNeutral
		case attackerID:
			return SideDefender
		case defenderID:
			return SideAttacker
		}
		return SideNeutral
	}
}

// leader returns the strongest opponent of voterID (most cities, then most territory),
// or "" if no opponent is ahead of everyone else.
func leader(s *game.GameState, voterID string) string {
	score := func(id string) int {
		return 100*s.CountCities(id) + len(s.GetPlayerTerritories(id))
	}
	best, bestScore, tied := "", -1, false
	for _, id := range sortedPlayerIDs(s) {
		if id == voterID || s.Players[id].Eliminated {
			continue
		}
		switch sc := score(id); {
		case sc > bestScore:
			best, bestScore, tied = id, sc, false
		case sc == bestScore:
			tied = true
		}
	}
	if tied {
		return ""
	}
	return best
}

// ResolveAllies sorts the third parties in an attack on target into attacker and defender
// allies: fixed alliance settings apply, and CPUs that would be asked vote immediately.
// Humans who would be asked are returned in ask for the caller to prompt (or ignore).
func ResolveAllies(s *game.GameState, attackerID string, target *game.Territory) (attackerAllies, defenderAllies, ask []string) {
	attackerAllies, defenderAllies, maybeAsk := s.SortAllies(attackerID, target)
	for _, id := range maybeAsk {
		p := s.Players[id]
		if p == nil || !p.IsAI {
			ask = append(ask, id)
			continue
		}
		switch VoteAlliance(s, id, attackerID, target.ID, p.AIDifficulty) {
		case SideAttacker:
			attackerAllies = append(attackerAllies, id)
		case SideDefender:
			defenderAllies = append(defenderAllies, id)
		}
	}
	return attackerAllies, defenderAllies, ask
}
