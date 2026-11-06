package ai

import (
	"fmt"
	"math/rand"

	"lords-of-conquest/internal/game"
)

// BuildDecision is one thing a CPU wants to build.
type BuildDecision struct {
	Type        game.BuildType
	Territory   string
	WaterBodyID string // boats only
	UseGold     bool
}

// minBuildScore is the least a build must be worth for Medium and Hard to make it.
const minBuildScore = 1.0

// goldSpendPenalty discourages paying gold (needed for cities) for weapons and boats.
const goldSpendPenalty = 1.5

// weaponAttackWeight scales the offensive worth of a weapon. Simulations show Hard wins
// more often when it arms its borders to take enemy cities.
var weaponAttackWeight = map[game.AIDifficulty]float64{
	game.AIEasy:   1,
	game.AIMedium: 1,
	game.AIHard:   2,
}

// PlanDevelopment picks the CPU's next build, or nil when it should stop building.
// Call it again after each build.
func PlanDevelopment(s *game.GameState, playerID string, d game.AIDifficulty, rng *rand.Rand) *BuildDecision {
	player := s.Players[playerID]
	if player == nil || player.Stockpile == nil {
		return nil
	}
	difficulty := game.ParseAIDifficulty(string(d))
	m := newDefenceModel(s, playerID)
	baseLoss := m.expectedLoss(player.StockpileTerritory)
	decisions := make(map[string]*BuildDecision)
	var opts []option
	add := func(dec *BuildDecision, score float64) {
		if dec.UseGold && dec.Type != game.BuildCity {
			score -= goldSpendPenalty
		}
		if difficulty != game.AIEasy && score < minBuildScore {
			return
		}
		key := fmt.Sprintf("%s:%s:%s", dec.Type, dec.Territory, dec.WaterBodyID)
		decisions[key] = dec
		opts = append(opts, option{id: key, score: score})
	}

	for _, id := range sortedTerritoryIDs(s) {
		t := s.Territories[id]
		if t.Owner != playerID {
			continue
		}
		if useGold, ok := payment(s, playerID, game.BuildCity, id); ok {
			add(&BuildDecision{Type: game.BuildCity, Territory: id, UseGold: useGold}, cityScore(s, m, playerID, t))
		}
		if useGold, ok := payment(s, playerID, game.BuildWeapon, id); ok {
			t.HasWeapon = true
			gain := baseLoss - m.expectedLoss(player.StockpileTerritory)
			t.HasWeapon = false
			add(&BuildDecision{Type: game.BuildWeapon, Territory: id, UseGold: useGold},
				gain+weaponAttackWeight[difficulty]*weaponAttackValue(s, playerID, t))
		}
		if useGold, ok := payment(s, playerID, game.BuildBoat, id); ok {
			for _, wb := range t.WaterBodies {
				t.AddBoat(wb)
				gain := baseLoss - m.expectedLoss(player.StockpileTerritory)
				t.RemoveBoat(wb)
				add(&BuildDecision{Type: game.BuildBoat, Territory: id, WaterBodyID: wb, UseGold: useGold},
					gain+boatAttackValue(s, playerID, t, wb))
			}
		}
	}

	key := choose(opts, d, rng)
	if key == "" {
		return nil
	}
	return decisions[key]
}

// payment reports whether playerID can build buildType on territoryID, and whether it
// must pay in gold. Resources are used whenever they suffice.
func payment(s *game.GameState, playerID string, buildType game.BuildType, territoryID string) (useGold, ok bool) {
	if s.CanBuild(playerID, buildType, territoryID, false) == nil {
		return false, true
	}
	if s.CanBuild(playerID, buildType, territoryID, true) == nil {
		return true, true
	}
	return false, false
}

// cityScore values a city on t: the production it doubles, progress toward victory,
// minus the risk of losing it.
func cityScore(s *game.GameState, m *defenceModel, playerID string, t *game.Territory) float64 {
	doubled := 0.0
	for _, n := range append(neighbours(s, t), t) {
		if n.Owner != playerID || n.Resource == game.ResourceNone || n.Resource == game.ResourceGrassland {
			continue
		}
		if !doubledByCity(s, playerID, n) {
			doubled += resourceValue(n.Resource)
		}
	}
	score := 4 + 1.5*doubled + 0.5*float64(ownedNeighbours(s, playerID, t))
	if s.CountCities(playerID)+1 >= s.Settings.VictoryCities {
		score += 20
	}
	return score - 6*m.lossChance(t)
}

// doubledByCity reports whether t's production is already doubled by a city on or next to it.
func doubledByCity(s *game.GameState, playerID string, t *game.Territory) bool {
	if t.HasCity {
		return true
	}
	for _, n := range neighbours(s, t) {
		if n.Owner == playerID && n.HasCity {
			return true
		}
	}
	return false
}

// weaponAttackValue is the offensive worth of a weapon on t: it adds 3 to attacks on
// each enemy neighbour. Scaled by weaponAttackWeight.
func weaponAttackValue(s *game.GameState, playerID string, t *game.Territory) float64 {
	return 0.75 * float64(min(enemyNeighbours(s, playerID, t), 3))
}

// boatAttackValue is the offensive worth of a boat on t in water body wb: it opens attacks
// on enemy coast that the player doesn't border.
func boatAttackValue(s *game.GameState, playerID string, t *game.Territory, wb string) float64 {
	water := s.WaterBodies[wb]
	if water == nil {
		return 0
	}
	targets := 0
	for _, id := range water.Territories {
		target := s.Territories[id]
		if target != nil && target.Owner != "" && target.Owner != playerID && ownedNeighbours(s, playerID, target) == 0 {
			targets++
		}
	}
	if targets == 0 {
		return 0
	}
	v := 1 + 0.25*float64(min(targets, 4))
	if !ownsBoatOn(s, playerID, wb) {
		v++
	}
	return v
}

// ownsBoatOn reports whether playerID has a boat on water body wb.
func ownsBoatOn(s *game.GameState, playerID, wb string) bool {
	for _, t := range s.Territories {
		if t.Owner == playerID && t.BoatsInWater(wb) > 0 {
			return true
		}
	}
	return false
}
