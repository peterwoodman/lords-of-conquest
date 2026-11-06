package ai

import (
	"fmt"
	"math/rand"
	"slices"

	"lords-of-conquest/internal/game"
)

// AttackDecision is an attack a CPU has chosen to make.
type AttackDecision struct {
	Target         string
	Brought        *game.BroughtUnit // nil when attacking without bringing a unit
	AttackerAllies []string
	DefenderAllies []string
	AttackCards    []string // card IDs to play (card combat only)
	WinChance      float64
}

// attackProfile tunes how willing a difficulty is to attack.
type attackProfile struct {
	minWinChance     float64 // ordinary targets
	minHighValue     float64 // stockpiles and cities
	minCritical      float64 // stockpile or city of a player one city from winning
	useReinforcement bool
	protectStockpile bool // never bring a unit whose loss endangers our stockpile
}

var attackProfiles = map[game.AIDifficulty]attackProfile{
	game.AIEasy:   {minWinChance: 0.75, minHighValue: 0.75, minCritical: 0.75},
	game.AIMedium: {minWinChance: 0.6, minHighValue: 0.6, minCritical: 0.5, useReinforcement: true, protectStockpile: true},
	game.AIHard:   {minWinChance: 0.65, minHighValue: 0.5, minCritical: 0.4, useReinforcement: true, protectStockpile: true},
}

// highValueTarget is the capture value at or above which minHighValue applies.
const highValueTarget = 9

// PlanAttack chooses the CPU's next attack, or returns nil to end its conquest turn.
// Allies come from fixed alliance settings and CPU votes; humans who would have to be
// asked are treated as neutral.
func PlanAttack(s *game.GameState, playerID string, d game.AIDifficulty, rng *rand.Rand) *AttackDecision {
	player := s.Players[playerID]
	if player == nil || player.Eliminated || player.AttacksRemaining <= 0 {
		return nil
	}
	profile := attackProfiles[game.ParseAIDifficulty(string(d))]
	defence := newDefenceModel(s, playerID)
	cardMode := s.Settings.CombatMode == game.CombatModeCards

	decisions := make(map[string]*AttackDecision)
	var opts []option
	for _, targetID := range attackCandidates(s, playerID) {
		target := s.Territories[targetID]
		attackerAllies, defenderAllies, _ := ResolveAllies(s, playerID, target)
		value := captureValue(s, playerID, target)

		minChance := profile.minWinChance
		switch {
		case isCritical(s, target):
			minChance = profile.minCritical
		case value >= highValueTarget:
			minChance = profile.minHighValue
		}

		broughtOptions := []*game.BroughtUnit{nil}
		if profile.useReinforcement {
			broughtOptions = append(broughtOptions, reinforcements(s, playerID, target)...)
		}
		for _, brought := range broughtOptions {
			if s.ValidateAttack(playerID, targetID, brought) != nil {
				continue
			}
			attack := s.CalculateAttackWithAllies(playerID, target, brought, attackerAllies)
			defense := s.CalculateDefenseWithAllies(target, defenderAllies)
			var cards []string
			if cardMode {
				defense += estimatedDefenseCards(s, target, d)
				cards, attack = chooseAttackCards(s, player, target, brought, attack, defense, minChance, d)
			}
			chance := winProbability(s, attack, defense)
			if chance < minChance {
				continue
			}
			if brought != nil && profile.protectStockpile && endangersStockpile(s, defence, player, brought) {
				continue
			}
			risk := broughtValue(s, brought)
			// Losing the first attack also forfeits the second
			if player.AttacksRemaining == 2 {
				risk++
			}
			score := chance*value - (1-chance)*risk
			if brought != nil {
				score -= 0.01 // prefer keeping units home when it makes no difference
			}
			score -= cardSpendCost * float64(len(cards))
			key := fmt.Sprintf("%s:%d", targetID, len(opts))
			decisions[key] = &AttackDecision{
				Target:         targetID,
				Brought:        brought,
				AttackerAllies: attackerAllies,
				DefenderAllies: defenderAllies,
				AttackCards:    cards,
				WinChance:      chance,
			}
			opts = append(opts, option{id: key, score: score})
		}
	}

	key := choose(opts, d, rng)
	if key == "" {
		return nil
	}
	return decisions[key]
}

// attackCandidates lists territories playerID could attack: those adjacent to its land,
// plus coastal territories its boats can reach.
func attackCandidates(s *game.GameState, playerID string) []string {
	var result []string
	for _, id := range sortedTerritoryIDs(s) {
		t := s.Territories[id]
		if t.Owner == playerID {
			continue
		}
		if ownedNeighbours(s, playerID, t) > 0 || boatCanReach(s, playerID, t) {
			result = append(result, id)
		}
	}
	return result
}

// boatCanReach reports whether playerID has a boat on a water body that t borders.
func boatCanReach(s *game.GameState, playerID string, t *game.Territory) bool {
	for _, wb := range t.WaterBodies {
		water := s.WaterBodies[wb]
		if water == nil {
			continue
		}
		for _, id := range water.Territories {
			if from := s.Territories[id]; from != nil && from.Owner == playerID && from.BoatsInWater(wb) > 0 {
				return true
			}
		}
	}
	return false
}

// reinforcements turns the attack planner's options into units to bring, carrying
// whatever adds strength.
func reinforcements(s *game.GameState, playerID string, target *game.Territory) []*game.BroughtUnit {
	plan := s.GetAttackPlan(playerID, target.ID)
	if plan == nil {
		return nil
	}
	var result []*game.BroughtUnit
	for _, r := range plan.Reinforcements {
		b := &game.BroughtUnit{FromTerritory: r.FromTerritory, WaterBodyID: r.WaterBodyID}
		fromAdjacent := s.IsAdjacent(r.FromTerritory, target.ID)
		switch r.UnitType {
		case "horse":
			b.UnitType = game.UnitHorse
			if slices.Contains(r.CanCarry, "weapon") {
				b.CarryingWeapon, b.WeaponFromTerritory = true, r.FromTerritory
			}
		case "boat":
			b.UnitType = game.UnitBoat
			if !fromAdjacent && slices.Contains(r.CanCarry, "weapon") {
				b.CarryingWeapon, b.WeaponFromTerritory = true, r.FromTerritory
			}
			if !fromAdjacent && slices.Contains(r.CanCarry, "horse") {
				b.CarryingHorse, b.HorseFromTerritory = true, r.FromTerritory
			}
		default:
			continue
		}
		result = append(result, b)
	}
	return result
}

// captureValue is what playerID gains by taking target.
func captureValue(s *game.GameState, playerID string, target *game.Territory) float64 {
	v := territoryValue(target) + 2*float64(ownedNeighbours(s, playerID, target))
	owner := s.Players[target.Owner]
	if owner == nil {
		return v
	}
	if owner.StockpileTerritory == target.ID {
		v += stockpileValue
	}
	if isCritical(s, target) {
		v += 6
	}
	if len(s.GetPlayerTerritories(owner.ID)) == 1 {
		v += 3 // eliminates the owner
	}
	if target.HasCity && leadsInCities(s, owner.ID) {
		v += 2
	}
	return v
}

// isCritical reports whether target is the stockpile or a city of a player one city
// short of winning.
func isCritical(s *game.GameState, target *game.Territory) bool {
	owner := s.Players[target.Owner]
	if owner == nil || s.CountCities(owner.ID) < s.Settings.VictoryCities-1 {
		return false
	}
	return target.HasCity || owner.StockpileTerritory == target.ID
}

// leadsInCities reports whether playerID has at least one city and no fewer than anyone else.
func leadsInCities(s *game.GameState, playerID string) bool {
	mine := s.CountCities(playerID)
	if mine == 0 {
		return false
	}
	for id, p := range s.Players {
		if id != playerID && !p.Eliminated && s.CountCities(id) > mine {
			return false
		}
	}
	return true
}

// broughtValue is what is lost if a brought unit is destroyed.
func broughtValue(s *game.GameState, b *game.BroughtUnit) float64 {
	if b == nil {
		return 0
	}
	v := 0.0
	switch b.UnitType {
	case game.UnitHorse:
		v = 1
	case game.UnitBoat:
		v = 2
	}
	if b.CarryingWeapon {
		v += 2
	}
	if b.CarryingHorse {
		v++
	}
	return v
}

// endangersStockpile reports whether losing the brought unit would make the player's
// stockpile territory more likely to fall.
func endangersStockpile(s *game.GameState, m *defenceModel, player *game.Player, b *game.BroughtUnit) bool {
	home := s.Territories[player.StockpileTerritory]
	from := s.Territories[b.FromTerritory]
	if home == nil || from == nil {
		return false
	}
	before := m.lossChance(home)

	hadHorse, hadWeapon := from.HasHorse, from.HasWeapon
	if b.UnitType == game.UnitHorse || b.CarryingHorse {
		from.HasHorse = false
	}
	if b.CarryingWeapon {
		from.HasWeapon = false
	}
	if b.UnitType == game.UnitBoat {
		from.RemoveBoat(b.WaterBodyID)
	}
	after := m.lossChance(home)
	from.HasHorse, from.HasWeapon = hadHorse, hadWeapon
	if b.UnitType == game.UnitBoat {
		from.AddBoat(b.WaterBodyID)
	}
	return after > before
}
